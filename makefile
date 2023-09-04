SERVER_PORT := $(shell random unused-port)
ADDRESS := localhost:$(SERVER_PORT)
TEMP_FILE := $(shell random tempfile)

.PHONY: build
build:
	go build -o ./cmd/agent/agent ./cmd/agent
	go build -o ./cmd/server/server ./cmd/server

.PHONY: static
static:
	go vet -vettool=$(shell which statictest) ./...

.PHONY: test
test:
	go test -short -race -timeout=10s -count=1 -cover ./...

.PHONY: iter1
iter1: build
	metricstest -test.v -test.run=^TestIteration1$$ \
		-binary-path=cmd/server/server

.PHONY: iter2
iter2: build
	metricstest -test.v -test.run=^TestIteration2[AB]*$$ \
		-source-path=. -agent-binary-path=cmd/agent/agent

.PHONY: iter3
iter3: build
	metricstest -test.v -test.run=^TestIteration3[AB]*$$ \
		-source-path=. \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server

.PHONY: iter4
iter4: build
	metricstest -test.v -test.run=^TestIteration4$$ \
		-source-path=. \
		-server-port=$(SERVER_PORT) \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server

.PHONY: iter5
iter5: build
	metricstest -test.v -test.run=^TestIteration5$$ \
		-source-path=. \
		-server-port=$(SERVER_PORT) \
		-agent-binary-path=cmd/agent/agent \
		-binary-path=cmd/server/server
