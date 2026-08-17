package cfg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// ComputeMigrationTargetVersion derives a deterministic version string from
// the set of AutoMigrate models (field names, types, and gorm tags). Any real
// schema change on any registered model changes the hash, so the migration
// gate in MigrateTenantWithControl/IsTenantReady detects drift on its own --
// nobody has to remember to bump a hand-maintained version string.
func ComputeMigrationTargetVersion(models []any) string {
	signatures := make([]string, 0, len(models))
	for _, m := range models {
		signatures = append(signatures, modelSignature(m))
	}
	sort.Strings(signatures)

	h := sha256.New()
	for _, s := range signatures {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	return "auto-" + hex.EncodeToString(h.Sum(nil))[:12]
}

func modelSignature(m any) string {
	t := reflect.TypeOf(m)
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return "nil"
	}
	if t.Kind() != reflect.Struct {
		return fmt.Sprintf("%s.%s", t.PkgPath(), t.Name())
	}

	fields := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fields = append(fields, fmt.Sprintf("%s:%s:%s", f.Name, f.Type.String(), f.Tag))
	}
	return fmt.Sprintf("%s.%s{%s}", t.PkgPath(), t.Name(), strings.Join(fields, ","))
}

// EffectiveMigrationTargetVersion returns the manual override if one was set
// via TENANT_MIGRATION_TARGET_VERSION (kept only as an emergency escape
// hatch), otherwise the version computed from the current AutoMigrate model
// set.
func (cfg *Config) EffectiveMigrationTargetVersion() string {
	if cfg.TenantMigrationTargetVersion != "" {
		return cfg.TenantMigrationTargetVersion
	}
	return ComputeMigrationTargetVersion(cfg.PostgresModels)
}
