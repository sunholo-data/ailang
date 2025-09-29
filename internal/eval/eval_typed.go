package eval

import (
	"fmt"
	"math"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/typedast"
	"github.com/sunholo/ailang/internal/types"
)

// TypedEvaluator evaluates TypedAST programs
type TypedEvaluator struct {
	env         *Environment
	trace       *TraceCollector
	seed        *int64
	virtualTime bool
}

// TraceCollector collects execution traces for training data
type TraceCollector struct {
	Entries []TraceEntry
	Enabled bool
}

// TraceEntry represents a single trace entry
type TraceEntry struct {
	CallSiteID  uint64
	FnID        uint64
	FnScheme    *types.Scheme
	CallEffects *types.Row
	Inputs      []string
	Output      string
	Seed        *int64
	VirtualTime bool
	Timestamp   int64
}

// NewTypedEvaluator creates a new typed evaluator
func NewTypedEvaluator(trace bool, seed int, virtualTime bool) *TypedEvaluator {
	env := NewEnvironment()
	registerBuiltins(env)

	var traceCollector *TraceCollector
	if trace {
		traceCollector = &TraceCollector{
			Enabled: true,
			Entries: []TraceEntry{},
		}
	}

	var seedPtr *int64
	if seed != 0 {
		s := int64(seed)
		seedPtr = &s
	}

	return &TypedEvaluator{
		env:         env,
		trace:       traceCollector,
		seed:        seedPtr,
		virtualTime: virtualTime,
	}
}

// NewTypedEvaluatorWithEnv creates evaluator with existing environment
func NewTypedEvaluatorWithEnv(env *Environment, trace bool, seed int, virtualTime bool) *TypedEvaluator {
	var traceCollector *TraceCollector
	if trace {
		traceCollector = &TraceCollector{
			Enabled: true,
			Entries: []TraceEntry{},
		}
	}

	var seedPtr *int64
	if seed != 0 {
		s := int64(seed)
		seedPtr = &s
	}

	return &TypedEvaluator{
		env:         env,
		trace:       traceCollector,
		seed:        seedPtr,
		virtualTime: virtualTime,
	}
}

// EvalTypedProgram evaluates a typed program
func (e *TypedEvaluator) EvalTypedProgram(prog *typedast.TypedProgram) (Value, error) {
	var lastResult Value = &UnitValue{}

	for _, decl := range prog.Decls {
		result, err := e.evalTypedNode(decl)
		if err != nil {
			return nil, err
		}
		lastResult = result
	}

	return lastResult, nil
}

// evalTypedNode evaluates a typed node
func (e *TypedEvaluator) evalTypedNode(node typedast.TypedNode) (Value, error) {
	// Note: We only use type information for tracing/errors, not for behavior

	switch n := node.(type) {
	case *typedast.TypedVar:
		return e.evalVar(n)

	case *typedast.TypedLit:
		return e.evalLit(n)

	case *typedast.TypedLambda:
		return e.evalLambda(n)

	case *typedast.TypedLet:
		return e.evalLet(n)

	case *typedast.TypedLetRec:
		return e.evalLetRec(n)

	case *typedast.TypedApp:
		return e.evalApp(n)

	case *typedast.TypedIf:
		return e.evalIf(n)

	case *typedast.TypedBinOp:
		return e.evalBinOp(n)

	case *typedast.TypedUnOp:
		return e.evalUnOp(n)

	case *typedast.TypedRecord:
		return e.evalRecord(n)

	case *typedast.TypedRecordAccess:
		return e.evalRecordAccess(n)

	case *typedast.TypedList:
		return e.evalList(n)

	case *typedast.TypedMatch:
		return e.evalMatch(n)

	default:
		return nil, fmt.Errorf("evaluation not implemented for %T", node)
	}
}

