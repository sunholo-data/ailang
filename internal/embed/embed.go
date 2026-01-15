// Package embed provides a high-level API for embedding AILANG in Go programs.
//
// This package simplifies using AILANG as a scripting/extension language within
// Go applications. It handles module compilation, caching, and provides convenient
// value conversion between Go and AILANG types.
//
// Basic usage:
//
//	engine := embed.New()
//	defer engine.Close()
//
//	// Load and call an AILANG function
//	result, err := engine.Call("my/module", "myFunction", 42, "hello")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Extract Go value
//	value, err := embed.ToInt(result)
package embed

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/runtime"
)

// Engine manages AILANG module compilation and execution.
// It caches compiled modules for efficient repeated calls.
type Engine struct {
	mu       sync.RWMutex
	runtime  *runtime.ModuleRuntime
	basePath string
	closed   bool
}

// New creates a new AILANG embedding engine.
// basePath is the root directory for resolving module imports.
func New(basePath string) *Engine {
	return &Engine{
		basePath: basePath,
		runtime:  runtime.NewModuleRuntime(basePath),
	}
}

// Close releases resources held by the engine.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	return nil
}

// Load compiles and loads a module, caching it for future calls.
// modulePath is the path relative to basePath (e.g., "internal/transforms/event_formatter").
func (e *Engine) Load(modulePath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return fmt.Errorf("engine is closed")
	}

	if e.runtime.HasInstance(modulePath) {
		return nil // Already loaded
	}

	_, err := e.runtime.LoadAndEvaluate(modulePath)
	return err
}

// Call invokes an exported function from a module with the given arguments.
// Arguments are automatically converted from Go types to AILANG values.
// The module is loaded if not already cached.
func (e *Engine) Call(modulePath, funcName string, args ...interface{}) (eval.Value, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil, fmt.Errorf("engine is closed")
	}

	// Load module if needed
	inst := e.runtime.GetInstance(modulePath)
	if inst == nil {
		var err error
		inst, err = e.runtime.LoadAndEvaluate(modulePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load module %s: %w", modulePath, err)
		}
	}

	// Get the function
	fnVal, err := inst.GetExport(funcName)
	if err != nil {
		return nil, fmt.Errorf("function %s not found in module %s: %w", funcName, modulePath, err)
	}

	if _, ok := fnVal.(*eval.FunctionValue); !ok {
		return nil, fmt.Errorf("%s is not a function (got %T)", funcName, fnVal)
	}

	// Convert arguments
	ailangArgs := make([]eval.Value, len(args))
	for i, arg := range args {
		converted, err := FromGo(arg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert argument %d: %w", i, err)
		}
		ailangArgs[i] = converted
	}

	// Call the function
	return runtime.CallEntrypoint(e.runtime, inst, funcName, ailangArgs)
}

// CallJSON is a convenience method that accepts JSON input and returns JSON output.
// Useful for language-agnostic integrations.
func (e *Engine) CallJSON(modulePath, funcName string, inputJSON []byte) ([]byte, error) {
	// Parse JSON input
	var input interface{}
	if len(inputJSON) > 0 {
		if err := json.Unmarshal(inputJSON, &input); err != nil {
			return nil, fmt.Errorf("invalid JSON input: %w", err)
		}
	}

	// Convert to AILANG value
	ailangInput, err := FromGo(input)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	// Call function
	result, err := e.Call(modulePath, funcName, ailangInput)
	if err != nil {
		return nil, err
	}

	// Convert result to Go
	goResult, err := ToGo(result)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result: %w", err)
	}

	// Encode as JSON
	return json.Marshal(goResult)
}

// Eval compiles and evaluates AILANG code directly (not from a file).
// Useful for one-off expressions or REPL-like functionality.
func (e *Engine) Eval(code string) (eval.Value, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return nil, fmt.Errorf("engine is closed")
	}

	cfg := pipeline.Config{
		Mode:         pipeline.ModeEval,
		RelaxModules: true,
	}
	src := pipeline.Source{
		Code:     code,
		Filename: "<embedded>",
		IsREPL:   true,
	}

	result, err := pipeline.RunWithContext(context.Background(), cfg, src)
	if err != nil {
		return nil, fmt.Errorf("compilation error: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("evaluation errors: %v", result.Errors)
	}

	return result.Value, nil
}

// ListExports returns the names of all exported functions/values in a module.
func (e *Engine) ListExports(modulePath string) ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.closed {
		return nil, fmt.Errorf("engine is closed")
	}

	inst := e.runtime.GetInstance(modulePath)
	if inst == nil {
		return nil, fmt.Errorf("module %s not loaded", modulePath)
	}

	return inst.ListExports(), nil
}

// HasExport checks if a module exports a value with the given name.
func (e *Engine) HasExport(modulePath, name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.closed {
		return false
	}

	inst := e.runtime.GetInstance(modulePath)
	if inst == nil {
		return false
	}

	return inst.HasExport(name)
}
