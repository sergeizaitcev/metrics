module := $(shell head -n1 go.mod | sed -e 's/module //')
branch_name := $(shell git rev-parse --abbrev-ref HEAD)
# VERSION := $(shell git describe --tags)
version := $(BRANCH_NAME)
date := $(shell date +'%Y/%m/%d %H:%M:%S')
commit := $(shell git rev-parse --short HEAD)
go_build := go build -ldflags "-X '$(MODULE)/version.Build=$(VERSION)' -X '$(MODULE)/version.Date=$(DATE)' -X '$(MODULE)/version.Commit=$(COMMIT)'"

static_lint := $(GOPATH)/bin/staticlint
$(static_lint):
	@go install ./cmd/staticlint

protoc_gen_go := $(GOPATH)/bin/protoc-gen-go
$(protoc_gen_go):
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

protoc_gen_go_grpc := $(GOPATH)/bin/protoc-gen-go-grpc
$(protoc_gen_go_grpc):
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

server_port := $(shell random unused-port)
temp_file := $(shell random tempfile)
database_dsn := postgres://postgres:postgres@localhost:5432/practicum?sslmode=disable
address := localhost:$(SERVER_PORT)

.DEFAULT_GOAL := all

.PHONY: all
all: test lint autotest

.PHONY: up
up:
	@docker-compose -f ./scripts/docker-compose.yml up -d postgres

.PHONY: down
down:
	@docker-compose -f ./scripts/docker-compose.yml down

.PHONY: lint
lint: $(static_lint)
	@go vet -vettool=$(shell which statictest) ./...
	@go vet -vettool=$(shell which staticlint) ./...

.PHONY: test
test:
	@go test -short -race -timeout=30s -count=1 -coverprofile=cover.out ./...

.PHONY: cover
cover:
	@go tool cover -func=cover.out

.PHONY: clean
clean:
	@rm -rf ./cmd/agent/agent ./cmd/server/server ./cover.out

.PHONY: build
build:
	@$(go_build) -o ./cmd/agent/agent ./cmd/agent
	@$(go_build) -o ./cmd/server/server ./cmd/server

.PHONY: proto
proto: $(protoc_gen_go) $(protoc_gen_go_grpc)
	@protoc --proto_path=api/proto --go_out=api/proto --go-grpc_out=api/proto metrics/metrics.proto

.PHONY: keygen
keygen:
	@openssl genrsa -out testdata/private.pem 4096
	@openssl rsa -in testdata/private.pem -outform PEM -pubout -out testdata/public.pem 

.PHONY: autotest
autotest: build $(branch_name)

iter%:
	@metricstest -test.v -test.run=^TestIteration$*[AB]?$$ \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server \
		-server-port=$(server_port) \
		-database-dsn=$(database_dsn) \
		-file-storage-path=$(temp_file) \
		-key=$(temp_file) \
		-source-path=.