// evalVar evaluates a variable
func (e *TypedEvaluator) evalVar(v *typedast.TypedVar) (Value, error) {
	val, ok := e.env.Get(v.Name)
	if !ok {
		// Should not happen in well-typed programs
		return nil, fmt.Errorf("undefined variable: %s at %s", v.Name, v.Span)
	}
	return val, nil
}

// evalLit evaluates a literal
func (e *TypedEvaluator) evalLit(lit *typedast.TypedLit) (Value, error) {
	switch lit.Kind {
	case core.IntLit:
		if i, ok := lit.Value.(int); ok {
			return &IntValue{Value: i}, nil
		}
		return nil, fmt.Errorf("invalid int literal: %v", lit.Value)

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

// evalLambda evaluates a lambda (creates closure)
func (e *TypedEvaluator) evalLambda(lam *typedast.TypedLambda) (Value, error) {
	return &FunctionValue{
		Params: lam.Params,
		Body:   lam.Body,      // Store typed body
		Env:    e.env.Clone(), // Capture environment
		Typed:  true,
	}, nil
}

// evalLet evaluates let binding
func (e *TypedEvaluator) evalLet(let *typedast.TypedLet) (Value, error) {
	// Evaluate value
	val, err := e.evalTypedNode(let.Value)
	if err != nil {
		return nil, err
	}

	// Extend environment
	oldEnv := e.env
	e.env = e.env.Extend(let.Name, val)

	// Evaluate body
	result, err := e.evalTypedNode(let.Body)

	// Restore environment
	e.env = oldEnv

	return result, err
}

// evalLetRec evaluates recursive bindings
func (e *TypedEvaluator) evalLetRec(letrec *typedast.TypedLetRec) (Value, error) {
	// Create new environment for recursion
	recEnv := e.env.NewChildEnvironment()

	// First pass: create function placeholders
	for _, binding := range letrec.Bindings {
		// For now, assume all recursive bindings are functions
		if lam, ok := binding.Value.(*typedast.TypedLambda); ok {
			fn := &FunctionValue{
				Params: lam.Params,
				Body:   lam.Body,
				Env:    recEnv, // Will be updated
				Typed:  true,
			}
			recEnv.Set(binding.Name, fn)
		} else {
			return nil, fmt.Errorf("non-function recursive binding not supported: %s", binding.Name)
		}
	}

	// Second pass: update closures with complete environment
	for _, binding := range letrec.Bindings {
		if fn, ok := recEnv.values[binding.Name].(*FunctionValue); ok {
			fn.Env = recEnv.Clone()
		}
	}

	// Evaluate body in recursive environment
	oldEnv := e.env
	e.env = recEnv
	result, err := e.evalTypedNode(letrec.Body)
	e.env = oldEnv

	return result, err
}

// evalApp evaluates function application
func (e *TypedEvaluator) evalApp(app *typedast.TypedApp) (Value, error) {
	// Evaluate function
	fnVal, err := e.evalTypedNode(app.Func)
	if err != nil {
		return nil, err
	}

	// Evaluate arguments
	var args []Value
	for _, arg := range app.Args {
		argVal, err := e.evalTypedNode(arg)
		if err != nil {
			return nil, err
		}
		args = append(args, argVal)
	}

	// Trace if enabled
	if e.trace != nil && e.trace.Enabled {
		e.recordTrace(app, fnVal, args)
	}

	// Apply function
	switch fn := fnVal.(type) {
	case *FunctionValue:
		return e.applyFunction(fn, args)

	case *BuiltinFunction:
		return fn.Fn(args)

	default:
		return nil, fmt.Errorf("cannot apply non-function: %T at %s", fnVal, app.Span)
	}
}

// applyFunction applies a user-defined function
func (e *TypedEvaluator) applyFunction(fn *FunctionValue, args []Value) (Value, error) {
	if len(args) != len(fn.Params) {
		return nil, fmt.Errorf("argument count mismatch: expected %d, got %d",
			len(fn.Params), len(args))
	}

	// Create new environment extending closure
	oldEnv := e.env
	e.env = fn.Env.NewChildEnvironment()

	// Bind parameters
	for i, param := range fn.Params {
		e.env.Set(param, args[i])
	}

	// Evaluate body
	var result Value
	var err error
	if typedBody, ok := fn.Body.(typedast.TypedNode); ok && fn.Typed {
		result, err = e.evalTypedNode(typedBody)
	} else if coreBody, ok := fn.Body.(core.CoreExpr); ok {
		result, err = e.evalCore(coreBody)
	} else {
		err = fmt.Errorf("invalid function body type: %T", fn.Body)
	}

	// Restore environment
	e.env = oldEnv

	return result, err
}

// evalIf evaluates conditional
func (e *TypedEvaluator) evalIf(ifExpr *typedast.TypedIf) (Value, error) {
	// Evaluate condition
	condVal, err := e.evalTypedNode(ifExpr.Cond)
	if err != nil {
		return nil, err
	}

	// Check condition
	boolVal, ok := condVal.(*BoolValue)
	if !ok {
		return nil, fmt.Errorf("condition must be boolean, got %T at %s", condVal, ifExpr.Span)
	}

	// Evaluate appropriate branch
	if boolVal.Value {
		return e.evalTypedNode(ifExpr.Then)
	} else {
		return e.evalTypedNode(ifExpr.Else)
	}
}

// evalBinOp evaluates binary operation
func (e *TypedEvaluator) evalBinOp(binop *typedast.TypedBinOp) (Value, error) {
	// Evaluate operands
	left, err := e.evalTypedNode(binop.Left)
	if err != nil {
		return nil, err
	}

	right, err := e.evalTypedNode(binop.Right)
	if err != nil {
		return nil, err
	}

	// Perform operation based on operator and types
	switch binop.Op {
	case "+":
		return e.evalAdd(left, right, binop.Span.String())
	case "-":
		return e.evalSub(left, right, binop.Span.String())
	case "*":
		return e.evalMul(left, right, binop.Span.String())
	case "/":
		return e.evalDiv(left, right, binop.Span.String())
	case "%":
		return e.evalMod(left, right, binop.Span.String())
	case "++":
		return e.evalConcat(left, right, binop.Span.String())
	case "<":
		return e.evalLess(left, right, binop.Span.String())
	case ">":
		return e.evalGreater(left, right, binop.Span.String())
	case "<=":
		return e.evalLessEq(left, right, binop.Span.String())
	case ">=":
		return e.evalGreaterEq(left, right, binop.Span.String())
	case "==":
		return e.evalEqual(left, right, binop.Span.String())
	case "!=":
		return e.evalNotEqual(left, right, binop.Span.String())
	case "&&":
		return e.evalAnd(left, right, binop.Span.String())
	case "||":
		return e.evalOr(left, right, binop.Span.String())
	default:
		return nil, fmt.Errorf("unknown binary operator: %s at %s", binop.Op, binop.Span)
	}
}

// evalUnOp evaluates unary operation
func (e *TypedEvaluator) evalUnOp(unop *typedast.TypedUnOp) (Value, error) {
	// Evaluate operand
	operand, err := e.evalTypedNode(unop.Operand)
	if err != nil {
		return nil, err
	}

	switch unop.Op {
	case "-":
		return e.evalNegate(operand, unop.Span.String())
	case "not":
		return e.evalNot(operand, unop.Span.String())
	default:
		return nil, fmt.Errorf("unknown unary operator: %s at %s", unop.Op, unop.Span)
	}
}

// evalRecord evaluates record construction
func (e *TypedEvaluator) evalRecord(rec *typedast.TypedRecord) (Value, error) {
	fields := make(map[string]Value)

	for name, value := range rec.Fields {
		val, err := e.evalTypedNode(value)
		if err != nil {
			return nil, err
		}
		fields[name] = val
	}

	return &RecordValue{Fields: fields}, nil
}

// evalRecordAccess evaluates field access
func (e *TypedEvaluator) evalRecordAccess(acc *typedast.TypedRecordAccess) (Value, error) {
	// Evaluate record
	recordVal, err := e.evalTypedNode(acc.Record)
	if err != nil {
		return nil, err
	}

	// Access field
	recVal, ok := recordVal.(*RecordValue)
	if !ok {
		return nil, fmt.Errorf("cannot access field of non-record: %T at %s", recordVal, acc.Span)
	}

	val, ok := recVal.Fields[acc.Field]
	if !ok {
		return nil, fmt.Errorf("field not found: %s at %s", acc.Field, acc.Span)
	}

	return val, nil
}

// evalList evaluates list construction
func (e *TypedEvaluator) evalList(list *typedast.TypedList) (Value, error) {
	var elements []Value

	for _, elem := range list.Elements {
		val, err := e.evalTypedNode(elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, val)
	}

	return &ListValue{Elements: elements}, nil
}

// evalMatch evaluates pattern matching
func (e *TypedEvaluator) evalMatch(match *typedast.TypedMatch) (Value, error) {
	// Evaluate scrutinee
	scrutinee, err := e.evalTypedNode(match.Scrutinee)
	if err != nil {
		return nil, err
	}

	// Try each arm
	for _, arm := range match.Arms {
		// Match pattern
		bindings, matched := e.matchPattern(arm.Pattern, scrutinee)
		if !matched {
			continue
		}

		// Check guard if present
		if arm.Guard != nil {
			oldEnv := e.env
			e.env = e.env.NewChildEnvironment()

			// Bind pattern variables
			for name, val := range bindings {
				e.env.Set(name, val)
			}

			guardVal, err := e.evalTypedNode(arm.Guard)
			e.env = oldEnv

			if err != nil {
				return nil, err
			}

			if boolVal, ok := guardVal.(*BoolValue); !ok || !boolVal.Value {
				continue
			}
		}

		// Evaluate body with pattern bindings
		oldEnv := e.env
		e.env = e.env.NewChildEnvironment()

		for name, val := range bindings {
			e.env.Set(name, val)
		}

		result, err := e.evalTypedNode(arm.Body)
		e.env = oldEnv

		return result, err
	}

	return nil, fmt.Errorf("non-exhaustive pattern match at %s", match.Span)
}

// matchPattern attempts to match a pattern against a value
func (e *TypedEvaluator) matchPattern(pat typedast.TypedPattern, val Value) (map[string]Value, bool) {
	switch p := pat.(type) {
	case typedast.TypedVarPattern:
		// Variable pattern always matches
		return map[string]Value{p.Name: val}, true

	case typedast.TypedLitPattern:
		// Literal pattern must match exactly
		if e.valuesEqual(val, p.Value) {
			return nil, true
		}
		return nil, false

	case typedast.TypedWildcardPattern:
		// Wildcard always matches
		return nil, true

	default:
		// TODO: Implement other pattern types
		return nil, false
	}
}

// evalCore evaluates a Core expression (fallback for untyped bodies)
func (e *TypedEvaluator) evalCore(expr core.CoreExpr) (Value, error) {
	// This is a simplified version - full implementation would mirror evalTypedNode
	return nil, fmt.Errorf("Core evaluation not implemented")
}

// Arithmetic operations

func (e *TypedEvaluator) evalAdd(left, right Value, loc string) (Value, error) {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return &IntValue{Value: l.Value + r.Value}, nil
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return &FloatValue{Value: l.Value + r.Value}, nil
		}
	}
	return nil, fmt.Errorf("cannot add %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalSub(left, right Value, loc string) (Value, error) {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return &IntValue{Value: l.Value - r.Value}, nil
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return &FloatValue{Value: l.Value - r.Value}, nil
		}
	}
	return nil, fmt.Errorf("cannot subtract %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalMul(left, right Value, loc string) (Value, error) {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return &IntValue{Value: l.Value * r.Value}, nil
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return &FloatValue{Value: l.Value * r.Value}, nil
		}
	}
	return nil, fmt.Errorf("cannot multiply %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalDiv(left, right Value, loc string) (Value, error) {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			if r.Value == 0 {
				return nil, fmt.Errorf("division by zero at %s", loc)
			}
			return &IntValue{Value: l.Value / r.Value}, nil
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			if r.Value == 0 {
				return nil, fmt.Errorf("division by zero at %s", loc)
			}
			return &FloatValue{Value: l.Value / r.Value}, nil
		}
	}
	return nil, fmt.Errorf("cannot divide %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalMod(left, right Value, loc string) (Value, error) {
	if l, ok := left.(*IntValue); ok {
		if r, ok := right.(*IntValue); ok {
			if r.Value == 0 {
				return nil, fmt.Errorf("modulo by zero at %s", loc)
			}
			return &IntValue{Value: l.Value % r.Value}, nil
		}
	} else if l, ok := left.(*FloatValue); ok {
		if r, ok := right.(*FloatValue); ok {
			if r.Value == 0 {
				return nil, fmt.Errorf("modulo by zero at %s", loc)
			}
			return &FloatValue{Value: math.Mod(l.Value, r.Value)}, nil
		}
	}
	return nil, fmt.Errorf("cannot mod %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalConcat(left, right Value, loc string) (Value, error) {
	l, ok1 := left.(*StringValue)
	r, ok2 := right.(*StringValue)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot concatenate %T and %T at %s", left, right, loc)
	}
	return &StringValue{Value: l.Value + r.Value}, nil
}

