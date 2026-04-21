package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/runtime/argdecode"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/vm"
)

// envFlags contains all environment-related command-line flags
type envFlags struct {
	allowEnv         string
	allowEnvFile     string
	env              string
	envSnapshot      string
	writeEnvSnapshot string
}

// setupEnvContext configures the effect context with environment variable settings
// Returns true if the program should exit (e.g., after writing snapshot)
func setupEnvContext(effCtx *effects.EffContext, flags envFlags) bool {
	// 1. Override env vars with --env KEY=VALUE,FOO=bar
	if flags.env != "" {
		for _, pair := range strings.Split(flags.env, ",") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				effCtx.EnvSnapshot[key] = value
			} else {
				fmt.Fprintf(os.Stderr, "%s: invalid --env format '%s' (expected KEY=VALUE)\n", red("Error"), pair)
				os.Exit(1)
			}
		}
	}

	// 2. Load snapshot from JSON file with --env-snapshot
	if flags.envSnapshot != "" {
		snapshotData, err := os.ReadFile(flags.envSnapshot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot read snapshot file '%s': %v\n", red("Error"), flags.envSnapshot, err)
			os.Exit(1)
		}
		var snapshot map[string]string
		if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "%s: invalid snapshot JSON in '%s': %v\n", red("Error"), flags.envSnapshot, err)
			os.Exit(1)
		}
		// Replace snapshot with loaded data
		effCtx.EnvSnapshot = snapshot
	}

	// 3. Set allowlist with --allow-env KEY1,KEY2
	if flags.allowEnv != "" {
		allowlist := []string{}
		for _, key := range strings.Split(flags.allowEnv, ",") {
			key = strings.TrimSpace(key)
			if key != "" {
				allowlist = append(allowlist, key)
			}
		}
		effCtx.EnvAllowlist = allowlist
	}

	// 4. Load allowlist from file with --allow-env-file
	if flags.allowEnvFile != "" {
		allowlistData, err := os.ReadFile(flags.allowEnvFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot read allowlist file '%s': %v\n", red("Error"), flags.allowEnvFile, err)
			os.Exit(1)
		}
		allowlist := []string{}
		for _, line := range strings.Split(string(allowlistData), "\n") {
			line = strings.TrimSpace(line)
			// Skip empty lines and comments
			if line != "" && !strings.HasPrefix(line, "#") {
				allowlist = append(allowlist, line)
			}
		}
		effCtx.EnvAllowlist = allowlist
	}

	// 5. Write snapshot and exit with --write-env-snapshot
	if flags.writeEnvSnapshot != "" {
		snapshotJSON, err := json.MarshalIndent(effCtx.EnvSnapshot, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot marshal snapshot: %v\n", red("Error"), err)
			os.Exit(1)
		}
		if err := os.WriteFile(flags.writeEnvSnapshot, snapshotJSON, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write snapshot file '%s': %v\n", red("Error"), flags.writeEnvSnapshot, err)
			os.Exit(1)
		}
		fmt.Printf("%s Environment snapshot written to %s\n", green("✓"), flags.writeEnvSnapshot)
		return true // Signal to exit
	}

	return false // Continue execution
}

// grantCapabilities parses capability string and grants them to the effect context
func grantCapabilities(effCtx *effects.EffContext, caps string) {
	if caps != "" {
		for _, capName := range strings.Split(caps, ",") {
			capName = strings.TrimSpace(capName)
			if capName != "" {
				effCtx.Grant(effects.NewCapability(capName))
			}
		}
	}
}

// setupSharedMemHandler initializes the SharedMem effect context if the capability is granted.
// SharedMem provides shared memory caching for semantic caching (M-DX15).
func setupSharedMemHandler(effCtx *effects.EffContext) {
	if effCtx.HasCap("SharedMem") {
		// Initialize with in-memory cache (default)
		// Future: could add flags for Redis, memcached, etc.
		effCtx.SharedMem = effects.NewSharedMemContext(nil)
	}
}

