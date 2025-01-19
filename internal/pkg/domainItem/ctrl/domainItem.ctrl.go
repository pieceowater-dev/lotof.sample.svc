package ctrl

import (
	pb "app/internal/core/grpc/generated/lotof.sample.proto/lotof.sample.svc/domainItem"
	"app/internal/pkg/domainItem/svc"
	"context"
	"fmt"
	"strconv"
)

type DomainItemController struct {
	domainItemService *svc.DomainItemService
	pb.UnimplementedDomainItemServiceServer
}

// NewDomainItemController creates a new DomainItemController instance.
func NewDomainItemController(service *svc.DomainItemService) *DomainItemController {
	return &DomainItemController{domainItemService: service}
}

// SomeQueryMethod handles the GetSomethingRequest and returns a GetSomethingResponse.
func (c *DomainItemController) SomeQueryMethod(ctx context.Context, req *pb.GetSomethingRequest) (*pb.GetSomethingResponse, error) {
	// Convert req.Id from string to int
	id, err := strconv.Atoi(req.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %v", err)
	}

	// Delegate to the service
	somethings, err := c.domainItemService.GetSomethings(ctx, id)
	if err != nil {
		return nil, err
	}

	// Convert entities to gRPC-compatible responses
	var pbSomethings []*pb.Something
	for _, item := range somethings {
		pbSomethings = append(pbSomethings, &pb.Something{
			Id:       fmt.Sprintf("%d", item.ID),
			SomeEnum: pb.SomeEnum(item.SomeEnum),
		})
	}

	return &pb.GetSomethingResponse{
		Somethings: pbSomethings,
	}, nil
}

// SomeMutationMethod handles the CreateSomethingRequest. Currently not implemented.
func (c *DomainItemController) SomeMutationMethod(ctx context.Context, req *pb.CreateSomethingRequest) (*pb.Something, error) {
	panic("Not implemented")
}
