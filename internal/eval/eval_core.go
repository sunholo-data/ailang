package eval

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// GlobalResolver resolves global references to values
type GlobalResolver interface {
	ResolveValue(ref core.GlobalRef) (Value, error)
}

// CoreEvaluator evaluates Core AST programs after dictionary elaboration
type CoreEvaluator struct {
	env                   *Environment
	registry              *types.DictionaryRegistry
	resolver              GlobalResolver // Resolver for global references
	experimentalBinopShim bool           // Feature flag for operator shim
}

// NewCoreEvaluatorWithRegistry creates a new Core evaluator with dictionary support
func NewCoreEvaluatorWithRegistry(registry *types.DictionaryRegistry) *CoreEvaluator {
	env := NewEnvironment()
	registerBuiltins(env)

	return &CoreEvaluator{
		env:      env,
		registry: registry,
	}
}

// NewCoreEvaluator creates a new core evaluator without a registry (for REPL)
func NewCoreEvaluator() *CoreEvaluator {
	env := NewEnvironment()
	registerBuiltins(env)

	return &CoreEvaluator{
		env:      env,
		registry: types.NewDictionaryRegistry(),
	}
}

// AddDictionary adds a dictionary to the evaluator (for REPL)
func (e *CoreEvaluator) AddDictionary(key string, dict core.DictValue) {
	// Register each method in the dictionary
	for method, impl := range dict.Methods {
		e.registry.Register("prelude", dict.TypeClass, dict.Type, method, impl)
	}
}

// SetGlobalResolver sets the resolver for global references
func (e *CoreEvaluator) SetGlobalResolver(resolver GlobalResolver) {
	e.resolver = resolver
}

// GetEnvironmentBindings returns all bindings in the current environment
func (e *CoreEvaluator) GetEnvironmentBindings() map[string]Value {
	return e.env.GetAllBindings()
}

// EvalLetRecBindings evaluates a LetRec and returns its bindings without evaluating the body
func (e *CoreEvaluator) EvalLetRecBindings(letrec *core.LetRec) (map[string]Value, error) {
	// Create new environment for recursive bindings
	newEnv := e.env.NewChildEnvironment()

	// First pass: create placeholders for recursive references
	for _, binding := range letrec.Bindings {
		newEnv.Set(binding.Name, &UnitValue{}) // Placeholder
	}

	// Second pass: evaluate bindings in the recursive environment
	oldEnv := e.env
	e.env = newEnv

	bindings := make(map[string]Value)
	for _, binding := range letrec.Bindings {
		val, err := e.evalCore(binding.Value)
		if err != nil {
			e.env = oldEnv
			return nil, err
		}
		newEnv.Set(binding.Name, val)
		bindings[binding.Name] = val
	}

	e.env = oldEnv
	return bindings, nil
}

// SetExperimentalBinopShim enables the experimental operator shim
func (e *CoreEvaluator) SetExperimentalBinopShim(enabled bool) {
	e.experimentalBinopShim = enabled
}

// SetResolver sets the resolver for global references
func (e *CoreEvaluator) SetResolver(resolver GlobalResolver) {
	e.resolver = resolver
}

// Eval evaluates a single expression (simplified for REPL)
func (e *CoreEvaluator) Eval(expr core.CoreExpr) (Value, error) {
	return e.evalCore(expr)
}

// EvalCoreProgram evaluates a Core program
func (e *CoreEvaluator) EvalCoreProgram(prog *core.Program) (Value, error) {
	var lastResult Value = &UnitValue{}

	for _, decl := range prog.Decls {
		result, err := e.evalCore(decl)
		if err != nil {
			return nil, err
		}
		lastResult = result
	}

	return lastResult, nil
}

