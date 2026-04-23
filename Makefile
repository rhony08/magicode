.PHONY: build clean test run install lint fmt help

# Binary name
BINARY_NAME := opencode
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR := ./bin
GO := go
GOFLAGS := -v

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

# Default target
help:
	@echo "OpenCode CLI - Go Implementation"
	@echo ""
	@echo "Usage:"
	@echo "  make build        Build the binary"
	@echo "  make run          Run the CLI"
	@echo "  make test         Run tests"
	@echo "  make lint         Run linter"
	@echo "  make fmt          Format code"
	@echo "  make clean        Clean build artifacts"
	@echo "  make install      Install binary"
	@echo ""

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/opencode
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/opencode
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/opencode
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/opencode
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/opencode
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/opencode
	@echo "Built all platforms"

# Run the CLI
run:
	$(GO) run ./cmd/opencode

# Run with arguments
run-args:
	$(GO) run ./cmd/opencode $(ARGS)

# Run tests
test:
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		$(GO) vet ./...; \
	fi

# Format code
fmt:
	$(GO) fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean

# Install binary to system
install: build
	@echo "Installing..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installed: /usr/local/bin/$(BINARY_NAME)"

# Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Update dependencies
update-deps:
	$(GO) get -u ./...
	$(GO) mod tidy

# Generate documentation
docs:
	$(GO) doc -all ./internal/... > docs/api.md

# Development mode with hot reload (requires reflex)
dev:
	@if command -v reflex >/dev/null 2>&1; then \
		reflex -g '*.go' -s -- sh -c 'make build && make run'; \
	else \
		@echo "reflex not installed. Install with: go install github.com/cespare/reflex@latest"; \
	fi

# Memory profiling
profile-memory:
	$(GO) test -memprofile=mem.prof ./internal/instance
	$(GO) tool pprof -http=:8080 mem.prof

# Benchmark tests
benchmark:
	$(GO) test -bench=. -benchmem ./internal/...