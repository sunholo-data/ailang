.PHONY: build test run clean install fmt vet lint deps

# Binary name
BINARY=ailang
VERSION=0.1.0
BUILD_DIR=bin

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

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

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -cover -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Vet complete"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
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

# Show help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make install       - Install binary to GOPATH/bin"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make lint          - Run linter"
	@echo "  make deps          - Download dependencies"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make repl          - Start the REPL"
	@echo "  make run FILE=...  - Run an AILANG file"
	@echo "  make watch         - Watch mode (local build)"
	@echo "  make watch-install - Watch mode (auto-install to PATH)"
	@echo "  make dev           - Quick development build"
	@echo "  make quick-install - Quick install without version info"
	@echo "  make help          - Show this help"