package cfg

import (
	"log"

	"gorm.io/gorm"
)

// ApplyRawTenantMigrations runs the handful of schema objects gorm's
// AutoMigrate can't express, against a tx already pinned to the tenant
// schema's search_path. Every statement is BEST-EFFORT: a failure is logged
// and swallowed so a missing privilege (e.g. CREATE EXTENSION on managed
// Postgres) never wedges tenant provisioning.
//
// The application-level guard in booking/repo (pg_advisory_xact_lock + explicit
// buffer-aware overlap check inside the write tx) is the AUTHORITATIVE
// double-booking protection. The EXCLUDE constraint here is defense in depth:
// it covers the raw [start_at, end_at) overlap only, because an index/EXCLUDE
// expression must be IMMUTABLE and "timestamptz + interval" is merely STABLE
// (it can't appear in the constraint). The post-service buffer therefore stays
// a concern of the app-level guard alone.
//
// Each statement runs inside its own SAVEPOINT: Postgres aborts the *whole*
// transaction on any error, so without a savepoint a single failing best-effort
// statement (e.g. CREATE EXTENSION denied) would poison the outer AutoMigrate
// tx and roll back every tenant table at COMMIT ("commit unexpectedly resulted
// in rollback"). Rolling back to the savepoint contains the damage.
func ApplyRawTenantMigrations(tx *gorm.DB) error {
	stmts := []struct {
		desc string
		sql  string
	}{
		{
			desc: "btree_gist extension",
			sql:  `CREATE EXTENSION IF NOT EXISTS btree_gist`,
		},
		{
			desc: "bookings no-overlap EXCLUDE constraint",
			sql: `DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'bookings_no_overlap') THEN
		ALTER TABLE bookings ADD CONSTRAINT bookings_no_overlap
			EXCLUDE USING gist (
				master_id WITH =,
				tstzrange(start_at, end_at) WITH &&
			)
			WHERE (status IN (0, 1, 2) AND master_id IS NOT NULL);
	END IF;
END $$;`,
		},
	}

	for _, s := range stmts {
		if err := tx.Exec(`SAVEPOINT raw_tenant_mig`).Error; err != nil {
			// Can't even open a savepoint — the tx is already unusable; let the
			// caller see the real failure rather than masking it.
			return err
		}
		if err := tx.Exec(s.sql).Error; err != nil {
			log.Printf("[ApplyRawTenantMigrations] skipped %q: %v", s.desc, err)
			if rbErr := tx.Exec(`ROLLBACK TO SAVEPOINT raw_tenant_mig`).Error; rbErr != nil {
				return rbErr
			}
			continue
		}
		if err := tx.Exec(`RELEASE SAVEPOINT raw_tenant_mig`).Error; err != nil {
			return err
		}
	}
	return nil
}