// Comparison operations

func (e *TypedEvaluator) evalLess(left, right Value, loc string) (Value, error) {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return &BoolValue{Value: l.Value < r.Value}, nil
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return &BoolValue{Value: l.Value < r.Value}, nil
		}
	}
	return nil, fmt.Errorf("cannot compare %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalGreater(left, right Value, loc string) (Value, error) {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return &BoolValue{Value: l.Value > r.Value}, nil
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return &BoolValue{Value: l.Value > r.Value}, nil
		}
	}
	return nil, fmt.Errorf("cannot compare %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalLessEq(left, right Value, loc string) (Value, error) {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return &BoolValue{Value: l.Value <= r.Value}, nil
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return &BoolValue{Value: l.Value <= r.Value}, nil
		}
	}
	return nil, fmt.Errorf("cannot compare %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalGreaterEq(left, right Value, loc string) (Value, error) {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return &BoolValue{Value: l.Value >= r.Value}, nil
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return &BoolValue{Value: l.Value >= r.Value}, nil
		}
	}
	return nil, fmt.Errorf("cannot compare %T and %T at %s", left, right, loc)
}

func (e *TypedEvaluator) evalEqual(left, right Value, loc string) (Value, error) {
	return &BoolValue{Value: e.valuesEqual(left, right)}, nil
}