// evalCore evaluates a Core expression
func (e *CoreEvaluator) evalCore(expr core.CoreExpr) (Value, error) {
	if expr == nil {
		return &UnitValue{}, nil
	}

	switch n := expr.(type) {
	case *core.Var:
		return e.evalCoreVar(n)

	case *core.VarGlobal:
		return e.evalCoreVarGlobal(n)

	case *core.Lit:
		return e.evalCoreLit(n)

	case *core.Lambda:
		return e.evalCoreLambda(n)

	case *core.Let:
		return e.evalCoreLet(n)

	case *core.LetRec:
		return e.evalCoreLetRec(n)

	case *core.App:
		return e.evalCoreApp(n)

	case *core.If:
		return e.evalCoreIf(n)

	case *core.BinOp:
		return e.evalCoreBinOp(n)

	case *core.UnOp:
		return e.evalCoreUnOp(n)

	case *core.Record:
		return e.evalCoreRecord(n)

	case *core.RecordAccess:
		return e.evalCoreRecordAccess(n)

	case *core.List:
		return e.evalCoreList(n)

	case *core.Match:
		return e.evalCoreMatch(n)

	case *core.DictRef:
		return e.evalDictRef(n)

	case *core.DictAbs:
		return e.evalDictAbs(n)

	case *core.DictApp:
		return e.evalDictApp(n)

	case *core.Intrinsic:
		return e.evalIntrinsic(n)

	default:
		return nil, fmt.Errorf("core evaluation not implemented for %T", expr)
	}
}

// evalCoreVar evaluates a variable
func (e *CoreEvaluator) evalCoreVar(v *core.Var) (Value, error) {
	val, ok := e.env.Get(v.Name)
	if !ok {
		return nil, fmt.Errorf("undefined variable: %s", v.Name)
	}
	// Check for cycle marker (immediate self-reference)
	if marker, ok := val.(*CycleMarker); ok {
		return nil, fmt.Errorf("RT009: value initialization cycle detected: %s references itself during initialization", marker.Name)
	}
	return val, nil
}

// evalCoreVarGlobal evaluates a global variable reference
func (e *CoreEvaluator) evalCoreVarGlobal(v *core.VarGlobal) (Value, error) {
	if e.resolver == nil {
		return nil, fmt.Errorf("no resolver available to resolve global reference: %s.%s", v.Ref.Module, v.Ref.Name)
	}

	// Resolve the value through the resolver
	val, err := e.resolver.ResolveValue(v.Ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve global %s.%s: %w", v.Ref.Module, v.Ref.Name, err)
	}

	return val, nil
}

// evalCoreLit evaluates a literal
func (e *CoreEvaluator) evalCoreLit(lit *core.Lit) (Value, error) {
	switch lit.Kind {
	case core.IntLit:
		// Handle various numeric types from parser
		switch v := lit.Value.(type) {
		case int:
			return &IntValue{Value: v}, nil
		case int64:
			return &IntValue{Value: int(v)}, nil
		case float64:
			return &IntValue{Value: int(v)}, nil
		default:
			return nil, fmt.Errorf("invalid int literal: %v (type %T)", lit.Value, lit.Value)
		}

	case core.FloatLit:
		if f, ok := lit.Value.(float64); ok {
			return &FloatValue{Value: f}, nil
		}
		return nil, fmt.Errorf("invalid float literal: %v", lit.Value)

	case core.StringLit:
		if s, ok := lit.Value.(string); ok {
			return &StringValue{Value: s}, nil
		}
		return nil, fmt.Errorf("invalid string literal: %v", lit.Value)

	case core.BoolLit:
		if b, ok := lit.Value.(bool); ok {
			return &BoolValue{Value: b}, nil
		}
		return nil, fmt.Errorf("invalid bool literal: %v", lit.Value)

	case core.UnitLit:
		return &UnitValue{}, nil

	default:
		return nil, fmt.Errorf("unknown literal kind: %v", lit.Kind)
	}
}

// evalCoreLambda evaluates a lambda (creates closure)
func (e *CoreEvaluator) evalCoreLambda(lam *core.Lambda) (Value, error) {
	return &FunctionValue{
		Params: lam.Params,
		Body:   lam.Body,
		Env:    e.env, // Capture environment by reference (needed for recursion)
		Typed:  false,
	}, nil
}

// evalCoreLet evaluates a let binding
func (e *CoreEvaluator) evalCoreLet(let *core.Let) (Value, error) {
	// Evaluate the value
	val, err := e.evalCore(let.Value)
	if err != nil {
		return nil, err
	}

	// Create new environment with binding
	newEnv := e.env.NewChildEnvironment()
	newEnv.Set(let.Name, val)

	// Evaluate body in new environment
	oldEnv := e.env
	e.env = newEnv
	result, err := e.evalCore(let.Body)
	e.env = oldEnv

	return result, err
}

