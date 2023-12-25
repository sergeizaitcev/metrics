BRANCH_NAME := $(shell git rev-parse --abbrev-ref HEAD)
SERVER_PORT := $(shell random unused-port)
TEMP_FILE := $(shell random tempfile)
DATABASE_DSN := postgres://postgres:postgres@localhost:5432/practicum?sslmode=disable
ADDRESS := localhost:$(SERVER_PORT)

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
lint:
	@go vet -vettool=$(shell which statictest) ./...

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
	@go build -o ./cmd/agent/agent ./cmd/agent
	@go build -o ./cmd/server/server ./cmd/server

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
