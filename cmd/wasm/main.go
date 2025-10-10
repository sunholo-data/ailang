// +build js,wasm

package main

import (
	"bytes"
	"syscall/js"

	"github.com/sunholo/ailang/internal/repl"
)

// WasmREPL wraps the REPL for browser use
type WasmREPL struct {
	repl   *repl.REPL
	output *bytes.Buffer
}

// NewWasmREPL creates a new browser-ready REPL
func NewWasmREPL() *WasmREPL {
	w := &WasmREPL{
		repl:   repl.New(),
		output: &bytes.Buffer{},
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
	return "Environment reset"
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
		"version":   "v0.3.0",
		"buildTime": "wasm",
		"platform":  "browser",
	}
}

func main() {
	// Initialize REPL
	replInstance = NewWasmREPL()

	// Register functions for JavaScript to call
	js.Global().Set("ailangEval", js.FuncOf(evalExpression))
	js.Global().Set("ailangReset", js.FuncOf(resetREPL))
	js.Global().Set("ailangVersion", js.FuncOf(getVersion))

	// Signal ready (safely check if console exists)
	if console := js.Global().Get("console"); !console.IsUndefined() {
		if logFunc := console.Get("log"); !logFunc.IsUndefined() {
			console.Call("log", "AILANG WASM REPL loaded")
		}
	}

	// Keep the program running
	select {} // Block forever
}