func (e *TypedEvaluator) evalNotEqual(left, right Value, loc string) (Value, error) {
	return &BoolValue{Value: !e.valuesEqual(left, right)}, nil
}

// Boolean operations

func (e *TypedEvaluator) evalAnd(left, right Value, loc string) (Value, error) {
	l, ok1 := left.(*BoolValue)
	r, ok2 := right.(*BoolValue)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot AND %T and %T at %s", left, right, loc)
	}
	return &BoolValue{Value: l.Value && r.Value}, nil
}

func (e *TypedEvaluator) evalOr(left, right Value, loc string) (Value, error) {
	l, ok1 := left.(*BoolValue)
	r, ok2 := right.(*BoolValue)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("cannot OR %T and %T at %s", left, right, loc)
	}
	return &BoolValue{Value: l.Value || r.Value}, nil
}

// Unary operations

func (e *TypedEvaluator) evalNegate(operand Value, loc string) (Value, error) {
	switch v := operand.(type) {
	case *IntValue:
		return &IntValue{Value: -v.Value}, nil
	case *FloatValue:
		return &FloatValue{Value: -v.Value}, nil
	default:
		return nil, fmt.Errorf("cannot negate %T at %s", operand, loc)
	}
}

func (e *TypedEvaluator) evalNot(operand Value, loc string) (Value, error) {
	if v, ok := operand.(*BoolValue); ok {
		return &BoolValue{Value: !v.Value}, nil
	}
	return nil, fmt.Errorf("cannot NOT %T at %s", operand, loc)
}

