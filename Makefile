.PHONY: build install clean test fmt vet

BINARY_NAME=yamldiff
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/yamldiff

install:
	go install $(LDFLAGS) ./cmd/yamldiff

clean:
	rm -f $(BINARY_NAME)
	go clean

test:
	go test -v ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

# Run with test files
demo: build
	@echo "=== Demo: Comparing test-old.yaml and test-new.yaml ==="
	@./$(BINARY_NAME) test-old.yaml test-new.yaml || true

demo-counts: build
	@echo "=== Demo with counts only ==="
	@./$(BINARY_NAME) -c test-old.yaml test-new.yaml || true

# Development helpers
dev-deps:
	go mod download
	go mod tidy
