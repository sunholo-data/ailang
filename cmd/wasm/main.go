//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/repl"
	"github.com/sunholo-data/ailang/std"
)

// Version information (injected at build time via -ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

// WasmREPL wraps the REPL for browser use
type WasmREPL struct {
	repl         *repl.REPL
	registry     *repl.ModuleRegistry
	output       *bytes.Buffer
	stdlibLoaded []string          // modules that loaded successfully
	stdlibFailed map[string]string // module name -> last load error
}

// loadEmbeddedStdlib loads all stdlib modules from the embedded filesystem
// into the module registry. This enables imports like `import std/list` in browser.
// Uses multi-pass loading to handle dependencies: modules that fail are retried
// until all succeed or no more progress can be made.
//
// Returns (loadedNames, failedNameToError, topLevelError). topLevelError is only
// non-nil for catastrophic failures (e.g. embedded FS unreadable).
func loadEmbeddedStdlib(registry *repl.ModuleRegistry) ([]string, map[string]string, error) {
	// Read all .ail files from embedded std/ directory
	entries, err := std.FS.ReadDir(".")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read embedded stdlib: %w", err)
	}

	// Collect all module sources
	type moduleSource struct {
		name    string
		content string
	}
	var pending []moduleSource

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ail") {
			continue
		}

		// Read file content
		content, err := std.FS.ReadFile(entry.Name())
		if err != nil {
			continue
		}

		// Module name is "std/<filename without .ail>"
		moduleName := "std/" + strings.TrimSuffix(entry.Name(), ".ail")
		pending = append(pending, moduleSource{name: moduleName, content: string(content)})
	}

	// Track per-module last error so we can expose diagnostics to JS.
	// Without this, silent failures in stdlib loading are invisible to the
	// browser-side debugging and produce mysterious "undefined global variable"
	// errors when user code imports a module that failed to load.
	lastErrors := make(map[string]string)
	var loaded []string

	// Multi-pass loading: retry failed modules until no progress
	const maxPasses = 5
	for pass := 0; pass < maxPasses && len(pending) > 0; pass++ {
		var stillPending []moduleSource

		for _, mod := range pending {
			// Panic recovery per module — prevents one bad module from crashing WASM
			func() {
				defer func() {
					if r := recover(); r != nil {
						msg := fmt.Sprintf("panic: %v", r)
						lastErrors[mod.name] = msg
						if console := js.Global().Get("console"); !console.IsUndefined() {
							console.Call("warn", fmt.Sprintf("Panic loading %s: %v", mod.name, r))
						}
						stillPending = append(stillPending, mod)
					}
				}()
				_, err := registry.LoadModule(mod.name, mod.content)
				if err != nil {
					// Module failed - might need dependencies loaded first
					lastErrors[mod.name] = err.Error()
					stillPending = append(stillPending, mod)
				} else {
					delete(lastErrors, mod.name)
					loaded = append(loaded, mod.name)
				}
			}()
		}

		// Check if we made progress
		if len(stillPending) == len(pending) {
			// No progress made - remaining modules have unresolvable issues.
			// Log per-module errors to console so browser devtools can surface them.
			if console := js.Global().Get("console"); !console.IsUndefined() {
				for _, mod := range stillPending {
					msg := lastErrors[mod.name]
					if msg == "" {
						msg = "(no error captured)"
					}
					console.Call("error", fmt.Sprintf("Failed to load stdlib module %s: %s", mod.name, msg))
				}
			}
			break
		}

		pending = stillPending
	}

	return loaded, lastErrors, nil
}

// NewWasmREPL creates a new browser-ready REPL
func NewWasmREPL() *WasmREPL {
	registry := repl.NewModuleRegistry()
	replInstance := repl.NewWithVersion(Version, BuildTime)

	// Connect registry to REPL for import resolution
	replInstance.SetRegistry(registry)

	// Share REPL's effect context with registry so InvokeExport can use
	// configured effect handlers (AI, IO, etc.)
	registry.SetEffContext(replInstance.GetEffContext())

	// Load embedded stdlib modules (enables import std/list, std/json, etc.)
	loaded, failed, err := loadEmbeddedStdlib(registry)
	if err != nil {
		// Log error but continue - REPL can still work without stdlib
		if console := js.Global().Get("console"); !console.IsUndefined() {
			console.Call("warn", "Failed to load embedded stdlib: "+err.Error())
		}
	}

	w := &WasmREPL{
		repl:         replInstance,
		registry:     registry,
		output:       &bytes.Buffer{},
		stdlibLoaded: loaded,
		stdlibFailed: failed,
	}

	// Auto-import prelude for numeric defaults (just like CLI REPL)
	// This is discarded since we don't want to show import message on init
	discardBuf := &bytes.Buffer{}
	w.repl.HandleCommand(":import std/prelude", discardBuf)

	return w
}

