package repl

import (
	"fmt"
	"io"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/typedast"
	"github.com/sunholo/ailang/internal/types"
)

// ProcessExpression runs an expression through the full pipeline (exported for WASM)
func (r *REPL) ProcessExpression(input string, out io.Writer) {
	// Step 1: Parse
	l := lexer.New(input, "<repl>")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		r.printParserErrors(p.Errors(), out)
		return
	}

	// Step 2: Elaborate to Core (with dictionary-passing)
	elaborator := elaborate.NewElaborator()
	coreProg, err := elaborator.Elaborate(program)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Elaboration error"), err)
		return
	}

	// Extract the first declaration as an expression
	if len(coreProg.Decls) == 0 {
		fmt.Fprintln(out, yellow("Empty expression"))
		return
	}
	coreExpr := coreProg.Decls[0]

	if r.config.ShowCore {
		fmt.Fprintf(out, "%s\n", dim("Core AST:"))
		fmt.Fprintln(out, formatCore(coreExpr, "  "))
	}

	// Step 3: Type check with constraints
	typeChecker := types.NewCoreTypeCheckerWithInstances(r.instEnv)
	typeChecker.EnableTraceDefaulting(r.config.TraceDefaulting)

	typedNode, updatedEnv, qualType, constraints, err := typeChecker.InferWithConstraints(coreExpr, r.typeEnv)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Type error"), err)
		if r.config.TraceDefaulting {
			r.printDefaultingFailure(constraints, out)
		}
		return
	}

	// Update REPL type environment with any new bindings
	// NOTE: For top-level let bindings, we'll explicitly persist the type below (Step 8)
	// For now, only update if it's not a Let (to avoid nested scope issues)
	if _, isLet := coreExpr.(*core.Let); !isLet {
		r.typeEnv = updatedEnv
	}

	// Step 4: Dictionary elaboration (resolve constraints to dictionaries)
	// Get resolved constraints from the type checker - this also triggers defaulting
	resolved := typeChecker.GetResolvedConstraints()

	// CRITICAL FIX: Manually call FillOperatorMethods to set correct method names
	// The REPL's InferWithConstraints doesn't call this automatically
	// Fill operator methods manually for dictionary elaboration
	typeChecker.FillOperatorMethods(coreExpr)

	// Get the final type after defaulting - prefer concrete types from post-defaulting
	typeToDisplay := r.getFinalTypeAfterDefaulting(typedNode, qualType, resolved)

	// Pretty print the final type
	prettyType := r.normalizeTypeName(typeToDisplay)

	if r.config.ShowTyped {
		fmt.Fprintf(out, "%s\n", dim("Typed AST:"))
		fmt.Fprintln(out, formatTyped(typedNode, "  "))
	}

	// Create a temporary program for elaboration
	tempProg := &core.Program{Decls: []core.CoreExpr{coreExpr}}
	elaboratedProg, err := elaborate.ElaborateWithDictionaries(tempProg, resolved)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Dictionary elaboration error"), err)
		r.suggestMissingInstances(constraints, out)
		return
	}

	// Extract the elaborated expression
	if len(elaboratedProg.Decls) == 0 {
		fmt.Fprintln(out, yellow("Empty result after elaboration"))
		return
	}
	elaboratedCore := elaboratedProg.Decls[0]

	// Step 5: Verify ANF
	if err := elaborate.VerifyANF(elaboratedProg); err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("ANF verification error"), err)
		return
	}

	// Step 5.5: Lower intrinsic operations to dictionary calls
	lowerer := pipeline.NewOpLowerer(r.typeEnv)
	loweredProg, err := lowerer.Lower(elaboratedProg)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Op lowering error"), err)
		return
	}

	// Update elaboratedCore with lowered version
	if len(loweredProg.Decls) > 0 {
		elaboratedCore = loweredProg.Decls[0]
	}

	// Step 6: Link dictionaries
	linker := link.NewLinker()

	// Add instances to linker with canonical keys
	r.registerDictionariesForLinker(linker)

	if r.config.DryLink {
		// Dry run to show required instances
		required := linker.DryRun(elaboratedCore)
		if len(required) > 0 {
			fmt.Fprintf(out, "%s\n", yellow("Required instances:"))
			for _, key := range required {
				fmt.Fprintf(out, "  • %s\n", key)
			}
		}
		return
	}

	linkedCore, err := linker.Link(elaboratedCore)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Linking error"), err)
		return
	}

	// Step 7: Evaluate (using persistent evaluator with builtin resolver)
	result, err := r.evaluator.Eval(linkedCore)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Runtime error"), err)
		return
	}

	// Step 8: Persist top-level let bindings in REPL environment
	// The evaluator's evalCoreLet restores the environment after evaluation,
	// but for REPL we want top-level bindings to persist across inputs (both value and type)
	if letExpr, ok := elaboratedCore.(*core.Let); ok {
		// Re-evaluate just the RHS to get the value
		val, err := r.evaluator.Eval(letExpr.Value)
		if err == nil {
			// Add VALUE binding to both REPL env and evaluator env (they're the same reference)
			r.env.Set(letExpr.Name, val)

			// TODO: Add TYPE binding persistence for let
			// Currently, type annotations are lost during Surface → Core elaboration,
			// so the TypedLet scheme contains a fresh type variable instead of the annotated type.
			// To fix this properly, we need to preserve annotations through elaboration.
			// For now, let bindings work for VALUES but type info isn't persisted across REPL inputs.
			if typedLet, ok := typedNode.(*typedast.TypedLet); ok {
				if typedLet.Scheme != nil {
					scheme := typedLet.Scheme.(*types.Scheme)
					// Persist type binding (though it may be generalized to a type variable)
					r.typeEnv.BindScheme(letExpr.Name, scheme)
				}
			}
		}
	}

	// Store result
	r.lastResult = result

	// Pretty print result with type on the same line
	fmt.Fprintf(out, "%s :: %s\n", formatValue(result), cyan(prettyType))
}