// CycleMarker is a special value used to detect initialization cycles
type CycleMarker struct {
	Name string
}

func (c *CycleMarker) String() string { return fmt.Sprintf("<cycle-marker:%s>", c.Name) }
func (c *CycleMarker) Type() string   { return "CycleMarker" }

// evalCoreLetRec evaluates recursive let bindings
func (e *CoreEvaluator) evalCoreLetRec(letrec *core.LetRec) (Value, error) {
	// Create new environment for recursive bindings
	newEnv := e.env.NewChildEnvironment()

	// First pass: create cycle markers for recursive references
	for _, binding := range letrec.Bindings {
		newEnv.Set(binding.Name, &CycleMarker{Name: binding.Name})
	}

	// Second pass: evaluate bindings in the recursive environment
	oldEnv := e.env
	e.env = newEnv

	for _, binding := range letrec.Bindings {
		// Check if we're trying to access ourselves during evaluation
		val, err := e.evalCore(binding.Value)
		if err != nil {
			e.env = oldEnv
			return nil, err
		}

		// Check if the value is still a cycle marker (immediate self-reference)
		if marker, ok := val.(*CycleMarker); ok {
			e.env = oldEnv
			return nil, fmt.Errorf("RT009: value initialization cycle detected: %s references itself", marker.Name)
		}

		newEnv.Set(binding.Name, val)
	}

	// Evaluate body in recursive environment
	result, err := e.evalCore(letrec.Body)
	e.env = oldEnv

	return result, err
}

// evalCoreApp evaluates function application
func (e *CoreEvaluator) evalCoreApp(app *core.App) (Value, error) {
	// Check if this is a builtin function call
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module == "$builtin" {
		// Evaluate arguments
		var args []Value
		for _, arg := range app.Args {
			argVal, err := e.evalCore(arg)
			if err != nil {
				return nil, err
			}
			args = append(args, argVal)
		}

		// Call the builtin
		return CallBuiltin(vg.Ref.Name, args)
	}

	// Evaluate function
	fnVal, err := e.evalCore(app.Func)
	if err != nil {
		return nil, err
	}

	// Evaluate arguments
	var args []Value
	for _, arg := range app.Args {
		argVal, err := e.evalCore(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, argVal)
	}

	// Apply function
	switch fn := fnVal.(type) {
	case *FunctionValue:
		if len(args) != len(fn.Params) {
			return nil, fmt.Errorf("function expects %d arguments, got %d", len(fn.Params), len(args))
		}

		// Create new environment with parameters bound
		newEnv := fn.Env.Clone()
		for i, param := range fn.Params {
			newEnv.Set(param, args[i])
		}

		// Evaluate body
		oldEnv := e.env
		e.env = newEnv

		// Body could be Core or TypedAST depending on origin
		var result Value
		if coreBody, ok := fn.Body.(core.CoreExpr); ok {
			result, err = e.evalCore(coreBody)
		} else {
			return nil, fmt.Errorf("function body is not Core AST")
		}

		e.env = oldEnv
		return result, err

	case *BuiltinFunction:
		return fn.Fn(args)

	default:
		return nil, fmt.Errorf("cannot apply non-function value: %T", fnVal)
	}
}

// evalCoreIf evaluates conditional
func (e *CoreEvaluator) evalCoreIf(ifExpr *core.If) (Value, error) {
	// Evaluate condition
	condVal, err := e.evalCore(ifExpr.Cond)
	if err != nil {
		return nil, err
	}

	// Check condition is boolean
	boolVal, ok := condVal.(*BoolValue)
	if !ok {
		return nil, fmt.Errorf("if condition must be boolean, got %T", condVal)
	}

	// Evaluate appropriate branch
	if boolVal.Value {
		return e.evalCore(ifExpr.Then)
	} else {
		return e.evalCore(ifExpr.Else)
	}
}

