package domainItem

import (
	"app/internal/pkg/domainItem/ctrl"
	"app/internal/pkg/domainItem/svc"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
)

type Module struct {
	name    string
	version string
	API     *ctrl.DomainItemController
}

// New creates a new instance of the DomainItem module.
func New(db gossiper.Database) *Module {
	// Create service and controller
	service := svc.NewDomainItemService(db)
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