// valuesEqual checks if two values are equal
func (e *TypedEvaluator) valuesEqual(left, right interface{}) bool {
	switch l := left.(type) {
	case *IntValue:
		if r, ok := right.(*IntValue); ok {
			return l.Value == r.Value
		}
	case *FloatValue:
		if r, ok := right.(*FloatValue); ok {
			return l.Value == r.Value
		}
	case *StringValue:
		if r, ok := right.(*StringValue); ok {
			return l.Value == r.Value
		}
	case *BoolValue:
		if r, ok := right.(*BoolValue); ok {
			return l.Value == r.Value
		}
	case *UnitValue:
		_, ok := right.(*UnitValue)
		return ok
	case int:
		if r, ok := right.(int); ok {
			return l == r
		}
	case float64:
		if r, ok := right.(float64); ok {
			return l == r
		}
	case string:
		if r, ok := right.(string); ok {
			return l == r
		}
	case bool:
		if r, ok := right.(bool); ok {
			return l == r
		}
	}
	return false
}

// recordTrace records a function call trace
func (e *TypedEvaluator) recordTrace(app *typedast.TypedApp, fn Value, args []Value) {
	if e.trace == nil || !e.trace.Enabled {
		return
	}

	// TODO: Extract scheme and effects from typed nodes
	// For now, create a placeholder trace
	var inputs []string
	for _, arg := range args {
		inputs = append(inputs, boundedShow(arg, 3, 10))
	}

	entry := TraceEntry{
		CallSiteID:  app.NodeID,
		FnID:        0,   // TODO: Extract from function
		FnScheme:    nil, // TODO: Extract scheme
		CallEffects: nil, // TODO: Type assertion needed
		Inputs:      inputs,
		Seed:        e.seed,
		VirtualTime: e.virtualTime,
		Timestamp:   e.getTimestamp(),
	}

	e.trace.Entries = append(e.trace.Entries, entry)
}

