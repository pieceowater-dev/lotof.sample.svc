PROTOC = protoc
PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GRPC_GO = $(GOPATH)/bin/protoc-gen-go-grpc
PROTOC_PKG = github.com/pieceowater-dev/lotof.sample.proto
PROTOC_PKG_PATH = $(shell go list -m -f '{{.Dir}}' $(PROTOC_PKG))
PROTOC_DIR = protos
PROTOC_OUT_DIR = ./internal/core/grpc/generated

DOCKER_COMPOSE = docker-compose

.PHONY: all generate run clean build-dev build-main compose-up compose-down

all: gql-gen run


# gRPC code generation
grpc-gen: grpc-clean
	mkdir -p $(PROTOC_OUT_DIR)
	$(PROTOC) \
		-I $(PROTOC_PKG_PATH)/$(PROTOC_DIR) \
		--go_out=$(PROTOC_OUT_DIR) \
		--go-grpc_out=$(PROTOC_OUT_DIR) \
		$(PROTOC_PKG_PATH)/$(PROTOC_DIR)/*/*/*.proto

grpc-clean:
	rm -rf $(PROTOC_OUT_DIR)

grpc-update:
	go get -u $(PROTOC_PKG)@latest

run:
	go run ./cmd/server/main.go


build:
	docker build -t gtw .


compose-up:
	$(DOCKER_COMPOSE) up -d

compose-down:
	$(DOCKER_COMPOSE) down

clean: gql-clean grpc-clean