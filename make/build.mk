# =============================================================================
# BUILD & INSTALL TARGETS
# =============================================================================

.PHONY: build install quick-install dev deps clean prepare-embed bootstrap-content microrag-mcp-build microrag-mcp-install

prepare-embed: ## Internal: Copy prompts and scorecard for embedding
	@if [ ! -d cmd/ailang/prompts ] || [ prompts/versions.json -nt cmd/ailang/prompts/versions.json ]; then \
		echo "Copying prompts for embedding..."; \
		rm -rf cmd/ailang/prompts; \
		cp -r prompts cmd/ailang/prompts; \
	fi
	@if [ docs/static/benchmarks/axiom_scorecard.json -nt cmd/ailang/axiom_scorecard.json ]; then \
		echo "Copying axiom scorecard for embedding..."; \
		cp docs/static/benchmarks/axiom_scorecard.json cmd/ailang/axiom_scorecard.json; \
	fi

build: prepare-embed ## Build the ailang binary to bin/
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/ailang
	@echo "Build complete: $(BUILD_DIR)/$(BINARY)"

install: prepare-embed ## Install ailang to GOPATH/bin + ~/.local/bin symlink (opencode-compat)
	@echo "Installing $(BINARY)..."
	@go install $(LDFLAGS) ./cmd/ailang
	@echo "$(GREEN)$(CHECKMARK) Installed to $$(go env GOPATH)/bin/$(BINARY)$(RESET)"
	@# Symlink into ~/.local/bin for tools like opencode that use a sanitized
	@# child-shell PATH (it doesn't include $$GOPATH/bin). Without this, the
	@# bash tool in opencode sessions fails with "command not found: ailang"
	@# and models fall into pathological filesystem-search loops trying to
	@# locate the binary. See 2026-05-23 incident in
	@# .claude/skills/local-ollama-eval/SKILL.md.
	@mkdir -p ~/.local/bin
	@ln -sf "$$(go env GOPATH)/bin/$(BINARY)" ~/.local/bin/$(BINARY)
	@echo "$(GREEN)$(CHECKMARK) Symlinked to ~/.local/bin/$(BINARY) (for opencode-compat)$(RESET)"
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

quick-install: prepare-embed ## Quick install (GOPATH/bin + ~/.local/bin symlink)
	@go install $(LDFLAGS) ./cmd/ailang
	@mkdir -p ~/.local/bin
	@ln -sf "$$(go env GOPATH)/bin/$(BINARY)" ~/.local/bin/$(BINARY)
	@echo "$(GREEN)$(CHECKMARK) ailang updated in $$(go env GOPATH)/bin (+ ~/.local/bin symlink)$(RESET)"

microrag-mcp-build: ## Build the μRAG MCP server (cmd/ailang-microrag-mcp)
	@echo "Building ailang-microrag-mcp..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/ailang-microrag-mcp ./cmd/ailang-microrag-mcp
	@echo "$(GREEN)$(CHECKMARK) Built $(BUILD_DIR)/ailang-microrag-mcp$(RESET)"

microrag-mcp-install: ## Install ailang-microrag-mcp to GOPATH/bin
	@go install $(LDFLAGS) ./cmd/ailang-microrag-mcp
	@echo "$(GREEN)$(CHECKMARK) ailang-microrag-mcp installed in $$(go env GOPATH)/bin$(RESET)"

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
	@# 1c. Active agent prompt (minimal agent coding guide)
	@AGENT=$$(python3 -c "import json; d=json.load(open('prompts/agent/versions.json')); print(d['active'])"); \
		cp "prompts/agent/$${AGENT}.md" $(BUILD_DIR)/bootstrap-content/agent-prompt.md; \
		echo "  $(ARROW) Agent prompt: $${AGENT}"
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
