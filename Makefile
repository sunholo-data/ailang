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
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY) cmd/ailang/main.go
	@echo "Installed to $(GOPATH)/bin/$(BINARY)"

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

# Watch mode for development
watch:
	@echo "Starting watch mode..."
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch (macOS) or apt-get install fswatch (Linux)" && exit 1)
	fswatch -o internal cmd | xargs -n1 -I{} make build

# Quick development build (no optimization)
dev:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) cmd/ailang/main.go

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
	@echo "  make watch         - Watch mode for development"
	@echo "  make dev           - Quick development build"
	@echo "  make help          - Show this help"