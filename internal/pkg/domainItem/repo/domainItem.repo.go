package repo

import (
	"context"

	"github.com/google/uuid"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"gorm.io/gorm"

	"app/internal/pkg/domainItem/ent"
)

// DomainItemRepository is defined as an interface (rather than the struct
// below being used directly) so the svc layer can be unit-tested against a
// fake without a database.
type DomainItemRepository interface {
	CreateDomainItem(ctx context.Context, item *ent.DomainItem) error
	GetDomainItem(ctx context.Context, id uuid.UUID) (*ent.DomainItem, error)
	ListDomainItems(ctx context.Context, limit, offset int) ([]*ent.DomainItem, int64, error)
	UpdateDomainItem(ctx context.Context, item *ent.DomainItem) error
	DeleteDomainItem(ctx context.Context, id uuid.UUID) error
}

type Repo struct {
	db gossiper.Database
}

func New(db gossiper.Database) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateDomainItem(ctx context.Context, item *ent.DomainItem) error {
	return gossiper.WithSchema(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Create(item).Error
	})
}

func (r *Repo) GetDomainItem(ctx context.Context, id uuid.UUID) (*ent.DomainItem, error) {
	var item ent.DomainItem
	err := gossiper.WithSchemaReadOnly(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).First(&item).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *Repo) ListDomainItems(ctx context.Context, limit, offset int) ([]*ent.DomainItem, int64, error) {
	var items []*ent.DomainItem
	var count int64
	err := gossiper.WithSchemaReadOnly(ctx, r.db, func(tx *gorm.DB) error {
		q := tx.Model(&ent.DomainItem{})
		if err := q.Count(&count).Error; err != nil {
			return err
		}
		return q.Order("created_at ASC").Limit(limit).Offset(offset).Find(&items).Error
	})
	return items, count, err
}

func (r *Repo) UpdateDomainItem(ctx context.Context, item *ent.DomainItem) error {
	return gossiper.WithSchema(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Model(&ent.DomainItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]any{
				"name":   item.Name,
				"status": item.Status,
			}).Error
	})
}

func (r *Repo) DeleteDomainItem(ctx context.Context, id uuid.UUID) error {
	return gossiper.WithSchema(ctx, r.db, func(tx *gorm.DB) error {
		return tx.Where("id = ?", id).Delete(&ent.DomainItem{}).Error
	})
}
