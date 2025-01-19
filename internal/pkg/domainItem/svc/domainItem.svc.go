package svc

import (
	"app/internal/pkg/domainItem/ent"
	"context"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"log"
)

type DomainItemService struct {
	db gossiper.Database
}

// NewDomainItemService creates a new DomainItemService instance.
func NewDomainItemService(db gossiper.Database) *DomainItemService {
	return &DomainItemService{db: db}
}

// GetSomethings fetches somethings from the database filtered by ID.
func (s *DomainItemService) GetSomethings(ctx context.Context, id int) ([]ent.Something, error) {
	log.Println("Fetching somethings from database...")

	// Fetch somethings using the database interface filtered by ID
	var items []ent.Something
	if err := s.db.GetDB().WithContext(ctx).Where("id = ?", id).Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}
