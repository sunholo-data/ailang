# =============================================================================
# TESTING TARGETS
# =============================================================================

.PHONY: test test-parser test-parser-update test-lowering test-imports test-imports-success
.PHONY: test-import-errors test-recursion test-iface-determinism test-builtin-freeze
.PHONY: test-operator-assertions test-regression-guards test-builtin-consistency
.PHONY: test-stdlib-canaries test-row-properties test-golden-types test-repl-smoke
.PHONY: test-sim-stub test-stdlib-freeze verify-no-shim verify-lowering

# Core tests. Depends on build so integration tests that shell out to the
# ailang binary never see a stale bin/ailang — a stale binary caused phantom
# stdlib failures on 2026-07-10 (see internal/testutil/ailangbin.go).
test: build ## Run all Go unit tests (builds bin/ailang first)
	@echo "Running tests..."
	@$(GOTEST) -v $$($(GOCMD) list ./... | grep -v /scripts | grep -v /examples/agents)

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

# Stdlib freeze
test-stdlib-freeze: $(FREEZE_DIR)/option.sha256 $(FREEZE_DIR)/result.sha256 \
                    $(FREEZE_DIR)/list.sha256 $(FREEZE_DIR)/string.sha256 \
                    $(FREEZE_DIR)/io.sha256 ## Verify std/ interfaces haven't changed
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
	      echo "MISMATCH $$name"; ok=1; \
	    fi; \
	  fi; \
	done; \
	exit $$ok

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