// setupNetHandler configures Net effect security settings if the capability is granted.
func setupNetHandler(effCtx *effects.EffContext, allowHTTP bool, allowDomains string, allowLocalhost bool, allowMetadata bool) {
	if effCtx.HasCap("Net") {
		effCtx.Net.AllowHTTP = allowHTTP
		effCtx.Net.AllowLocalhost = allowLocalhost
		effCtx.Net.AllowMetadata = allowMetadata
		if allowDomains != "" {
			for _, d := range strings.Split(allowDomains, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					effCtx.Net.AllowedDomains = append(effCtx.Net.AllowedDomains, d)
				}
			}
		}
	}
}

// setupStreamHandler initializes the Stream effect context if the capability is granted.
// Stream provides bidirectional WebSocket connections (M-STREAM-BIDI).
func setupStreamHandler(effCtx *effects.EffContext, allowHTTP bool, allowDomains string, allowLocalhost bool) {
	if effCtx.HasCap("Stream") {
		effCtx.Stream = effects.NewStreamContext()
		effCtx.Stream.AllowHTTP = allowHTTP
		effCtx.Stream.AllowLocalhost = allowLocalhost
		if allowDomains != "" {
			for _, d := range strings.Split(allowDomains, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					effCtx.Stream.AllowedDomains = append(effCtx.Stream.AllowedDomains, d)
				}
			}
		}
	}
}

// setupProcessHandler initializes the Process effect context if the capability is granted.
// Process provides external command execution (M-PROCESS).
func setupProcessHandler(effCtx *effects.EffContext, timeout string, allowlist string, maxOutput int64) error {
	if !effCtx.HasCap("Process") {
		return nil
	}

	pc := effects.NewProcessContext()

	// Parse timeout duration
	if timeout != "" {
		d, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("invalid --process-timeout %q: %w", timeout, err)
		}
		pc.Timeout = d
	}

	// Set max output
	if maxOutput > 0 {
		pc.MaxOutput = maxOutput
	}

	// Resolve allowlist (path-pinned at startup)
	if err := pc.ResolveAllowlist(allowlist); err != nil {
		return fmt.Errorf("invalid --process-allowlist: %w", err)
	}

	effCtx.Process = pc
	return nil
}

// setupSharedIndexHandler initializes the SharedIndex effect context if the capability is granted.
// SharedIndex provides similarity-based semantic retrieval (M-DX16).
func setupSharedIndexHandler(effCtx *effects.EffContext) {
	if effCtx.HasCap("SharedIndex") {
		// Initialize with in-memory index (default)
		// Future: could add flags for external index backends
		effCtx.SharedIndex = effects.NewSharedIndexContext(nil)
	}
}

// moduleExecParams contains all parameters needed for module execution
type moduleExecParams struct {
	filename          string
	iface             *iface.Iface
	modules           map[string]*loader.LoadedModule
	entry             string
	argsJSON          string
	print             bool
	noprint           bool
	maxRecursionDepth int

	// M-BYTECODE-2D M3: bytecode VM dispatch path. When bytecodeMode is set,
	// executeModuleEntrypoint compiles the entry function (and its module)
	// to bytecode and runs it on the VM, with an evaluator-bridge for any
	// function the compiler couldn't lower. pipelineResult must be non-nil
	// in this mode — the bytecode compile reuses its AST/Core/CoreTI rather
	// than re-running the pipeline.
	bytecodeMode   bool
	strictBytecode bool
	quiet          bool
	pipelineResult *pipeline.Result
}

