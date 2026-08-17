package ent

import (
	"app/internal/core/generic/entbase"

	"github.com/google/uuid"
)

type Status int

const (
	Active Status = iota
	Archived
)

// DomainItem is the template's example entity -- delete/rename this whole
// module (ent/repo/svc/ctrl + the proto service it implements) when
// bootstrapping a real domain, and copy its shape for your first real
// entity in the meantime.
type DomainItem struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name   string    `gorm:"type:varchar(256);not null"`
	Status Status    `gorm:"not null;default:0"`

	entbase.Timestamps
}

func (DomainItem) TableName() string {
	return "domain_items"
}
