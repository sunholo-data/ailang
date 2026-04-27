# =============================================================================
# CI & INTEGRATION TARGETS
# =============================================================================

.PHONY: ci ci-strict all

# Default target
all: test build ## Run tests and build

# CI verification
ci: deps fmt-check vet lint test test-coverage-badge test-lowering verify-no-shim verify-examples verify-mcp-tools ## Run full CI verification
	@echo "$(GREEN)$(CHECKMARK) CI verification complete$(RESET)"

ci-strict: deps fmt-check vet lint test test-coverage-badge verify-lowering test-lowering test-builtin-freeze test-operator-assertions test-imports test-recursion test-iface-determinism verify-examples verify-mcp-tools ## Extended CI with A2 milestone gates
	@echo "$(GREEN)$(CHECKMARK) Strict CI verification complete (A2 milestone)$(RESET)"
