# lotof.sample.svc

Template gRPC microservice ("msvc") for bootstrapping a new LOTOF domain. It
ships with the full multi-tenant scaffolding every current domain service
(menu, issues, contacts, atrace) already runs — namespace-per-tenant schema
switching, migration state tracking with retry/backoff, and Hub-driven
tenant provisioning — plus one worked example module (`domainItem`) so the
wiring is visible end to end instead of left as an exercise.

Pairs with [lotof.sample.proto](https://github.com/pieceowater-dev/lotof.sample.proto)
(the gRPC contracts) and [lotof.sample.gtw](https://github.com/pieceowater-dev/lotof.sample.gtw)
(the GraphQL gateway that calls this service).

## What's generic vs what's the example

Keep as-is — this is shared multi-tenant plumbing, not domain logic:

```
internal/core/cfg/
  cfg.go                       # add your own env vars here; keep the tenant fields
  tenants.go                   # Hub discovery/provisioning calls (EnsureAppTenant, ConfigTenants, AddAppToNamespace)
  tenant_migration_control.go  # migration state table, advisory locks, retry/backoff
  tenant_migration_version.go  # auto-computed migration version hash from PostgresModels
internal/core/generic/
  entbase/timestamps.go        # embed entbase.Timestamps in your entities instead of gorm.Model
  middleware/
    middleware.go               # GrpcMiddleware interface
    userdata.middleware.go      # unpacks namespace/user-id from gRPC metadata into ctx
    tenant_readiness.middleware.go  # blocks requests until the tenant schema is migrated
internal/pkg/_tenant/          # implements AppTenantsService (NewTenant/GetTenantStatus) -- register every new entity's &ent.Thing{} in tenants.module.go
```

Delete or rename — this is the disposable example:

```
internal/pkg/domainItem/       # ent + repo + svc + ctrl for one CRUD entity
```

## Bootstrapping a new service

1. Fork this repo as `lotof.<domain>.msvc.<name>`.
2. Update `go.mod`'s module-relative import path is `app` (unchanged) but repoint the proto dependency: `go get github.com/pieceowater-dev/lotof.<domain>.proto@latest` (see the matching `.proto` template's README for forking that repo first).
3. Rename `internal/pkg/domainItem/` to your first real entity (ent/repo/svc/ctrl, same shape), update `internal/pkg/router.go` and `internal/pkg/_tenant/tenants.module.go` to reference it instead.
4. Update `.env` / `cfg.go` defaults: `APP_BUNDLE_NAME` (must match what you register in Hub's `namespace_apps`), `SERVICE_NAME`, `POSTGRES_DB_DSN`.
5. Register the new app bundle in Hub so `ConfigTenants` can discover namespaces for it.
6. `make setup && make generate && make build`.

## Multi-tenancy model

Each namespace (tenant) gets its own PostgreSQL schema. `internal/pkg/_tenant`
implements the `AppTenantsService.NewTenant` RPC Hub calls to provision a
schema on demand; `tenant_readiness.middleware.go` blocks any other RPC for a
namespace until that namespace's schema has finished migrating, retrying
with backoff instead of failing outright. `ComputeMigrationTargetVersion`
hashes the registered `PostgresModels` set, so a schema change on any entity
alone is enough to trigger a re-migration for every tenant — no hand-bumped
version string to forget.

Repos read the tenant namespace out of context via
`middleware.GetNamespaceFromContext` and switch schema with
`db.WithSchema`/`db.WithSchemaReadOnly` (see `domainItem/repo`) — copy that
pattern for every new entity's repository.

## Prerequisites

- [Go](https://golang.org/doc/install) 1.25+
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)
- `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` (`make setup` installs the Go generators; `protoc` itself must already be on PATH)

## Common tasks

```
make setup     # go get the latest proto package + go mod tidy
make generate  # regenerate gRPC stubs (own proto + Hub's, for tenant provisioning)
make build     # compile to bin/lotof.sample.svc
make run       # build + run
make test      # go build ./... (no test suite yet)
compose-up     # local postgres + pgadmin
```

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author
![PCWT Dev Logo](https://avatars.githubusercontent.com/u/168465239?s=50)
### [PCWT Dev](https://github.com/pieceowater-dev)
