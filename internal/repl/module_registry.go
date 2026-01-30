package repl

import (
	"fmt"
	"sync"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/runtime"
	"github.com/sunholo/ailang/internal/types"
)

// ModuleRegistry stores compiled modules for REPL access.
// It enables loading AILANG modules in the browser via ailangLoadModule()
// and makes their exports available for subsequent ailangEval() calls.
type ModuleRegistry struct {
	mu      sync.RWMutex
	modules map[string]*RegisteredModule
}

// RegisteredModule represents a compiled and evaluated AILANG module.
type RegisteredModule struct {
	Name    string             // Module name (e.g., "invoice_processor")
	Source  string             // Original source code
	Exports map[string]*Export // Exported functions and values
}

// Export represents a single exported function or value from a module.
type Export struct {
	Name   string        // Export name
	Value  interface{}   // Evaluated value (closure, constant, etc.)
	Scheme *types.Scheme // Type signature for type checking
}

// NewModuleRegistry creates an empty module registry.
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]*RegisteredModule),
	}
}

// LoadModule compiles and registers a module from source code.
// Returns the list of exported names on success.
func (mr *ModuleRegistry) LoadModule(name, sourceCode string) ([]string, error) {
	// Step 1: Parse
	l := lexer.New(sourceCode, name+".ail")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse error: %s", p.Errors()[0])
	}

	// Step 2: Elaborate to Core
	elaborator := elaborate.NewElaborator()
	coreProg, err := elaborator.Elaborate(program)
	if err != nil {
		return nil, fmt.Errorf("elaboration error: %w", err)
	}

	if len(coreProg.Decls) == 0 {
		return nil, fmt.Errorf("module %s has no declarations", name)
	}

	// Step 3: Type check
	typeEnv := types.NewTypeEnvWithBuiltins()
	typeEnv = pipeline.InjectPrelude(typeEnv)

	// Use NewCoreTypeChecker which sets up Num/Fractional defaults and loads builtin instances
	typeChecker := types.NewCoreTypeChecker()

	// Type check each declaration and collect types
	declTypes := make(map[string]*types.Scheme)
	for _, decl := range coreProg.Decls {
		if letDecl, ok := decl.(*core.Let); ok {
			_, updatedEnv, _, _, err := typeChecker.InferWithConstraints(decl, typeEnv)
			if err != nil {
				return nil, fmt.Errorf("type error in %s: %w", letDecl.Name, err)
			}
			typeEnv = updatedEnv

			// Extract the scheme for this binding
			if result, err := typeEnv.Lookup(letDecl.Name); err == nil {
				if scheme, ok := result.(*types.Scheme); ok {
					declTypes[letDecl.Name] = scheme
				}
			}
		}
	}

	// Step 4: Dictionary elaboration
	resolved := typeChecker.GetResolvedConstraints()
	elaboratedProg, err := elaborate.ElaborateWithDictionaries(coreProg, resolved)
	if err != nil {
		return nil, fmt.Errorf("dictionary elaboration error: %w", err)
	}

	// Step 5: Op lowering
	lowerer := pipeline.NewOpLowerer(typeEnv, typeChecker.CoreTI)
	loweredProg, err := lowerer.Lower(elaboratedProg)
	if err != nil {
		return nil, fmt.Errorf("op lowering error: %w", err)
	}

	// Step 6: Link dictionaries
	linker := link.NewLinker()
	registerPreludeInstances(linker) // Register standard instances

	linkedDecls := make([]core.CoreExpr, 0, len(loweredProg.Decls))
	for _, decl := range loweredProg.Decls {
		linked, err := linker.Link(decl)
		if err != nil {
			return nil, fmt.Errorf("linking error: %w", err)
		}
		linkedDecls = append(linkedDecls, linked)
	}

	// Step 7: Evaluate to get values
	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	builtinResolver := runtime.NewBuiltinOnlyResolver(builtinRegistry)
	evaluator.SetGlobalResolver(builtinResolver)

	exports := make(map[string]*Export)
	for i, decl := range linkedDecls {
		if letDecl, ok := coreProg.Decls[i].(*core.Let); ok {
			val, err := evaluator.Eval(decl)
			if err != nil {
				return nil, fmt.Errorf("eval error in %s: %w", letDecl.Name, err)
			}

			// Store in evaluator's environment for subsequent declarations
			evaluator.Env().Set(letDecl.Name, val)

			// All top-level declarations are exported for now
			// TODO: Support explicit export syntax when parser supports it
			exports[letDecl.Name] = &Export{
				Name:   letDecl.Name,
				Value:  val,
				Scheme: declTypes[letDecl.Name],
			}
		}
	}

	// Register module
	mr.mu.Lock()
	mr.modules[name] = &RegisteredModule{
		Name:    name,
		Source:  sourceCode,
		Exports: exports,
	}
	mr.mu.Unlock()

	// Return export names
	exportNames := make([]string, 0, len(exports))
	for name := range exports {
		exportNames = append(exportNames, name)
	}
	return exportNames, nil
}