// executeModuleEntrypoint loads a module, resolves the entrypoint, and executes it
func executeModuleEntrypoint(rt *runtime.ModuleRuntime, params moduleExecParams) error {
	// Set recursion depth limit
	if params.maxRecursionDepth > 0 {
		rt.GetEvaluator().SetMaxRecursionDepth(params.maxRecursionDepth)
	}

	// Pre-load modules from pipeline result
	if params.modules != nil {
		for path, loaded := range params.modules {
			rt.PreloadModule(path, loaded)
		}
	}

	// Load and evaluate module
	inst, err := rt.LoadAndEvaluate(params.iface.Module)
	if err != nil {
		return fmt.Errorf("module evaluation failed: %w", err)
	}

	// Resolve entrypoint
	entry := params.entry
	fnExport, exists := params.iface.Exports[entry]

	if !exists {
		// Auto-select entrypoint if possible
		if entry == "main" {
			var zeroArgFuncs []string
			for name, export := range params.iface.Exports {
				if export.Type != nil {
					if fnType, isFn := export.Type.Type.(*types.TFunc2); isFn {
						if len(fnType.Params) == 0 {
							zeroArgFuncs = append(zeroArgFuncs, name)
						}
					}
				}
			}

			// Case 1: Exactly one zero-arg function
			if len(zeroArgFuncs) == 1 {
				entry = zeroArgFuncs[0]
				fnExport = params.iface.Exports[entry]
				exists = true
			} else if len(zeroArgFuncs) > 1 {
				// Case 2: Multiple zero-arg functions, try "test"
				for _, name := range zeroArgFuncs {
					if name == "test" {
						entry = name
						fnExport = params.iface.Exports[entry]
						exists = true
						break
					}
				}
			}
		}

		if !exists {
			exportNames := []string{}
			for name := range params.iface.Exports {
				exportNames = append(exportNames, name)
			}
			return fmt.Errorf("entrypoint '%s' not found in module\nAvailable exports: %v", entry, exportNames)
		}
	}

	// Check function type
	scheme := fnExport.Type
	if scheme == nil {
		return fmt.Errorf("entrypoint '%s' has no type information", entry)
	}

	fnType, isFn := scheme.Type.(*types.TFunc2)
	if !isFn {
		return fmt.Errorf("entrypoint '%s' is not a function (has type %s)", entry, scheme.Type)
	}

	// Get entrypoint value
	entrypointVal, err := inst.GetExport(entry)
	if err != nil {
		exportNames := runtime.GetExportNames(inst)
		return fmt.Errorf("entrypoint '%s' not found in module %s\nAvailable exports: %s",
			entry, params.iface.Module, strings.Join(exportNames, ", "))
	}

	// Check arity
	arity, err := runtime.GetArity(entrypointVal)
	if err != nil {
		return fmt.Errorf("entrypoint '%s' is not a function: %w", entry, err)
	}
	if arity > 1 {
		return fmt.Errorf("entrypoint '%s' takes %d parameters. v0.2.0 supports 0 or 1.\nSuggestion: wrap as 'wrapper(p:{...}) -> ...' and pass --args-json", entry, arity)
	}

	// Decode arguments
	args, err := decodeEntrypointArgs(params.argsJSON, fnType, entry)
	if err != nil {
		return err
	}

	// M-BYTECODE-2D M3: bytecode VM dispatch path. The bridge is wired in
	// non-strict mode so that EvalOnly functions (effectful, polymorphic-
	// dictionary, ADT-shaped) trap back to the evaluator transparently. In
	// strict mode the bridge is intentionally NOT wired: any call into an
	// EvalOnly stub becomes a hard VM error so the strict gate fails loudly.
	if params.bytecodeMode {
		ranOnVM, vmErr := tryRunEntryViaVM(rt, inst, params, entry, args)
		if ranOnVM {
			return nil
		}
		if params.strictBytecode {
			return fmt.Errorf("bytecode execution failed: %w", vmErr)
		}
		if !params.quiet {
			fmt.Fprintf(os.Stderr, "%s bytecode path unavailable (%v); falling back to evaluator\n", yellow("⚠"), vmErr)
		}
		// Fall through to evaluator path below.
	}

	// Call the entrypoint function
	execResult, err := runtime.CallEntrypoint(rt, inst, entry, args)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Print result if not Unit and not suppressed
	if execResult.Type() != "unit" && !params.noprint {
		if params.print {
			fmt.Println(execResult.String())
		}
	}

	return nil
}