// getTimestamp returns current timestamp (virtual or real)
func (e *TypedEvaluator) getTimestamp() int64 {
	if e.virtualTime {
		// TODO: Implement virtual time
		return 0
	}
	// TODO: Get real timestamp
	return 0
}

// boundedShow produces bounded string representation
func boundedShow(v Value, maxDepth, maxWidth int) string {
	// TODO: Implement bounded show with depth/width limits
	return showValue(v, 0)
}

// registerBuiltins registers builtin functions
func registerBuiltins(env *Environment) {
	// Register print builtin
	env.Set("print", &BuiltinFunction{
		Name: "print",
		Fn: func(args []Value) (Value, error) {
			for _, arg := range args {
				fmt.Print(arg.String())
			}
			fmt.Println()
			return &UnitValue{}, nil
		},
	})

	// Register show builtin
	env.Set("show", &BuiltinFunction{
		Name: "show",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("show expects exactly 1 argument, got %d", len(args))
			}
			return &StringValue{Value: showValue(args[0], 0)}, nil
		},
	})

	// Register toText builtin
	env.Set("toText", &BuiltinFunction{
		Name: "toText",
		Fn: func(args []Value) (Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("toText expects exactly 1 argument, got %d", len(args))
			}
			return &StringValue{Value: toTextValue(args[0])}, nil
		},
	})
}
