# =============================================================================
# CODE COVERAGE TARGETS
# =============================================================================

.PHONY: test-coverage test-coverage-badge test-coverage-detailed coverage-clean
.PHONY: test-coverage-gate test-coverage-html

# Coverage settings
COVERAGE_DIR := coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html
COVERAGE_THRESHOLD := 29

# Generate coverage report
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic $$($(GOCMD) list ./... | grep -v /scripts | grep -v /examples/agents)
	@echo "$(GREEN)$(CHECKMARK) Coverage report generated$(RESET)"

# Display quick coverage badge
test-coverage-badge: test-coverage ## Show coverage percentage badge
	@echo ""
	@coverage=$$($(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}'); \
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
test-coverage-gate: test-coverage ## Verify coverage meets threshold
	@coverage=$$($(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}' | sed 's/%//'); \
	threshold=$(COVERAGE_THRESHOLD); \
	if (( $$(echo "$$coverage >= $$threshold" | bc -l) )); then \
		echo "$(GREEN)$(CHECKMARK) Coverage $$coverage% meets threshold $$threshold%$(RESET)"; \
	else \
		echo "$(RED)$(CROSS) Coverage $$coverage% below threshold $$threshold%$(RESET)"; \
		exit 1; \
	fi

# Clean coverage files
coverage-clean: ## Remove coverage files
	@rm -rf $(COVERAGE_DIR) coverage.out
	@echo "$(GREEN)$(CHECKMARK) Coverage files cleaned$(RESET)"