// Eval evaluates a single expression and returns the result
func (w *WasmREPL) Eval(input string) string {
	w.output.Reset()
	// Process expression through the REPL pipeline
	// Note: This bypasses the Start() method which requires stdin/stdout
	w.repl.ProcessExpression(input, w.output)
	return w.output.String()
}

// HandleCommand processes REPL commands like :type, :help
func (w *WasmREPL) HandleCommand(cmd string) string {
	w.output.Reset()
	w.repl.HandleCommand(cmd, w.output)
	return w.output.String()
}

// Reset clears the REPL environment, reconnects registry, and reloads stdlib.
func (w *WasmREPL) Reset() string {
	w.registry = repl.NewModuleRegistry()
	w.repl = repl.NewWithVersion(Version, BuildTime)

	// Reconnect registry to REPL (required for import resolution)
	w.repl.SetRegistry(w.registry)

	// Share REPL's effect context with registry (required for effect handlers)
	w.registry.SetEffContext(w.repl.GetEffContext())

	// Reload embedded stdlib modules
	loaded, failed, err := loadEmbeddedStdlib(w.registry)
	if err != nil {
		if console := js.Global().Get("console"); !console.IsUndefined() {
			console.Call("warn", "Failed to reload stdlib after reset: "+err.Error())
		}
	}
	w.stdlibLoaded = loaded
	w.stdlibFailed = failed

	// Re-import prelude for numeric defaults
	discardBuf := &bytes.Buffer{}
	w.repl.HandleCommand(":import std/prelude", discardBuf)

	return "Environment reset"
}

// LoadModule compiles and registers a module from source code.
// Returns (exports []string, error) where error is nil on success.
func (w *WasmREPL) LoadModule(name, code string) (exports []string, err error) {
	// Panic recovery to prevent WASM runtime crash
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("internal error: %v", r)
			exports = nil
		}
	}()

	return w.registry.LoadModule(name, code)
}

// GetRegistry returns the module registry for import resolution
func (w *WasmREPL) GetRegistry() *repl.ModuleRegistry {
	return w.registry
}

// Global REPL instance
var replInstance *WasmREPL

// evalExpression is the main entry point called from JavaScript
func evalExpression(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return "Error: no input provided"
	}

	input := args[0].String()

	// Handle commands (start with :)
	if len(input) > 0 && input[0] == ':' {
		return replInstance.HandleCommand(input)
	}

	// Evaluate expression
	return replInstance.Eval(input)
}

// resetREPL resets the REPL environment
func resetREPL(this js.Value, args []js.Value) interface{} {
	return replInstance.Reset()
}

// getVersion returns version info
func getVersion(this js.Value, args []js.Value) interface{} {
	return map[string]interface{}{
		"version":   Version,
		"buildTime": BuildTime,
		"platform":  "browser/wasm",
	}
}

// loadModule loads an AILANG module and returns its exports
// JavaScript: ailangLoadModule(name, code) -> {success: bool, exports?: string[], error?: string}
func loadModule(this js.Value, args []js.Value) interface{} {
	// Validate arguments
	if len(args) < 2 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangLoadModule requires 2 arguments: name and code",
		}
	}

	name := args[0].String()
	code := args[1].String()

	// Validate module name
	if name == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "module name cannot be empty",
		}
	}

	// Load the module (includes panic recovery)
	exports, err := replInstance.LoadModule(name, code)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	// Convert exports to JavaScript array
	jsExports := make([]interface{}, len(exports))
	for i, exp := range exports {
		jsExports[i] = exp
	}

	return map[string]interface{}{
		"success": true,
		"exports": jsExports,
	}
}

// listModules returns the list of loaded modules
// JavaScript: ailangListModules() -> string[]
func listModules(this js.Value, args []js.Value) interface{} {
	modules := replInstance.GetRegistry().ListModules()
	jsModules := make([]interface{}, len(modules))
	for i, mod := range modules {
		jsModules[i] = mod
	}
	return jsModules
}

// stdlibStatus returns diagnostic info about embedded stdlib loading.
// JavaScript: ailangStdlibStatus() -> {loaded: string[], failed: {name, error}[]}
//
// This surfaces silent stdlib load failures so that callers can diagnose
// "undefined global variable: X from std/Y" errors originating from a
// partially-loaded stdlib (e.g. a module that silently failed multi-pass
// loading because of a dependency or codegen issue).
func stdlibStatus(this js.Value, args []js.Value) interface{} {
	loaded := make([]interface{}, len(replInstance.stdlibLoaded))
	for i, name := range replInstance.stdlibLoaded {
		loaded[i] = name
	}
	failed := make([]interface{}, 0, len(replInstance.stdlibFailed))
	for name, errMsg := range replInstance.stdlibFailed {
		failed = append(failed, map[string]interface{}{
			"name":  name,
			"error": errMsg,
		})
	}
	return map[string]interface{}{
		"loaded": loaded,
		"failed": failed,
	}
}

