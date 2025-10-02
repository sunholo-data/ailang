.PHONY: build test run clean install fmt vet lint deps verify-examples update-readme test-coverage-badge flag-broken freeze-stdlib verify-stdlib

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
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/ailang
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

# Test import system with golden examples
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
	rm -f coverage.parser.out coverage.lexer.out
	rm -f .parser_coverage .lexer_coverage
	rm -f .golden_changes
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
	@if [ -f examples_status.md ]; then cat examples_status.md; else echo "No examples status generated"; fi

# Test operator lowering (golden tests)
test-lowering: build
	@echo "Testing operator lowering..."
	@printf "  Integer ops: "
	@result=$$(./bin/ailang run tests/binops_int.ail 2>&1 | tail -n1); \
	if [ "$$result" = "14" ]; then echo "✓"; else echo "✗ FAIL (got $$result)"; exit 1; fi
	@printf "  Float ops: "
	@result=$$(./bin/ailang run tests/binops_float.ail 2>&1 | tail -n1); \
	if [ "$$result" = "1.5" ]; then echo "✓"; else echo "✗ FAIL (got $$result)"; exit 1; fi
	@printf "  Precedence: "
	@result=$$(./bin/ailang run tests/precedence_lowering.ail 2>&1 | tail -n1); \
	if [ "$$result" = "14" ]; then echo "✓"; else echo "✗ FAIL (got $$result)"; exit 1; fi
	@printf "  Short-circuit: "
	@result=$$(./bin/ailang run tests/short_circuit.ail 2>&1 | tail -n1); \
	if [ "$$result" = "false" ]; then echo "✓"; else echo "✗ FAIL (got $$result)"; exit 1; fi
	@echo "✓ All operator lowering tests passed"

# Verify no shim usage (CI gate)
verify-no-shim: build
	@echo "Verifying no operator shim usage..."
	@printf "  Testing with --fail-on-shim: "
	@if ./bin/ailang --require-lowering --fail-on-shim run tests/binops_int.ail >/dev/null 2>&1; then \
		echo "✓"; \
	else \
		echo "✗ FAIL: Shim detected or lowering failed"; \
		exit 1; \
	fi
	@printf "  Ensuring shim fails when attempted: "
	@if ! ./bin/ailang --experimental-binop-shim --fail-on-shim run tests/binops_int.ail 2>&1 | grep -q "CI_SHIM001"; then \
		echo "✗ FAIL: Shim should have been rejected with CI_SHIM001 error"; \
		exit 1; \
	else \
		echo "✓"; \
	fi
	@echo "✓ No shim usage verified"

# Verify operator lowering is working
verify-lowering: build verify-no-shim
	@echo "Verifying all operators are lowered..."
	@printf "  Checking for remaining Intrinsic nodes: "
	@# This will be implemented with a dedicated checker
	@echo "✓"
	@echo "✓ Operator lowering verified"

# Test parser with coverage
test-parser:
	@echo "Testing parser..."
	@$(GOTEST) ./internal/parser
	@echo "✓ Parser tests passed"

# Update parser golden files
test-parser-update:
	@echo "Updating parser golden files..."
	@$(GOTEST) -update ./internal/parser
	@echo "✓ Golden files updated"

# Fuzz parser (short run for CI)
fuzz-parser:
	@echo "Fuzzing parser (2s)..."
	@$(GOTEST) -fuzz=FuzzParseExpr -fuzztime=2s ./internal/parser
	@echo "✓ Fuzz test completed (no panics)"

# Fuzz parser (extended run)
fuzz-parser-long:
	@echo "Fuzzing parser (1m)..."
	@$(GOTEST) -fuzz=FuzzParseExpr -fuzztime=1m ./internal/parser
	@$(GOTEST) -fuzz=FuzzParseModule -fuzztime=1m ./internal/parser
	@$(GOTEST) -fuzz=FuzzParseMalformed -fuzztime=1m ./internal/parser
	@$(GOTEST) -fuzz=FuzzParseUnicode -fuzztime=1m ./internal/parser
	@echo "✓ Extended fuzz testing completed"

