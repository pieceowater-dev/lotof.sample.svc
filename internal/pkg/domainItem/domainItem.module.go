package domainItem

import (
	"app/internal/pkg/domainItem/ctrl"
	"app/internal/pkg/domainItem/repo"
	"app/internal/pkg/domainItem/svc"

	gossiper "github.com/pieceowater-dev/lotof.lib.gossiper/v2"
)

type Module struct {
	name    string
	version string
	API     *ctrl.DomainItemController
	Service *svc.DomainItemService
}

// New creates a new instance of the DomainItem module.
func New(db gossiper.Database) *Module {
	repository := repo.New(db)
	service := svc.NewDomainItemService(repository)
	controller := ctrl.NewDomainItemController(service)

	return &Module{
		name:    "DomainItem",
		version: "v1",
		API:     controller,
		Service: service,
	}
}

func (m Module) Initialize() error { return nil }
func (m Module) Version() string   { return m.version }
func (m Module) Name() string      { return m.name }