// initBuiltins initializes built-in type class instances
func (r *REPL) initBuiltins() {
	// Wrapper functions to convert Go functions to uniform eval signatures
	wrapInt2 := func(f func(int64, int64) int64) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
			}
			x, ok1 := args[0].(*eval.IntValue)
			y, ok2 := args[1].(*eval.IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			return &eval.IntValue{Value: int(f(int64(x.Value), int64(y.Value)))}, nil
		}
	}

	wrapFloat2 := func(f func(float64, float64) float64) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
			}
			x, ok1 := args[0].(*eval.FloatValue)
			y, ok2 := args[1].(*eval.FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			return &eval.FloatValue{Value: f(x.Value, y.Value)}, nil
		}
	}

	wrapFloat1 := func(f func(float64) float64) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 argument, got %d", len(args))
			}
			x, ok := args[0].(*eval.FloatValue)
			if !ok {
				return nil, fmt.Errorf("expected float argument")
			}
			return &eval.FloatValue{Value: f(x.Value)}, nil
		}
	}

	wrapIntCmp2 := func(f func(int64, int64) bool) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
			}
			x, ok1 := args[0].(*eval.IntValue)
			y, ok2 := args[1].(*eval.IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			return &eval.BoolValue{Value: f(int64(x.Value), int64(y.Value))}, nil
		}
	}

	wrapFloatCmp2 := func(f func(float64, float64) bool) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
			}
			x, ok1 := args[0].(*eval.FloatValue)
			y, ok2 := args[1].(*eval.FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			return &eval.BoolValue{Value: f(x.Value, y.Value)}, nil
		}
	}

	// Register built-in instances with wrapped methods as BuiltinFunction
	r.instances["Num[Int]"] = core.DictValue{
		TypeClass: "Num",
		Type:      "Int",
		Methods: map[string]interface{}{
			"add": &eval.BuiltinFunction{
				Name: "add",
				Fn: wrapInt2(func(a, b int64) int64 {
					result := a + b
					// Integer addition
					return result
				}),
			},
			"sub": &eval.BuiltinFunction{
				Name: "sub",
				Fn:   wrapInt2(func(a, b int64) int64 { return a - b }),
			},
			"mul": &eval.BuiltinFunction{
				Name: "mul",
				Fn: wrapInt2(func(a, b int64) int64 {
					result := a * b
					// Integer multiplication
					return result
				}),
			},
			"div": &eval.BuiltinFunction{
				Name: "div",
				Fn: wrapInt2(func(a, b int64) int64 {
					if b == 0 {
						panic("division by zero")
					}
					return a / b
				}),
			},
		},
	}

	r.instances["Num[Float]"] = core.DictValue{
		TypeClass: "Num",
		Type:      "Float",
		Methods: map[string]interface{}{
			"add": &eval.BuiltinFunction{Name: "add", Fn: wrapFloat2(func(a, b float64) float64 { return a + b })},
			"sub": &eval.BuiltinFunction{Name: "sub", Fn: wrapFloat2(func(a, b float64) float64 { return a - b })},
			"mul": &eval.BuiltinFunction{Name: "mul", Fn: wrapFloat2(func(a, b float64) float64 { return a * b })},
			"div": &eval.BuiltinFunction{Name: "div", Fn: wrapFloat2(func(a, b float64) float64 { return a / b })},
		},
	}

	// Fractional[Float] - extends Num with fractional operations
	r.instances["Fractional[Float]"] = core.DictValue{
		TypeClass: "Fractional",
		Type:      "Float",
		Methods: map[string]interface{}{
			// Inherit all Num methods
			"add": &eval.BuiltinFunction{Name: "add", Fn: wrapFloat2(func(a, b float64) float64 { return a + b })},
			"sub": &eval.BuiltinFunction{Name: "sub", Fn: wrapFloat2(func(a, b float64) float64 { return a - b })},
			"mul": &eval.BuiltinFunction{Name: "mul", Fn: wrapFloat2(func(a, b float64) float64 { return a * b })},
			"div": &eval.BuiltinFunction{Name: "div", Fn: wrapFloat2(func(a, b float64) float64 { return a / b })},
			"neg": &eval.BuiltinFunction{Name: "neg", Fn: wrapFloat1(func(a float64) float64 { return -a })},
			"abs": &eval.BuiltinFunction{Name: "abs", Fn: wrapFloat1(func(a float64) float64 {
				if a < 0 {
					return -a
				}
				return a
			})},
			"fromInt": &eval.BuiltinFunction{Name: "fromInt", Fn: func(args []eval.Value) (eval.Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("expected 1 argument, got %d", len(args))
				}
				if iv, ok := args[0].(*eval.IntValue); ok {
					return &eval.FloatValue{Value: float64(iv.Value)}, nil
				}
				return nil, fmt.Errorf("expected int argument")
			}},
			// Fractional-specific methods
			"divide": &eval.BuiltinFunction{Name: "divide", Fn: wrapFloat2(func(a, b float64) float64 { return a / b })},
			"recip":  &eval.BuiltinFunction{Name: "recip", Fn: wrapFloat1(func(a float64) float64 { return 1.0 / a })},
			"fromRational": &eval.BuiltinFunction{Name: "fromRational", Fn: func(args []eval.Value) (eval.Value, error) {
				// For now, just convert from float (simplified)
				if len(args) != 1 {
					return nil, fmt.Errorf("expected 1 argument, got %d", len(args))
				}
				if fv, ok := args[0].(*eval.FloatValue); ok {
					return fv, nil // Identity for now
				}
				return nil, fmt.Errorf("expected float argument")
			}},
		},
		Provides: []string{"Num[Float]"}, // Fractional provides Num
	}

	r.instances["Eq[Int]"] = core.DictValue{
		TypeClass: "Eq",
		Type:      "Int",
		Methods: map[string]interface{}{
			"eq":  &eval.BuiltinFunction{Name: "eq", Fn: wrapIntCmp2(func(a, b int64) bool { return a == b })},
			"neq": &eval.BuiltinFunction{Name: "neq", Fn: wrapIntCmp2(func(a, b int64) bool { return a != b })},
		},
	}

	r.instances["Eq[Float]"] = core.DictValue{
		TypeClass: "Eq",
		Type:      "Float",
		Methods: map[string]interface{}{
			"eq": &eval.BuiltinFunction{Name: "eq", Fn: wrapFloatCmp2(func(a, b float64) bool {
				// Law-compliant: reflexive for NaN
				if a != a && b != b {
					return true
				}
				return a == b
			})},
			"neq": &eval.BuiltinFunction{Name: "neq", Fn: wrapFloatCmp2(func(a, b float64) bool {
				if a != a && b != b {
					return false
				}
				return a != b
			})},
		},
	}

	r.instances["Ord[Int]"] = core.DictValue{
		TypeClass: "Ord",
		Type:      "Int",
		Methods: map[string]interface{}{
			"lt":  &eval.BuiltinFunction{Name: "lt", Fn: wrapIntCmp2(func(a, b int64) bool { return a < b })},
			"lte": &eval.BuiltinFunction{Name: "lte", Fn: wrapIntCmp2(func(a, b int64) bool { return a <= b })},
			"gt":  &eval.BuiltinFunction{Name: "gt", Fn: wrapIntCmp2(func(a, b int64) bool { return a > b })},
			"gte": &eval.BuiltinFunction{Name: "gte", Fn: wrapIntCmp2(func(a, b int64) bool { return a >= b })},
		},
		Provides: []string{"Eq[Int]"}, // Ord provides Eq
	}

	r.instances["Ord[Float]"] = core.DictValue{
		TypeClass: "Ord",
		Type:      "Float",
		Methods: map[string]interface{}{
			"lt":  &eval.BuiltinFunction{Name: "lt", Fn: wrapFloatCmp2(func(a, b float64) bool { return a < b })},
			"lte": &eval.BuiltinFunction{Name: "lte", Fn: wrapFloatCmp2(func(a, b float64) bool { return a <= b })},
			"gt":  &eval.BuiltinFunction{Name: "gt", Fn: wrapFloatCmp2(func(a, b float64) bool { return a > b })},
			"gte": &eval.BuiltinFunction{Name: "gte", Fn: wrapFloatCmp2(func(a, b float64) bool { return a >= b })},
		},
		Provides: []string{"Eq[Float]"}, // Ord provides Eq
	}

	// Register with dictionary registry
	for key, dict := range r.instances {
		r.dictReg.RegisterInstance(key, dict)
	}
}