// tryRunEntryViaVM compiles the entry module to bytecode (reusing the
// pipeline result that the evaluator path already produced) and runs the
// named entry function on the VM. Functions the compiler couldn't lower are
// dispatched back to the evaluator through bytecodeBridge — that bridge
// requires the module instance to already be evaluated, which is why this
// helper runs *after* rt.LoadAndEvaluate.
//
// Returns (true, nil) on a successful VM run (result already printed).
// Returns (false, err) when the bytecode path could not be used at all
// (compile failure for the entry, missing prototype, unsupported arity).
// Runtime VM errors that occur after dispatch starts are returned as
// (false, err) too — the caller decides strict vs. fallback.
func tryRunEntryViaVM(rt *runtime.ModuleRuntime, inst *runtime.ModuleInstance, params moduleExecParams, entry string, args []eval.Value) (bool, error) {
	if params.pipelineResult == nil {
		return false, fmt.Errorf("internal: bytecodeMode set but pipelineResult is nil")
	}

	// Recover from lower/compile panics so they degrade to a normal compile
	// error and let the caller fall back to the evaluator in non-strict mode.
	// The lower pass still contains a few `panic(...)` fast-fails for shapes
	// that haven't been bridged yet (e.g. non-tail-position match lowering);
	// treating them as compile errors is the M3 contract.
	var img *bytecode.BytecodeImage
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("compile panic: %v", r)
			}
		}()
		img, err = compileBytecodeFromResult(*params.pipelineResult, params.iface.Module)
	}()
	if err != nil {
		return false, fmt.Errorf("compile: %w", err)
	}
	if err := img.Validate(); err != nil {
		return false, fmt.Errorf("validate: %w", err)
	}

	proto := findEntryProto(img, entry)
	if proto == nil {
		return false, fmt.Errorf("entry function %q not found in bytecode image", entry)
	}

	// Convert decoded args from eval.Value to bytecode.Value via the bridge.
	bcArgs := make([]bytecode.Value, 0, len(args))
	for i, a := range args {
		bv, err := evalValueToBytecode(a)
		if err != nil {
			return false, fmt.Errorf("entry arg %d: %w", i, err)
		}
		bcArgs = append(bcArgs, bv)
	}

	// The lower pass synthesizes a Unit parameter for nullary functions so
	// they round-trip through the call ABI. Pad if needed.
	if proto.NumParams == 1 && len(bcArgs) == 0 {
		bcArgs = []bytecode.Value{bytecode.Unit()}
	}
	if int(proto.NumParams) != len(bcArgs) {
		return false, fmt.Errorf("entry %q expects %d args, got %d", entry, proto.NumParams, len(bcArgs))
	}

	if !params.quiet {
		fmt.Fprintf(os.Stderr, "%s Running %s via bytecode VM\n", green("✓"), params.filename)
	}

	machine := vm.NewVM(img)
	// Wire the eval bridge in non-strict mode so EvalOnly stubs trap back
	// to the evaluator. In strict mode, leaving Interop nil makes any trap
	// a loud VM error.
	if !params.strictBytecode {
		machine.Interop = newBytecodeBridge(rt, inst, rt.GetEvaluator())
	}

	// If the entry itself is EvalOnly the VM adds nothing on top of the
	// evaluator — dispatch through runtime.CallEntrypoint directly so we
	// get the normal per-request evaluator fork, global resolver setup and
	// goroutine-evaluator registration. Going through bridge.CallEvalFunc
	// here (which was the original fallback) skips those steps and
	// subtly breaks effects that depend on resolver/fork state — e.g.
	// Process subprocess stdout ordering.
	if proto.EvalOnly {
		if params.strictBytecode {
			return false, fmt.Errorf("entry %q is evaluator-only (%s); strict mode disables bridge dispatch", entry, proto.EvalReason)
		}
		result, err := runtime.CallEntrypoint(rt, inst, entry, args)
		if err != nil {
			return false, fmt.Errorf("eval-only entry %q: %w", entry, err)
		}
		if !params.noprint && params.print && result != nil && result.Type() != "unit" {
			fmt.Println(result.String())
		}
		return true, nil
	}

	result, err := machine.Run(proto, bcArgs)
	if err != nil {
		return false, fmt.Errorf("vm: %w", err)
	}
	printVMResult(result, params)
	return true, nil
}