// evalCoreBinOp evaluates binary operation
func (e *CoreEvaluator) evalCoreBinOp(binop *core.BinOp) (Value, error) {
	// Evaluate operands
	leftVal, err := e.evalCore(binop.Left)
	if err != nil {
		return nil, err
	}

	rightVal, err := e.evalCore(binop.Right)
	if err != nil {
		return nil, err
	}

	// Apply operation based on operator and types
	return e.applyBinOp(binop.Op, leftVal, rightVal)
}

// evalCoreUnOp evaluates unary operation
func (e *CoreEvaluator) evalCoreUnOp(unop *core.UnOp) (Value, error) {
	// Evaluate operand
	operandVal, err := e.evalCore(unop.Operand)
	if err != nil {
		return nil, err
	}

	// Apply operation
	return applyUnOp(unop.Op, operandVal)
}

// evalCoreRecord evaluates record construction
func (e *CoreEvaluator) evalCoreRecord(record *core.Record) (Value, error) {
	fields := make(map[string]Value)

	for name, fieldExpr := range record.Fields {
		val, err := e.evalCore(fieldExpr)
		if err != nil {
			return nil, err
		}
		fields[name] = val
	}

	return &RecordValue{Fields: fields}, nil
}

// evalCoreRecordAccess evaluates field access
func (e *CoreEvaluator) evalCoreRecordAccess(access *core.RecordAccess) (Value, error) {
	recordVal, err := e.evalCore(access.Record)
	if err != nil {
		return nil, err
	}

	record, ok := recordVal.(*RecordValue)
	if !ok {
		return nil, fmt.Errorf("cannot access field of non-record value: %T", recordVal)
	}

	val, ok := record.Fields[access.Field]
	if !ok {
		return nil, fmt.Errorf("record has no field: %s", access.Field)
	}

	return val, nil
}

// evalCoreList evaluates list construction
func (e *CoreEvaluator) evalCoreList(list *core.List) (Value, error) {
	var elements []Value

	for _, elemExpr := range list.Elements {
		val, err := e.evalCore(elemExpr)
		if err != nil {
			return nil, err
		}
		elements = append(elements, val)
	}

	return &ListValue{Elements: elements}, nil
}

// evalCoreMatch evaluates pattern matching
func (e *CoreEvaluator) evalCoreMatch(match *core.Match) (Value, error) {
	// Evaluate scrutinee
	scrutineeVal, err := e.evalCore(match.Scrutinee)
	if err != nil {
		return nil, err
	}

	// Try each arm
	for _, arm := range match.Arms {
		bindings, matched := matchPattern(arm.Pattern, scrutineeVal)
		if !matched {
			continue
		}

		// Pattern matched - evaluate body with bindings
		newEnv := e.env.NewChildEnvironment()
		for name, val := range bindings {
			newEnv.Set(name, val)
		}

		oldEnv := e.env
		e.env = newEnv
		result, err := e.evalCore(arm.Body)
		e.env = oldEnv

		return result, err
	}

	return nil, fmt.Errorf("no pattern matched in match expression")
}

// evalDictRef evaluates a dictionary reference
func (e *CoreEvaluator) evalDictRef(ref *core.DictRef) (Value, error) {
	// Create a dictionary value that contains the methods
	// The dictionary is a record with method implementations

	// Look up all methods for this class/type combination
	methods := make(map[string]Value)

	// Create type for normalized key generation
	typ := &types.TCon{Name: ref.TypeName}

	// Common methods for each class
	var methodNames []string
	switch ref.ClassName {
	case "Num":
		methodNames = []string{"add", "sub", "mul", "div", "neg", "abs", "fromInt"}
	case "Fractional":
		methodNames = []string{"add", "sub", "mul", "div", "neg", "abs", "fromInt", "divide", "recip", "fromRational"}
	case "Eq":
		methodNames = []string{"eq", "neq"}
	case "Ord":
		methodNames = []string{"lt", "lte", "gt", "gte", "min", "max"}
	default:
		return nil, fmt.Errorf("unknown type class: %s", ref.ClassName)
	}

	// Collect all methods
	for _, method := range methodNames {
		key := types.MakeDictionaryKey("prelude", ref.ClassName, typ, method)
		entry, ok := e.registry.Lookup(key)
		if !ok {
			return nil, fmt.Errorf("missing dictionary method: %s", key)
		}

		// Check if the implementation is already a BuiltinFunction
		if builtin, ok := entry.Impl.(*BuiltinFunction); ok {
			methods[method] = builtin
		} else {
			// Wrap the implementation as a builtin function
			methods[method] = &BuiltinFunction{
				Name: method,
				Fn:   wrapDictionaryMethod(entry.Impl),
			}
		}
	}

	// Return dictionary as a record
	return &RecordValue{Fields: methods}, nil
}

