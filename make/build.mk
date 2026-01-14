# =============================================================================
# BUILD & INSTALL TARGETS
# =============================================================================

.PHONY: build install quick-install dev deps clean prepare-embed

prepare-embed: ## Internal: Copy prompts for embedding
	@if [ ! -d cmd/ailang/prompts ] || [ prompts/versions.json -nt cmd/ailang/prompts/versions.json ]; then \
		echo "Copying prompts for embedding..."; \
		rm -rf cmd/ailang/prompts; \
		cp -r prompts cmd/ailang/prompts; \
	fi

build: prepare-embed ## Build the ailang binary to bin/
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/ailang
	@echo "Build complete: $(BUILD_DIR)/$(BINARY)"

install: prepare-embed ## Install ailang to GOPATH/bin (with version info)
	@echo "Installing $(BINARY)..."
	@go install $(LDFLAGS) ./cmd/ailang
	@echo "$(GREEN)$(CHECKMARK) Installed to $$(go env GOPATH)/bin/$(BINARY)$(RESET)"
	@echo ""
	@if echo "$$PATH" | grep -q "$$(go env GOPATH)/bin"; then \
		echo "$(GREEN)$(CHECKMARK) Your PATH is correctly configured$(RESET)"; \
		echo "  You can now run 'ailang' from anywhere!"; \
	else \
		echo "$(YELLOW)$(WARNING) WARNING: $$(go env GOPATH)/bin is not in your PATH$(RESET)"; \
		echo ""; \
		echo "  To use 'ailang' from anywhere, add this to your shell profile:"; \
		echo "  export PATH=\"$$(go env GOPATH)/bin:\$$PATH\""; \
	fi

quick-install: prepare-embed ## Quick install without version info (faster)
	@go install ./cmd/ailang
	@echo "$(GREEN)$(CHECKMARK) ailang updated in $$(go env GOPATH)/bin$(RESET)"

dev: ## Quick development build (no optimization)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY) cmd/ailang/main.go

deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies downloaded"

clean: ## Clean build artifacts and coverage files
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf cmd/ailang/prompts
	rm -f coverage.out coverage.html
	rm -f coverage.parser.out coverage.lexer.out
	rm -f .parser_coverage .lexer_coverage
	rm -f .golden_changes
	rm -f examples_report.json examples_status.md coverage.txt
	@echo "Clean complete"

# Watch modes
watch: ## Watch mode - rebuild on changes (local bin/)
	@echo "Starting watch mode (local build)..."
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch" && exit 1)
	fswatch -o internal cmd | xargs -n1 -I{} make build

watch-install: ## Watch mode - auto-install on changes
	@echo "Starting watch mode (auto-install)..."
	@which fswatch > /dev/null || (echo "fswatch not found. Install with: brew install fswatch" && exit 1)
	fswatch -o internal cmd examples | xargs -n1 -I{} sh -c 'clear && echo "Rebuilding..." && make install && echo "$(GREEN)$(CHECKMARK) ailang updated!$(RESET)" || echo "$(RED)$(CROSS) Build failed$(RESET)"'

# WASM build
build-wasm: ## Build WASM binary for browser REPL
	@echo "Building WASM binary..."
	@mkdir -p $(BUILD_DIR)
	GOOS=js GOARCH=wasm $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY).wasm ./cmd/wasm
	@echo "$(GREEN)$(CHECKMARK) WASM binary: $(BUILD_DIR)/$(BINARY).wasm$(RESET)"
