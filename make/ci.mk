# =============================================================================
# CI & INTEGRATION TARGETS
# =============================================================================

.PHONY: ci ci-quick ci-strict all

# Default target
all: test build ## Run tests and build

# CI verification
ci: deps fmt-check test-fmt-check shellcheck-autopush test-shellcheck-autopush vet lint check-wasm-build test test-nightly-classifier test-coverage-badge check-coverage-threshold test-lowering verify-no-shim verify-examples verify-examples-toplevel verify-stdlib verify-stdlib-selftest verify-examples-gate-selftest verify-mcp-tools verify-install-guide verify-pi-assets verify-maintenance-assets check-boundaries check-no-fma check-architecture-closure check-referenced-paths check-git-exec test-check-git-exec check-changelog check-file-sizes check-golden-drift check-protocol-closure check-skills check-context-docs test-check-context-docs check-doc-review test-check-doc-review check-doc-samples test-check-doc-samples check-tmpfile-hygiene check-home-isolation check-no-personal-email test-check-no-personal-email test-check-autoclose test-check-referenced-paths test-check-changelog test-check-protocol-closure test-check-tmpfile-hygiene test-check-home-isolation check-prompt-freeze check-prompt-commands check-llms-txt test-check-llms-txt check-cli-docs test-parser test-stdlib-ail test-regression-guards test-imports-success test-import-errors test-launchd-drivers ## Run full CI verification, including workflow gates
	@echo "$(GREEN)$(CHECKMARK) CI verification complete$(RESET)"

# Fast (~55 s with the doc-sample build) gates over generated files and docs that a direct push to dev can leave
# stale: gofmt, the ARCHITECTURE.md closure section, the CLI reference, the devtools
# prompt's commands, file sizes, the changelog index, layer boundaries and the examples
# manifest. The pre-push hook runs this in a throwaway worktree of the pushed commit, so
# a push to dev cannot turn dev red on them. 2026-09-27: cli.md (rig-gate), the closure
# count and manifest statistics each broke dev or a PR after a direct push.
ci-quick: fmt-check check-architecture-closure check-cli-docs check-prompt-commands check-llms-txt check-file-sizes check-changelog check-boundaries ## Fast (~55s) generated-file and doc gates (run by the pre-push hook on dev pushes)
	@go run ./scripts/validate_manifest.go --ci
	@# Doc samples with the ratchet against origin/dev, which in the pre-push hook is the
	@# remote tip BEFORE this push. Builds bin/ailang from the pushed commit (~10-15 s).
	@$(MAKE) --no-print-directory check-doc-samples DOC_SAMPLES_ARGS="-base-ref origin/dev"
	@echo "$(GREEN)$(CHECKMARK) ci-quick gates passed$(RESET)"

ci-strict: deps fmt-check vet lint test test-coverage-badge verify-lowering test-lowering test-builtin-freeze test-operator-assertions test-imports test-recursion test-iface-determinism verify-examples verify-examples-toplevel verify-mcp-tools verify-install-guide ## Extended CI with A2 milestone gates
	@echo "$(GREEN)$(CHECKMARK) Strict CI verification complete (A2 milestone)$(RESET)"

.PHONY: check-llms-txt
check-llms-txt: ## llms.txt (3 committed copies) embeds the active teaching prompt; fix with make generate-llms-txt
	@scripts/check_llms_txt.sh

.PHONY: test-check-llms-txt
test-check-llms-txt: ## Self-test of the llms.txt gate against the real generator (bash 3.2)
	@/bin/bash scripts/test_check_llms_txt.sh