// printVMResult mirrors the evaluator path's print policy for VM-side
// results: skip Unit, honor --no-print, and render the return value in
// evaluator-compatible spelling. We route the VM value through the bridge
// back to an eval.Value so that String() matches the tree-walking path
// byte-for-byte — bytecode.Value.String() is debug-oriented (it prints
// strings with Go %q quoting and would break parity tests).
func printVMResult(v bytecode.Value, params moduleExecParams) {
	if params.noprint {
		return
	}
	if v.Tag == bytecode.TagUnit {
		return
	}
	if !params.print {
		return
	}
	ev, err := bytecodeValueToEval(v)
	if err != nil || ev == nil {
		// Fall back to the bytecode value's own formatting for shapes
		// the bridge doesn't know how to convert (currently none on
		// the success path — all Tier-1 shapes round-trip).
		fmt.Println(v.String())
		return
	}
	if ev.Type() == "unit" {
		return
	}
	fmt.Println(ev.String())
}

// decodeEntrypointArgs decodes JSON arguments based on function type signature
func decodeEntrypointArgs(argsJSON string, fnType *types.TFunc2, entry string) ([]eval.Value, error) {
	var args []eval.Value

	if len(fnType.Params) == 0 {
		// M-DX10: Zero-arg functions in surface syntax are actually unit-arg in core
		// Pass unit argument for entry invocation
		if argsJSON != "null" {
			return nil, fmt.Errorf("entrypoint '%s' takes no arguments, but --args-json was provided", entry)
		}
		args = []eval.Value{&eval.UnitValue{}} // Unit-argument model
	} else if len(fnType.Params) == 1 {
		// Single-arg function - decode JSON to match parameter type
		argVal, err := argdecode.DecodeJSON(argsJSON, fnType.Params[0])
		if err != nil {
			return nil, fmt.Errorf("failed to decode arguments: %w", err)
		}
		args = []eval.Value{argVal}
	} else {
		// Multi-arg functions not yet supported
		return nil, fmt.Errorf("entrypoint '%s' has %d parameters (only 0 or 1 supported in v0.2.0)", entry, len(fnType.Params))
	}

	return args, nil
}

// printNonModuleResult prints the result of non-module file execution
func printNonModuleResult(value eval.Value, print, noprint bool) {
	if value != nil && value.Type() != "unit" && !noprint {
		if print {
			fmt.Println(value.String())
		}
	}
}

// executeBatchItem runs one batch input with a fresh runtime and effect context.
// The pipeline result (compilation) is reused — only the execution is per-input.
// M-PERF7: This avoids re-compiling 19 modules per input file.
func executeBatchItem(ctx context.Context, result pipeline.Result, input string,
	entry string, argsJSON string, printResult bool, noprint bool, caps string,
	maxRecursionDepth int, noBudgets bool, budgetReport string, debugEffect bool,
	verifyContracts bool, emitTrace string, binopShim bool, quiet bool,
	netAllowHTTP bool, netAllowDomains string, netAllowLocalhost bool, netAllowMetadata bool,
	streamAllowHTTP bool, streamAllowDomains string, streamAllowLocalhost bool,
	processTimeout string, processAllowlist string, processMaxOutput int64,
	aiStub bool, aiModel string,
	allowEnv string, allowEnvFile string, env string, envSnapshot string, writeEnvSnapshot string,
	filename string, bytecodeMode bool, strictBytecode bool) error {

	// Fresh runtime per input — prevents state leaks between batch items
	rt := runtime.NewModuleRuntime(".")

	// Each input gets its own args: the input path is the sole program argument
	effCtx := effects.NewEffContext([]string{input})
	grantCapabilities(effCtx, caps)

	effCtx.GoCtx = ctx

	if noBudgets {
		effCtx.DisableBudgets = true
	}

	// Set up effect handlers
	setupSharedMemHandler(effCtx)
	setupSharedIndexHandler(effCtx)
	setupNetHandler(effCtx, netAllowHTTP, netAllowDomains, netAllowLocalhost, netAllowMetadata)
	setupStreamHandler(effCtx, streamAllowHTTP, streamAllowDomains, streamAllowLocalhost)
	if err := setupProcessHandler(effCtx, processTimeout, processAllowlist, processMaxOutput); err != nil {
		return fmt.Errorf("process handler setup: %w", err)
	}
	if err := setupAIHandler(effCtx, aiStub, aiModel); err != nil {
		return fmt.Errorf("AI handler setup: %w", err)
	}
	// Debug is auto-granted as a ghost effect in NewEffContext() — no conditional setup needed
	if verifyContracts {
		effCtx.Contracts = effects.NewContractContextWithMode(effects.ContractModePanic)
	}

	// Process environment variable flags
	envConfig := envFlags{
		allowEnv:         allowEnv,
		allowEnvFile:     allowEnvFile,
		env:              env,
		envSnapshot:      envSnapshot,
		writeEnvSnapshot: writeEnvSnapshot,
	}
	setupEnvContext(effCtx, envConfig)

	rt.GetEvaluator().SetEffContext(effCtx)

	// Wire function callers for builtins that invoke AILANG closures
	{
		evaluator := rt.GetEvaluator()
		effCtx.FnCaller = evaluator.CallValue
		effCtx.FnCallerN = evaluator.CallValueN
	}

	if binopShim {
		rt.GetEvaluator().SetExperimentalBinopShim(true)
	}
	if result.DictReg != nil {
		rt.GetEvaluator().SetDictionaryRegistry(result.DictReg)
	}

	// Execute module entrypoint
	// M-BYTECODE-BATCH: thread --bytecode / --strict-bytecode and the
	// precompiled pipeline result through so each batch item runs on the
	// VM path (via tryRunEntryViaVM). The BytecodeImage is currently
	// recompiled per item because compileBytecodeFromResult is cheap
	// relative to per-item runtime setup; a shared-image optimization is
	// tracked for a follow-up.
	execParams := moduleExecParams{
		filename:          filename,
		iface:             result.Interface,
		modules:           result.Modules,
		entry:             entry,
		argsJSON:          argsJSON,
		print:             printResult,
		noprint:           noprint,
		maxRecursionDepth: maxRecursionDepth,
		bytecodeMode:      bytecodeMode,
		strictBytecode:    strictBytecode,
		quiet:             quiet,
		pipelineResult:    &result,
	}
	return executeModuleEntrypoint(rt, execParams)
}