// GetExport retrieves a specific export from a loaded module.
func (mr *ModuleRegistry) GetExport(moduleName, funcName string) (*Export, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	mod, ok := mr.modules[moduleName]
	if !ok {
		return nil, fmt.Errorf("module %s not loaded (use ailangLoadModule first)", moduleName)
	}

	exp, ok := mod.Exports[funcName]
	if !ok {
		return nil, fmt.Errorf("symbol %s not exported by %s", funcName, moduleName)
	}

	return exp, nil
}

// GetModule retrieves a loaded module by name.
func (mr *ModuleRegistry) GetModule(name string) (*RegisteredModule, bool) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	mod, ok := mr.modules[name]
	return mod, ok
}

// CallExport formats a function call expression string for use with the REPL.
// This returns the expression that can be evaluated via ProcessExpression.
// Arguments are converted to their AILANG string representation.
func (mr *ModuleRegistry) CallExport(moduleName, funcName string, args []eval.Value) (string, error) {
	// Verify the export exists
	_, err := mr.GetExport(moduleName, funcName)
	if err != nil {
		return "", err
	}

	// Build curried call expression: moduleName.funcName(arg1)(arg2)...
	// First we need to import the module, then call the function
	expr := funcName
	for _, arg := range args {
		argStr := formatArgument(arg)
		expr = fmt.Sprintf("%s(%s)", expr, argStr)
	}

	return expr, nil
}

// formatArgument converts an eval.Value to its AILANG source representation
func formatArgument(v eval.Value) string {
	switch val := v.(type) {
	case *eval.IntValue:
		return fmt.Sprintf("%d", val.Value)
	case *eval.FloatValue:
		return fmt.Sprintf("%g", val.Value)
	case *eval.StringValue:
		return fmt.Sprintf("%q", val.Value)
	case *eval.BoolValue:
		if val.Value {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ListModules returns the names of all loaded modules.
func (mr *ModuleRegistry) ListModules() []string {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	names := make([]string, 0, len(mr.modules))
	for name := range mr.modules {
		names = append(names, name)
	}
	return names
}

// registerPreludeInstances registers standard type class instances with the linker.
// This mirrors the instances registered in the REPL.
func registerPreludeInstances(linker *link.Linker) {
	// Helper to create builtin functions
	wrapInt2 := func(name string, f func(int64, int64) int64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("expected 2 arguments")
				}
				x, ok1 := args[0].(*eval.IntValue)
				y, ok2 := args[1].(*eval.IntValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("expected int arguments")
				}
				return &eval.IntValue{Value: int(f(int64(x.Value), int64(y.Value)))}, nil
			},
		}
	}

	wrapFloat2 := func(name string, f func(float64, float64) float64) *eval.BuiltinFunction {
		return &eval.BuiltinFunction{
			Name: name,
			Fn: func(args []eval.Value) (eval.Value, error) {
				if len(args) != 2 {
					return nil, fmt.Errorf("expected 2 arguments")
				}
				x, ok1 := args[0].(*eval.FloatValue)
				y, ok2 := args[1].(*eval.FloatValue)
				if !ok1 || !ok2 {
					return nil, fmt.Errorf("expected float arguments")
				}
				return &eval.FloatValue{Value: f(x.Value, y.Value)}, nil
			},
		}
	}

	// Num[Int]
	numInt := core.DictValue{
		TypeClass: "Num",
		Type:      "Int",
		Methods: map[string]interface{}{
			"add": wrapInt2("add", func(a, b int64) int64 { return a + b }),
			"sub": wrapInt2("sub", func(a, b int64) int64 { return a - b }),
			"mul": wrapInt2("mul", func(a, b int64) int64 { return a * b }),
			"div": wrapInt2("div", func(a, b int64) int64 { return a / b }),
		},
	}

	// Num[Float]
	numFloat := core.DictValue{
		TypeClass: "Num",
		Type:      "Float",
		Methods: map[string]interface{}{
			"add": wrapFloat2("add", func(a, b float64) float64 { return a + b }),
			"sub": wrapFloat2("sub", func(a, b float64) float64 { return a - b }),
			"mul": wrapFloat2("mul", func(a, b float64) float64 { return a * b }),
			"div": wrapFloat2("div", func(a, b float64) float64 { return a / b }),
		},
	}

	// Register with canonical keys
	for methodName := range numInt.Methods {
		key := types.MakeDictionaryKey("prelude", "Num", &types.TCon{Name: "Int"}, methodName)
		linker.AddDictionary(key, numInt)
	}

	for methodName := range numFloat.Methods {
		key := types.MakeDictionaryKey("prelude", "Num", &types.TCon{Name: "Float"}, methodName)
		linker.AddDictionary(key, numFloat)
	}
}
