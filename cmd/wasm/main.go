//go:build js && wasm
// +build js,wasm

package main

import (
	"bytes"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/sunholo/ailang/internal/repl"
	"github.com/sunholo/ailang/std"
)

// Version information (injected at build time via -ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

// WasmREPL wraps the REPL for browser use
type WasmREPL struct {
	repl     *repl.REPL
	registry *repl.ModuleRegistry
	output   *bytes.Buffer
}

// loadEmbeddedStdlib loads all stdlib modules from the embedded filesystem
// into the module registry. This enables imports like `import std/list` in browser.
func loadEmbeddedStdlib(registry *repl.ModuleRegistry) error {
	// Read all .ail files from embedded std/ directory
	entries, err := std.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read embedded stdlib: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ail") {
			continue
		}

		// Read file content
		content, err := std.FS.ReadFile(entry.Name())
		if err != nil {
			// Log but continue - some modules might have issues
			continue
		}

		// Module name is "std/<filename without .ail>"
		moduleName := "std/" + strings.TrimSuffix(entry.Name(), ".ail")

		// Load module into registry (ignore errors - some may have dependencies)
		_, _ = registry.LoadModule(moduleName, string(content))
	}

	return nil
}

// NewWasmREPL creates a new browser-ready REPL
func NewWasmREPL() *WasmREPL {
	registry := repl.NewModuleRegistry()
	replInstance := repl.NewWithVersion(Version, BuildTime)

	// Connect registry to REPL for import resolution
	replInstance.SetRegistry(registry)

	// Load embedded stdlib modules (enables import std/list, std/json, etc.)
	if err := loadEmbeddedStdlib(registry); err != nil {
		// Log error but continue - REPL can still work without stdlib
		if console := js.Global().Get("console"); !console.IsUndefined() {
			console.Call("warn", "Failed to load embedded stdlib: "+err.Error())
		}
	}

	w := &WasmREPL{
		repl:     replInstance,
		registry: registry,
		output:   &bytes.Buffer{},
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

// Reset clears the REPL environment
func (w *WasmREPL) Reset() string {
	w.repl = repl.New()
	w.registry = repl.NewModuleRegistry()
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

func main() {
	// Initialize REPL
	replInstance = NewWasmREPL()

	// Register functions for JavaScript to call
	js.Global().Set("ailangEval", js.FuncOf(evalExpression))
	js.Global().Set("ailangReset", js.FuncOf(resetREPL))
	js.Global().Set("ailangVersion", js.FuncOf(getVersion))
	js.Global().Set("ailangLoadModule", js.FuncOf(loadModule))
	js.Global().Set("ailangListModules", js.FuncOf(listModules))

	// Signal ready (safely check if console exists)
	if console := js.Global().Get("console"); !console.IsUndefined() {
		if logFunc := console.Get("log"); !logFunc.IsUndefined() {
			console.Call("log", "AILANG WASM REPL loaded")
		}
	}

	// Keep the program running
	select {} // Block forever
}
