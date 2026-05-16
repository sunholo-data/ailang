# =============================================================================
# AILANG MAKEFILE
# =============================================================================
#
# Run 'make' or 'make help' to see available targets organized by category.
# Run 'make help-<category>' for detailed help on specific categories.
#
# Categories:
#   build     - Building and installing
#   test      - Testing
#   coverage  - Code coverage
#   eval      - Benchmarks & evaluation
#   docs      - Documentation
#   services  - Server & coordinator
#   examples  - Example verification
#   health    - Code quality & organization
#   claude    - Claude CLI integration
#   ci        - CI/CD targets
#
# =============================================================================

# Configuration
BINARY := ailang
BUILD_DIR := bin

# Version from git
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Build flags
LDFLAGS := -ldflags "-X github.com/sunholo-data/ailang/internal/version.Version=$(VERSION) -X github.com/sunholo-data/ailang/internal/version.Commit=$(COMMIT) -X github.com/sunholo-data/ailang/internal/version.BuildTime=$(BUILD_TIME)"

# Colors and symbols (for pretty output)
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[0;33m
CYAN := \033[0;36m
BOLD := \033[1m
RESET := \033[0m
CHECKMARK := ✓
CROSS := ✗
WARNING := ⚠️
ARROW := →
LINE := ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Stdlib paths (for freeze/verify)
STDLIB := std/option.ail std/result.ail std/list.ail std/string.ail std/io.ail
FREEZE_DIR := goldens/stdlib
TOOLS := ailang

# Export for sub-Makefiles
export BINARY BUILD_DIR VERSION COMMIT BUILD_TIME
export GOCMD GOBUILD GOCLEAN GOTEST GOGET GOMOD GOFMT GOVET LDFLAGS
export GREEN RED YELLOW CYAN BOLD RESET CHECKMARK CROSS WARNING ARROW LINE
export STDLIB FREEZE_DIR TOOLS

# =============================================================================
# INCLUDE SUB-MAKEFILES
# =============================================================================

include make/build.mk
include make/test.mk
include make/coverage.mk
include make/eval.mk
include make/docs.mk
include make/services.mk
include make/examples.mk
include make/code-health.mk
include make/claude.mk
include make/ci.mk

# =============================================================================
# HELP SYSTEM
# =============================================================================

.PHONY: help help-build help-test help-coverage help-eval help-docs help-services
.PHONY: help-examples help-health help-claude help-ci help-release

# Default help - show categories
help: ## Show this help
	@echo ""
	@echo "$(BOLD)AILANG Makefile$(RESET) - $(VERSION)"
	@echo "$(LINE)"
	@echo ""
	@echo "$(BOLD)Quick Start:$(RESET)"
	@echo "  make build          Build ailang binary"
	@echo "  make install        Install to PATH"
	@echo "  make test           Run all tests"
	@echo "  make repl           Start REPL"
	@echo ""
	@echo "$(BOLD)Categories:$(RESET) (use 'make help-<category>' for details)"
	@echo ""
	@echo "  $(CYAN)build$(RESET)      Building, installing, dependencies"
	@echo "  $(CYAN)test$(RESET)       Unit tests, integration tests, fuzzing"
	@echo "  $(CYAN)coverage$(RESET)   Code coverage reports and gates"
	@echo "  $(CYAN)eval$(RESET)       AI benchmarks and evaluation"
	@echo "  $(CYAN)docs$(RESET)       Documentation, website, package docs"
	@echo "  $(CYAN)services$(RESET)   Server, coordinator, REPL"
	@echo "  $(CYAN)examples$(RESET)   Example verification and README updates"
	@echo "  $(CYAN)health$(RESET)     Code quality, linting, file sizes"
	@echo "  $(CYAN)claude$(RESET)     Claude CLI integration"
	@echo "  $(CYAN)ci$(RESET)         CI/CD targets"
	@echo ""
	@echo "$(BOLD)All Targets:$(RESET) (sorted alphabetically)"
	@echo ""
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-build: ## Show build targets
	@echo ""
	@echo "$(BOLD)Build & Install Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/build.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-test: ## Show test targets
	@echo ""
	@echo "$(BOLD)Testing Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/test.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-coverage: ## Show coverage targets
	@echo ""
	@echo "$(BOLD)Coverage Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/coverage.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-eval: ## Show evaluation targets
	@echo ""
	@echo "$(BOLD)Evaluation & Benchmark Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/eval.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-docs: ## Show documentation targets
	@echo ""
	@echo "$(BOLD)Documentation Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/docs.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-services: ## Show service targets
	@echo ""
	@echo "$(BOLD)Service Management Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/services.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-examples: ## Show example targets
	@echo ""
	@echo "$(BOLD)Example Verification Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/examples.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-health: ## Show code health targets
	@echo ""
	@echo "$(BOLD)Code Health & Quality Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/code-health.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-claude: ## Show Claude CLI targets
	@echo ""
	@echo "$(BOLD)Claude CLI Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/claude.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-ci: ## Show CI targets
	@echo ""
	@echo "$(BOLD)CI/CD Targets$(RESET)"
	@echo "$(LINE)"
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' make/ci.mk | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2}'

help-release: ## Show release workflow
	@echo ""
	@echo "$(BOLD)Release Workflow$(RESET)"
	@echo "$(LINE)"
	@echo ""
	@echo "$(BOLD)Step 1:$(RESET) Run baseline evaluation"
	@echo "  make eval-baseline EVAL_VERSION=v0.X.Y              # 3 dev models (~\$$0.22)"
	@echo "  make eval-baseline EVAL_VERSION=v0.X.Y FULL=true    # All 6 models (~\$$1.50)"
	@echo ""
	@echo "$(BOLD)Step 2:$(RESET) Update website dashboard"
	@echo "  ailang eval-report eval_results/baselines/v0.X.Y v0.X.Y --format=json"
	@echo ""
	@echo "$(BOLD)Step 3:$(RESET) Clear Docusaurus cache and restart"
	@echo "  make docs-restart"
	@echo ""
	@echo "$(BOLD)Using Skills:$(RESET) (recommended)"
	@echo "  - Use 'release-manager' skill for full release workflow"
	@echo "  - Use 'post-release' skill to update dashboard"
	@echo ""

# =============================================================================
# CODEGEN HARNESS
# =============================================================================

## test-codegen: Compile multi-module harness to Go and run go build (M-CODEGEN-SUSTAINABILITY)
test-codegen: build
	@echo "$(BOLD)Running codegen harness test...$(RESET)"
	@rm -rf tests/codegen-harness/gen
	@./bin/ailang compile -emit-go -out tests/codegen-harness/gen -relax-modules tests/codegen-harness/*.ail
	@cd tests/codegen-harness/gen && go build ./...
	@echo "$(GREEN)✓ Codegen harness: compile + go build passed$(RESET)"
	@rm -rf tests/codegen-harness/gen

# =============================================================================
# MICRO-RAG (μRAG) CORPUS INDEXER
# =============================================================================

## brain-index-syntax: Index AILANG syntax/builtins/examples into the brain (additive, idempotent)
brain-index-syntax: build
	@echo "$(BOLD)Indexing μRAG corpora (active prompt resolved at runtime)...$(RESET)"
	@AILANG_BIN=./bin/ailang ./tools/index_ailang_syntax.sh

## brain-index-syntax-reset: Drop and rebuild the μRAG namespaces (used at release time)
brain-index-syntax-reset: build
	@echo "$(BOLD)Resetting and rebuilding μRAG corpora...$(RESET)"
	@AILANG_BIN=./bin/ailang ./tools/index_ailang_syntax.sh --reset

# =============================================================================
# AGENT MCP SERVER (M-AGENT-MCP)
# =============================================================================

## snapshot: Build the MCP snapshot tree from current repo state (M2)
snapshot: bin/build-snapshot
	@./bin/build-snapshot --repo . --out build/snapshot

## verify-mcp-tools: Type-check every AILANG file under mcp_tools/ (M-AGENT-MCP M8)
verify-mcp-tools: build
	@echo "$(BOLD)Verifying mcp_tools/...$(RESET)"
	@FAIL=0; for f in mcp_tools/*.ail; do \
		AILANG_RELAX_MODULES=1 ./bin/ailang check $$f > /dev/null 2>&1 \
			&& echo "  ✓ $$f" \
			|| { echo "  ✗ $$f"; FAIL=1; }; \
	done; \
	if [ $$FAIL -ne 0 ]; then \
		echo "$(BOLD)mcp_tools/ verification FAILED$(RESET)"; exit 1; \
	fi
	@echo "$(BOLD)✓ All mcp_tools/ AILANG modules type-check$(RESET)"

## test-feedback-integration: Run integration tests for submit_feedback against test env Firestore (M-PKG-FEEDBACK-LOOP M1)
test-feedback-integration:
	@echo "$(BOLD)Running submit_feedback integration tests against ailang-multivac-test...$(RESET)"
	@AILANG_STORAGE=gcp AILANG_CLOUD_PROJECT=ailang-multivac-test AILANG_TOPIC_PREFIX=ailang-test \
		go test -tags integration -count=1 -v ./internal/feedback/

## verify-install-guide: Drift-check install_guide_overrides.json against canonical sources (M-AGENT-MCP-ONBOARDING M2)
verify-install-guide:
	@echo "$(BOLD)Verifying install_guide_overrides.json against canonical sources...$(RESET)"
	@OVERRIDES=tools/build-snapshot/install_guide_overrides.json; \
	GETTING_STARTED=docs/docs/guides/getting-started.mdx; \
	BOOTSTRAP_README=../ailang_bootstrap/README.md; \
	FAIL=0; \
	for cmd in "/plugin marketplace add sunholo-data/ailang_bootstrap" "gemini extensions install" "codex plugin add"; do \
		if ! grep -q -F "$$cmd" $$OVERRIDES 2>/dev/null; then \
			echo "  ✗ override missing: $$cmd"; FAIL=1; \
		else \
			echo "  ✓ override present: $$cmd"; \
		fi; \
	done; \
	if [ -f "$$GETTING_STARTED" ]; then \
		for cmd in "/plugin marketplace add" "gemini extensions install" "codex plugin add"; do \
			if grep -q -F "$$cmd" $$GETTING_STARTED && ! grep -q -F "$$cmd" $$OVERRIDES; then \
				echo "  ✗ canonical $$GETTING_STARTED has '$$cmd' but overrides do not"; FAIL=1; \
			fi; \
		done; \
	else \
		echo "  ⚠ canonical $$GETTING_STARTED missing (skipping that diff)"; \
	fi; \
	if [ $$FAIL -ne 0 ]; then \
		echo "$(BOLD)install_guide drift detected — update install_guide_overrides.json$(RESET)"; exit 1; \
	fi
	@echo "$(BOLD)✓ install_guide_overrides.json is in sync with canonical sources$(RESET)"

## error-codes: Generate dist/error_codes.json — machine-readable registry of all AILANG error codes
error-codes:
	@echo "$(BOLD)Generating dist/error_codes.json...$(RESET)"
	@mkdir -p dist
	@go run ./tools/gen-error-codes/ internal/errors/codes.go dist/error_codes.json
	@echo "$(GREEN)✓ dist/error_codes.json generated$(RESET)"

## vscode-extension-bundle: Rebuild the AILANG VS Code extension bundle (extension.js)
##                          embedded into the ailang binary. Run after editing
##                          tools/vscode-extension-build/extension-src.js or bumping
##                          its vscode-languageclient dependency.
vscode-extension-bundle:
	@echo "$(BOLD)Bundling AILANG VS Code extension...$(RESET)"
	@bash tools/vscode-extension-build/bundle.sh
	@echo "$(GREEN)✓ cmd/ailang/editor_assets/vscode/extension.js updated$(RESET)"
	@echo "  Remember to re-run 'make install' so the new bundle is embedded in your local ailang binary."

## bin/build-snapshot: Build the snapshot tool
bin/build-snapshot: tools/build-snapshot/main.go
	@echo "$(BOLD)Building build-snapshot tool...$(RESET)"
	@go build -o bin/build-snapshot ./tools/build-snapshot/

## install-notify-daemon: Install the macOS notification daemon under launchd (env=ENV, default prod)
install-notify-daemon: build
	@echo "$(BOLD)Installing ailang notify daemon (env=$${ENV:-prod})...$(RESET)"
	@./bin/ailang daemon install --env $${ENV:-prod} --binary $$(pwd)/bin/ailang
	@echo "$(BOLD)✓ Daemon installed. Status:$(RESET)"
	@./bin/ailang daemon status

## uninstall-notify-daemon: Remove the macOS notification daemon
uninstall-notify-daemon: build
	@./bin/ailang daemon uninstall

## smoke-notify-daemon: End-to-end smoke test against ailang-multivac-dev
smoke-notify-daemon: build
	@./scripts/smoke-notify-daemon.sh

## mcp-local: Run the MCP server locally against the build/snapshot/ directory
mcp-local: build snapshot
	@echo "$(BOLD)Starting MCP server on http://localhost:8080/mcp/ ...$(RESET)"
	@MCP_SNAPSHOT_DIR=$$(pwd)/build/snapshot AILANG_RELAX_MODULES=1 \
		./bin/ailang serve-api --mcp-http --routes-only \
		--caps FS,Env --port 8080 ./mcp_tools/

# =============================================================================
# DEFAULT TARGET
# =============================================================================

.DEFAULT_GOAL := help