// registerDictionariesForLinker registers all dictionaries with canonical keys for the linker
func (r *REPL) registerDictionariesForLinker(linker *link.Linker) {
	for _, dict := range r.instances {
		// Convert "Num[Int]" to canonical keys like "prelude::Num::Int::add"
		className := dict.TypeClass

		// Create a proper Type for key generation
		typeForKey := &types.TCon{Name: dict.Type}

		// Register each method with its canonical key
		for methodName := range dict.Methods {
			canonicalKey := types.MakeDictionaryKey("prelude", className, typeForKey, methodName)
			linker.AddDictionary(canonicalKey, dict)
		}
	}
}

// registerDictionariesForEvaluator registers all dictionaries with canonical keys for the evaluator
func (r *REPL) registerDictionariesForEvaluator(evaluator *eval.CoreEvaluator) {
	for _, dict := range r.instances {
		// Convert "Num[Int]" to canonical keys like "prelude::Num::Int::add"
		className := dict.TypeClass

		// Create a proper Type for key generation
		typeForKey := &types.TCon{Name: dict.Type}

		// Register each method with its canonical key
		for methodName := range dict.Methods {
			canonicalKey := types.MakeDictionaryKey("prelude", className, typeForKey, methodName)
			evaluator.AddDictionary(canonicalKey, dict)
		}

		// Also register the base dictionary for lookups (no method name)
		baseKey := types.MakeDictionaryKey("prelude", className, typeForKey, "")
		evaluator.AddDictionary(baseKey, dict)
	}
}

// suggestMissingInstances provides helpful suggestions for missing instances
func (r *REPL) suggestMissingInstances(constraints []types.Constraint, out io.Writer) {
	fmt.Fprintf(out, "%s\n", yellow("Missing instances:"))
	for _, c := range constraints {
		key := constraintToKey(c)
		if _, exists := r.instances[key]; !exists {
			fmt.Fprintf(out, "  • %s\n", key)

			// Suggest import if in prelude
			if isInPrelude(key) {
				fmt.Fprintf(out, "    %s\n", dim("Try: :import std/prelude"))
			}
		}
	}
}

// printDefaultingFailure shows why defaulting failed
func (r *REPL) printDefaultingFailure(constraints []types.Constraint, out io.Writer) {
	fmt.Fprintf(out, "%s\n", yellow("Defaulting failure details:"))
	fmt.Fprintln(out, "  Ambiguous constraints:")
	for _, c := range constraints {
		if isAmbiguous(c) {
			fmt.Fprintf(out, "    • %s\n", formatConstraint(c))
		}
	}
	fmt.Fprintln(out, "  Current defaults:")
	fmt.Fprintln(out, "    • Num → Int")
	fmt.Fprintln(out, "    • Fractional → Float")
}
