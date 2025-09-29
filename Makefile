.PHONY: build test run clean install fmt vet lint deps verify-examples update-readme test-coverage-badge flag-broken

# Binary name
BINARY=ailang
BUILD_DIR=bin

# Version handling - get from git tag or use dev version
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags with version info
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: test build

# Build the binary
build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) cmd/ailang/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY)"

# Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY)..."
	@go install $(LDFLAGS) ./cmd/ailang
	@echo "✓ Installed to $$(go env GOPATH)/bin/$(BINARY)"
	@echo ""
	@if echo "$$PATH" | grep -q "$$(go env GOPATH)/bin"; then \
		echo "✓ Your PATH is correctly configured"; \
		echo "  You can now run 'ailang' from anywhere!"; \
	else \
		echo "⚠️  WARNING: $$(go env GOPATH)/bin is not in your PATH"; \
		echo ""; \
		echo "  To use 'ailang' from anywhere, add this to your shell profile:"; \
		echo "  export PATH=\"$$(go env GOPATH)/bin:\$$PATH\""; \
		echo ""; \
		echo "  For zsh (~/.zshrc):"; \
		echo "    echo 'export PATH=\"$$(go env GOPATH)/bin:\$$PATH\"' >> ~/.zshrc"; \
		echo "    source ~/.zshrc"; \
		echo ""; \
		echo "  For bash (~/.bashrc or ~/.bash_profile):"; \
		echo "    echo 'export PATH=\"$$(go env GOPATH)/bin:\$$PATH\"' >> ~/.bashrc"; \
		echo "    source ~/.bashrc"; \
	fi

# Run tests (excluding scripts directory which contains standalone executables)
test:
	@echo "Running tests..."
	@$(GOTEST) -v $$($(GOCMD) list ./... | grep -v /scripts)

# Run tests with coverage (excluding scripts directory)
test-coverage:
	@echo "Running tests with coverage..."
	@$(GOTEST) -v -cover -coverprofile=coverage.out $$($(GOCMD) list ./... | grep -v /scripts)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Code formatted"

# Check code formatting (for CI)
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Go code is not formatted. Please run 'make fmt'"; \
		echo "Files that need formatting:"; \
		gofmt -l .; \
		exit 1; \
	fi
	@echo "Code formatting check passed"

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

# Install golangci-lint
install-lint:
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2
	@echo "golangci-lint installed"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with 'make install-lint' or 'brew install golangci-lint'" && exit 1)
	golangci-lint run
	@echo "Lint complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -f examples_report.json examples_status.md coverage.txt
	@echo "Clean complete"

# Run the REPL
repl: build
	@$(BUILD_DIR)/$(BINARY) repl

# Run an AILANG file
run: build
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make run FILE=path/to/file.ail"; \
		exit 1; \
	fi
	@$(BUILD_DIR)/$(BINARY) run $(FILE)

# Watch mode for development (rebuilds to bin/)
watch:
	@echo "Starting watch mode (local build)..."
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch (macOS) or apt-get install fswatch (Linux)" && exit 1)
	fswatch -o internal cmd | xargs -n1 -I{} make build

# Watch and install mode (auto-installs to GOPATH/bin on changes)
watch-install:
	@echo "Starting watch mode (auto-install)..."
	@echo "ailang will be automatically updated in $$(go env GOPATH)/bin on every change"
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch (macOS) or apt-get install fswatch (Linux)" && exit 1)
	fswatch -o internal cmd examples | xargs -n1 -I{} sh -c 'clear && echo "🔄 Rebuilding and installing..." && make install && echo "✓ ailang updated!" || echo "❌ Build failed"'

# Quick development build (no optimization)
dev:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) cmd/ailang/main.go

# Quick install (useful for development)
quick-install:
	@go install ./cmd/ailang
	@echo "✓ ailang updated in $$(go env GOPATH)/bin"

# Verify all examples
verify-examples: build
	@echo "Verifying examples..."
	@go run ./scripts/verify_examples.go --json > examples_report.json 2>&1 || true
	@go run ./scripts/verify_examples.go --markdown > examples_status.md 2>&1 || true
	@cat examples_status.md

# Update README with example status
update-readme: build
	@echo "Verifying examples..."
	@go run ./scripts/verify_examples.go --json > examples_report.json 2>&1 || true
	@go run ./scripts/verify_examples.go --markdown > examples_status.md 2>&1 || true
	@cat examples_status.md
	@echo "Updating README with example status..."
	@go run ./scripts/update_readme.go

# Generate test coverage badge
test-coverage-badge:
	@echo "Generating coverage badge..."
	@$(GOTEST) -coverprofile=coverage.out ./... > /dev/null 2>&1 || true
	@go tool cover -func=coverage.out | grep total: | awk '{print $$3}' | sed 's/%//' > coverage.txt
	@echo "Coverage: $$(cat coverage.txt)%"

# Flag broken examples with warning headers
flag-broken: verify-examples
	@echo "Flagging broken examples..."
	@go run ./scripts/flag_broken_examples.go

# CI verification target  
ci: deps fmt-check vet lint test test-coverage-badge verify-examples
	@echo "CI verification complete"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build            - Build the binary"
	@echo "  make install          - Install binary to GOPATH/bin"
	@echo "  make test             - Run tests"
	@echo "  make test-coverage    - Run tests with coverage"
	@echo "  make verify-examples  - Verify all examples"
	@echo "  make flag-broken      - Add warning headers to broken examples"
	@echo "  make update-readme    - Update README with example status"
	@echo "  make ci               - Run full CI verification"
	@echo "  make fmt              - Format code"
	@echo "  make fmt-check        - Check code formatting"
	@echo "  make vet              - Run go vet"
	@echo "  make lint             - Run linter"
	@echo "  make install-lint     - Install golangci-lint"
	@echo "  make deps             - Download dependencies"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make repl             - Start the REPL"
	@echo "  make run FILE=...     - Run an AILANG file"
	@echo "  make watch            - Watch mode (local build)"
	@echo "  make watch-install    - Watch mode (auto-install to PATH)"
	@echo "  make dev              - Quick development build"
	@echo "  make quick-install    - Quick install without version info"
	@echo "  make help             - Show this help"