package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/sunholo-data/ailang/internal/agentprompt"
	"github.com/sunholo-data/ailang/internal/devtoolsprompt"
	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/riglock"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"github.com/sunholo-data/ailang/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// evalTracer is the OpenTelemetry tracer for eval harness instrumentation.
var evalTracer = otel.Tracer("ailang.eval")

func runEvalSuite() {
	ctx := context.Background()

	// Check if we already have a task ID from coordinator/parent process
	taskID, sessionID := telemetry.ExtractCorrelationIDs()

	// If no task ID provided, generate one for this eval run
	// This enables eval runs to appear in the Observatory task hierarchy
	// Generate assignment ID as well - needed for span grouping in UI
	var assignmentID string
	if taskID == "" {
		taskID = fmt.Sprintf("eval-%d", time.Now().UnixNano())
		assignmentID = fmt.Sprintf("aa_%d", time.Now().UnixNano())
		// Set in environment so it gets picked up by telemetry.NewResource()
		// Both task_id and assignment_id are needed for full hierarchy visibility
		existingAttrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES")
		newAttrs := fmt.Sprintf("ailang.task_id=%s,ailang.assignment_id=%s", taskID, assignmentID)
		if existingAttrs != "" {
			os.Setenv("OTEL_RESOURCE_ATTRIBUTES", existingAttrs+","+newAttrs)
		} else {
			os.Setenv("OTEL_RESOURCE_ATTRIBUTES", newAttrs)
		}
	}

	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-eval")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: telemetry init failed: %v\n", err)
	} else {
		defer shutdownTelemetry(ctx)
	}

	// Extract trace context from environment (enables cross-process trace linking)
	// If TRACEPARENT is set (e.g., by coordinator or CI), this eval suite will be
	// a child span in the parent trace
	ctx = telemetry.ExtractTraceContext(ctx)

	// Start parent span for the entire eval suite
	ctx, suiteSpan := evalTracer.Start(ctx, "eval.suite")
	defer suiteSpan.End()

	// Record correlation IDs as span attributes for fallback trace linking
	if taskID != "" {
		suiteSpan.SetAttributes(attribute.String("ailang.task_id", taskID))
	}
	if sessionID != "" {
		suiteSpan.SetAttributes(attribute.String("ailang.session_id", sessionID))
	}

	// Parse eval-suite subcommand flags
	fs := flag.NewFlagSet("eval-suite", flag.ExitOnError)
	models := fs.String("models", "", "Comma-separated list of models (default: dev models)")
	fullSuite := fs.Bool("full", false, "Run full benchmark suite with all models from extended_suite (gpt5-2-codex, claude-opus-4-6, claude-sonnet-4-6, gemini-3-pro, gemini-2-5-pro)")
	benchmarks := fs.String("benchmarks", "", "Comma-separated list of benchmarks (empty = auto-discover from benchmarks/)")
	tier := fs.String("tier", "", "Comma-separated list of tiers to include (smoke|core|stretch|vision). Empty = all tiers. Applied after benchmark discovery.")
	byConfidence := fs.String("benchmarks-by-confidence", "", "Select the most informative (non-saturated) benchmarks for the run mode from a ratings DB instead of --benchmarks/--tier. Pass a db path, or 'auto' for ~/.ailang/state/observatory.db (M-EVAL-RATING-EFFICIENCY).")
	maxBenchmarks := fs.Int("max-benchmarks", 0, "Cap on benchmarks selected by --benchmarks-by-confidence (0 = no cap)")
	langs := fs.String("langs", "python,ailang", "Comma-separated list of languages")
	seed := fs.Int64("seed", 42, "Random seed for deterministic runs")
	outputDir := fs.String("output", "eval_results", "Output directory for results")
	timeout := fs.Duration("timeout", 30*time.Second, "Timeout for code execution")
	maxConcurrent := fs.Int("parallel", 10, "Maximum concurrent API calls across all providers (0 = sequential, recommended: 10-15)")
	selfRepair := fs.Bool("self-repair", true, "Enable single-shot self-repair on errors (default: true)")
	noSelfRepair := fs.Bool("no-self-repair", false, "Disable self-repair (run without error correction)")
	promptVersion := fs.String("prompt-version", "", "Prompt version ID for all benchmarks")
	skipExisting := fs.Bool("skip-existing", false, "Skip benchmarks that already have result files (resume interrupted run)")
	bankByVersion := fs.Bool("bank-by-version", false, "Namespace the output dir by the AILANG build version (eval_results/.../<version>/). A new build re-evals from scratch and history accumulates per release; with --skip-existing the rotation banks per-version (M-EVAL-VERSION-BANKING)")
	dryRun := fs.Bool("dry-run", false, "Print the planned (model, harness, benchmark) runs and exit without executing")
	noRigLock := fs.Bool("no-rig-lock", false, "Skip the shared rig lock. The rig is a single GPU; by default eval-suite refuses to start if another rig job (nightly/lang-eval/rotation) holds the lock, to prevent thrash/model-reload hangs. Use only on an isolated box.")

	// Agent mode flags
	agent := fs.Bool("agent", false, "Use agent-based evaluation (Claude Code or Gemini CLI)")
	agentModel := fs.String("agent-model", "", "Override agent CLI model (default: use first model from -models flag). Advanced use only.")
	// NOTE: `-agent-parallel` was historically wired here but never actually
	// governed dispatch concurrency. The real semaphore is `-parallel` (see
	// runBenchmarksParallel). The dead flag was a recurring source of confusion
	// (eg. 2026-05-22 and 2026-05-23 rotations where users passed `-agent-parallel 1`
	// expecting serial execution and got 10x oversubscription instead). Removed
	// here; the AgentBenchmarkConfig.MaxConcurrent field is likewise removed.
	agentRequestsPerSecond := fs.Int("agent-rate", 1, "API requests per second (agent mode only)")
	agentTimeout := fs.Int("agent-timeout", 60, "Timeout per benchmark in seconds (agent mode only)")
	maxTokensPerBench := fs.Int("max-tokens-per-bench", 0, "Hard token-budget ceiling per benchmark; aborts mid-run if exceeded (0 = unlimited). M-EVAL-OS-LONGITUDINAL Phase 1: thrash detection for free local models.")
	numTrials := fs.Int("trials", 1, "Number of trials per (benchmark, model, language) combination. M-EVAL-OS-LONGITUDINAL Phase 3: N>=3 produces a per-benchmark pass-rate distribution in summary.json. Default 1 preserves single-trial behaviour.")

	// Contract verification flags (M-CONTRACT-EVAL)
	benchmarkDir := fs.String("benchmark-dir", "", "Directory containing benchmark YAML files (default: benchmarks/ in CWD)")
	verify := fs.Bool("verify", false, "Enable contract verification (run ailang ai-check on solutions)")
	verifyTimeout := fs.Duration("verify-timeout", 5*time.Second, "Per-function Z3 timeout for contract verification")
	devtoolsPrompt := fs.Bool("devtools-prompt", false, "Append devtools prompt to agent system prompt (enables full experiment condition)")
	conditions := fs.String("conditions", "", "Comma-separated experimental conditions (baseline,contract,z3_guided,full,tool_aware). Creates separate jobs per condition like --langs. Overrides --verify and --devtools-prompt.")

	// μRAG injection toggle (M-BRAIN-MICRORAG)
	// auto: respect inherited env (default).
	// on:   force AILANG_MICRORAG_ENABLED=1 in subprocess env.
	// off:  force AILANG_MICRORAG_ENABLED=0 (use for the baseline arm of an A/B run).
	microragMode := fs.String("microrag", "auto", "μRAG knowledge injection: on | off | auto (default: auto). For A/B comparison, run twice with on/off.")

	// Message-based coordination flags (M-UNIFIED-AI-CONTROL-PLANE)
	queueMode := fs.Bool("queue", false, "Run benchmarks via message queue (coordinator processes, crash recovery)")
	queueInbox := fs.String("queue-inbox", "eval-runner", "Inbox for queue mode benchmark jobs")
	queueWait := fs.Bool("queue-wait", true, "Wait for all queued benchmarks to complete (queue mode only)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// M-EVAL-VERSION-BANKING: namespace the output dir by the AILANG build version BEFORE any use of
	// *outputDir (cleanResults, the --skip-existing glob, the metrics writer, the rotation summarizer
	// all read it downstream), so the whole pipeline is version-consistent. A new build -> empty
	// <version>/ dir -> re-evals from scratch; history accumulates one banked set per build/release.
	if *bankByVersion {
		ver := version.Version
		if ver == "" {
			ver = "unknown"
		}
		ver = strings.ReplaceAll(ver, "/", "-") // defensive: keep it a single path segment
		*outputDir = filepath.Join(*outputDir, ver)
		fmt.Printf("%s Version-banking: output → %s\n", cyan("→"), *outputDir)
	}

	// Shared rig lock (M-RIG-LOCK-ENFORCE): the rig is a single GPU. Two eval
	// jobs hitting ollama concurrently thrash and can trigger a model reload that
	// stalls a stream — which has hung runs for hours. Refuse to start if another
	// rig job (nightly/lang-eval/rotation, or another eval-suite) holds the lock,
	// surfacing the conflict immediately instead of relying on anyone to source
	// rig-lock.sh. --dry-run touches no GPU, so it skips the lock. The launchd
	// wrapper already holds the lock and exports AILANG_RIG_LOCK_HELD=1, which
	// riglock.Acquire honours so it never deadlocks against its own parent.
	if !*dryRun && !*noRigLock {
		acquired, release, lockErr := riglock.Acquire(riglock.NoWait)
		if lockErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: rig lock unavailable (%v) — proceeding without it\n", lockErr)
		} else if !acquired {
			holder := riglock.Holder()
			if holder == "" {
				holder = "another rig job"
			}
			fmt.Fprintf(os.Stderr, "Error: the rig is busy — %s holds the rig lock.\n", holder)
			fmt.Fprintf(os.Stderr, "       The rig is a single GPU; running concurrently thrashes ollama and can hang.\n")
			fmt.Fprintf(os.Stderr, "       Wait for it to finish, or pass --no-rig-lock to override (isolated box only).\n")
			os.Exit(1)
		} else {
			defer release()
		}
	}

	// Set package-level eval config from flags (M-CONTRACT-EVAL)
	if *benchmarkDir != "" {
		evalBenchmarkDir = *benchmarkDir
	}
	evalVerifyFlag = *verify
	evalVerifyTimeout = *verifyTimeout
	evalDevtoolsPromptFlag = *devtoolsPrompt
	evalMicroragMode = eval_harness.ParseMicroragMode(*microragMode)

	// Parse --conditions flag into a list
	var conditionList []string
	if *conditions != "" {
		validConditions := map[string]bool{}
		for _, c := range eval_harness.ValidConditionNames {
			validConditions[c] = true
		}
		for _, c := range strings.Split(*conditions, ",") {
			c = strings.TrimSpace(c)
			if !validConditions[c] {
				fmt.Fprintf(os.Stderr, "Error: unknown condition %q (valid: %v)\n", c, eval_harness.ValidConditionNames)
				os.Exit(1)
			}
			conditionList = append(conditionList, c)
		}
	} else {
		// Legacy mode: single job with empty condition (uses --verify/--devtools-prompt flags)
		conditionList = []string{""}
	}

	// Initialize models configuration
	if err := eval_harness.InitModelsConfig(); err != nil {
		if *agent {
			// Agent mode REQUIRES models.yml for executor routing -- fail fast
			fmt.Fprintf(os.Stderr, "Error: Could not load models.yml: %v\n", err)
			fmt.Fprintf(os.Stderr, "Agent mode requires models.yml for executor routing (agent_cli, agent_model_name).\n")
			fmt.Fprintf(os.Stderr, "Ensure models.yml exists at internal/eval_harness/models.yml or is embedded in the binary.\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Warning: Could not load models.yml: %v\n", err)
		fmt.Fprintf(os.Stderr, "Continuing with hardcoded fallback model lists (may be stale).\n")
	}

	// Determine model list
	var modelList []string
	if *models != "" {
		// User specified models explicitly. Recognize named suites as a
		// single token (e.g. --models agent_suite) and expand them to the
		// composite from models.yml. Otherwise fall back to comma-split.
		modelList = expandModelSuite(*models, eval_harness.GlobalModelsConfig)
	} else if *fullSuite {
		// Full suite: use extended suite (5 models) from models.yml
		if eval_harness.GlobalModelsConfig != nil && len(eval_harness.GlobalModelsConfig.ExtendedSuite) > 0 {
			modelList = eval_harness.GlobalModelsConfig.ExtendedSuite
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5-2-codex", "claude-opus-4-6", "claude-sonnet-4-6", "gemini-3-pro", "gemini-2-5-pro"}
		}
	} else {
		// Default: use dev models from models.yml
		if eval_harness.GlobalModelsConfig != nil && len(eval_harness.GlobalModelsConfig.DevModels) > 0 {
			modelList = eval_harness.GlobalModelsConfig.DevModels
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5-mini", "claude-haiku-4-5", "gemini-2-5-flash"}
		}
	}
	// SAFETY: block standard (direct-API) mode only for models that are GENUINELY
	// agent-only — i.e. have an agent_cli AND no cloud standard path (provider is
	// local/CLI-bound, e.g. ollama). Without this guard those degrade silently to
	// junk (the 2026-05-23 incident: 102 trials, total_tokens=0, provider=ollama).
	// Cloud models (anthropic/openai/google/openrouter) that ALSO have an agent_cli
	// — Claude, GPT, Gemini — are dual-mode and run standard natively; they must
	// NOT be blocked (they have hundreds of standard-mode runs across baselines).
	if !*agent && eval_harness.GlobalModelsConfig != nil {
		var agentOnlyModels []string
		for _, m := range modelList {
			if eval_harness.GlobalModelsConfig.SupportsAgentEval(m) &&
				!eval_harness.GlobalModelsConfig.SupportsStandardEval(m) {
				cli, _ := eval_harness.GlobalModelsConfig.GetAgentCLI(m)
				agentOnlyModels = append(agentOnlyModels, fmt.Sprintf("%s (agent_cli: %q)", m, cli))
			}
		}
		if len(agentOnlyModels) > 0 {
			fmt.Fprintf(os.Stderr, "Error: model(s) require agent mode but -agent was not passed:\n")
			for _, m := range agentOnlyModels {
				fmt.Fprintf(os.Stderr, "  - %s\n", m)
			}
			fmt.Fprintf(os.Stderr, "\nThese models are only usable via their agent CLI; standard-mode runs would silently produce empty/garbage results.\n")
			fmt.Fprintf(os.Stderr, "Re-run with -agent and -benchmarks <list>, e.g.:\n")
			fmt.Fprintf(os.Stderr, "  ailang eval-suite -agent -benchmarks fizzbuzz,adt_option -models %s\n", modelList[0])
			os.Exit(1)
		}
	}

	// SAFETY (M-RIG single-GPU): agent runs on agent-only / local models (ollama-backed, e.g. qwen)
	// MUST be serial. The rig is ONE GPU; concurrent trials thrash ollama and crash motoko (each
	// produces a 0-byte JSONL -> "terminated without emitting run_summary"). --parallel defaults to
	// 10, so a raw `eval-suite --agent` silently oversubscribes. Clamp to 1 here. Recurring footgun:
	// the dead -agent-parallel flag, the 2026-05-22/23 rotations, and the 2026-06-22 contract_leap_year
	// run that lost 7/8 trials to GPU contention. (Override with an isolated box only.)
	if *agent && *maxConcurrent > 1 && eval_harness.GlobalModelsConfig != nil {
		for _, m := range modelList {
			if eval_harness.GlobalModelsConfig.SupportsAgentEval(m) && !eval_harness.GlobalModelsConfig.SupportsStandardEval(m) {
				fmt.Fprintf(os.Stderr, "\u26a0 Local/agent-only model on the single-GPU rig \u2014 forcing --parallel 1 (was %d) to avoid ollama thrash + motoko crashes.\n", *maxConcurrent)
				*maxConcurrent = 1
				break
			}
		}
	}

	var benchmarkList []string
	if *byConfidence != "" {
		// Confidence-based selection from a ratings DB (M-EVAL-RATING-EFFICIENCY M3):
		// the harness picks the benchmarks worth running, so it satisfies agent mode's
		// explicit-list requirement too.
		dbPath := *byConfidence
		if dbPath == "auto" {
			dbPath = defaultObservatoryDB()
		}
		evalMode := "standard"
		if *agent {
			evalMode = "agent"
		}
		sel, err := selectBenchmarksByConfidence(dbPath, evalMode, *maxBenchmarks)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: --benchmarks-by-confidence: %v\n", err)
			os.Exit(1)
		}
		benchmarkList = sel
		fmt.Fprintf(os.Stderr, "Selected %d %s benchmarks by confidence from %s\n", len(sel), evalMode, dbPath)
	} else if *benchmarks == "" {
		// SAFETY: Agent mode requires explicit benchmark list to prevent accidental large runs
		if *agent {
			fmt.Fprintf(os.Stderr, "Error: --agent mode requires explicit --benchmarks list\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Agent mode is expensive and time-consuming. You must explicitly specify which benchmarks to run.\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Example:\n")
			fmt.Fprintf(os.Stderr, "  ailang eval-suite --agent --benchmarks cli_args,fizzbuzz\n")
			fmt.Fprintf(os.Stderr, "\n")
			os.Exit(1)
		}

		// Auto-discover benchmarks from benchmarks/ directory (standard mode only)
		benchmarkList = discoverBenchmarks()
		if len(benchmarkList) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No benchmarks found in benchmarks/ directory\n")
			os.Exit(1)
		}
	} else {
		benchmarkList = strings.Split(*benchmarks, ",")
	}
	langList := strings.Split(*langs, ",")

	// Validate models and benchmarks
	for i := range modelList {
		modelList[i] = strings.TrimSpace(modelList[i])
	}
	for i := range benchmarkList {
		benchmarkList[i] = strings.TrimSpace(benchmarkList[i])
	}

	// Apply --tier filter (M-EVAL-SUITE-PREP M3)
	if strings.TrimSpace(*tier) != "" {
		tiers, err := parseTierList(*tier)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: --tier: %v\n", err)
			os.Exit(1)
		}
		before := len(benchmarkList)
		benchmarkList = filterBenchmarksByTier(benchmarkList, tiers)
		fmt.Fprintf(os.Stderr, "Tier filter (%s): %d -> %d benchmarks\n",
			strings.Join(tiers, ","), before, len(benchmarkList))
		if len(benchmarkList) == 0 {
			fmt.Fprintf(os.Stderr, "Error: no benchmarks match tier %s\n", strings.Join(tiers, ","))
			os.Exit(1)
		}
	}
	for i := range langList {
		langList[i] = strings.TrimSpace(langList[i])
	}

	// --dry-run: enumerate planned runs and exit before any execution.
	// Used by M-EXEC-EXPAND to verify agent_suite expands correctly.
	if *dryRun {
		// M-EVAL-OS-LONGITUDINAL Phase 3: include trial dimension in count.
		trials := *numTrials
		if trials < 1 {
			trials = 1
		}
		fmt.Printf("Dry-run: %d model(s) x %d benchmark(s) x %d language(s) x %d trial(s) = %d planned runs\n",
			len(modelList), len(benchmarkList), len(langList), trials,
			len(modelList)*len(benchmarkList)*len(langList)*trials)
		fmt.Printf("Models:     %s\n", strings.Join(modelList, ", "))
		fmt.Printf("Benchmarks: %s\n", strings.Join(benchmarkList, ", "))
		fmt.Printf("Languages:  %s\n", strings.Join(langList, ", "))
		if eval_harness.GlobalModelsConfig != nil {
			fmt.Printf("\nHarness routing (agent_cli per model):\n")
			for _, m := range modelList {
				if cfg, ok := eval_harness.GlobalModelsConfig.Models[m]; ok {
					cli := "<none>"
					if cfg.AgentCLI != nil && *cfg.AgentCLI != "" {
						cli = *cfg.AgentCLI
					}
					fmt.Printf("  %-24s -> %s\n", m, cli)
				} else {
					fmt.Printf("  %-24s -> <unknown model>\n", m)
				}
			}
		}
		return
	}

	// Filter models for agent mode
	if *agent {
		if eval_harness.GlobalModelsConfig == nil {
			fmt.Fprintf(os.Stderr, "Error: models.yml not loaded, cannot determine agent support\n")
			os.Exit(1)
		}

		// Filter to only models that support agent eval
		originalModels := modelList
		modelList = eval_harness.GlobalModelsConfig.FilterAgentSupportedModels(modelList)

		// Warn about skipped models
		if len(modelList) < len(originalModels) {
			skipped := []string{}
			for _, model := range originalModels {
				if !eval_harness.GlobalModelsConfig.SupportsAgentEval(model) {
					skipped = append(skipped, model)
				}
			}
			fmt.Fprintf(os.Stderr, "%s Agent mode: Skipping %d unsupported model(s): %v\n",
				yellow("⚠️"), len(skipped), skipped)
			fmt.Fprintf(os.Stderr, "   These models have no agent_cli configured in models.yml\n")
			fmt.Println()
		}

		if len(modelList) == 0 {
			fmt.Fprintf(os.Stderr, "Error: No models support agent evaluation\n")
			fmt.Fprintf(os.Stderr, "Agent mode requires models with agent_cli configured in models.yml\n")
			fmt.Fprintf(os.Stderr, "Supported: Claude (claude-haiku-4-5, claude-sonnet-4-6) and Gemini (gemini-2-5-flash, gemini-3-flash, etc.)\n")
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "Example:\n")
			fmt.Fprintf(os.Stderr, "  ailang eval-suite --agent --models claude-haiku-4-5,gemini-2-5-flash --benchmarks fizzbuzz\n")
			fmt.Fprintf(os.Stderr, "\n")
			os.Exit(1)
		}
	}

	// Calculate total runs (conditions are a dimension like languages)
	totalRuns := len(modelList) * len(benchmarkList) * len(langList) * len(conditionList)

	// Set suite span attributes now that we know the configuration
	suiteSpan.SetAttributes(
		attribute.StringSlice("eval.models", modelList),
		attribute.StringSlice("eval.benchmarks", benchmarkList),
		attribute.StringSlice("eval.languages", langList),
		attribute.Int("eval.total_runs", totalRuns),
		attribute.Bool("eval.agent_mode", *agent),
		attribute.Bool("eval.full_suite", *fullSuite),
		attribute.Int64("eval.seed", *seed),
	)

	// Create Task in Observatory for this eval run
	// This enables eval suites to appear in the task hierarchy
	createEvalTask(taskID, assignmentID, modelList, benchmarkList, langList, totalRuns, *agent)

	// M-EVAL-CHAINS: Create execution chain for ALL eval runs (standard + agent)
	// Each benchmark × model × language × condition becomes a chain stage
	var evalChain *EvalChainContext
	{
		obsStore, err := observatory.OpenDefaultStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s Warning: Could not open observatory database: %v\n", yellow("⚠️"), err)
			fmt.Fprintf(os.Stderr, "   Eval results will be stored as JSON files only\n")
		} else {
			evalMode := "standard"
			if *agent {
				evalMode = "agent"
			}
			condRef := ""
			if len(conditionList) == 1 && conditionList[0] != "" {
				condRef = "/" + conditionList[0]
			}
			cwd, _ := os.Getwd()
			chain, err := obsStore.CreateChain(ctx, &observatory.ChainCreateRequest{
				SourceType:    observatory.ChainSourceEvalSuite,
				SourceRef:     fmt.Sprintf("%s/%s%s", taskID, evalMode, condRef),
				WorkspacePath: cwd,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s Warning: Could not create eval chain: %v\n", yellow("⚠️"), err)
			} else {
				evalChain = &EvalChainContext{
					Store:   obsStore,
					ChainID: chain.ID,
				}
				fmt.Printf("  Chain ID: %s\n", chain.ID[:8])
			}
		}
	}

	fmt.Printf("%s AILANG Benchmark Suite\n", cyan("🚀"))
	fmt.Println("==========================")
	fmt.Println()
	fmt.Printf("Models:     %v\n", modelList)
	fmt.Printf("Benchmarks: %v\n", benchmarkList)
	fmt.Printf("Languages:  %v\n", langList)
	if conditionList[0] != "" {
		fmt.Printf("Conditions: %v\n", conditionList)
	}
	fmt.Printf("Seed:       %d\n", *seed)
	fmt.Printf("Parallel:   %d concurrent\n", *maxConcurrent)
	fmt.Printf("Total runs: %d\n", totalRuns)
	fmt.Println()

	// Check API keys
	checkAPIKeys(modelList)

	// M-EVAL-GUARD: Start watchdog to detect and kill orphaned eval processes
	watchdog := eval_harness.NewWatchdog(15*time.Minute, 60*time.Second)
	watchdogDone := make(chan struct{})
	go watchdog.Start(watchdogDone)
	defer func() {
		close(watchdogDone)
		if report := watchdog.Report(); report != "No orphaned processes detected" {
			fmt.Printf("%s %s\n", yellow("⚠️"), report)
		}
	}()

	// M-EVAL-GUARD: Setup signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Printf("\n%s Received interrupt, cleaning up...\n", yellow("⚠️"))
		// Kill any remaining orphaned processes
		killed := watchdog.KillOrphans()
		if killed > 0 {
			fmt.Printf("Killed %d orphaned process(es)\n", killed)
		}
		os.Exit(1)
	}()

	// Clean previous results (unless resuming)
	if !*skipExisting {
		fmt.Printf("%s Cleaning previous results...\n", cyan("→"))
		cleanResults(*outputDir)
	} else {
		fmt.Printf("%s Resuming run (skipping existing results)...\n", cyan("→"))
	}

	// Build job list in round-robin order by model
	// This interleaves models to distribute API calls across providers (OpenAI, Anthropic, Google)
	// allowing higher parallelism without hitting single-provider rate limits.
	//
	// Example with 3 models, 2 benchmarks, 2 languages:
	//   Old order: [m1/b1/l1, m1/b1/l2, m1/b2/l1, m1/b2/l2, m2/b1/l1, ...]
	//   New order: [m1/b1/l1, m2/b1/l1, m3/b1/l1, m1/b1/l2, m2/b1/l2, ...]
	//
	// With --parallel 10 and 3 providers, this means ~3-4 concurrent calls per provider
	// instead of 10 calls to the same provider.
	var jobs []Job
	skippedCount := 0
	// M-EVAL-OS-LONGITUDINAL Phase 3: outer trial loop. Default --trials 1
	// generates the same single-job-per-(model,benchmark,lang,condition) tuple
	// as before. --trials N > 1 produces N independent invocations per tuple,
	// each writing its own result JSON with a "trial" field for downstream
	// aggregation by SummarizeRotation.
	trialsToRun := *numTrials
	if trialsToRun < 1 {
		trialsToRun = 1
	}
	if trialsToRun > 1 {
		fmt.Printf("📊 N-trial mode: %d trials per (model, benchmark, language)\n", trialsToRun)
	}
	for trial := 1; trial <= trialsToRun; trial++ {
		for _, condition := range conditionList {
			for _, lang := range langList {
				// For each benchmark, create jobs for all models (round-robin)
				for benchIdx := 0; benchIdx < len(benchmarkList); benchIdx++ {
					for _, model := range modelList {
						benchmark := benchmarkList[benchIdx]
						job := Job{
							Model:     model,
							Benchmark: benchmark,
							Language:  lang,
							Condition: condition,
							Trial:     trial,
						}

						// Check if result already exists (if resuming)
						if *skipExisting {
							// Result filename format: benchmarkID_lang_model_timestamp.json
							// Check in appropriate subdirectory based on eval mode and condition
							var patterns []string
							modeDir := "standard"
							if *agent {
								modeDir = "agent"
							}
							if condition != "" {
								// With conditions: check mode/condition/ subdirectory
								patterns = append(patterns, filepath.Join(*outputDir, modeDir, condition, fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model)))
							} else {
								// Legacy: check mode/ subdirectory
								patterns = append(patterns, filepath.Join(*outputDir, modeDir, fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model)))
							}
							// Also check root directory for legacy results
							patterns = append(patterns, filepath.Join(*outputDir, fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model)))

							foundExisting := false
							for _, pattern := range patterns {
								matches, _ := filepath.Glob(pattern)
								if len(matches) > 0 {
									foundExisting = true
									break
								}
							}

							if foundExisting {
								skippedCount++
								continue // Skip this job
							}
						}

						jobs = append(jobs, job)
					}
				}
			}
		}
	} // close trial loop (M-EVAL-OS-LONGITUDINAL Phase 3)

	if *skipExisting && skippedCount > 0 {
		fmt.Printf("Skipped %d existing results\n", skippedCount)
		fmt.Println()
	}

	// Handle --no-self-repair flag (overrides --self-repair=true default)
	finalSelfRepair := *selfRepair
	if *noSelfRepair {
		finalSelfRepair = false
		fmt.Println("Self-repair DISABLED (--no-self-repair)")
	} else {
		fmt.Println("Self-repair ENABLED (default)")
	}

	// Configure agent mode if requested
	var agentConfig *eval_harness.AgentBenchmarkConfig
	if *agent {
		// Agent CLI model will be determined per-job based on the code generation model
		// (unless --agent-model is explicitly provided as override)
		agentModelOverride := *agentModel

		fmt.Println()
		fmt.Printf("%s Agent mode ENABLED\n", cyan("🤖"))
		fmt.Printf("  - Models: %v\n", modelList)
		if agentModelOverride != "" {
			fmt.Printf("  - Agent CLI model: %s (override)\n", agentModelOverride)
		} else {
			fmt.Printf("  - Agent CLI model: per-model lookup from models.yml\n")
		}
		// Dispatch parallelism is governed by -parallel (the runBenchmarksParallel
		// semaphore). Print THAT value so the banner doesn't mislead the user.
		fmt.Printf("  - Dispatch parallelism: %d (-parallel flag)\n", *maxConcurrent)
		fmt.Printf("  - Rate limit: %d req/sec\n", *agentRequestsPerSecond)
		fmt.Printf("  - Timeout: %d seconds\n", *agentTimeout)
		fmt.Println()

		// M-CONTRACT-EVAL: Load devtools prompt if flag is active or "full" condition is requested
		var devtoolsContent string
		needDevtools := evalDevtoolsPromptFlag
		if !needDevtools {
			for _, c := range conditionList {
				if c == "full" {
					needDevtools = true
					break
				}
			}
		}
		if needDevtools {
			var dtErr error
			devtoolsContent, dtErr = devtoolsprompt.LoadPrompt("v0.8.0-compact")
			if dtErr != nil {
				fmt.Fprintf(os.Stderr, "%s Failed to load devtools prompt: %v\n", yellow("⚠️"), dtErr)
			}
		}

		// Load agent coding prompt if "agent_prompt" condition is requested
		var agentPromptContent string
		for _, c := range conditionList {
			if c == "agent_prompt" {
				var apErr error
				agentPromptContent, apErr = agentprompt.LoadPrompt("latest")
				if apErr != nil {
					fmt.Fprintf(os.Stderr, "%s Failed to load agent prompt: %v\n", yellow("⚠️"), apErr)
				} else {
					fmt.Printf("  - Agent prompt loaded (%d bytes)\n", len(agentPromptContent))
				}
				break
			}
		}

		agentConfig = &eval_harness.AgentBenchmarkConfig{
			MaxTokensPerBench:  *maxTokensPerBench,
			RequestsPerSecond:  *agentRequestsPerSecond,
			TimeoutSeconds:     *agentTimeout,
			WorkspaceDir:       filepath.Join(os.TempDir(), "ailang_eval"),
			AllowedTools:       []string{"Bash", "Read", "Write", "Edit", "Grep"},
			ClaudePath:         "claude",           // Use PATH
			ClaudeModel:        agentModelOverride, // Empty unless override specified
			Verify:             evalVerifyFlag,     // M-CONTRACT-EVAL: enable contract verification
			DevtoolsPrompt:     devtoolsContent,    // M-CONTRACT-EVAL: devtools prompt for "full" condition
			AgentPromptContent: agentPromptContent, // Agent coding prompt for "agent_prompt" condition
			MicroragMode:       evalMicroragMode,   // M-BRAIN-MICRORAG: subprocess env mode
		}
	}

	// M-BRAIN-MICRORAG: For executor-based paths (claude/gemini via internal/executor),
	// the executors call BuildEnvironment() which inherits os.Environ(). Setting the
	// var on this process ensures inheritance reaches those subprocesses too. Direct
	// agent_runner paths use ApplyToEnv on the per-cmd env explicitly.
	switch evalMicroragMode {
	case eval_harness.MicroragModeOn:
		_ = os.Setenv("AILANG_MICRORAG_ENABLED", "1")
	case eval_harness.MicroragModeOff:
		_ = os.Setenv("AILANG_MICRORAG_ENABLED", "0")
	}

	// Queue mode: Submit benchmarks as messages for coordinator processing (M-UNIFIED-AI-CONTROL-PLANE)
	if *queueMode {
		results := runBenchmarksViaQueue(ctx, jobs, taskID, *queueInbox, *seed, *outputDir, *agent, *queueWait, suiteSpan)
		_ = time.Since(time.Now()) // Duration tracked in queue function

		// Summary for queue mode
		successCount := 0
		failCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			} else {
				failCount++
			}
		}

		suiteSpan.SetAttributes(
			attribute.Int("eval.success_count", successCount),
			attribute.Int("eval.fail_count", failCount),
			attribute.Bool("eval.queue_mode", true),
		)
		if failCount > 0 {
			suiteSpan.SetStatus(codes.Error, fmt.Sprintf("%d/%d benchmarks failed", failCount, len(jobs)))
		} else {
			suiteSpan.SetStatus(codes.Ok, "all benchmarks passed")
		}

		fmt.Println()
		fmt.Printf("%s Benchmark suite complete (queue mode)!\n", green("✓"))
		fmt.Printf("Queued: %d jobs to inbox '%s'\n", len(jobs), *queueInbox)
		if *queueWait {
			fmt.Printf("Completed: %d/%d (%.1f%%)\n", successCount, len(jobs), float64(successCount)/float64(len(jobs))*100)
		} else {
			fmt.Println("Jobs queued - use 'ailang messages list --inbox eval-runner' to monitor")
		}
		fmt.Println()
		completeEvalTask(taskID, failCount == 0)
		return
	}

	// Open messaging store for dashboard event visibility (non-blocking on failure)
	var evalStore messaging.MessageStore
	if store, err := openStore(); err == nil {
		evalStore = store
		defer evalStore.Close()

		// Create "Suite Started" message for event queue visibility
		startPayload := map[string]interface{}{
			"task_id":    taskID,
			"models":     modelList,
			"benchmarks": benchmarkList,
			"languages":  langList,
			"total_jobs": len(jobs),
			"agent_mode": *agent,
		}
		payloadBytes, _ := json.Marshal(startPayload)

		startMsg := &messaging.InboxMessage{
			FromAgent:     "eval-suite",
			ToInbox:       "controlplane",
			MessageType:   messaging.InboxTypeNotification,
			Title:         fmt.Sprintf("Eval Suite Started: %d models, %d benchmarks", len(modelList), len(benchmarkList)),
			Payload:       string(payloadBytes),
			Category:      "eval",
			CorrelationID: taskID,
		}
		if err := evalStore.InsertInboxMessageWithContext(ctx, startMsg); err == nil {
			// Broadcast to dashboard via HTTP (non-blocking)
			go broadcastEvalEvent(startMsg)
			// Emit span so event appears in ExecHierarchy (Milestone 12)
			emitEventSpan(ctx, "suite_started", taskID, startMsg)
		}
	}

	// Run benchmarks with concurrency control (direct mode)
	startTime := time.Now()
	results := runBenchmarksParallel(ctx, jobs, *seed, *outputDir, *timeout, *maxConcurrent, finalSelfRepair, *promptVersion, agentConfig, taskID, evalChain)
	duration := time.Since(startTime)

	finalizeSuiteRun(suiteSummaryParams{
		ctx:         ctx,
		results:     results,
		outputDir:   *outputDir,
		totalRuns:   totalRuns,
		trialsToRun: trialsToRun,
		taskID:      taskID,
		evalStore:   evalStore,
		evalChain:   evalChain,
		suiteSpan:   suiteSpan,
		duration:    duration,
	})
}
