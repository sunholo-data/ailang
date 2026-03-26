# =============================================================================
# EXAMPLE VERIFICATION TARGETS
# =============================================================================

.PHONY: verify-examples verify-examples-all verify-examples-trace verify-cli-examples examples-status
.PHONY: update-readme update-trace-baselines flag-broken freeze-stdlib verify-stdlib
.PHONY: compile-examples-go

# Example verification (parallel by default, ~8s vs ~3min sequential)
verify-examples: build ## Verify examples in examples/runnable/ (CI mode)
	@echo "Verifying examples..."
	@go run ./scripts/verify_examples.go --parallel 8 --json > examples_report.json 2>&1 || true
	@go run ./scripts/verify_examples.go --parallel 8 --markdown > examples_status.md 2>&1 || true
	@if [ -f examples_status.md ]; then cat examples_status.md; fi

verify-examples-all: build ## Verify ALL examples with threshold gate (60%)
	@echo "Verifying all examples with threshold gate..."
	@go run ./scripts/verify_examples.go --all --parallel 8 --threshold 60

verify-examples-trace: build ## Verify examples with trace capture + determinism replay
	@echo "Verifying examples with trace determinism..."
	@go run ./scripts/verify_examples.go --trace

update-trace-baselines: build ## Regenerate trace baselines for all passing examples
	@echo "Updating trace baselines..."
	@go run ./scripts/verify_examples.go --trace --update-baselines

verify-cli-examples: ## Verify CLI examples from documentation
	@echo "Verifying CLI examples..."
	@./tools/verify_cli_examples.sh

examples-status: build ## Quick one-line example status
	@go run ./scripts/verify_examples.go --all 2>&1 | grep "Examples:"

# Go codegen compile verification
# M-CODEGEN-COMPILE-GATE: Compile all runnable examples to Go and verify with go build
compile-examples-go: build ## Compile examples to Go and verify with go build
	@echo "Compiling runnable examples to Go with verification..."
	@passed=0; failed=0; skipped=0; \
	for f in examples/runnable/*.ail; do \
		name=$$(basename "$$f" .ail); \
		outdir=$$(mktemp -d); \
		if ./bin/ailang compile --emit-go --out "$$outdir" "$$f" >/dev/null 2>&1; then \
			passed=$$((passed + 1)); \
		else \
			echo "  ✗ $$name"; \
			failed=$$((failed + 1)); \
		fi; \
		rm -rf "$$outdir"; \
	done; \
	echo ""; \
	echo "Go compile gate: $$passed passed, $$failed failed"; \
	if [ $$failed -gt 0 ]; then exit 1; fi

# README & documentation updates
update-readme: build ## Update README with example status
	@echo "Verifying examples..."
	@go run ./scripts/verify_examples.go --json > examples_report.json 2>&1 || true
	@go run ./scripts/verify_examples.go --markdown > examples_status.md 2>&1 || true
	@if [ -f examples_report.json ]; then go run ./scripts/update_readme.go; fi
	@if [ -f examples_report.json ]; then go run ./scripts/update_docs_examples.go; fi

flag-broken: verify-examples ## Add warning headers to broken examples
	@echo "Flagging broken examples..."
	@go run ./scripts/flag_broken_examples.go

# Standard library interface management
freeze-stdlib: ## Freeze std/ library interfaces (create golden checksums)
	@echo "Freezing std/ library interfaces..."
	@tools/freeze-stdlib.sh

verify-stdlib: ## Verify std/ library interface stability
	@echo "Verifying std/ library interfaces..."
	@tools/verify-stdlib.sh