// callExport calls a function from a loaded module with arguments
// JavaScript: ailangCall(moduleName, funcName, ...args) -> {success: bool, result?: string, error?: string}
func callExport(this js.Value, args []js.Value) interface{} {
	// Validate arguments
	if len(args) < 2 {
		return map[string]interface{}{
			"success": false,
			"error":   "ailangCall requires at least 2 arguments: moduleName and funcName",
		}
	}

	moduleName := args[0].String()
	funcName := args[1].String()

	// Convert remaining JS arguments to AILANG values
	ailangArgs := make([]eval.Value, 0, len(args)-2)
	for i := 2; i < len(args); i++ {
		arg := args[i]
		var ailangVal eval.Value

		switch arg.Type() {
		case js.TypeNumber:
			f := arg.Float()
			if f == float64(int(f)) {
				ailangVal = &eval.IntValue{Value: int(f)}
			} else {
				ailangVal = &eval.FloatValue{Value: f}
			}
		case js.TypeString:
			ailangVal = &eval.StringValue{Value: arg.String()}
		case js.TypeBoolean:
			ailangVal = &eval.BoolValue{Value: arg.Bool()}
		default:
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("unsupported argument type at position %d: %v", i-1, arg.Type()),
			}
		}

		ailangArgs = append(ailangArgs, ailangVal)
	}

	// Call the function directly using InvokeExport
	result, err := replInstance.GetRegistry().InvokeExport(moduleName, funcName, ailangArgs)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	// Convert result to native JS type (bool→boolean, int→number, etc.)
	// Also include string representation for display/logging.
	return map[string]interface{}{
		"success": true,
		"result":  ailangValueToJS(result),
		"display": formatValue(result),
	}
}

// formatValue converts an eval.Value to a string representation for JavaScript
func formatValue(v eval.Value) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case *eval.IntValue:
		return fmt.Sprintf("%d", val.Value)
	case *eval.FloatValue:
		return fmt.Sprintf("%g", val.Value)
	case *eval.StringValue:
		return val.Value // Return raw string, not quoted
	case *eval.BoolValue:
		if val.Value {
			return "true"
		}
		return "false"
	case *eval.ListValue:
		var parts []string
		for _, elem := range val.Elements {
			parts = append(parts, formatValue(elem))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *eval.RecordValue:
		var parts []string
		for k, v := range val.Fields {
			parts = append(parts, fmt.Sprintf("%s: %s", k, formatValue(v)))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case *eval.TaggedValue:
		if len(val.Fields) == 0 {
			return val.CtorName
		}
		var parts []string
		for _, f := range val.Fields {
			parts = append(parts, formatValue(f))
		}
		return val.CtorName + "(" + strings.Join(parts, ", ") + ")"
	case *eval.UnitValue:
		return "()"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// evalAsync: ailangEvalAsync(expr) -> Promise<string>
// Evaluates an expression asynchronously, returning a JS Promise.
// Required when the expression may trigger effect handlers that return Promises.
func evalAsync(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return newRejectedPromise("ailangEvalAsync requires 1 argument: expression")
	}

	input := args[0].String()

	handler := js.FuncOf(func(_ js.Value, pArgs []js.Value) interface{} {
		resolve := pArgs[0]
		reject := pArgs[1]
		go func() {
			defer func() {
				if r := recover(); r != nil {
					reject.Invoke(fmt.Sprintf("panic: %v", r))
				}
			}()

			// Handle commands
			if len(input) > 0 && input[0] == ':' {
				result := replInstance.HandleCommand(input)
				resolve.Invoke(result)
				return
			}

			result := replInstance.Eval(input)
			resolve.Invoke(result)
		}()
		return nil
	})

	return js.Global().Get("Promise").New(handler)
}

// callAsync: ailangCallAsync(moduleName, funcName, ...args) -> Promise<{success, result?, error?}>
// Calls a module export asynchronously, returning a JS Promise.
// Required when the function may trigger effect handlers that return Promises.
func callAsync(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return newRejectedPromise("ailangCallAsync requires at least 2 arguments: moduleName and funcName")
	}

	moduleName := args[0].String()
	funcName := args[1].String()

	// Convert remaining JS arguments to AILANG values
	ailangArgs := make([]eval.Value, 0, len(args)-2)
	for i := 2; i < len(args); i++ {
		arg := args[i]
		var ailangVal eval.Value

		switch arg.Type() {
		case js.TypeNumber:
			f := arg.Float()
			if f == float64(int(f)) {
				ailangVal = &eval.IntValue{Value: int(f)}
			} else {
				ailangVal = &eval.FloatValue{Value: f}
			}
		case js.TypeString:
			ailangVal = &eval.StringValue{Value: arg.String()}
		case js.TypeBoolean:
			ailangVal = &eval.BoolValue{Value: arg.Bool()}
		default:
			return newRejectedPromise(fmt.Sprintf("unsupported argument type at position %d: %v", i-1, arg.Type()))
		}

		ailangArgs = append(ailangArgs, ailangVal)
	}

	handler := js.FuncOf(func(_ js.Value, pArgs []js.Value) interface{} {
		resolve := pArgs[0]
		reject := pArgs[1]
		go func() {
			defer func() {
				if r := recover(); r != nil {
					reject.Invoke(fmt.Sprintf("panic: %v", r))
				}
			}()

			result, err := replInstance.GetRegistry().InvokeExport(moduleName, funcName, ailangArgs)
			if err != nil {
				resolve.Invoke(map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				})
				return
			}

			resolve.Invoke(map[string]interface{}{
				"success": true,
				"result":  ailangValueToJS(result),
				"display": formatValue(result),
			})
		}()
		return nil
	})

	return js.Global().Get("Promise").New(handler)
}

