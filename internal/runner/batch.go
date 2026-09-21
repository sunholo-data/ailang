package runner

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
)

// ExecuteBatchItem runs one batch input with a fresh runtime and effect context.
// The pipeline result (compilation) is reused — only the execution is per-input.
// M-PERF7: This avoids re-compiling 19 modules per input file.
//
// routingPolicy and attr are resolved once per batch run by the caller (the
// declared mode is the same for every input because the same entry function
// is invoked).
func ExecuteBatchItem(ctx context.Context, result pipeline.Result, input string, opts Options,
	routingPolicy *ai.AIRoutingPolicy, attr *ai.Attribution) error {

	// Fresh runtime per input — prevents state leaks between batch items
	rt := runtime.NewModuleRuntime(".")

	// Each input gets its own args: the input path is the sole program argument
	effCtx := effects.NewEffContext([]string{input})
	defer effCtx.CloseFSRoot() // per-input owner of the sandbox root (M-EXECUTOR-POLICY-HARDENING M1)
	effCtx.Debug = effects.NewDebugContext()
	effects.DebugSink{MinLevel: opts.DebugLogLevel, Label: input}.Attach(effCtx.Debug) // W nil: current os.Stderr
	defer func() {
		FlushDebugOutput(effCtx, opts.DebugLogLevel, input)
	}()
	if err := GrantCapabilities(effCtx, opts.Caps); err != nil {
		return err
	}

	effCtx.GoCtx = ctx

	if opts.NoBudgets {
		effCtx.DisableBudgets = true
	}

	// Set up effect handlers
	SetupSharedMemHandler(effCtx)
	SetupSharedIndexHandler(effCtx)
	if err := SetupNetHandler(effCtx, opts.Net.AllowHTTP, opts.Net.AllowDomains, opts.Net.AllowLocalhost, opts.Net.AllowMetadata, opts.Net.Timeout); err != nil {
		return err
	}
	SetupStreamHandler(effCtx, opts.Stream.AllowHTTP, opts.Stream.AllowDomains, opts.Stream.AllowLocalhost)
	if err := SetupFSLimit(effCtx, opts.FSMaxBytes); err != nil {
		return err
	}
	if err := SetupProcessHandler(effCtx, opts.Process.Timeout, opts.Process.Allowlist, opts.Process.MaxOutput); err != nil {
		return fmt.Errorf("process handler setup: %w", err)
	}
	if opts.AIHandler != nil {
		if err := opts.AIHandler(effCtx, routingPolicy, attr); err != nil {
			return fmt.Errorf("AI handler setup: %w", err)
		}
	}
	// Debug is auto-granted as a ghost effect in NewEffContext() — no conditional setup needed
	if opts.VerifyContracts {
		effCtx.Contracts = effects.NewContractContextWithMode(effects.ContractModePanic)
	}

	// Process environment variable flags. The exit signal (--write-env-snapshot)
	// is ignored per item, as it always was in batch mode; a flag error aborts
	// the whole run (the caller checks for *EnvFlagError).
	if _, err := SetupEnvContext(effCtx, opts.Env); err != nil {
		return err
	}

	rt.GetEvaluator().SetEffContext(effCtx)

	// Wire function callers for builtins that invoke AILANG closures
	{
		evaluator := rt.GetEvaluator()
		effCtx.FnCaller = evaluator.CallValue
		effCtx.FnCallerN = evaluator.CallValueN
	}

	if opts.BinopShim {
		rt.GetEvaluator().SetExperimentalBinopShim(true)
	}
	if result.DictReg != nil {
		rt.GetEvaluator().SetDictionaryRegistry(result.DictReg)
	}

	// Execute module entrypoint
	// M-BYTECODE-BATCH: thread --bytecode / --strict-bytecode and the
	// precompiled pipeline result through so each batch item runs on the
	// VM path (via tryRunEntryViaVM). The BytecodeImage is currently
	// recompiled per item because CompileBytecodeFromResult is cheap
	// relative to per-item runtime setup; a shared-image optimization is
	// tracked for a follow-up.
	execParams := ModuleExecParams{
		Filename:          opts.Filename,
		Iface:             result.Interface,
		Modules:           result.Modules,
		Entry:             opts.Entry,
		ArgsJSON:          opts.ArgsJSON,
		Print:             opts.Print,
		NoPrint:           opts.NoPrint,
		MaxRecursionDepth: opts.MaxRecursionDepth,
		BytecodeMode:      opts.BytecodeMode,
		StrictBytecode:    opts.StrictBytecode,
		Quiet:             opts.Quiet,
		PipelineResult:    &result,
	}
	return runBatchItemEntrypoint(rt, execParams)
}

// runBatchItemEntrypoint executes one batch item's entrypoint, converting the
// exit() sentinel panic into a per-item outcome.
//
// #607: exit() raises a *eval.EvalExitCode sentinel panic (effects/io.go). The
// single-file run path recovers it (Run) and turns it into a clean exit code;
// ExecuteBatchItem called ExecuteModuleEntrypoint directly, so the sentinel
// unwound through the whole batch loop — the process died with rc=2 and a raw
// Go stack, and every remaining input was silently skipped. That is the
// guard-the-helper-miss-the-call-site shape: the recover existed, one call
// site did not have it.
//
// Batch mode's contract is per-item isolation — the "Batch complete: X/Y
// succeeded" summary already promises it — so a non-zero exit fails THAT item
// (the caller counts it and continues to the next input) and exit(0) succeeds.
// Non-exit panics are re-raised unchanged: a genuine crash must stay loud.
func runBatchItemEntrypoint(rt *runtime.ModuleRuntime, params ModuleExecParams) error {
	return RecoverBatchItemExit(func() error {
		return ExecuteModuleEntrypoint(rt, params)
	})
}

// RecoverBatchItemExit runs one batch item and maps the exit() sentinel panic
// onto that item's error result. Split out from runBatchItemEntrypoint so each
// branch is reachable from a unit test without standing up a module runtime.
func RecoverBatchItemExit(run func() error) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		ec, ok := r.(*eval.EvalExitCode)
		if !ok {
			panic(r) // re-panic: not an exit(), so it is a real crash
		}
		if ec.Code != 0 {
			err = fmt.Errorf("program called exit(%d)", ec.Code)
			return
		}
		err = nil // exit(0) — the item finished successfully
	}()
	return run()
}
