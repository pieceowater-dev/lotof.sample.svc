package tenant

import (
	"app/internal/core/cfg"
	"app/internal/pkg/_tenant/svc"
	domainItemEnt "app/internal/pkg/domainItem/ent"
	"log"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
)

// Module implements generic.tenants.AppTenantsService (see
// protos/generic/tenants/tenants.proto) -- Hub calls NewTenant when a
// namespace is created/attached to this app so the service can provision
// (create schema + AutoMigrate) the tenant on demand.
type Module struct {
	name    string
	version string
	API     *svc.TenantServer
}

// New wires the tenant module against every entity that needs its own
// per-namespace table. Add every new domain entity's &ent.Thing{} here so
// AutoMigrate picks it up for each tenant schema.
func New() *Module {
	entities := []any{
		&domainItemEnt.DomainItem{},
	}

	database, err := gossiper.NewDB(
		gossiper.PostgresDB,
		cfg.Inst().PostgresDatabaseDSN,
		cfg.Inst().DebugSQL,
		entities,
	)
	if err != nil {
		log.Printf("Failed to create database instance: %v", err)
		return nil
	}

	return &Module{
		name:    "tenant",
		version: "v1",
		API:     svc.NewTenantServer(database, entities),
	}
}

// Initialize initializes the module.
func (m Module) Initialize() error { return nil }

// Version returns the version of the module.
func (m Module) Version() string { return m.version }

// Name returns the name of the module.
func (m Module) Name() string { return m.name }
