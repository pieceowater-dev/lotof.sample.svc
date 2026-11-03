package cfg

import (
	"app/internal/core/grpc/generated/lotof.hub.gtw/gtw"
	"app/internal/core/grpc/generated/lotof.hub.msvc.namespaces/ns"
	"context"
	"fmt"
	"log"
	"sync"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"github.com/pieceowater-dev/lotof.lib.gossiper/v2/tenancy"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

// The provisioning logic itself (namespace discovery, schema sync, AutoMigrate
// under the migration-state bookkeeping, readiness) lives in gossiper's
// tenancy package -- it was copied into every service and drifted. What stays
// here is the service-specific part: the Hub calls (their gRPC clients are
// generated per service) and the hooks around AutoMigrate.

var (
	tenantsOnce sync.Once
	tenantsMgr  *tenancy.Manager
)

// Tenants is this service's tenant manager.
func (cfg *Config) Tenants() *tenancy.Manager {
	tenantsOnce.Do(func() {
		tenantsMgr = tenancy.NewManager(tenancy.ManagerConfig{
			ServiceName:     cfg.ServiceName,
			AppBundleName:   cfg.AppBundleName,
			AppBundleSecret: cfg.AppBundleSecret,
			DatabaseDSN:     cfg.PostgresDatabaseDSN,
			DebugSQL:        cfg.DebugSQL,
			Models:          func() []any { return cfg.PostgresModels },
			// Emergency manual override only; the version is normally computed
			// from the AutoMigrate model set.
			VersionOverride: func() string { return cfg.TenantMigrationTargetVersion },
			Hub:             hubClient{cfg: cfg},
			OnSynced:        cfg.AddNamespace,
			AfterModels: func(ctx context.Context, tx *gorm.DB, namespace string) error {
				return ApplyRawTenantMigrations(tx)
			},
		})
	})
	return tenantsMgr
}

var namespacesMu sync.Mutex

// AddNamespace records a synced tenant; re-syncing one (every warmup does) is not a new entry.
func (cfg *Config) AddNamespace(namespace string) {
	namespacesMu.Lock()
	defer namespacesMu.Unlock()
	for _, existing := range cfg.Namespaces {
		if existing == namespace {
			return
		}
	}
	cfg.Namespaces = append(cfg.Namespaces, namespace)
}

// hubClient is tenancy.Hub over this service's generated Hub clients.
type hubClient struct{ cfg *Config }

// Tenants lists the namespaces Hub has registered for this app.
func (h hubClient) Tenants(ctx context.Context) ([]tenancy.EncryptedTenant, error) {
	factory := gossiper.NewTransportFactory()
	grpcTransport := factory.CreateTransport(gossiper.GRPC, h.cfg.HubApplicationAddr)

	client, err := grpcTransport.CreateClient(gtw.NewGatewayServiceClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create hub client: %w", err)
	}
	c := client.(gtw.GatewayServiceClient)

	response, err := grpcTransport.Send(ctx, c, "GatewayNamespacesByApp", &gtw.GatewayNamespacesByAppRequest{AppBundle: h.cfg.AppBundleName})
	if err != nil {
		return nil, err
	}
	res, ok := response.(*gtw.GatewayNamespacesByAppResponse)
	if !ok {
		return nil, fmt.Errorf("response type mismatch, got: %T", response)
	}

	tenants := make([]tenancy.EncryptedTenant, 0, len(res.Tenants))
	for _, t := range res.Tenants {
		tenants = append(tenants, tenancy.EncryptedTenant{Namespace: t.Namespace, Credentials: t.Credentials})
	}
	return tenants, nil
}

// EnsureAppTenant asks hub to provision this app's tenant schema for a
// namespace the given user genuinely belongs to, WITHOUT recording an app
// installation (that stays gated on a confirmed billing subscription). This
// exists for namespaces the periodic warmup will never discover on its own --
// it only finds namespaces already associated via namespace_apps, so a
// namespace's very first connection to this app would otherwise leave
// TenantReadiness permanently stuck.
func (h hubClient) EnsureAppTenant(ctx context.Context, namespaceSlug string, userID string) error {
	if namespaceSlug == "" || userID == "" {
		return fmt.Errorf("namespaceSlug and userID are required")
	}

	factory := gossiper.NewTransportFactory()
	grpcTransport := factory.CreateTransport(gossiper.GRPC, h.cfg.HubApplicationAddr)

	client, err := grpcTransport.CreateClient(gtw.NewGatewayServiceClient)
	if err != nil {
		return fmt.Errorf("failed to create hub client: %w", err)
	}
	c := client.(gtw.GatewayServiceClient)

	md := metadata.New(map[string]string{"user-id": userID})
	outCtx := metadata.NewOutgoingContext(ctx, md)

	_, err = grpcTransport.Send(outCtx, c, "EnsureAppTenant", &gtw.GatewayEnsureAppTenantRequest{
		NamespaceSlug: namespaceSlug,
		AppBundle:     h.cfg.AppBundleName,
	})
	return err
}

// AddAppToNamespace registers this app to a namespace in Hub (used when the app
// is attached to an existing namespace).
func (cfg *Config) AddAppToNamespace(namespaceID string) error {
	log.Printf("[AddAppToNamespace] Registering app '%s' to namespace '%s'", cfg.AppBundleName, namespaceID)

	factory := gossiper.NewTransportFactory()
	grpcTransport := factory.CreateTransport(gossiper.GRPC, cfg.HubApplicationAddr)

	client, err := grpcTransport.CreateClient(ns.NewNamespaceServiceClient)
	if err != nil {
		log.Printf("❌ [AddAppToNamespace] Error creating client: %v", err)
		return err
	}
	c := client.(ns.NamespaceServiceClient)

	ctx := context.Background()
	response, err := grpcTransport.Send(ctx, c, "AddAppToNamespace", &ns.AddAppToNamespaceRequest{
		NamespaceId: namespaceID,
		AppBundle:   cfg.AppBundleName,
	})
	if err != nil {
		log.Printf("❌ [AddAppToNamespace] Error sending request: %v", err)
		return err
	}

	res, ok := response.(*ns.NamespaceApp)
	if !ok {
		return fmt.Errorf("response type mismatch: got %T", response)
	}

	log.Printf("✅ [AddAppToNamespace] Registered. NamespaceApp ID: %s", res.Id)
	return nil
}
