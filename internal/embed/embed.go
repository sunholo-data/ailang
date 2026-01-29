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
	"os"
	"path/filepath"
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
	compiled map[string]bool // tracks which modules have been compiled through pipeline
}

// New creates a new AILANG embedding engine.
// basePath is the root directory for resolving module imports.
// The basePath should be the root of the project containing both
// user modules and the stdlib directory.
func New(basePath string) *Engine {
	// Resolve basePath to absolute for consistent path handling
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		absBasePath = basePath
	}

	// Set AILANG_STDLIB_PATH to basePath so stdlib modules can be found
	// This is needed because the loader resolves stdlib relative to CWD by default
	if os.Getenv("AILANG_STDLIB_PATH") == "" {
		os.Setenv("AILANG_STDLIB_PATH", absBasePath)
	}

	return &Engine{
		basePath: absBasePath,
		runtime:  runtime.NewModuleRuntime(absBasePath),
		compiled: make(map[string]bool),
	}
}

// Close releases resources held by the engine.
func (e *Engine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	return nil
}

// InvalidateModule clears all caches for a module, forcing recompilation on next call.
// This is used by hot reload to pick up source file changes.
func (e *Engine) InvalidateModule(modulePath string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.compiled, modulePath)
	e.runtime.DeleteInstance(modulePath)
	e.runtime.GetLoader().DeleteCached(modulePath)
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

	// Compile module through pipeline first (applies OpLowering)
	if !e.compiled[modulePath] {
		if err := e.compileModule(modulePath); err != nil {
			return err
		}
	}

	_, err := e.runtime.LoadAndEvaluate(modulePath)
	return err
}

// compileModule runs the module through the pipeline to apply all transforms
// including OpLowering, then preloads the result into the runtime.
func (e *Engine) compileModule(modulePath string) error {
	// Construct the file path (make absolute for pipeline)
	filePath := filepath.Join(e.basePath, modulePath+".ail")
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Run through pipeline with ModeCheck to get compiled modules
	cfg := pipeline.Config{
		Mode:         pipeline.ModeCheck,
		RelaxModules: true, // Allow module path to not match file path exactly
	}
	src := pipeline.Source{
		Filename: absPath,
	}

	result, err := pipeline.RunWithContext(context.Background(), cfg, src)
	if err != nil {
		return fmt.Errorf("compilation error: %w", err)
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("compilation errors: %v", result.Errors)
	}

	// Preload all compiled modules to the runtime
	if result.Modules != nil {
		for path, loaded := range result.Modules {
			e.runtime.PreloadModule(path, loaded)
			e.compiled[path] = true
		}
	}

	// Also preload with the requested modulePath since pipeline may use different paths
	// (e.g., absolute vs relative, or canonical vs declared path)
	if result.Interface != nil {
		// Get the module that was actually loaded (by declared module path)
		declaredPath := result.Interface.Module
		if loaded, ok := result.Modules[declaredPath]; ok {
			// Preload with the user's requested path for lookup
			e.runtime.PreloadModule(modulePath, loaded)
			e.compiled[modulePath] = true
		}
	}

	return nil
}

// Call invokes an exported function from a module with the given arguments.
// Arguments are automatically converted from Go types to AILANG values.
// The module is loaded if not already cached.
func (e *Engine) Call(modulePath, funcName string, args ...interface{}) (eval.Value, error) {
	// Fast path: check if module is already loaded (read lock only)
	e.mu.RLock()
	if e.closed {
		e.mu.RUnlock()
		return nil, fmt.Errorf("engine is closed")
	}
	inst := e.runtime.GetInstance(modulePath)
	alreadyCompiled := e.compiled[modulePath]
	e.mu.RUnlock()

	// Slow path: compile and load module (write lock)
	if inst == nil {
		e.mu.Lock()
		// Double-check after acquiring write lock
		inst = e.runtime.GetInstance(modulePath)
		if inst == nil {
			if !alreadyCompiled {
				if err := e.compileModule(modulePath); err != nil {
					e.mu.Unlock()
					return nil, fmt.Errorf("failed to compile module %s: %w", modulePath, err)
				}
			}

			var err error
			inst, err = e.runtime.LoadAndEvaluate(modulePath)
			if err != nil {
				e.mu.Unlock()
				return nil, fmt.Errorf("failed to load module %s: %w", modulePath, err)
			}
		}
		e.mu.Unlock()
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

// CallPreserveFloats is like Call but preserves Go float64 values as AILANG FloatValue,
// even for whole numbers like 100.0. Use this for direct Go calls where you need
// float arguments to stay as floats (not be converted to int for JSON compatibility).
func (e *Engine) CallPreserveFloats(modulePath, funcName string, args ...interface{}) (eval.Value, error) {
	// Fast path: check if module is already loaded (read lock only)
	e.mu.RLock()
	if e.closed {
		e.mu.RUnlock()
		return nil, fmt.Errorf("engine is closed")
	}
	inst := e.runtime.GetInstance(modulePath)
	alreadyCompiled := e.compiled[modulePath]
	e.mu.RUnlock()

	// Slow path: compile and load module (write lock)
	if inst == nil {
		e.mu.Lock()
		// Double-check after acquiring write lock
		inst = e.runtime.GetInstance(modulePath)
		if inst == nil {
			if !alreadyCompiled {
				if err := e.compileModule(modulePath); err != nil {
					e.mu.Unlock()
					return nil, fmt.Errorf("failed to compile module %s: %w", modulePath, err)
				}
			}

			var err error
			inst, err = e.runtime.LoadAndEvaluate(modulePath)
			if err != nil {
				e.mu.Unlock()
				return nil, fmt.Errorf("failed to load module %s: %w", modulePath, err)
			}
		}
		e.mu.Unlock()
	}

	// Get the function
	fnVal, err := inst.GetExport(funcName)
	if err != nil {
		return nil, fmt.Errorf("function %s not found in module %s: %w", funcName, modulePath, err)
	}

	if _, ok := fnVal.(*eval.FunctionValue); !ok {
		return nil, fmt.Errorf("%s is not a function (got %T)", funcName, fnVal)
	}

	// Convert arguments, preserving float types
	ailangArgs := make([]eval.Value, len(args))
	for i, arg := range args {
		converted, err := FromGoPreserveFloats(arg)
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

// SetEffContext configures the effect context for all subsequent calls.
// This enables effectful AILANG functions (IO, FS, Net, AI, etc.).
// The ctx parameter should be an *effects.EffContext; it uses interface{}
// to avoid an import cycle between embed and effects packages.
func (e *Engine) SetEffContext(ctx interface{}) {
	e.runtime.GetEvaluator().SetEffContext(ctx)
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
