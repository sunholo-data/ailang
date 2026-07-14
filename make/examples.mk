# =============================================================================
# EXAMPLE VERIFICATION TARGETS
# =============================================================================

.PHONY: verify-examples verify-examples-toplevel verify-examples-all verify-examples-trace verify-cli-examples examples-status
.PHONY: update-readme update-trace-baselines flag-broken freeze-stdlib verify-stdlib
.PHONY: compile-examples-go verify-examples-gate-selftest backfill-manifest-modules

# Example verification (parallel by default, ~8s vs ~3min sequential)
#
# This is a REAL gate: it exits non-zero when any example fails, while STILL
# writing both report artifacts unconditionally (5+ consumers: tools/build-snapshot,
# scripts/update_readme.go, scripts/flag_broken_examples.go, scripts/update_docs_examples.go,
# docusaurus-deploy.yml). The verifier's own exit code (non-zero on failure) is
# captured and re-raised AFTER the status markdown is printed. Also runs the
# manifest `modules` drift-lint (validate_manifest --ci) so a stale/new example
# whose manifest entry is out of date turns the gate red too.
verify-examples: build ## Verify examples in examples/runnable/ (CI gate — fails on any red example)
	@echo "Verifying examples..."
	@go run ./scripts/verify_examples.go --parallel 8 --json > examples_report.json 2>&1; echo $$? > .examples_rc.tmp
	@go run ./scripts/verify_examples.go --parallel 8 --markdown > examples_status.md 2>&1 || true
	@if [ -f examples_status.md ]; then cat examples_status.md; fi
	@echo "Validating manifest (schema + modules drift)..."
	@go run ./scripts/validate_manifest.go --ci; echo $$? > .manifest_rc.tmp
	@rc=$$(cat .examples_rc.tmp); mrc=$$(cat .manifest_rc.tmp); rm -f .examples_rc.tmp .manifest_rc.tmp; \
		if [ "$$rc" != "0" ]; then echo "❌ verify-examples: one or more examples failed (exit $$rc)"; exit $$rc; fi; \
		if [ "$$mrc" != "0" ]; then echo "❌ verify-examples: manifest modules drift (exit $$mrc)"; exit $$mrc; fi; \
		echo "✅ verify-examples: all examples pass and manifest is in sync"

# Gate self-test: guard the CALL-SITE, not just the helper (env-forward lesson).
# Drops a deliberately-broken fixture into examples/runnable/, runs the verifier
# the same way the gate does, and asserts (a) non-zero exit AND (b) both report
# artifacts were STILL written. Always removes the fixture, even on failure.
verify-examples-gate-selftest: build ## Prove the verify-examples gate fails (and still writes artifacts) on a broken example
	@echo "Gate self-test: injecting a deliberately-broken example..."
	@fixture=examples/runnable/_gate_selftest_broken.ail; \
	rpt=$$(mktemp); md=$$(mktemp); \
	printf 'module examples/runnable/_gate_selftest_broken\nexport func main() -> int = "not an int"\n' > $$fixture; \
	trap "rm -f $$fixture $$rpt $$md" EXIT; \
	go run ./scripts/verify_examples.go --parallel 8 --json > $$rpt 2>&1; rc=$$?; \
	go run ./scripts/verify_examples.go --parallel 8 --markdown > $$md 2>&1 || true; \
	fail=0; \
	if [ "$$rc" = "0" ]; then echo "❌ SELF-TEST FAIL: verifier exited 0 on a broken example"; fail=1; \
		else echo "✓ verifier exited non-zero ($$rc) on the broken example"; fi; \
	if [ ! -s $$rpt ]; then echo "❌ SELF-TEST FAIL: examples_report.json not written"; fail=1; \
		else echo "✓ examples_report.json written on failure"; fi; \
	if [ ! -s $$md ]; then echo "❌ SELF-TEST FAIL: examples_status.md not written"; fail=1; \
		else echo "✓ examples_status.md written on failure"; fi; \
	if [ "$$fail" != "0" ]; then exit 1; fi; \
	echo "✅ gate self-test passed: broken example -> non-zero exit AND artifacts written"

# Regenerate the manifest `modules` field (parser-backed). Commit the result.
backfill-manifest-modules: ## Backfill examples/manifest.json `modules` field from actual imports
	@go run ./scripts/backfill_manifest_modules.go

# Gate the TOP-LEVEL examples/*.ail directory, which verify-examples (above)
# does NOT cover — it only checks examples/runnable/. That gap let 8 examples
# ship using `++` on strings (list-only since v0.13.0) plus 2 missing imports,
# none caught by CI. Every top-level example must type-check; runnable ones must
# run; AI/network + known-bug examples are skipped WITH a logged reason.
verify-examples-toplevel: build ## Gate top-level examples/*.ail (type-check all + run runnable)
	@./tools/verify_examples.sh

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
