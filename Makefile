APP_NAME = lotof.sample.svc
BUILD_DIR = bin
MAIN_FILE = cmd/server/main.go
PG_MIGRATION_DIR = cmd/server/db/pg/migrations
PROTOC = protoc
PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GRPC_GO = $(GOPATH)/bin/protoc-gen-go-grpc
PROTOC_PKG = github.com/pieceowater-dev/lotof.sample.proto
PROTOC_PKG_PATH = $(shell go list -m -f '{{.Dir}}' $(PROTOC_PKG))
PROTOC_DIR = protos
PROTOC_OUT_DIR = ./internal/core/grpc/generated
PG_DB_DSN = $(shell grep POSTGRES_DB_DSN .env | cut -d '"' -f2)
DOCKER_COMPOSE = docker-compose

export PATH := /usr/local/bin:$(PATH)

.PHONY: all clean build run update migration migrate db-sync setup install-flyway install-atlas install-postgres install-atlas-cli \
        grpc-gen grpc-clean grpc-update compose-up compose-down gql-gen gql-clean

# Setup the environment
setup: install-atlas-cli grpc-update
	@echo "Setup completed!"; \
	go mod tidy

# Default build target
all: build

# Update dependencies
update:
	go mod tidy

# Build the project
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

# Run the application
run: build
	./$(BUILD_DIR)/$(APP_NAME)

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) gql-clean grpc-clean

# Install Atlas CLI
install-atlas-cli:
	@brew install ariga/tap/atlas

# Generate new migration files with Atlas
pg-migration:
	@mkdir -p $(PG_MIGRATION_DIR); \
	PATH=/usr/local/bin:$$PATH atlas migrate diff --env postgres; \
	echo "Migration files generated in $(PG_MIGRATION_DIR)"; \
	git add $(PG_MIGRATION_DIR)/*

# Apply migrations with Atlas
pg-migrate:
	@PATH=/usr/local/bin:$$PATH atlas migrate apply --url "$(PG_DB_DSN)" --dir="file://$(shell pwd)/$(PG_MIGRATION_DIR)"

# Sync migrations: generate and apply them
db-sync: pg-migration pg-migrate

# gRPC code generation
grpc-gen:
	@echo "Generating gRPC code from proto files..."
	mkdir -p $(PROTOC_OUT_DIR)
	find $(PROTOC_PKG_PATH)/$(PROTOC_DIR) -name "*.proto" | xargs $(PROTOC) \
		-I $(PROTOC_PKG_PATH)/$(PROTOC_DIR) \
		--go_out=$(PROTOC_OUT_DIR) \
		--go-grpc_out=$(PROTOC_OUT_DIR)
	@echo "gRPC code generation completed!"

# Clean gRPC generated files
grpc-clean:
	rm -rf $(PROTOC_OUT_DIR)

# Update gRPC dependencies
grpc-update:
	go get -u $(PROTOC_PKG)@latest

# Docker build target
build-docker:
	docker build -t $(APP_NAME) .

# Build Docker image and run the container
build-and-run-docker: build-docker
	docker stop $(APP_NAME)
	docker rm $(APP_NAME)
	docker run -d -p 50051:50051 \
		-e POSTGRES_DB_DSN="$(PG_DB_DSN)" \
		--network lotofsamplesvc_pieceonetwork \
		--name $(APP_NAME) \
		$(APP_NAME)

# Start Docker Compose services
compose-up:
	$(DOCKER_COMPOSE) up -d

# Stop Docker Compose services
compose-down:
	$(DOCKER_COMPOSE) down