// newRejectedPromise creates a Promise that immediately rejects with the given message.
func newRejectedPromise(msg string) js.Value {
	return js.Global().Get("Promise").Call("reject", msg)
}

func main() {
	// Initialize REPL
	replInstance = NewWasmREPL()

	// Register existing sync functions
	js.Global().Set("ailangEval", js.FuncOf(evalExpression))
	js.Global().Set("ailangReset", js.FuncOf(resetREPL))
	js.Global().Set("ailangVersion", js.FuncOf(getVersion))
	js.Global().Set("ailangLoadModule", js.FuncOf(loadModule))
	js.Global().Set("ailangListModules", js.FuncOf(listModules))
	js.Global().Set("ailangStdlibStatus", js.FuncOf(stdlibStatus))
	js.Global().Set("ailangCall", js.FuncOf(callExport))

	// Register effect handler functions (M-WASM-EFFECTS)
	js.Global().Set("ailangSetEffectHandler", js.FuncOf(setEffectHandler))
	js.Global().Set("ailangSetAIHandler", js.FuncOf(setAIHandler))
	js.Global().Set("ailangSetAIStepHandler", js.FuncOf(setAIStepHandler))
	js.Global().Set("ailangSetAIStepWithCacheHandler", js.FuncOf(setAIStepWithCacheHandler))
	js.Global().Set("ailangSetAIStepWithStreamHandler", js.FuncOf(setAIStepWithStreamHandler))
	js.Global().Set("ailangGrantCapability", js.FuncOf(grantCapability))

	// Register trace handler (M-WASM-TRACE)
	js.Global().Set("ailangSetTraceHandler", js.FuncOf(setTraceHandler))

	// Register Cognitive OS handlers (M-COG-RUNTIME, v0.21.x)
	js.Global().Set("ailangSetDOMApplyPatchHandler", js.FuncOf(setDOMApplyPatchHandler))
	js.Global().Set("ailangSetDOMApplyBatchHandler", js.FuncOf(setDOMApplyBatchHandler))
	js.Global().Set("ailangSetMsgSendHandler", js.FuncOf(setMsgSendHandler))
	js.Global().Set("ailangSetMsgRecvHandler", js.FuncOf(setMsgRecvHandler))
	// M-COG-RUNTIME-BROWSER M4: Subscribe wiring
	js.Global().Set("ailangSetDOMSubscribeHandler", js.FuncOf(setDOMSubscribeHandler))
	js.Global().Set("ailangSetMsgSubscribeHandler", js.FuncOf(setMsgSubscribeHandler))

	// Register async evaluation functions (for effects that use Promises)
	js.Global().Set("ailangEvalAsync", js.FuncOf(evalAsync))
	js.Global().Set("ailangCallAsync", js.FuncOf(callAsync))

	// Register ADT constructor helper (M-WASM-STREAM-BRIDGE Phase 2)
	// ailangADT("Ok", ailangADT("StreamConn", 42)) → {_ctor: "Ok", _fields: [{_ctor: "StreamConn", _fields: [42]}]}
	js.Global().Set("ailangADT", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return nil
		}
		ctor := args[0].String()
		fields := make([]interface{}, len(args)-1)
		for i := 1; i < len(args); i++ {
			fields[i-1] = args[i]
		}
		obj := map[string]interface{}{
			"_ctor":   ctor,
			"_fields": fields,
		}
		return obj
	}))

	// Signal ready (safely check if console exists)
	if console := js.Global().Get("console"); !console.IsUndefined() {
		if logFunc := console.Get("log"); !logFunc.IsUndefined() {
			console.Call("log", "AILANG WASM REPL loaded (with effects support)")
		}
	}

	// Keep the program running
	select {} // Block forever
}