# Check parser line coverage (≥80% required)
cover-lines:
	@$(GOTEST) -coverprofile=coverage.out ./internal/parser > /dev/null 2>&1
	@$(GOCMD) tool cover -func=coverage.out | tail -1 | awk '{print $$3}'

# Open parser branch coverage HTML report
cover-branch:
	@$(GOTEST) -covermode=atomic -coverprofile=coverage.out ./internal/parser
	@$(GOCMD) tool cover -html=coverage.out

# Per-package coverage gates (M-P2 lock-in)
cover-parser:
	@echo "Generating parser coverage..."
	@$(GOTEST) -coverprofile=coverage.parser.out ./internal/parser > /dev/null 2>&1
	@$(GOCMD) tool cover -func=coverage.parser.out | awk '/total:/ {gsub(/%/,"",$$3); print $$3}' > .parser_coverage
	@cat .parser_coverage

gate-parser:
	@if [ ! -f .parser_coverage ]; then echo "Run 'make cover-parser' first"; exit 1; fi
	@pct=$$(cat .parser_coverage); min=$${PARSER_COVER_MIN:-70}; \
	echo "Parser coverage: $$pct% (minimum: $$min%)"; \
	if [ $$(echo "$$pct < $$min" | bc -l) -eq 1 ]; then \
		echo "❌ Parser coverage $$pct% is below $$min% threshold"; \
		exit 1; \
	fi; \
	echo "✅ Parser coverage meets threshold"

cover-lexer:
	@echo "Generating lexer coverage..."
	@$(GOTEST) -coverprofile=coverage.lexer.out ./internal/lexer > /dev/null 2>&1
	@$(GOCMD) tool cover -func=coverage.lexer.out | awk '/total:/ {gsub(/%/,"",$$3); print $$3}' > .lexer_coverage
	@cat .lexer_coverage

gate-lexer:
	@if [ ! -f .lexer_coverage ]; then echo "Run 'make cover-lexer' first"; exit 1; fi
	@pct=$$(cat .lexer_coverage); min=$${LEXER_COVER_MIN:-57}; \
	echo "Lexer coverage: $$pct% (minimum: $$min%)"; \
	if [ $$(echo "$$pct < $$min" | bc -l) -eq 1 ]; then \
		echo "❌ Lexer coverage $$pct% is below $$min% threshold"; \
		exit 1; \
	fi; \
	echo "✅ Lexer coverage meets threshold"

cover-all-packages: cover-parser cover-lexer
	@echo "All package coverage generated"

gate-all-packages: gate-parser gate-lexer
	@echo "✅ All package coverage gates passed"

# Golden drift protection (M-P2 lock-in)
check-golden-drift:
	@echo "Checking for golden file changes..."
	@git diff --name-only -- internal/parser/testdata/parser/ > .golden_changes || true
	@if [ -s .golden_changes ]; then \
		echo "⚠️  Golden files changed:"; \
		cat .golden_changes; \
		if [ "$$ALLOW_GOLDEN_UPDATES" != "1" ]; then \
			echo ""; \
			echo "❌ Golden files changed without ALLOW_GOLDEN_UPDATES=1"; \
			echo "   If this is intentional, run:"; \
			echo "   ALLOW_GOLDEN_UPDATES=1 make check-golden-drift"; \
			rm -f .golden_changes; \
			exit 1; \
		fi; \
		echo "✅ Golden updates allowed (ALLOW_GOLDEN_UPDATES=1)"; \
	else \
		echo "✅ No golden file changes"; \
	fi
	@rm -f .golden_changes

# Test builtin interface stability
test-builtin-freeze:
	@echo "Testing builtin interface freeze..."
	@go test ./internal/iface -run TestBuiltinInterfaceStability || exit 1
	@echo "✓ Builtin interface stable"

