# =============================================================================
# TESTING TARGETS
# =============================================================================

.PHONY: test test-parser test-parser-update test-lowering test-imports test-imports-success
.PHONY: test-import-errors test-recursion test-iface-determinism test-builtin-freeze
.PHONY: test-operator-assertions test-regression-guards test-builtin-consistency
.PHONY: test-stdlib-canaries test-row-properties test-golden-types test-repl-smoke
.PHONY: test-sim-stub test-stdlib-freeze verify-no-shim verify-lowering
.PHONY: test-nightly-classifier test-launchd-drivers test-check-changelog test-check-protocol-closure test-check-autoclose

# Core tests. Depends on build so integration tests that shell out to the
# ailang binary never see a stale bin/ailang — a stale binary caused phantom
# stdlib failures on 2026-07-10 (see internal/testutil/ailangbin.go).
#
# The prefetch below runs UNPOISONED so the poisoned test run cannot fail on a
# module-cache miss. It is deliberately plain `download`, NOT `download all`:
# `all` resolves the full transitive graph and writes checksums for modules this
# repo never imports, adding ~394 lines to the TRACKED go.sum on every run —
# which leaves the developer's tree dirty and stamps every later build `-dirty`
# (Makefile:27 uses `git describe --dirty`). Measured 2026-08-05: plain
# `download` into a cold GOMODCACHE followed by this poisoned run gives
# rc=0 / 107 ok / zero module-resolution failures — indistinguishable from the
# warm run, with go.sum untouched. The CI workflows keep `download all`: they
# run on three OSes whose build constraints were not measured here, and their
# checkout is ephemeral so the go.sum churn is harmless there.
test: build ## Run all Go unit tests (builds bin/ailang first)
	@echo "Running tests..."
	@$(GOCMD) mod download
	@HTTP_PROXY=http://127.0.0.1:9 HTTPS_PROXY=http://127.0.0.1:9 NO_PROXY=localhost,127.0.0.1 GOPROXY=off $(GOTEST) -v $$($(GOCMD) list ./... | grep -v /scripts | grep -v /examples/agents)

test-nightly-classifier: ## Run nightly variance-guard contract and replay tests
	@python3 tools/test_nightly_classify.py -v

