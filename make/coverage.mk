# =============================================================================
# CODE COVERAGE TARGETS
# =============================================================================

.PHONY: test-coverage test-coverage-badge test-coverage-detailed coverage-clean
.PHONY: test-coverage-gate test-coverage-html coverage-from-profile check-coverage-threshold

# Coverage settings
COVERAGE_DIR := coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
# Whole-suite profile written by CI's single `go test ./...` pass (ci.yml).
COVERAGE_RAW := $(COVERAGE_DIR)/raw.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html
COVERAGE_THRESHOLD := 29
# The packages the coverage % counts. ONE definition: test-coverage tests exactly
# these, and coverage-from-profile cuts CI's whole-suite profile down to them, so
# the gate and SonarCloud measure the same set either way.
COVERAGE_PKGS_CMD = $(GOCMD) list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' ./... | grep -v /scripts | grep -v /examples/agents

# Generate coverage report
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) -covermode=set $$($(COVERAGE_PKGS_CMD))
	@echo "$(GREEN)$(CHECKMARK) Coverage report generated$(RESET)"

# CI path: reuse the profile the main test step already wrote instead of running
# the whole suite a second time (that rerun cost ~8.5 min of every CI run).
# Keeps the mode line plus every block whose package is in COVERAGE_PKGS_CMD.
coverage-from-profile: ## Derive coverage.out from CI's whole-suite profile (no test rerun)
	@test -s $(COVERAGE_RAW) || { echo "$(RED)$(CROSS) $(COVERAGE_RAW) missing or empty — run go test -coverprofile=$(COVERAGE_RAW) first$(RESET)"; exit 1; }
	@$(COVERAGE_PKGS_CMD) | awk 'NR==FNR { keep[$$0] = 1; next } FNR == 1 { print; next } { p = $$0; sub(/:.*/, "", p); sub(/\/[^\/]*$$/, "", p); if (p in keep) print }' - $(COVERAGE_RAW) > $(COVERAGE_FILE)
	@test $$(wc -l < $(COVERAGE_FILE)) -gt 1 || { echo "$(RED)$(CROSS) $(COVERAGE_FILE) kept no coverage blocks — filter or profile is broken$(RESET)"; exit 1; }
	@echo "$(GREEN)$(CHECKMARK) $(COVERAGE_FILE) derived from $(COVERAGE_RAW)$(RESET)"

# Display quick coverage badge
test-coverage-badge: test-coverage ## Show coverage percentage badge
	@echo ""
	@coverage=$$($(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep "^total:" | awk '{print $$3}'); \
	echo "$(BOLD)Coverage: $$coverage$(RESET)"

# Generate HTML coverage report
test-coverage-html: test-coverage ## Generate HTML coverage report
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o=$(COVERAGE_HTML)
	@echo "$(GREEN)$(CHECKMARK) HTML report generated: $(COVERAGE_HTML)$(RESET)"
	@echo "  Open in browser: open $(COVERAGE_HTML)"

# Detailed coverage report
test-coverage-detailed: test-coverage ## Show detailed coverage by package
	@echo ""
	@echo "$(BOLD)Coverage by package:$(RESET)"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep -v "total" | sort -t':' -k2 -rn | head -20

# Coverage gate (fails if below threshold)
# Uses awk for POSIX-compatible float comparison (dash/sh doesn't support bash's (( )) )
test-coverage-gate: test-coverage ## Verify coverage meets threshold
	@$(MAKE) --no-print-directory check-coverage-threshold

# Threshold check on an existing $(COVERAGE_FILE) — no test run.
check-coverage-threshold: ## Check an existing coverage.out against the threshold
	@coverage=$$($(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep "^total:" | awk '{print $$3}' | sed 's/%//'); \
	threshold=$(COVERAGE_THRESHOLD); \
	if awk "BEGIN {exit !($$coverage >= $$threshold)}"; then \
		echo "$(GREEN)$(CHECKMARK) Coverage $$coverage% meets threshold $$threshold%$(RESET)"; \
	else \
		echo "$(RED)$(CROSS) Coverage $$coverage% below threshold $$threshold%$(RESET)"; \
		exit 1; \
	fi

# Clean coverage files
coverage-clean: ## Remove coverage files
	@rm -rf $(COVERAGE_DIR) coverage.out
	@echo "$(GREEN)$(CHECKMARK) Coverage files cleaned$(RESET)"