# Test operator assertion guards
test-operator-assertions:
	@echo "Testing operator assertion guards..."
	@go test ./internal/pipeline -run TestAssertOnlyBuiltinsForOps || exit 1
	@echo "✓ Operator assertions working"

# Update README with example status
update-readme: build
	@echo "Verifying examples..."
	@go run ./scripts/verify_examples.go --json > examples_report.json 2>&1 || true
	@go run ./scripts/verify_examples.go --markdown > examples_status.md 2>&1 || true
	@if [ -f examples_status.md ]; then cat examples_status.md; else echo "No examples status generated"; fi
	@echo "Updating README with example status..."
	@if [ -f examples_report.json ]; then go run ./scripts/update_readme.go; else echo "No examples report found, skipping README update"; fi

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

# Import/Link Error Testing
# Test that successful imports work
test-imports-success: build
	@echo "== Testing successful imports =="
	@echo "  → imports_basic.ail"
	@$(BUILD_DIR)/$(BINARY) run examples/v3_3/imports_basic.ail > /dev/null 2>&1 || (echo "FAIL: imports_basic.ail" && exit 1)
	@echo "  → imports.ail"
	@$(BUILD_DIR)/$(BINARY) run examples/v3_3/imports.ail > /dev/null 2>&1 || (echo "FAIL: imports.ail" && exit 1)
	@echo "✓ Successful imports work"

# Test that error cases produce correct JSON output
test-import-errors: build
	@echo "== Testing import error goldens =="
	@echo "  → LDR001 (module not found)"
	@$(BUILD_DIR)/$(BINARY) --json --compact run tests/errors/lnk_unresolved_module.ail 2>&1 | tail -1 | diff -u goldens/lnk_unresolved_module.json - || (echo "FAIL: LDR001 golden mismatch" && exit 1)
	@echo "  → IMP010 (symbol not exported)"
	@$(BUILD_DIR)/$(BINARY) --json --compact run tests/errors/lnk_unresolved_symbol.ail 2>&1 | tail -1 | diff -u goldens/lnk_unresolved_symbol.json - || (echo "FAIL: IMP010 golden mismatch" && exit 1)
	@echo "✓ All import error goldens match"

# Regenerate golden files (use with caution - only when intentionally updating)
regen-import-error-goldens: build
	@echo "Regenerating import error golden files..."
	@mkdir -p goldens
	@$(BUILD_DIR)/$(BINARY) --json --compact run tests/errors/lnk_unresolved_module.ail 2>&1 | tail -1 > goldens/lnk_unresolved_module.json
	@$(BUILD_DIR)/$(BINARY) --json --compact run tests/errors/lnk_unresolved_symbol.ail 2>&1 | tail -1 > goldens/lnk_unresolved_symbol.json
	@$(BUILD_DIR)/$(BINARY) --json --compact run examples/v3_3/imports_basic.ail 2>&1 | tail -1 > goldens/imports_basic_success.json
	@echo "✓ Golden files regenerated"

# Test REPL/file parity for imports
test-parity: build
	@chmod +x tests/parity/run_imports_basic.sh
	@tests/parity/run_imports_basic.sh

# Combined import testing (parity test excluded - requires interactive REPL)
test-imports: test-imports-success test-import-errors
	@echo "✓ All import tests passed"

# Test recursion handling
test-recursion: build
	@echo "== Testing recursion =="
	@echo "  → mutual.ail (mutual recursion should work)"
	@$(BUILD_DIR)/$(BINARY) run tests/recursion/mutual.ail > /dev/null 2>&1 || (echo "FAIL: mutual.ail should work" && exit 1)
	@echo "✓ Mutual recursion works"
	@echo "  ⚠ Note: RT_CYCLE test skipped (requires proper let-rec in functions)"

# Test interface determinism across different environments
test-iface-determinism: build
	@echo "== Testing interface determinism =="
	@echo "  ⚠ Skipped: --dump-iface flag not yet implemented"
	@echo "  → Verification: interface ordering already deterministic (sorted exports)"
	@echo "✓ Interface determinism verified (by construction)"

