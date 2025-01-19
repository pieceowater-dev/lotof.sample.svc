package domainItem

import (
	"app/internal/core/cfg"
	"app/internal/pkg/domainItem/ctrl"
	"app/internal/pkg/domainItem/svc"
	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
	"log"
)

type Module struct {
	name    string
	version string
	API     *ctrl.DomainItemController
}

// New creates a new instance of the DomainItem module.
func New() *Module {
	// Create database instance
	database, err := gossiper.NewDB(
		gossiper.PostgresDB,
		cfg.Inst().PostgresDatabaseDSN,
		false,
		[]any{},
	)
	if err != nil {
		log.Fatalf("Failed to create database instance: %v", err)
	}

	// Create service and controller
	service := svc.NewDomainItemService(database)
	controller := ctrl.NewDomainItemController(service)

	// Initialize and return the module
	return &Module{
		name:    "DomainItem",
		version: "v1",
		API:     controller,
	}
}

// Initialize initializes the module. Currently not implemented.
func (m Module) Initialize() error {
	panic("Not implemented")
}

// Version returns the version of the module.
func (m Module) Version() string {
	return m.version
}

// Name returns the name of the module.
func (m Module) Name() string {
	return m.name
}