// debugLogLevel is the minimum severity level to print. Set by --log-level flag.
// 0=DEBUG/TRACE, 1=INFO, 2=WARNING, 3=ERROR, 4=NONE (suppress all)
var debugLogLevel int

// parseLogLevel converts a log level string to a numeric severity threshold.
func parseLogLevel(level string) int {
	switch strings.ToLower(level) {
	case "debug", "trace":
		return 0
	case "info", "":
		return 1
	case "warn", "warning":
		return 2
	case "error", "err":
		return 3
	case "none", "off":
		return 4
	default:
		return 1 // default to INFO
	}
}

// severityLevel returns the numeric level for a severity string found in JSON log messages.
func severityLevel(severity string) int {
	switch severity {
	case "DEBUG", "TRACE":
		return 0
	case "INFO":
		return 1
	case "WARNING":
		return 2
	case "ERROR":
		return 3
	default:
		return 1
	}
}

// extractSeverity extracts the "severity" field from a JSON log message.
// Returns "" if the message is not JSON or has no severity field.
func extractSeverity(msg string) string {
	// Fast path: check if it looks like JSON with severity
	if len(msg) < 2 || msg[0] != '{' {
		return ""
	}
	var parsed struct {
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal([]byte(msg), &parsed); err != nil {
		return ""
	}
	return parsed.Severity
}

// flushDebugOutput collects Debug effect logs and prints them to stderr.
// Respects debugLogLevel for severity filtering. Called after execution completes.
func flushDebugOutput(effCtx *effects.EffContext) {
	if effCtx == nil || effCtx.Debug == nil {
		return
	}
	out := effCtx.Debug.Collect()
	for _, log := range out.Logs {
		// Filter by log level if set
		if debugLogLevel > 0 {
			severity := extractSeverity(log.Message)
			if severity != "" && severityLevel(severity) < debugLogLevel {
				continue
			}
		}
		fmt.Fprintf(os.Stderr, "%s\n", log.Message)
	}
	for _, a := range out.Assertions {
		if !a.Passed {
			fmt.Fprintf(os.Stderr, "[ASSERT FAIL] %s at %s\n", a.Message, a.Location)
		}
	}
}
