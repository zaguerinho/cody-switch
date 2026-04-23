.PHONY: build test lint clean

BINARY := video-tutorial
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/video-tutorial/

test:
	go test ./... -v -race -count=1

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run

clean:
	rm -f $(BINARY)

# Run just the unit tests (no integration)
test-unit:
	go test ./internal/... -v -count=1

# Sync browser JS to embed directory (for go:embed)
sync-embed:
	cp browser/*.js embed/browser/

# Show binary size
size: build
	@ls -lh $(BINARY) | awk '{print $$5, $$9}'
