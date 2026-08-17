// Package entbase provides shared, embeddable field sets for domain entities
// (ent structs). It plays the same role as GORM's gorm.Model, but keeps our
// UUID primary keys instead of gorm.Model's auto-increment uint ID.
package entbase

import (
	"time"

	"gorm.io/gorm"
)

// AuditTimestamps tracks creation and last-update times. Embed this in
// entities that are never deleted (e.g. append-only records such as
// activity log entries).
type AuditTimestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Timestamps extends AuditTimestamps with GORM's native soft-delete support.
// Because DeletedAt is a gorm.DeletedAt, GORM automatically scopes every
// query to exclude soft-deleted rows (no manual "is_deleted = false" filter
// needed), and a plain .Delete(&entity) call soft-deletes instead of hard
// removing the row. Use db.Unscoped() to bypass the filter when needed.
type Timestamps struct {
	AuditTimestamps
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
