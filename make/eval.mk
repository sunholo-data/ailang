# =============================================================================
# EVALUATION & BENCHMARK TARGETS
# =============================================================================

.PHONY: eval eval-suite eval-models eval-report eval-clean eval-analyze eval-analyze-fresh
.PHONY: eval-to-design eval-prompt-ab eval-prompt-list eval-prompt-hash
.PHONY: eval-baseline eval-diff eval-summary eval-matrix
.PHONY: eval-auto-improve eval-auto-improve-apply
.PHONY: eval-smoke eval-core eval-stretch
.PHONY: bench-phase2a bench-phase2a-quick bench-workloads bench-workloads-quick

# Basic eval commands
eval: build ## Run a single benchmark (mock mode)
	@echo "Running evaluation benchmark..."
	@$(BUILD_DIR)/$(BINARY) eval --benchmark fizzbuzz --mock

eval-suite: build ## Run full benchmark suite (all models, parallel)
	@echo "Running full benchmark suite..."
	@$(BUILD_DIR)/$(BINARY) eval-suite

# Tier-specific eval targets (M-EVAL-SUITE-PREP). Each target runs a
# single tier so "signal" (smoke) and "headroom" (vision) stay separate.
# MODELS=... overrides the default model set; pass --full flags via EXTRA.
eval-smoke: build ## Fast tier (15 benchmarks, dev models) — target <90s warm cache
	@echo "Running SMOKE tier (dev models)..."
	@# M-EVAL-OS-LONGITUDINAL M0: fast-fail if the local-Ollama rig is degraded
	@# (ollama down, observatory server down, model missing) before burning
	@# benchmark wall-clock on guaranteed timeouts. Skip the check when not
	@# running against an opencode-* local-Ollama model.
	@if echo "$(MODELS)" | grep -qE "opencode-(gemma4|qwen2\.5-coder|qwen3|phi4|gemma3|mistral-small|nemotron|granite-code|starcoder2|deepseek-coder)"; then \
		if [ -x .claude/skills/local-ollama-eval/scripts/verify_setup.sh ]; then \
			.claude/skills/local-ollama-eval/scripts/verify_setup.sh || { \
				echo ""; \
				echo "✗ Rig precondition failed. Fix issues above before running eval."; \
				exit 1; \
			}; \
		fi; \
	fi
	@# When MODELS contains an opencode-* (agent-CLI) model, eval-suite would
	@# reject the run unless -agent + -benchmarks are passed (post-2026-05-23
	@# silent-fallback guard in eval_suite.go). Auto-inject both here so the
	@# canonical `make eval-smoke MODELS=opencode-<x>` invocation just works
	@# without the user having to know about the agent-mode flag dance.
	@#
	@# Detect whether `-agent` is already in EXTRA by surrounding with spaces
	@# and looking for ` -agent ` literally. The naive `-agent\b` regex falsely
	@# matched `-agent-parallel` / `-agent-timeout` (because `\b` triggers at
	@# the `t-` boundary), causing the auto-injection to silently skip.
	@if echo "$(MODELS)" | grep -qE "opencode-" && ! echo " $(EXTRA) " | grep -q -- " -agent "; then \
		SMOKE_LIST=$$(grep -l '^tier: smoke' benchmarks/*.yml 2>/dev/null | xargs -n1 basename | sed 's/\.yml//' | sort | paste -sd ',' -); \
		echo "  Auto-enabling agent mode for opencode-* model (pass '-agent' in EXTRA to opt out of auto-injection)."; \
		echo "  Auto-benchmarks: $$SMOKE_LIST"; \
		$(BUILD_DIR)/$(BINARY) eval-suite -agent -benchmarks $$SMOKE_LIST $(if $(MODELS),-models $(MODELS)) $(EXTRA); \
	else \
		$(BUILD_DIR)/$(BINARY) eval-suite -tier smoke $(if $(MODELS),-models $(MODELS)) $(EXTRA); \
	fi

eval-core: build ## Core tier (~20 benchmarks, dev models)
	@echo "Running CORE tier (dev models)..."
	@$(BUILD_DIR)/$(BINARY) eval-suite -tier core $(if $(MODELS),-models $(MODELS)) $(EXTRA)

eval-stretch: build ## Stretch tier (~8 benchmarks, extended suite)
	@echo "Running STRETCH tier (extended suite)..."
	@$(BUILD_DIR)/$(BINARY) eval-suite -tier stretch $(if $(MODELS),-models $(MODELS),-full) $(EXTRA)

eval-models: build ## List available AI models
	@echo "Available models:"
	@$(BUILD_DIR)/$(BINARY) eval --list-models

eval-report: ## Generate evaluation report from results
	@echo "Generating evaluation report..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev"); \
	$(BUILD_DIR)/$(BINARY) eval-report eval_results/ $$VERSION --format=md

eval-clean: ## Clean evaluation results
	@echo "Cleaning evaluation results..."
	@rm -rf eval_results/*.json eval_results/*.csv eval_results/*.md

# Analysis & Design Doc Generation
eval-analyze: build ## Analyze eval results, generate design docs (with dedup)
	@echo "$(ARROW) Analyzing eval results..."
	@$(BUILD_DIR)/$(BINARY) eval-analyze --results eval_results/ \
		--model gpt5 --output design_docs/planned/ \
		--min-frequency 2

eval-analyze-fresh: build ## Analyze with forced new docs (disable dedup)
	@echo "$(ARROW) Analyzing eval results (forcing new docs)..."
	@$(BUILD_DIR)/$(BINARY) eval-analyze --results eval_results/ \
		--model gpt5 --output design_docs/planned/ \
		--min-frequency 2 --force-new

eval-to-design: eval-suite eval-analyze ## Full workflow: evals -> analysis -> design docs
	@echo "$(GREEN)$(CHECKMARK) Design docs generated in design_docs/planned/$(RESET)"

# Prompt A/B Testing
eval-prompt-ab: build ## Run A/B comparison of prompt versions
	@if [ -z "$(A)" ] || [ -z "$(B)" ]; then \
		echo "Usage: make eval-prompt-ab A=<version1> B=<version2>"; \
		echo "Example: make eval-prompt-ab A=v0.3.0-baseline B=v0.3.0-hints"; \
		exit 1; \
	fi
	@./tools/eval_prompt_ab.sh "$(A)" "$(B)" --model $(MODEL) --langs $(LANGS)

eval-prompt-list: ## List all available prompt versions
	@echo "Available prompt versions:"
	@cat prompts/versions.json | jq -r '.versions | to_entries[] | "  \(.key): \(.value.description)"'
	@echo ""
	@echo "Active version: $$(cat prompts/versions.json | jq -r '.active')"

eval-prompt-hash: ## Compute SHA256 hashes for all prompts
	@for file in prompts/*.md; do \
		hash=$$(shasum -a 256 "$$file" | awk '{print $$1}'); \
		echo "$$(basename $$file): $$hash"; \
	done

# Baseline & Comparison
eval-baseline: build ## Store baseline for a version
	@if [ -z "$(EVAL_VERSION)" ]; then \
		echo "Usage: make eval-baseline EVAL_VERSION=v0.3.X [FULL=true] [RESUME=true] [TIER=core,stretch]"; \
		exit 1; \
	fi
	@echo "Storing baseline for version $(EVAL_VERSION)..."
	@VERSION=$(EVAL_VERSION) FULL=$(FULL) RESUME=$(RESUME) TIER=$(TIER) ./tools/eval_baseline.sh

eval-diff: build ## Compare two eval result directories
	@if [ -z "$(BASELINE)" ] || [ -z "$(NEW)" ]; then \
		echo "Usage: make eval-diff BASELINE=<dir> NEW=<dir>"; \
		exit 1; \
	fi
	@bin/ailang eval-compare "$(BASELINE)" "$(NEW)"

eval-summary: ## Show summary of eval results
	@if [ -z "$(DIR)" ]; then \
		echo "Usage: make eval-summary DIR=<results_dir>"; \
		exit 1; \
	fi
	@bin/ailang eval-summary "$(DIR)"

eval-matrix: ## Generate performance matrix
	@if [ -z "$(DIR)" ] || [ -z "$(VERSION)" ]; then \
		echo "Usage: make eval-matrix DIR=<results_dir> VERSION=<version>"; \
		exit 1; \
	fi
	@bin/ailang eval-matrix "$(DIR)" "$(VERSION)"

# Automated Improvement
eval-auto-improve: ## Automated fix implementation (dry run)
	@echo "M-EVAL-LOOP: Automated Fix Implementation"
	@if [ -n "$(BENCH)" ]; then \
		./tools/eval_auto_improve.sh --benchmark "$(BENCH)"; \
	else \
		./tools/eval_auto_improve.sh; \
	fi

eval-auto-improve-apply: ## Automated fix implementation (apply mode)
	@if [ -n "$(BENCH)" ]; then \
		./tools/eval_auto_improve.sh --benchmark "$(BENCH)" --apply; \
	else \
		./tools/eval_auto_improve.sh --apply; \
	fi

# Phase 2A: Evaluator vs Native Go runtime benchmarks
bench-phase2a: ## Run Phase 2A benchmarks (evaluator vs native Go, -count=3)
	@echo "Phase 2A: Evaluator Performance Benchmarks"
	@echo "==========================================="
	@echo "Running native Go baselines + AILANG evaluator benchmarks..."
	@echo "Startup time EXCLUDED — measures evaluation only."
	@echo ""
	go test -run='^$$' -bench='Benchmark(Native|Eval)_' -benchmem -count=3 \
		./internal/eval/ -timeout=600s

bench-phase2a-quick: ## Run Phase 2A benchmarks (quick, -count=1)
	go test -run='^$$' -bench='Benchmark(Native|Eval)_' -benchmem -count=1 \
		./internal/eval/ -timeout=600s

# M-LAT-BUDGET: Latency-budget canonical workload baselines.
# Runs benchmarks/workloads/*.ail end-to-end with AILANG_NO_TRACE=1 and writes
# benchmarks/latency_budgets.json. The on-disk JSON is the input to bench-check
# (regression gate, deferred to M-LAT-BUDGET Phase 4).
bench-workloads: ## Capture latency-budget baseline (5 runs each, all workloads)
	@./tools/bench_workloads.sh

bench-workloads-quick: ## Quick latency-budget smoke test (3 runs, no file write)
	@./tools/bench_workloads.sh --runs 3 --no-write
