package svc

import (
	"app/internal/core/grpc/generated/generic/tenants"
	"context"
	"time"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
)

// activitySpecs are the usage metrics the admin console's namespace groups
// page reads for this product ("does lota grow these businesses").
// Each product lists its own tables here; domain_items is the sample entity.
var activitySpecs = []gossiper.ActivitySpec{
	{Key: "items", Table: "domain_items", TimeColumn: "created_at"},
}

// GetTenantActivity counts the tenant's usage of this product over time.
// Excluded from tenantReadinessMiddleware like GetTenantStatus: a tenant
// that is not ready reports per-metric errors instead of hanging.
func (s *TenantServer) GetTenantActivity(ctx context.Context, req *tenants.GetTenantActivityRequest) (*tenants.GetTenantActivityResponse, error) {
	counts, err := gossiper.CountActivity(ctx, s.database, req.GetNamespace(), activitySpecs, int(req.GetDays()), int(req.GetWeeks()), time.Now())
	if err != nil {
		return nil, err
	}
	out := &tenants.GetTenantActivityResponse{Metrics: make([]*tenants.ActivityMetric, 0, len(counts))}
	for _, c := range counts {
		out.Metrics = append(out.Metrics, &tenants.ActivityMetric{
			Key:      c.Key,
			Total:    c.Total,
			Current:  c.Current,
			Previous: c.Previous,
			Weekly:   c.Weekly,
			Error:    c.Error,
		})
	}
	return out, nil
}