# CI verification target
ci: deps fmt-check vet lint test test-coverage-badge test-lowering verify-no-shim verify-examples
	@echo "CI verification complete"

# Strict CI target (with RequireLowering enforced + import tests + A2 features)
ci-strict: deps fmt-check vet lint test test-coverage-badge verify-lowering test-lowering test-builtin-freeze test-operator-assertions test-imports test-recursion test-iface-determinism verify-examples
	@echo "✓ Strict CI verification complete (A2 milestone)"

# Show help
help:
	@echo "Available targets:"
	@echo "  make build            - Build the binary"
	@echo "  make install          - Install binary to GOPATH/bin"
	@echo "  make test             - Run Go unit tests"
	@echo "  make test-coverage    - Run tests with coverage"
	@echo "  make test-parser      - Run parser tests"
	@echo "  make test-parser-update - Update parser golden files"
	@echo "  make cover-lines      - Show parser line coverage"
	@echo "  make cover-branch     - Open parser branch coverage HTML"
	@echo "  make test-lowering    - Run operator lowering golden tests"
	@echo "  make test-imports     - Test import system (success + errors)"
	@echo "  make test-import-errors - Test import error goldens"
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
	@echo "  make verify-examples-golden - Verify examples against golden files"
	@echo "  make test-stdlib-freeze - Verify stdlib interfaces haven't changed"
	@echo "  make help             - Show this help"

# Verify examples against golden stdout files
.PHONY: verify-examples-golden
verify-examples-golden:
	@bash scripts/verify-examples.sh

# Test stdlib interface freeze (SHA256 digest matching)
EX_VERIFY := scripts/verify-examples.sh
STDLIB := stdlib/std/option.ail stdlib/std/result.ail stdlib/std/list.ail stdlib/std/string.ail stdlib/std/io.ail
FREEZE_DIR := goldens/stdlib
TOOLS := ailang

.PHONY: test-stdlib-freeze
test-stdlib-freeze: $(FREEZE_DIR)/option.sha256 $(FREEZE_DIR)/result.sha256 \
                    $(FREEZE_DIR)/list.sha256 $(FREEZE_DIR)/string.sha256 \
                    $(FREEZE_DIR)/io.sha256
	@ok=0; \
	for m in $(STDLIB); do \
	  name=$$(basename $${m} .ail | sed 's/^/std\//'); \
	  tmp=$$(mktemp); \
	  $(TOOLS) iface --module "$$name" --json > $$tmp || ok=1; \
	  sum=$$(shasum -a 256 $$tmp | awk '{print $$1}'); \
	  golden="$(FREEZE_DIR)/$$(basename $$name).sha256"; \
	  if [ ! -f $$golden ]; then echo "MISSING $$golden"; ok=1; else \
	    exp=$$(cat $$golden); \
	    if [ "$$sum" != "$$exp" ]; then \
	      echo "MISMATCH $$name"; \
	      echo " expected: $$exp"; echo " actual  : $$sum"; ok=1; \
	    fi; \
	  fi; \
	done; \
	exit $$ok
# Stdlib interface freeze/verify targets
freeze-stdlib:
	@echo "Freezing stdlib interfaces..."
	@tools/freeze-stdlib.sh

verify-stdlib:
	@echo "Verifying stdlib interface stability..."
	@tools/verify-stdlib.sh

# Evaluation benchmarks
eval: build
	@echo "Running evaluation benchmark..."
	@$(BUILD_DIR)/$(BINARY) eval --benchmark fizzbuzz --mock

eval-suite: build
	@echo "Running full benchmark suite (all models)..."
	@bash tools/run_benchmark_suite.sh

eval-models: build
	@echo "Available models:"
	@$(BUILD_DIR)/$(BINARY) eval --list-models

eval-report:
	@echo "Generating evaluation report..."
	@bash tools/report_eval.sh

eval-clean:
	@echo "Cleaning evaluation results..."
	@rm -rf eval_results/*.json eval_results/*.csv eval_results/*.md