# The launchd drivers carried ZERO automated coverage until #558's second recurrence — a large
# part of why two silent-staleness bugs shipped unnoticed. /bin/bash explicitly, not $$SHELL:
# the rig runs 3.2.57, so a suite that only passes under a newer bash proves nothing about it.
# The motoko connection probe's routing verdict is load-bearing; run its self-test here so CI
# refuses when the instrument can no longer prove both treatment absence and control visibility.
test-launchd-drivers: ## Run launchd driver tests (pin-root + routing + notices + hook stdout, bash 3.2)
	@/bin/bash tools/launchd/test_pin_root.sh
	@/bin/bash tools/launchd/test_driver_notify.sh
	@/bin/bash tools/launchd/test_mission_routing.sh
	@/bin/bash tools/launchd/test_hook_stdout.sh
	@/bin/bash tools/launchd/test_fmt_ab_schedule.sh
	@/bin/bash tools/launchd/test_controller_chain.sh
	@/bin/bash tools/eval/test_motoko_connection_probe.sh
	@for f in tools/launchd/*.sh tools/launchd/lib/*.sh; do /bin/bash -n "$$f" || exit 1; done
	@/bin/bash -n tools/eval/motoko_connection_probe.sh
	@/bin/bash -n tools/eval/test_motoko_connection_probe.sh
	@/bin/bash -n scripts/mission_decisions.sh
	@echo "launchd drivers: tests + bash 3.2 syntax OK"

# `make check-changelog` is a refusal gate that shipped with no coverage of WHICH release-note
# shapes it can see. Measured 2026-08-17 on dev at 0002c9b0b: five stranded sections in root
# CHANGELOG.md, one flagged. A gate is not a gate until something reds when you remove it, so
# each of its four refusal branches now has an arm that dies when that branch is neutered.
# /bin/bash explicitly: the rig runs 3.2.57 (same reason as test-launchd-drivers).
test-check-changelog: ## Run the changelog-index gate's own self-test (bash 3.2)
	@/bin/bash scripts/test_check_changelog.sh
	@/bin/bash -n scripts/check_changelog.sh

test-check-protocol-closure: ## Run the protocol-closure gate's own self-test (bash 3.2)
	@/bin/bash scripts/test_check_protocol_closure.sh
	@/bin/bash -n scripts/check_protocol_closure.sh

test-check-autoclose: ## Run the issue-autoclose gate's own self-test (bash 3.2)
	@/bin/bash scripts/test_check_autoclose.sh
	@/bin/bash -n scripts/check_autoclose.sh

test-parser: ## Run parser tests only
	@echo "Testing parser..."
	@$(GOTEST) ./internal/parser
	@echo "$(GREEN)$(CHECKMARK) Parser tests passed$(RESET)"

test-parser-update: ## Update parser golden files
	@echo "Updating parser golden files..."
	@$(GOTEST) -update ./internal/parser
	@echo "$(GREEN)$(CHECKMARK) Golden files updated$(RESET)"

# Operator lowering tests
test-lowering: build ## Run operator lowering golden tests
	@echo "Testing operator lowering..."
	@printf "  Integer ops: "
	@result=$$(./bin/ailang run tests/binops_int.ail 2>&1 | tail -n1); \
	if [ "$$result" = "14" ]; then echo "$(GREEN)$(CHECKMARK)$(RESET)"; else echo "$(RED)$(CROSS) FAIL (got $$result)$(RESET)"; exit 1; fi
	@printf "  Float ops: "
	@result=$$(./bin/ailang run tests/binops_float.ail 2>&1 | tail -n1); \
	if [ "$$result" = "1.5" ]; then echo "$(GREEN)$(CHECKMARK)$(RESET)"; else echo "$(RED)$(CROSS) FAIL (got $$result)$(RESET)"; exit 1; fi
	@printf "  Precedence: "
	@result=$$(./bin/ailang run tests/precedence_lowering.ail 2>&1 | tail -n1); \
	if [ "$$result" = "14" ]; then echo "$(GREEN)$(CHECKMARK)$(RESET)"; else echo "$(RED)$(CROSS) FAIL (got $$result)$(RESET)"; exit 1; fi
	@printf "  Short-circuit: "
	@result=$$(./bin/ailang run tests/short_circuit.ail 2>&1 | tail -n1); \
	if [ "$$result" = "false" ]; then echo "$(GREEN)$(CHECKMARK)$(RESET)"; else echo "$(RED)$(CROSS) FAIL (got $$result)$(RESET)"; exit 1; fi
	@echo "$(GREEN)$(CHECKMARK) All operator lowering tests passed$(RESET)"

verify-no-shim: build ## Verify no operator shim usage (CI gate)
	@echo "Verifying no operator shim usage..."
	@printf "  Testing with --fail-on-shim: "
	@if ./bin/ailang run --require-lowering --fail-on-shim tests/binops_int.ail >/dev/null 2>&1; then \
		echo "$(GREEN)$(CHECKMARK)$(RESET)"; \
	else \
		echo "$(RED)$(CROSS) FAIL: Shim detected$(RESET)"; exit 1; \
	fi
	@echo "$(GREEN)$(CHECKMARK) No shim usage verified$(RESET)"

verify-lowering: build verify-no-shim ## Verify all operators are lowered
	@echo "$(GREEN)$(CHECKMARK) Operator lowering verified$(RESET)"

# Import tests
test-imports: test-imports-success test-import-errors ## Run all import tests
	@echo "$(GREEN)$(CHECKMARK) All import tests passed$(RESET)"

test-imports-success: build ## Test successful imports work
	@echo "== Testing successful imports =="
	@echo "  $(ARROW) imports_basic.ail"
	@$(BUILD_DIR)/$(BINARY) run --caps IO examples/runnable/imports_basic.ail > /dev/null 2>&1 || (echo "FAIL: imports_basic.ail" && exit 1)
	@echo "  $(ARROW) imports.ail"
	@$(BUILD_DIR)/$(BINARY) run --caps IO examples/runnable/imports.ail > /dev/null 2>&1 || (echo "FAIL: imports.ail" && exit 1)
	@echo "$(GREEN)$(CHECKMARK) Successful imports work$(RESET)"

test-import-errors: build ## Test import error goldens
	@echo "== Testing import error goldens =="
	@echo "  $(ARROW) LDR001 (module not found)"
	@$(BUILD_DIR)/$(BINARY) run --json --compact tests/errors/lnk_unresolved_module.ail 2>&1 | tail -1 | diff -u goldens/lnk_unresolved_module.json - || (echo "FAIL: LDR001 golden mismatch" && exit 1)
	@echo "  $(ARROW) IMP010 (symbol not exported)"
	@$(BUILD_DIR)/$(BINARY) run --json --compact tests/errors/lnk_unresolved_symbol.ail 2>&1 | tail -1 | diff -u goldens/lnk_unresolved_symbol.json - || (echo "FAIL: IMP010 golden mismatch" && exit 1)
	@echo "$(GREEN)$(CHECKMARK) All import error goldens match$(RESET)"

# Other test categories
test-recursion: build ## Test mutual recursion handling
	@echo "== Testing recursion =="
	@$(BUILD_DIR)/$(BINARY) run tests/recursion/mutual.ail > /dev/null 2>&1 || (echo "FAIL: mutual.ail should work" && exit 1)
	@echo "$(GREEN)$(CHECKMARK) Mutual recursion works$(RESET)"

test-iface-determinism: build ## Test interface determinism
	@echo "$(GREEN)$(CHECKMARK) Interface determinism verified (by construction)$(RESET)"

test-builtin-freeze: ## Test builtin interface stability
	@echo "Testing builtin interface freeze..."
	@go test ./internal/iface -run TestBuiltinInterfaceStability || exit 1
	@echo "$(GREEN)$(CHECKMARK) Builtin interface stable$(RESET)"

test-operator-assertions: ## Test operator assertion guards
	@echo "Testing operator assertion guards..."
	@go test ./internal/pipeline -run TestAssertOnlyBuiltinsForOps || exit 1
	@echo "$(GREEN)$(CHECKMARK) Operator assertions working$(RESET)"

# Regression guards
test-regression-guards: ## Run all regression guard tests (critical)
	@echo "Running regression guard tests..."
	@echo "  $(ARROW) Builtin consistency"
	@$(GOTEST) -v ./internal/pipeline -run TestBuiltinConsistency
	@echo "  $(ARROW) Builtin type golden snapshots"
	@$(GOTEST) -v ./internal/pipeline -run TestBuiltinTypes
	@echo "  $(ARROW) REPL smoke tests"
	@$(GOTEST) -v ./internal/repl -run TestREPLSmoke
	@echo "  $(ARROW) Stdlib canaries"
	@$(GOTEST) -v ./internal/pipeline -run TestStdlibCanary
	@echo "  $(ARROW) Row unification properties"
	@$(GOTEST) -v ./internal/types -run TestRowUnification
	@echo "$(GREEN)$(CHECKMARK) All regression guards passed$(RESET)"

test-builtin-consistency: ## Test builtin three-way parity
	@$(GOTEST) -v ./internal/pipeline -run TestBuiltinConsistency

test-stdlib-canaries: ## Test std/ library health
	@$(GOTEST) -v ./internal/pipeline -run TestStdlibCanary

test-row-properties: ## Test row unification properties
	@$(GOTEST) -v ./internal/types -run TestRowUnification

test-golden-types: ## Test builtin type golden snapshots
	@$(GOTEST) -v ./internal/pipeline -run TestBuiltinTypes

test-repl-smoke: ## REPL smoke tests (:type command)
	@$(GOTEST) -v ./internal/repl -run TestREPLSmoke

test-sim-stub: install ## Test Go codegen pipeline (sim_stub example)
	@echo "Testing sim_stub example..."
	@cd examples/sim_stub && make clean && make test

# Stdlib freeze — historical name, kept because v0.2.0 acceptance docs and the
# sprint-executor skill reference it. The live gate is verify-stdlib
# (tools/verify-stdlib.sh over .stdlib-golden/); do not grow a second implementation.
test-stdlib-freeze: verify-stdlib ## Verify std/ interfaces haven't changed (alias of verify-stdlib)

# Fuzzing
# fuzz-parser retries the Go-fuzzing fuzztime-boundary "context deadline exceeded"
# artifact once: when -fuzztime expires while a slow (deeply-nested) input is still
# executing on a loaded CI runner, the coordinator cancels the worker context and Go
# reports it as a FAIL even though no crasher was found. A REAL crasher writes
# "Failing input written to testdata/fuzz/..." and fails immediately; a genuine
# slow-parse regression that ALWAYS times out fails on the second attempt too. Only
# the single transient boundary timeout is retried.
fuzz-parser: ## Fuzz parser (2s - for CI; retries transient fuzztime-boundary timeout once)
	@echo "Fuzzing parser (2s)..."
	@for attempt in 1 2; do \
	  out="$$($(GOTEST) -fuzz=FuzzParseExpr -fuzztime=2s ./internal/parser 2>&1)"; rc=$$?; \
	  printf '%s\n' "$$out"; \
	  [ $$rc -eq 0 ] && exit 0; \
	  if printf '%s' "$$out" | grep -q "Failing input written"; then \
	    echo "fuzz-parser: real crasher found — failing"; exit $$rc; \
	  fi; \
	  if printf '%s' "$$out" | grep -q "context deadline exceeded"; then \
	    echo "fuzz-parser: transient fuzztime-boundary timeout (no crasher persisted) — retry $$attempt/2"; \
	    continue; \
	  fi; \
	  echo "fuzz-parser: unexpected failure — not retrying"; exit $$rc; \
	done; \
	echo "fuzz-parser: transient timeout on both attempts — FAIL (investigate slow-parse inputs)"; exit 1

fuzz-parser-long: ## Fuzz parser (extended - 1m per target)
	@echo "Fuzzing parser (1m)..."
	@$(GOTEST) -fuzz=FuzzParseExpr -fuzztime=1m ./internal/parser
	@$(GOTEST) -fuzz=FuzzParseModule -fuzztime=1m ./internal/parser
	@$(GOTEST) -fuzz=FuzzParseMalformed -fuzztime=1m ./internal/parser
	@$(GOTEST) -fuzz=FuzzParseUnicode -fuzztime=1m ./internal/parser

# Golden file management
regen-import-error-goldens: build ## Regenerate import error golden files (use with caution)
	@mkdir -p goldens
	@$(BUILD_DIR)/$(BINARY) run --json --compact tests/errors/lnk_unresolved_module.ail 2>&1 | tail -1 > goldens/lnk_unresolved_module.json
	@$(BUILD_DIR)/$(BINARY) run --json --compact tests/errors/lnk_unresolved_symbol.ail 2>&1 | tail -1 > goldens/lnk_unresolved_symbol.json
	@echo "$(GREEN)$(CHECKMARK) Golden files regenerated$(RESET)"

check-golden-drift: ## Check for uncommitted golden file changes
	@echo "Checking for golden file changes..."
	@git diff --name-only -- internal/parser/testdata/parser/ > .golden_changes || true
	@if [ -s .golden_changes ]; then \
		echo "$(YELLOW)$(WARNING) Golden files changed:$(RESET)"; \
		cat .golden_changes; \
		if [ "$$ALLOW_GOLDEN_UPDATES" != "1" ]; then \
			echo "$(RED)$(CROSS) Golden files changed without ALLOW_GOLDEN_UPDATES=1$(RESET)"; \
			exit 1; \
		fi; \
	else \
		echo "$(GREEN)$(CHECKMARK) No golden file changes$(RESET)"; \
	fi
	@rm -f .golden_changes

# Test parity (manual)
test-parity: build ## Test REPL/file parity for imports (interactive)
	@chmod +x tests/parity/run_imports_basic.sh
	@tests/parity/run_imports_basic.sh

# The `.ail` test suites under tests/stdlib/ ran in NO make target and NO CI job until
# iteration 183 measured it: `grep -rn "ailang test" make/ Makefile` returned ZERO while the
# control (`.ail` in make/) returned 35 — i.e. the product's own test runner had no CI reach
# over any .ail suite. M-TAKE-FLATMAP-PEAK-MEMORY M2's entire behavioural pin (the fused
# delegation instrument and the parity suite) lived there, so it would have shipped as
# decoration: a guard is not a gate until something reds when you remove it.
# Both loops carry an EXACT COUNT PIN — adding, removing, renaming, or moving a test must be a
# deliberate, reviewable change rather than silently shrinking or growing the gate.
# Enumeration is `find -L` (symlinks FOLLOWED, so a symlinked fixture is a real fixture: with a
# bare `-type f` a committed symlink is type `l`, invisible to both the run and the count, and a
# suite that cannot pass sat one directory down at rc=0). A DANGLING symlink is still type `l`
# under -L, so it is rejected explicitly rather than silently skipped. The file list is read from
# a temp file, not word-split from `$(...)`, so a path containing a space stays one path.
# Known residual, declared rather than assumed: name matching is case-SENSITIVE, so `*_TEST.ail`
# and `*.EXPECTED` are still invisible, and a real suite not ending `_test.ail` is not enumerated.
# The .expected files are captured from the product's OWN stdout and diffed against a verbatim
# re-run, rather than reconstructed by the gate — a reconstruction verifies our arithmetic, not
# the artifact.
STDLIB_AIL_SUITES_EXPECTED := 4
STDLIB_AIL_FIXTURES_EXPECTED := 4

test-stdlib-ail: build ## Run the .ail test suites + run-fixtures under tests/stdlib/
	@echo "Running stdlib .ail test suites..."
	@test -d tests/stdlib || { echo "instrument failure: tests/stdlib does not exist"; exit 1; }; \
	find -L tests/stdlib -type l -name '*_test.ail' > /tmp/ailang_stdlib_dangling.$$$$; \
	if [ -s /tmp/ailang_stdlib_dangling.$$$$ ]; then \
	  echo "instrument failure: broken symlink(s) matching *_test.ail — a suite that cannot be read is not a suite:"; \
	  sed 's/^/  /' /tmp/ailang_stdlib_dangling.$$$$; rm -f /tmp/ailang_stdlib_dangling.$$$$; exit 1; \
	fi; rm -f /tmp/ailang_stdlib_dangling.$$$$; \
	suites=0; find -L tests/stdlib -type f -name '*_test.ail' | sort > /tmp/ailang_stdlib_suites.$$$$; \
	while IFS= read -r f; do \
	  suites=$$((suites+1)); \
	  echo "  test: $$f"; \
	  ./bin/ailang test --no-color "$$f" > /tmp/ailang_stdlib_test.$$$$ 2>&1 || \
	    { echo "FAIL: $$f"; cat /tmp/ailang_stdlib_test.$$$$; rm -f /tmp/ailang_stdlib_test.$$$$; exit 1; }; \
	  rm -f /tmp/ailang_stdlib_test.$$$$; \
	done < /tmp/ailang_stdlib_suites.$$$$; \
	[ "$$suites" -eq "$(STDLIB_AIL_SUITES_EXPECTED)" ] || \
	  { echo "instrument failure: expected $(STDLIB_AIL_SUITES_EXPECTED) stdlib .ail test suite(s), actual $$suites"; \
	    echo "actual files:"; sed 's/^/  /' /tmp/ailang_stdlib_suites.$$$$; rm -f /tmp/ailang_stdlib_suites.$$$$; \
	    echo "If this change was intentional, update STDLIB_AIL_SUITES_EXPECTED in make/test.mk."; exit 1; }; \
	rm -f /tmp/ailang_stdlib_suites.$$$$; \
	echo "  $$suites .ail test suite(s) passed"
	@find -L tests/stdlib -type l -name '*.expected' > /tmp/ailang_stdlib_dangling.$$$$; \
	if [ -s /tmp/ailang_stdlib_dangling.$$$$ ]; then \
	  echo "instrument failure: broken symlink(s) matching *.expected:"; \
	  sed 's/^/  /' /tmp/ailang_stdlib_dangling.$$$$; rm -f /tmp/ailang_stdlib_dangling.$$$$; exit 1; \
	fi; rm -f /tmp/ailang_stdlib_dangling.$$$$; \
	fixtures=0; find -L tests/stdlib -type f -name '*.expected' | sort > /tmp/ailang_stdlib_fixtures.$$$$; \
	while IFS= read -r e; do \
	  f=`echo "$$e" | sed 's/\.expected$$/.ail/'`; \
	  [ -e "$$f" ] || { echo "instrument failure: $$e has no matching $$f"; exit 1; }; \
	  fixtures=$$((fixtures+1)); \
	  echo "  run: $$f"; \
	  ./bin/ailang run --caps IO "$$f" 2>/tmp/ailang_stdlib_err.$$$$ | grep -v '^→\|^✓' > /tmp/ailang_stdlib_run.$$$$ || true; \
	  diff -u "$$e" /tmp/ailang_stdlib_run.$$$$ || \
	    { echo "FAIL: $$f stdout differs from $$e"; \
	      echo "--- stderr from $$f ---"; cat /tmp/ailang_stdlib_err.$$$$; \
	      rm -f /tmp/ailang_stdlib_run.$$$$ /tmp/ailang_stdlib_err.$$$$; exit 1; }; \
	  rm -f /tmp/ailang_stdlib_err.$$$$; \
	  rm -f /tmp/ailang_stdlib_run.$$$$; \
	done < /tmp/ailang_stdlib_fixtures.$$$$; \
	[ "$$fixtures" -eq "$(STDLIB_AIL_FIXTURES_EXPECTED)" ] || \
	  { echo "instrument failure: expected $(STDLIB_AIL_FIXTURES_EXPECTED) stdlib run-fixture(s), actual $$fixtures"; \
	    echo "actual files:"; sed 's/^/  /' /tmp/ailang_stdlib_fixtures.$$$$; rm -f /tmp/ailang_stdlib_fixtures.$$$$; \
	    echo "If this change was intentional, update STDLIB_AIL_FIXTURES_EXPECTED in make/test.mk."; exit 1; }; \
	rm -f /tmp/ailang_stdlib_fixtures.$$$$; \
	echo "  $$fixtures run-fixture(s) matched expected stdout"
	@echo "$(GREEN)✓ stdlib .ail suites and run-fixtures pass$(NC)"
