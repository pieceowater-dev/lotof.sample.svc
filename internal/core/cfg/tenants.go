package cfg

import (
	"app/internal/core/grpc/generated/lotof.hub.gtw/gtw"
	"app/internal/core/grpc/generated/lotof.hub.msvc.namespaces/ns"
	"context"
	"fmt"
	"log"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
)

// EnsureAppTenant asks hub to provision this app's tenant schema for a
// namespace the given user genuinely belongs to, WITHOUT recording an app
// installation (that stays gated on a confirmed billing subscription). This
// exists for namespaces the periodic ConfigTenants warmup will never
// discover on its own -- it only finds namespaces already associated via
// namespace_apps, so a namespace's very first connection to this app would
// otherwise leave TenantReadiness permanently stuck.
func (cfg *Config) EnsureAppTenant(ctx context.Context, namespaceSlug string, userID string) error {
	if namespaceSlug == "" || userID == "" {
		return fmt.Errorf("namespaceSlug and userID are required")
	}

	factory := gossiper.NewTransportFactory()
	grpcTransport := factory.CreateTransport(gossiper.GRPC, cfg.HubApplicationAddr)

	client, err := grpcTransport.CreateClient(gtw.NewGatewayServiceClient)
	if err != nil {
		return fmt.Errorf("failed to create hub client: %w", err)
	}
	c := client.(gtw.GatewayServiceClient)

	md := metadata.New(map[string]string{"user-id": userID})
	outCtx := metadata.NewOutgoingContext(ctx, md)

	_, err = grpcTransport.Send(outCtx, c, "EnsureAppTenant", &gtw.GatewayEnsureAppTenantRequest{
		NamespaceSlug: namespaceSlug,
		AppBundle:     cfg.AppBundleName,
	})
	return err
}

// SyncTenant creates/switches the PostgreSQL schema for a single tenant using
// its encrypted credentials.
func (cfg *Config) SyncTenant(database gossiper.Database, tenant *gossiper.EncryptedTenant) bool {
	log.Printf("[SyncTenant] Starting schema sync for namespace: %s", tenant.Namespace)

	tm, err := gossiper.NewTenantManager(database.GetDB(), cfg.AppBundleSecret)
	if err != nil {
		log.Printf("[SyncTenant] ❌ Failed to create TenantManager: %v", err)
		return false
	}

	if err := tm.SyncTenants(&[]gossiper.EncryptedTenant{*tenant}); err != nil {
		log.Printf("[SyncTenant] ❌ Failed to sync tenant '%s': %v", tenant.Namespace, err)
		return false
	}

	cfg.Namespaces = append(cfg.Namespaces, tenant.Namespace)
	log.Printf("[SyncTenant] ✅ Successfully synced schema for namespace: %s", tenant.Namespace)
	return true
}

// ConfigTenants discovers all namespaces registered for this app from the Hub
// gateway and provisions (creates schema + AutoMigrate) each one on startup.
func (cfg *Config) ConfigTenants() {
	log.Printf("=== ConfigTenants: Starting namespace discovery and schema initialization ===")

	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ ConfigTenants recovered from panic: %v", r)
		}
	}()

	factory := gossiper.NewTransportFactory()
	grpcTransport := factory.CreateTransport(gossiper.GRPC, cfg.HubApplicationAddr)

	client, err := grpcTransport.CreateClient(gtw.NewGatewayServiceClient)
	if err != nil {
		log.Printf("❌ [ConfigTenants] Error creating client: %v", err)
		return
	}
	c := client.(gtw.GatewayServiceClient)

	ctx := context.Background()

	log.Printf("[ConfigTenants] Requesting namespaces for app bundle: '%s'", cfg.AppBundleName)
	response, err := grpcTransport.Send(ctx, c, "GatewayNamespacesByApp", &gtw.GatewayNamespacesByAppRequest{AppBundle: cfg.AppBundleName})
	if err != nil {
		log.Printf("⚠️ [ConfigTenants] Error sending GatewayNamespacesByApp request: %v", err)
		log.Printf("ℹ️ [ConfigTenants] Not critical — schemas will be created on-demand via NewTenant gRPC endpoint")
		return
	}

	res, ok := response.(*gtw.GatewayNamespacesByAppResponse)
	if !ok {
		log.Printf("⚠️ [ConfigTenants] Response type mismatch, got: %T — schemas will be created on-demand", response)
		return
	}

	tenantCount := len(res.Tenants)
	log.Printf("[ConfigTenants] ✅ Received %d namespace(s) from Hub Gateway", tenantCount)
	if tenantCount == 0 {
		log.Printf("[ConfigTenants] No existing namespaces for app '%s' — schemas will be created on first access", cfg.AppBundleName)
		return
	}

	database, err := gossiper.NewDB(gossiper.PostgresDB, cfg.PostgresDatabaseDSN, cfg.DebugSQL, cfg.PostgresModels)
	if err != nil {
		log.Printf("❌ [ConfigTenants] Failed to create database instance: %v", err)
		return
	}

	for i, tenant := range res.Tenants {
		log.Printf("[ConfigTenants] [%d/%d] Processing namespace: '%s'", i+1, tenantCount, tenant.Namespace)

		t := gossiper.EncryptedTenant{Namespace: tenant.Namespace, Credentials: tenant.Credentials}

		err := cfg.MigrateTenantWithControl(ctx, database, &t, func(ctx context.Context, namespace string) error {
			log.Printf("[ConfigTenants] Running AutoMigrate for namespace: '%s' (models: %d)", namespace, len(cfg.PostgresModels))
			return database.WithSchema(ctx, namespace, func(tx *gorm.DB) error {
				for _, model := range cfg.PostgresModels {
					if err := tx.AutoMigrate(model); err != nil {
						return fmt.Errorf("auto-migrate failed for namespace '%s', model '%T': %w", namespace, model, err)
					}
				}
				return nil
			})
		})
		if err != nil {
			log.Printf("❌ [ConfigTenants] Migration failed for namespace '%s': %v", t.Namespace, err)
			continue
		}

		log.Printf("[ConfigTenants] ✅ AutoMigrate completed for namespace: '%s'", t.Namespace)
	}

	log.Printf("=== ConfigTenants: Warmup finished for %d namespace(s) ===", tenantCount)
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