// evalDictAbs evaluates dictionary abstraction
func (e *CoreEvaluator) evalDictAbs(abs *core.DictAbs) (Value, error) {
	// Dictionary abstraction introduces dictionary parameters
	// We need to evaluate the body with dictionaries in scope

	// For now, we'll just evaluate the body
	// In a full implementation, this would handle polymorphic dictionary passing
	return e.evalCore(abs.Body)
}

// evalIntrinsic evaluates an intrinsic operation
// This should typically be handled by OpLowering pass, but we provide
// a fallback implementation using the experimental binop shim
func (e *CoreEvaluator) evalIntrinsic(intrinsic *core.Intrinsic) (Value, error) {
	// Evaluate arguments
	args := make([]Value, len(intrinsic.Args))
	for i, arg := range intrinsic.Args {
		val, err := e.evalCore(arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Map intrinsic to operator for shim
	if e.experimentalBinopShim {
		// Binary operations
		if len(args) == 2 {
			var op string
			switch intrinsic.Op {
			case core.OpAdd:
				op = "+"
			case core.OpSub:
				op = "-"
			case core.OpMul:
				op = "*"
			case core.OpDiv:
				op = "/"
			case core.OpMod:
				op = "%"
			case core.OpEq:
				op = "=="
			case core.OpNe:
				op = "!="
			case core.OpLt:
				op = "<"
			case core.OpLe:
				op = "<="
			case core.OpGt:
				op = ">"
			case core.OpGe:
				op = ">="
			case core.OpConcat:
				op = "++"
			case core.OpAnd:
				op = "&&"
			case core.OpOr:
				op = "||"
			default:
				return nil, fmt.Errorf("unknown intrinsic operation: %v", intrinsic.Op)
			}
			return e.applyBinOp(op, args[0], args[1])
		}

		// Unary operations
		if len(args) == 1 {
			var op string
			switch intrinsic.Op {
			case core.OpNot:
				op = "not"
			case core.OpNeg:
				op = "-"
			default:
				return nil, fmt.Errorf("unknown unary intrinsic: %v", intrinsic.Op)
			}
			return applyUnOp(op, args[0])
		}
	}

	return nil, fmt.Errorf("intrinsic operations require OpLowering pass or --experimental-binop-shim flag")
}

// evalDictApp evaluates dictionary application
func (e *CoreEvaluator) evalDictApp(app *core.DictApp) (Value, error) {
	// Evaluate the dictionary
	dictVal, err := e.evalCore(app.Dict)
	if err != nil {
		return nil, err
	}

	// Dictionary should be a record with methods
	dict, ok := dictVal.(*RecordValue)
	if !ok {
		return nil, fmt.Errorf("dictionary must be a record, got %T", dictVal)
	}

	// Look up the method
	// fmt.Printf("DEBUG: DictApp looking for method '%s' in dictionary with fields: %v\n", app.Method, getFieldNames(dict.Fields))
	methodVal, ok := dict.Fields[app.Method]
	if !ok {
		return nil, fmt.Errorf("dictionary missing method: %s", app.Method)
	}

	// Evaluate arguments
	var args []Value
	for _, argExpr := range app.Args {
		argVal, err := e.evalCore(argExpr)
		if err != nil {
			return nil, err
		}
		args = append(args, argVal)
	}

	// Apply the method with proper type checking
	switch method := methodVal.(type) {
	case *BuiltinFunction:
		// Proper BuiltinFunction - use its Fn
		return method.Fn(args)
	default:
		// Raw function that slipped through - this should not happen with proper registration
		return nil, fmt.Errorf("unsupported dictionary method type: %T", methodVal)
	}
}

// wrapDictionaryMethod wraps a Go function as a Value function
func wrapDictionaryMethod(impl interface{}) func([]Value) (Value, error) {
	// If it's already a BuiltinFunction, extract its Fn
	if builtin, ok := impl.(*BuiltinFunction); ok {
		return builtin.Fn
	}

	return func(args []Value) (Value, error) {
		// This is a simplified wrapper - a full implementation would handle
		// all type conversions properly

		switch fn := impl.(type) {
		case func(int64, int64) int64:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*IntValue)
			y, ok2 := args[1].(*IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			result := fn(int64(x.Value), int64(y.Value))
			return &IntValue{Value: int(result)}, nil

		case func(int, int) int:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*IntValue)
			y, ok2 := args[1].(*IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			result := fn(x.Value, y.Value)
			return &IntValue{Value: result}, nil

		case func(float64, float64) float64:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*FloatValue)
			y, ok2 := args[1].(*FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			result := fn(x.Value, y.Value)
			return &FloatValue{Value: result}, nil

		case func(int64, int64) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*IntValue)
			y, ok2 := args[1].(*IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			result := fn(int64(x.Value), int64(y.Value))
			return &BoolValue{Value: result}, nil

		case func(int, int) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*IntValue)
			y, ok2 := args[1].(*IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			result := fn(int(x.Value), int(y.Value))
			return &BoolValue{Value: result}, nil

		case func(float64, float64) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*FloatValue)
			y, ok2 := args[1].(*FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			result := fn(x.Value, y.Value)
			return &BoolValue{Value: result}, nil

		case func(int) int:
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 argument")
			}
			x, ok := args[0].(*IntValue)
			if !ok {
				return nil, fmt.Errorf("expected int argument")
			}
			result := fn(x.Value)
			return &IntValue{Value: result}, nil

		case func(float64) float64:
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 argument")
			}
			x, ok := args[0].(*FloatValue)
			if !ok {
				return nil, fmt.Errorf("expected float argument")
			}
			result := fn(x.Value)
			return &FloatValue{Value: result}, nil

		case func(bool) bool:
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 argument")
			}
			x, ok := args[0].(*BoolValue)
			if !ok {
				return nil, fmt.Errorf("expected bool argument")
			}
			result := fn(x.Value)
			return &BoolValue{Value: result}, nil

		case func(string, string) bool:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*StringValue)
			y, ok2 := args[1].(*StringValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected string arguments")
			}
			result := fn(x.Value, y.Value)
			return &BoolValue{Value: result}, nil

		case func(string, string) string:
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments")
			}
			x, ok1 := args[0].(*StringValue)
			y, ok2 := args[1].(*StringValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected string arguments")
			}
			result := fn(x.Value, y.Value)
			return &StringValue{Value: result}, nil

		default:
			return nil, fmt.Errorf("unsupported dictionary method type: %T", impl)
		}
	}
}

