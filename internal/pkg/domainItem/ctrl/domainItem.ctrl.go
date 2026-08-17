package ctrl

import (
	"context"

	"github.com/google/uuid"

	genericpb "app/internal/core/grpc/generated/generic/utils"
	pb "app/internal/core/grpc/generated/lotof.sample.svc/domainItem"
	"app/internal/pkg/domainItem/ent"
	"app/internal/pkg/domainItem/svc"
)

type DomainItemController struct {
	pb.UnimplementedSampleDomainItemServiceServer
	svc *svc.DomainItemService
}

func NewDomainItemController(service *svc.DomainItemService) *DomainItemController {
	return &DomainItemController{svc: service}
}

func entityToPb(item *ent.DomainItem) *pb.DomainItem {
	return &pb.DomainItem{
		Id:     item.ID.String(),
		Name:   item.Name,
		Status: pb.DomainItemStatus(item.Status),
	}
}

func (c *DomainItemController) CreateDomainItem(ctx context.Context, req *pb.CreateDomainItemRequest) (*pb.DomainItem, error) {
	item := &ent.DomainItem{
		ID:   uuid.New(),
		Name: req.Name,
	}
	if err := c.svc.CreateDomainItem(ctx, item); err != nil {
		return nil, err
	}
	return entityToPb(item), nil
}

func (c *DomainItemController) GetDomainItem(ctx context.Context, req *pb.GetDomainItemRequest) (*pb.DomainItem, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, err
	}
	item, err := c.svc.GetDomainItem(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return entityToPb(item), nil
}

func (c *DomainItemController) ListDomainItems(ctx context.Context, req *pb.ListDomainItemsRequest) (*pb.ListDomainItemsResponse, error) {
	limit, offset := paginationParams(req.Pagination)
	items, total, err := c.svc.ListDomainItems(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	pbItems := make([]*pb.DomainItem, len(items))
	for i, item := range items {
		pbItems[i] = entityToPb(item)
	}
	return &pb.ListDomainItemsResponse{
		DomainItems:    pbItems,
		PaginationInfo: &genericpb.PaginationInfo{Count: int32(total)},
	}, nil
}

func (c *DomainItemController) UpdateDomainItem(ctx context.Context, req *pb.UpdateDomainItemRequest) (*pb.DomainItem, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, err
	}
	item := &ent.DomainItem{
		ID:     id,
		Name:   req.Name,
		Status: ent.Status(req.Status),
	}
	if err := c.svc.UpdateDomainItem(ctx, item); err != nil {
		return nil, err
	}
	return c.GetDomainItem(ctx, &pb.GetDomainItemRequest{Id: req.Id})
}

func (c *DomainItemController) DeleteDomainItem(ctx context.Context, req *pb.DeleteDomainItemRequest) (*pb.DeleteDomainItemResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return &pb.DeleteDomainItemResponse{Success: false}, err
	}
	if err := c.svc.DeleteDomainItem(ctx, id); err != nil {
		return &pb.DeleteDomainItemResponse{Success: false}, err
	}
	return &pb.DeleteDomainItemResponse{Success: true}, nil
}

func paginationParams(p *genericpb.Pagination) (limit, offset int) {
	limit = 20
	offset = 0
	if p != nil && p.Length > 0 {
		limit = int(p.Length)
		offset = (int(p.Page) - 1) * limit
	}
	return
}
