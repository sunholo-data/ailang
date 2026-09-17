package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"runtime/debug"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/iface"
	otelplatform "github.com/sunholo-data/ailang/internal/platform/otel"
	"github.com/sunholo-data/ailang/internal/runner"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// runFile is the `ailang run` process wrapper: GC tuning, telemetry init, the
// root span and the task-id hierarchy env, then one runner.Run with the parsed
// flags and an os.Exit on its code. The run pipeline itself lives in
// internal/runner (M-V1-SIMPLIFY-S2 M3).
func runFile(filename string, programArgs []string, trace bool, seed int, virtualTime bool, jsonOutput bool, compact bool, quiet bool, binopShim bool, failOnShim bool, requireLowering bool, trackInstantiations bool, noMono bool, debugCompile bool, strictSyntax bool, entry string, argsJSON string, print bool, noprint bool, batch bool, caps string, maxRecursionDepth int, stdlibPath string, traceLoader bool, strictVersion bool, allowEnv string, allowEnvFile string, env string, envSnapshot string, writeEnvSnapshot string, aiStub bool, aiModel string, aiRoutingValues routingFlagValues, debugEffect bool, relaxModules bool, debugTypes bool, debugTypesNode uint64, noBudgets bool, budgetReport string, verifyContracts bool, emitTrace string, traceTier string, netAllowHTTP bool, netAllowDomains string, netAllowLocalhost bool, netAllowMetadata bool, netTimeout string, streamAllowHTTP bool, streamAllowDomains string, streamAllowLocalhost bool, processTimeout string, processAllowlist string, processMaxOutput int64, release bool, bytecodeMode bool, strictBytecode bool, orReferer string, orTitle string, orCategories string, fsMaxBytes string) {
	// M-PERF-DOCPARSE: Reduce GC pressure for batch/CLI workloads.
	// Default GOGC=100 triggers GC when heap doubles — too aggressive for short-lived CLI runs.
	// GOGC=500 allows heap to grow 6x before GC, trading ~50MB extra memory for 25%+ speedup.
	// Only applies when GOGC is not already set by the user.
	if !config.GOGCSet() {
		debug.SetGCPercent(500)
	}

	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := otelplatform.Init(ctx, "ailang-run")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: telemetry init failed: %v\n", err)
	} else {
		defer shutdownTelemetry(ctx)
	}

	// Extract parent trace context from environment (if running as subprocess)
	// This enables distributed tracing when ailang is spawned by coordinator/executors
	ctx = telemetry.ExtractTraceContext(ctx)

	// Extract correlation IDs for span attributes (fallback linking)
	corrTaskID, sessionID := telemetry.ExtractCorrelationIDs()

	// Generate task ID for this run command
	runTaskID := fmt.Sprintf("run_%d", time.Now().UnixNano())

	// Inherit parent task from environment if set
	// This enables automatic hierarchy linking when ailang exec spawns ailang run
	parentTaskID := config.ParentTaskID()

	// If no parent task, use generic root marker for analytics
	// This ensures all runs appear in Observatory hierarchy views
	if parentTaskID == "" {
		parentTaskID = "root"
	}

	// Export current task ID so child processes inherit the hierarchy
	os.Setenv("AILANG_PARENT_TASK_ID", runTaskID)

	// Create root span for the entire run command
	// All child spans (compile, execute) will be linked under this
	tracer := otel.Tracer("ailang.cli")
	ctx, rootSpan := tracer.Start(ctx, "ailang run: "+filename,
		oteltrace.WithAttributes(
			attribute.String("file.path", filename),
			attribute.String("entry.function", entry),
			attribute.String("exec.task_id", runTaskID),
			attribute.String("exec.parent_task_id", parentTaskID),
		),
	)
	defer rootSpan.End()

	// Add correlation IDs as span attributes (if present)
	if corrTaskID != "" {
		rootSpan.SetAttributes(attribute.String("ailang.task_id", corrTaskID))
	}
	if sessionID != "" {
		rootSpan.SetAttributes(attribute.String("ailang.session_id", sessionID))
	}

	// Add capabilities to span (if any granted)
	if caps != "" {
		capList := strings.Split(caps, ",")
		rootSpan.SetAttributes(attribute.StringSlice("caps.granted", capList))
	}

	opts := runner.Options{
		Filename:            filename,
		ProgramArgs:         programArgs,
		Trace:               trace,
		Seed:                seed,
		VirtualTime:         virtualTime,
		JSONOutput:          jsonOutput,
		Compact:             compact,
		Quiet:               quiet,
		BinopShim:           binopShim,
		FailOnShim:          failOnShim,
		RequireLowering:     requireLowering,
		TrackInstantiations: trackInstantiations,
		NoMono:              noMono,
		DebugCompile:        debugCompile,
		StrictSyntax:        strictSyntax,
		RelaxModules:        relaxModules,
		DebugTypes:          debugTypes,
		DebugTypesNode:      debugTypesNode,
		Release:             release,
		StdlibPath:          stdlibPath,
		TraceLoader:         traceLoader,
		StrictVersion:       strictVersion,
		Entry:               entry,
		ArgsJSON:            argsJSON,
		Print:               print,
		NoPrint:             noprint,
		Batch:               batch,
		Caps:                caps,
		MaxRecursionDepth:   maxRecursionDepth,
		BytecodeMode:        bytecodeMode,
		StrictBytecode:      strictBytecode,
		Env: runner.EnvFlags{
			AllowEnv:         allowEnv,
			AllowEnvFile:     allowEnvFile,
			Env:              env,
			EnvSnapshot:      envSnapshot,
			WriteEnvSnapshot: writeEnvSnapshot,
		},
		Net: runner.NetOptions{
			AllowHTTP:      netAllowHTTP,
			AllowDomains:   netAllowDomains,
			AllowLocalhost: netAllowLocalhost,
			AllowMetadata:  netAllowMetadata,
			Timeout:        netTimeout,
		},
		Stream: runner.StreamOptions{
			AllowHTTP:      streamAllowHTTP,
			AllowDomains:   streamAllowDomains,
			AllowLocalhost: streamAllowLocalhost,
		},
		Process: runner.ProcessOptions{
			Timeout:   processTimeout,
			Allowlist: processAllowlist,
			MaxOutput: processMaxOutput,
		},
		FSMaxBytes:      fsMaxBytes,
		DebugEffect:     debugEffect,
		DebugLogLevel:   debugLogLevel,
		NoBudgets:       noBudgets,
		BudgetReport:    budgetReport,
		VerifyContracts: verifyContracts,
		EmitTrace:       emitTrace,
		TraceTier:       traceTier,
		RoutingPolicy: func(programIface *iface.Iface, entry string) (*ai.AIRoutingPolicy, error) {
			return resolveRoutingPolicy(aiRoutingValues, programIface, entry)
		},
		AIHandler: func(effCtx *effects.EffContext, policy *ai.AIRoutingPolicy, attr *ai.Attribution) error {
			return setupAIHandler(effCtx, aiStub, aiModel, policy, attr)
		},
		ORReferer:    orReferer,
		ORTitle:      orTitle,
		ORCategories: orCategories,
	}

	if code := runner.Run(ctx, opts); code != 0 {
		os.Exit(code)
	}
}