// matchPattern attempts to match a pattern against a value
func matchPattern(pattern core.CorePattern, value Value) (map[string]Value, bool) {
	bindings := make(map[string]Value)

	switch p := pattern.(type) {
	case *core.VarPattern:
		// Variable pattern always matches and binds
		bindings[p.Name] = value
		return bindings, true

	case *core.LitPattern:
		// Literal pattern matches if values are equal
		switch v := value.(type) {
		case *IntValue:
			if i, ok := p.Value.(int); ok && i == v.Value {
				return bindings, true
			}
		case *FloatValue:
			if f, ok := p.Value.(float64); ok && f == v.Value {
				return bindings, true
			}
		case *StringValue:
			if s, ok := p.Value.(string); ok && s == v.Value {
				return bindings, true
			}
		case *BoolValue:
			if b, ok := p.Value.(bool); ok && b == v.Value {
				return bindings, true
			}
		}
		return nil, false

	case *core.WildcardPattern:
		// Wildcard always matches without binding
		return bindings, true

	default:
		// Other patterns not yet implemented
		return nil, false
	}
}

// applyBinOp should NOT be called in dictionary-passing system except for special operators
// This is a fail-fast guard to ensure BinOp nodes are properly elaborated to DictApp
func (e *CoreEvaluator) applyBinOp(op string, left, right Value) (Value, error) {
	// Special case: string concatenation doesn't use type classes
	if op == "++" {
		lStr, lOk := left.(*StringValue)
		rStr, rOk := right.(*StringValue)
		if !lOk || !rOk {
			return nil, fmt.Errorf("'++' requires string operands")
		}
		return &StringValue{Value: lStr.Value + rStr.Value}, nil
	}

	// Special case: boolean operators don't use type classes
	if op == "&&" || op == "||" {
		lBool, lOk := left.(*BoolValue)
		rBool, rOk := right.(*BoolValue)
		if !lOk || !rOk {
			return nil, fmt.Errorf("'%s' requires boolean operands", op)
		}

		switch op {
		case "&&":
			return &BoolValue{Value: lBool.Value && rBool.Value}, nil
		case "||":
			return &BoolValue{Value: lBool.Value || rBool.Value}, nil
		}
	}

	// Experimental operator shim for basic arithmetic
	if e.experimentalBinopShim {
		// Try Int operations
		if lInt, lOk := left.(*IntValue); lOk {
			if rInt, rOk := right.(*IntValue); rOk {
				switch op {
				case "+":
					return &IntValue{Value: lInt.Value + rInt.Value}, nil
				case "-":
					return &IntValue{Value: lInt.Value - rInt.Value}, nil
				case "*":
					return &IntValue{Value: lInt.Value * rInt.Value}, nil
				case "/":
					if rInt.Value == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					return &IntValue{Value: lInt.Value / rInt.Value}, nil
				case "%":
					if rInt.Value == 0 {
						return nil, fmt.Errorf("modulo by zero")
					}
					return &IntValue{Value: lInt.Value % rInt.Value}, nil
				case "==":
					return &BoolValue{Value: lInt.Value == rInt.Value}, nil
				case "!=":
					return &BoolValue{Value: lInt.Value != rInt.Value}, nil
				case "<":
					return &BoolValue{Value: lInt.Value < rInt.Value}, nil
				case ">":
					return &BoolValue{Value: lInt.Value > rInt.Value}, nil
				case "<=":
					return &BoolValue{Value: lInt.Value <= rInt.Value}, nil
				case ">=":
					return &BoolValue{Value: lInt.Value >= rInt.Value}, nil
				}
			}
		}

		// Try Float operations
		if lFloat, lOk := left.(*FloatValue); lOk {
			if rFloat, rOk := right.(*FloatValue); rOk {
				switch op {
				case "+":
					return &FloatValue{Value: lFloat.Value + rFloat.Value}, nil
				case "-":
					return &FloatValue{Value: lFloat.Value - rFloat.Value}, nil
				case "*":
					return &FloatValue{Value: lFloat.Value * rFloat.Value}, nil
				case "/":
					if rFloat.Value == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					return &FloatValue{Value: lFloat.Value / rFloat.Value}, nil
				case "==":
					return &BoolValue{Value: lFloat.Value == rFloat.Value}, nil
				case "!=":
					return &BoolValue{Value: lFloat.Value != rFloat.Value}, nil
				case "<":
					return &BoolValue{Value: lFloat.Value < rFloat.Value}, nil
				case ">":
					return &BoolValue{Value: lFloat.Value > rFloat.Value}, nil
				case "<=":
					return &BoolValue{Value: lFloat.Value <= rFloat.Value}, nil
				case ">=":
					return &BoolValue{Value: lFloat.Value >= rFloat.Value}, nil
				}
			}
		}
	}

	// All other operators must go through dictionary elaboration
	return nil, fmt.Errorf("internal: BinOp reached evaluator; dictionaries not elaborated (op='%s')", op)
}

// applyUnOp applies a unary operator to a value
func applyUnOp(op string, operand Value) (Value, error) {
	switch op {
	case "-":
		switch v := operand.(type) {
		case *IntValue:
			return &IntValue{Value: -v.Value}, nil
		case *FloatValue:
			return &FloatValue{Value: -v.Value}, nil
		}

	case "!":
		if v, ok := operand.(*BoolValue); ok {
			return &BoolValue{Value: !v.Value}, nil
		}
	}

	return nil, fmt.Errorf("cannot apply unary operator %s to %T", op, operand)
}
