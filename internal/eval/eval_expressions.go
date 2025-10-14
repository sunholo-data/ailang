package eval

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
)

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

	case *core.Tuple:
		return e.evalCoreTuple(n)

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
	// Force IndirectValue if needed (for LetRec recursion)
	if iv, ok := val.(*IndirectValue); ok {
		forced, err := iv.Force()
		if err != nil {
			return nil, err
		}
		return forced, nil
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

// evalCoreLetRec evaluates recursive let bindings using indirection cells
// Implements function-first semantics (OCaml/Haskell style) for safe recursion
func (e *CoreEvaluator) evalCoreLetRec(letrec *core.LetRec) (Value, error) {
	// Phase 1: Pre-allocate indirection cells and extend environment
	recEnv := e.env.NewChildEnvironment()
	cells := make(map[string]*RefCell, len(letrec.Bindings))

	for _, binding := range letrec.Bindings {
		cell := &RefCell{} // Uninitialized cell
		cells[binding.Name] = cell
		recEnv.Set(binding.Name, &IndirectValue{Cell: cell})
	}

	// Phase 2: Evaluate RHS under recursive environment
	oldEnv := e.env
	e.env = recEnv
	defer func() { e.env = oldEnv }()

	for _, binding := range letrec.Bindings {
		// Optimize for lambda RHS: build closure immediately (safe, body executes later)
		if lam, ok := isLambda(binding.Value); ok {
			fv, err := e.buildClosure(lam, recEnv)
			if err != nil {
				return nil, err
			}
			cells[binding.Name].Val = fv
			cells[binding.Name].Init = true
			continue
		}

		// Non-lambda RHS: strict evaluation
		// Mark visiting to detect immediate cycles
		cells[binding.Name].Visiting = true
		val, err := e.evalCore(binding.Value)
		cells[binding.Name].Visiting = false
		if err != nil {
			return nil, err
		}

		cells[binding.Name].Val = val
		cells[binding.Name].Init = true
	}

	// Phase 3: Evaluate body under recursive environment
	return e.evalCore(letrec.Body)
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

// evalCoreTuple evaluates tuple construction
func (e *CoreEvaluator) evalCoreTuple(tuple *core.Tuple) (Value, error) {
	var elements []Value

	for _, elemExpr := range tuple.Elements {
		val, err := e.evalCore(elemExpr)
		if err != nil {
			return nil, err
		}
		elements = append(elements, val)
	}

	return &TupleValue{Elements: elements}, nil
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
