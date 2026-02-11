# =============================================================================
# BUILD & INSTALL TARGETS
# =============================================================================

.PHONY: build install quick-install dev deps clean prepare-embed bootstrap-content

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

# Bootstrap content bundle for ailang_bootstrap plugin repo
bootstrap-content: build ## Generate content bundle for bootstrap plugin sync
	@echo "Generating bootstrap content bundle..."
	@rm -rf $(BUILD_DIR)/bootstrap-content
	@mkdir -p $(BUILD_DIR)/bootstrap-content/examples $(BUILD_DIR)/bootstrap-content/examples/runnable
	@# 1a. Active teaching prompt (language syntax)
	@ACTIVE=$$(python3 -c "import json; d=json.load(open('prompts/versions.json')); print(d['active'])"); \
		cp "prompts/$${ACTIVE}.md" $(BUILD_DIR)/bootstrap-content/teaching-prompt.md; \
		echo "  $(ARROW) Teaching prompt: $${ACTIVE}"
	@# 1b. Active dev tools prompt (toolchain reference)
	@DEVTOOLS=$$(python3 -c "import json; d=json.load(open('prompts/devtools/versions.json')); print(d['active'])"); \
		cp "prompts/devtools/$${DEVTOOLS}.md" $(BUILD_DIR)/bootstrap-content/devtools-prompt.md; \
		echo "  $(ARROW) Devtools prompt: $${DEVTOOLS}"
	@# 2. Examples manifest
	@cp examples/manifest.json $(BUILD_DIR)/bootstrap-content/manifest.json
	@echo "  $(ARROW) Examples manifest"
	@# 3. Working example files (top-level)
	@for f in examples/*.ail; do \
		cp "$$f" $(BUILD_DIR)/bootstrap-content/examples/; \
	done
	@echo "  $(ARROW) Top-level examples"
	@# 4. Runnable examples
	@if [ -d examples/runnable ]; then \
		for f in examples/runnable/*.ail; do \
			[ -f "$$f" ] && cp "$$f" $(BUILD_DIR)/bootstrap-content/examples/runnable/; \
		done; \
		echo "  $(ARROW) Runnable examples"; \
	fi
	@# 5. Builtins list
	@$(BUILD_DIR)/$(BINARY) builtins list --by-module > $(BUILD_DIR)/bootstrap-content/builtins-by-module.txt 2>/dev/null || \
		echo "(builtins list not available)" > $(BUILD_DIR)/bootstrap-content/builtins-by-module.txt
	@echo "  $(ARROW) Builtins reference"
	@# 6. Recent changelog
	@head -100 CHANGELOG.md > $(BUILD_DIR)/bootstrap-content/changelog-recent.md
	@echo "  $(ARROW) Recent changelog"
	@# 7. Version
	@echo "$(VERSION)" > $(BUILD_DIR)/bootstrap-content/version.txt
	@echo "  $(ARROW) Version: $(VERSION)"
	@# 8. Create tarball
	@cd $(BUILD_DIR) && tar czf bootstrap-content.tar.gz bootstrap-content/
	@echo "$(GREEN)$(CHECKMARK) Bootstrap content bundle: $(BUILD_DIR)/bootstrap-content.tar.gz$(RESET)"
