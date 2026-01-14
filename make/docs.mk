# =============================================================================
# DOCUMENTATION TARGETS
# =============================================================================

.PHONY: docs docs-install docs-serve docs-build docs-preview docs-clean docs-restart
.PHONY: sync-prompts sync-versions generate-llms-txt
.PHONY: doctor doc

# Documentation generation
docs: sync-prompts sync-versions generate-llms-txt ## Generate all documentation
	@echo "$(GREEN)$(CHECKMARK) All documentation generated$(RESET)"

sync-prompts: ## Sync prompts/ to docs/docs/prompts/ (Docusaurus)
	@echo "Syncing prompts to docs..."
	@./docs/scripts/sync-prompts.sh

sync-versions: ## Sync version constants
	@echo "Syncing version constants..."
	@bash docs/scripts/generate-version-constants.sh

generate-llms-txt: ## Generate llms.txt
	@./tools/generate-llms-txt.sh

# Docusaurus website
docs-install: ## Install Docusaurus dependencies
	@echo "Installing Docusaurus dependencies..."
	@cd docs && npm install

docs-serve: ## Start Docusaurus dev server (localhost:3000)
	@echo "Starting Docusaurus development server..."
	@echo "Website: http://localhost:3000/ailang/"
	@cd docs && npm start

docs-build: build-wasm ## Build Docusaurus production site
	@echo "Copying WASM assets to docs..."
	@mkdir -p docs/static/wasm docs/static/js docs/src/components
	@cp bin/ailang.wasm docs/static/wasm/
	@curl -sL -o docs/static/wasm/wasm_exec.js https://raw.githubusercontent.com/golang/go/go1.22.0/misc/wasm/wasm_exec.js
	@cp web/ailang-repl.js docs/static/js/
	@cp web/AilangRepl.jsx docs/src/components/
	@echo "Building Docusaurus site..."
	@cd docs && npm run build

docs-preview: docs docs-build ## Build and serve production site
	@echo "Serving production build..."
	@cd docs && npm run serve

docs-clean: ## Clear Docusaurus cache
	@echo "Cleaning Docusaurus cache..."
	@cd docs && npm run clear
	@rm -rf docs/build docs/.docusaurus

docs-restart: docs-clean ## Clear cache and restart dev server
	@echo "Restarting Docusaurus..."
	@cd docs && npm start

# Developer documentation
doctor: build ## Run builtin registry validation
	@echo "Running builtin registry validation..."
	@AILANG_BUILTINS_REGISTRY=1 $(BUILD_DIR)/$(BINARY) doctor builtins

doc: ## Show Go package documentation
	@if [ -z "$(PKG)" ]; then \
		echo "Usage: make doc PKG=<package>"; \
		echo ""; \
		echo "Examples:"; \
		echo "  make doc PKG=internal/parser"; \
		echo "  make doc PKG=internal/types"; \
		echo "  make doc PKG=internal/eval"; \
		echo ""; \
		echo "Common packages:"; \
		echo "  internal/testing    - Test collection and property-based testing"; \
		echo "  internal/elaborate  - Surface AST to Core AST elaboration"; \
		echo "  internal/types      - Type system and type checking"; \
		echo "  internal/parser     - Parser (see docs/guides/parser_development.md)"; \
		echo "  internal/eval       - Core AST evaluator"; \
		echo "  internal/builtins   - Builtin function registry"; \
		exit 1; \
	fi
	@go doc -all github.com/sunholo/ailang/$(PKG)
