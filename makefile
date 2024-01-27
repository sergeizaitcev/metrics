MODULE := $(shell head -n1 go.mod | sed -e 's/module //')
BRANCH_NAME := $(shell git rev-parse --abbrev-ref HEAD)
# VERSION := $(shell git describe --tags)
VERSION := $(BRANCH_NAME)
DATE := $(shell date +'%Y/%m/%d %H:%M:%S')
COMMIT := $(shell git rev-parse --short HEAD)
GO_BUILD := go build -ldflags "-X '$(MODULE)/version.Build=$(VERSION)' -X '$(MODULE)/version.Date=$(DATE)' -X '$(MODULE)/version.Commit=$(COMMIT)'"

STATIC_LINT := $(GOPATH)/bin/staticlint

$(STATIC_LINT):
	@go install ./cmd/staticlint

RSA_KEYGEN := $(GOPATH)/bin/rsakeygen

$(RSA_KEYGEN):
	@go install ./cmd/rsakeygen

SERVER_PORT := $(shell random unused-port)
TEMP_FILE := $(shell random tempfile)
DATABASE_DSN := postgres://postgres:postgres@localhost:5432/practicum?sslmode=disable
ADDRESS := localhost:$(SERVER_PORT)

.DEFAULT_GOAL := all

.PHONY: all
all: test lint autotest

.PHONY: keygen
keygen: $(RSA_KEYGEN)
ifeq ("$(wildcard server.rsa")", "")
	@$(RSA_KEYGEN) -b 4096 -p server
endif

.PHONY: up
up:
	@docker-compose -f ./scripts/docker-compose.yml up -d postgres

.PHONY: down
down:
	@docker-compose -f ./scripts/docker-compose.yml down

.PHONY: lint
lint: $(STATIC_LINT)
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
	@$(GO_BUILD) -o ./cmd/agent/agent ./cmd/agent
	@$(GO_BUILD) -o ./cmd/server/server ./cmd/server

.PHONY: autotest
autotest: build $(BRANCH_NAME)

iter%:
	@metricstest -test.v -test.run=^TestIteration$*[AB]?$$ \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server \
		-server-port=$(SERVER_PORT) \
		-database-dsn=$(DATABASE_DSN) \
		-file-storage-path=$(TEMP_FILE) \
		-key=$(TEMP_FILE) \
		-source-path=.
