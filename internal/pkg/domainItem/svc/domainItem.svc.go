package svc

import (
	"context"

	"github.com/google/uuid"

	"app/internal/pkg/domainItem/ent"
	"app/internal/pkg/domainItem/repo"
)

type DomainItemService struct {
	repo repo.DomainItemRepository
}

func NewDomainItemService(repo repo.DomainItemRepository) *DomainItemService {
	return &DomainItemService{repo: repo}
}

func (s *DomainItemService) CreateDomainItem(ctx context.Context, item *ent.DomainItem) error {
	return s.repo.CreateDomainItem(ctx, item)
}

func (s *DomainItemService) GetDomainItem(ctx context.Context, id uuid.UUID) (*ent.DomainItem, error) {
	return s.repo.GetDomainItem(ctx, id)
}

func (s *DomainItemService) ListDomainItems(ctx context.Context, limit, offset int) ([]*ent.DomainItem, int64, error) {
	return s.repo.ListDomainItems(ctx, limit, offset)
}

func (s *DomainItemService) UpdateDomainItem(ctx context.Context, item *ent.DomainItem) error {
	return s.repo.UpdateDomainItem(ctx, item)
}

func (s *DomainItemService) DeleteDomainItem(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDomainItem(ctx, id)
}
