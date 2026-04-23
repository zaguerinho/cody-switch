.PHONY: build test lint clean size

BINARY := agent-hub
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/agent-hub/

test:
	go test ./... -v -race -count=1

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run

clean:
	rm -f $(BINARY)

test-unit:
	go test ./internal/... -v -count=1

size: build
	@ls -lh $(BINARY) | awk '{print $$5, $$9}'
