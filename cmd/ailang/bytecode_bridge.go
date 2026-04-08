package main

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/runtime"
)

// bytecodeBridge implements vm.EvalInterop for the `ailang run --bytecode`
// path. When the VM hits a function whose prototype is marked EvalOnly
// (because the bytecode compiler couldn't lower it), the VM hands the call
// to the bridge. The bridge:
//
//  1. Looks up the function value in the entry module's binding map.
//  2. Converts the bridged VM-side arguments to evaluator values.
//  3. Calls the evaluator's CallValueN with the converted args.
//  4. Converts the result back to a VM value and returns it.
//
// The bridge supports M-BYTECODE-VM Tier-1 value shapes (Int, Float, Bool,
// Unit, String, List, Tuple, Record). ADTs require a tag↔ctor-name registry
// that the bridge does not yet hold; passing an ADT across the boundary
// returns a clear error rather than silently mis-converting. Closures are
// likewise out of scope for M3 — the bridge will land them in M-BYTECODE-2E
// alongside effect-trap callbacks.
//
// Per M-BYTECODE-VM §11, this type lives in cmd/ailang (not in internal/vm)
// because the VM package must not import internal/eval. The vm.EvalInterop
// interface is the seam.
type bytecodeBridge struct {
	rt        *runtime.ModuleRuntime
	inst      *runtime.ModuleInstance
	evaluator *eval.CoreEvaluator
}

// newBytecodeBridge constructs a bridge bound to the given runtime, entry
// module instance, and evaluator. All three are required — the bridge
// returns errors rather than panicking if any is nil at call time.
func newBytecodeBridge(rt *runtime.ModuleRuntime, inst *runtime.ModuleInstance, evaluator *eval.CoreEvaluator) *bytecodeBridge {
	return &bytecodeBridge{rt: rt, inst: inst, evaluator: evaluator}
}

// CallEvalFunc resolves the named function in the entry module instance and
// dispatches it through the evaluator with bridged arguments. It implements
// vm.EvalInterop.
func (b *bytecodeBridge) CallEvalFunc(name string, args []bytecode.Value) (bytecode.Value, error) {
	if b == nil || b.inst == nil || b.evaluator == nil {
		return bytecode.Value{}, fmt.Errorf("bridge not fully initialized")
	}
	// Canonical names ("module/path.name") may refer to functions in a
	// different module than the entry instance. Resolve in this order:
	//   1. Try the entry module's binding map with the full name (legacy).
	//   2. If the name contains a "." split at the LAST dot and look up the
	//      function in the named module's instance via the runtime.
	//   3. Fall back to resolving the bare tail name in the entry instance.
	//
	// GetBinding (not GetExport) so we can call private functions too — the
	// EvalOnly stub may be a helper that is only reachable from inside the
	// module.
	fnVal, err := b.resolveFunc(name)
	if err != nil {
		return bytecode.Value{}, fmt.Errorf("resolve %q: %w", name, err)
	}

	bridgedArgs := make([]eval.Value, len(args))
	for i, a := range args {
		ev, convErr := bytecodeValueToEval(a)
		if convErr != nil {
			return bytecode.Value{}, fmt.Errorf("arg %d: %w", i, convErr)
		}
		bridgedArgs[i] = ev
	}

	result, err := b.evaluator.CallValueN(fnVal, bridgedArgs)
	if err != nil {
		return bytecode.Value{}, err
	}

	return evalValueToBytecode(result)
}

// resolveFunc walks the name resolution fallback chain described in
// CallEvalFunc. Returns an error if no module instance in the runtime holds
// the binding under any of the tried spellings.
func (b *bytecodeBridge) resolveFunc(name string) (eval.Value, error) {
	// 1. Entry instance, full name — handles the legacy bare-name case and
	//    any function already bound under its canonical form.
	if v, err := b.inst.GetBinding(name); err == nil {
		return v, nil
	}

	// 2. Canonical "module/path.name" → look up that module's instance.
	if dot := strings.LastIndex(name, "."); dot > 0 {
		modPath := name[:dot]
		bare := name[dot+1:]
		if b.rt != nil {
			if modInst := b.rt.GetInstance(modPath); modInst != nil {
				if v, err := modInst.GetBinding(bare); err == nil {
					return v, nil
				}
			}
		}
		// 3. Fall back to the bare tail name in the entry instance (e.g.
		//    cross-module helper registered bare during Phase 1 compat).
		if v, err := b.inst.GetBinding(bare); err == nil {
			return v, nil
		}
		return nil, fmt.Errorf("binding not found in module %q or entry instance", modPath)
	}

	return nil, fmt.Errorf("binding %q not found in entry instance", name)
}

// --- Value conversion -------------------------------------------------------

// bytecodeValueToEval converts a VM value to an evaluator value. Tier-1 only.
// Returns an error for shapes the bridge does not yet handle (Closure, ADT).
func bytecodeValueToEval(v bytecode.Value) (eval.Value, error) {
	switch v.Tag {
	case bytecode.TagInt:
		return &eval.IntValue{Value: int(v.Int)}, nil
	case bytecode.TagFloat:
		return &eval.FloatValue{Value: v.Flt}, nil
	case bytecode.TagBool:
		return &eval.BoolValue{Value: v.Bool}, nil
	case bytecode.TagUnit:
		return &eval.UnitValue{}, nil
	case bytecode.TagString:
		return &eval.StringValue{Value: v.AsString()}, nil
	case bytecode.TagList:
		src := v.AsList()
		dst := make([]eval.Value, len(src))
		for i, e := range src {
			ev, err := bytecodeValueToEval(e)
			if err != nil {
				return nil, fmt.Errorf("list[%d]: %w", i, err)
			}
			dst[i] = ev
		}
		return &eval.ListValue{Elements: dst}, nil
	case bytecode.TagTuple:
		src := v.AsTuple()
		dst := make([]eval.Value, len(src))
		for i, e := range src {
			ev, err := bytecodeValueToEval(e)
			if err != nil {
				return nil, fmt.Errorf("tuple[%d]: %w", i, err)
			}
			dst[i] = ev
		}
		return &eval.TupleValue{Elements: dst}, nil
	case bytecode.TagRecord:
		src := v.AsRecord()
		dst := make(map[string]eval.Value, len(src))
		for _, f := range src {
			ev, err := bytecodeValueToEval(f.Value)
			if err != nil {
				return nil, fmt.Errorf("record field %q: %w", f.Name, err)
			}
			dst[f.Name] = ev
		}
		return &eval.RecordValue{Fields: dst}, nil
	case bytecode.TagADT:
		return nil, fmt.Errorf("bridge: ADT values not yet supported (M-BYTECODE-2E scope)")
	case bytecode.TagClosure:
		return nil, fmt.Errorf("bridge: closure values not yet supported (M-BYTECODE-2E scope)")
	}
	return nil, fmt.Errorf("bridge: unknown bytecode tag %d", v.Tag)
}

// evalValueToBytecode converts an evaluator value to a VM value. Mirror of
// bytecodeValueToEval. Returns an error for shapes the bridge does not yet
// handle (functions, builtins, tagged ADTs, etc.).
func evalValueToBytecode(v eval.Value) (bytecode.Value, error) {
	if v == nil {
		return bytecode.Unit(), nil
	}
	switch ev := v.(type) {
	case *eval.IntValue:
		return bytecode.NewInt(int64(ev.Value)), nil
	case *eval.FloatValue:
		return bytecode.NewFloat(ev.Value), nil
	case *eval.BoolValue:
		return bytecode.NewBool(ev.Value), nil
	case *eval.UnitValue:
		return bytecode.Unit(), nil
	case *eval.StringValue:
		return bytecode.NewString(ev.Value), nil
	case *eval.ListValue:
		dst := make([]bytecode.Value, len(ev.Elements))
		for i, e := range ev.Elements {
			bv, err := evalValueToBytecode(e)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("list[%d]: %w", i, err)
			}
			dst[i] = bv
		}
		return bytecode.NewList(dst), nil
	case *eval.TupleValue:
		dst := make([]bytecode.Value, len(ev.Elements))
		for i, e := range ev.Elements {
			bv, err := evalValueToBytecode(e)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("tuple[%d]: %w", i, err)
			}
			dst[i] = bv
		}
		return bytecode.NewTuple(dst), nil
	case *eval.RecordValue:
		fields := make([]bytecode.RecordField, 0, len(ev.Fields))
		for name, val := range ev.Fields {
			bv, err := evalValueToBytecode(val)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("record field %q: %w", name, err)
			}
			fields = append(fields, bytecode.RecordField{Name: name, Value: bv})
		}
		// NewRecord sorts alphabetically — record-iteration order from
		// map[string]Value is non-deterministic, but the constructor handles it.
		return bytecode.NewRecord(fields), nil
	case *eval.TaggedValue:
		return bytecode.Value{}, fmt.Errorf("bridge: TaggedValue (%s.%s) not yet supported (M-BYTECODE-2E scope)", ev.TypeName, ev.CtorName)
	case *eval.FunctionValue, *eval.BuiltinFunction:
		return bytecode.Value{}, fmt.Errorf("bridge: function values not yet supported (M-BYTECODE-2E scope)")
	}
	return bytecode.Value{}, fmt.Errorf("bridge: unsupported eval value type %T", v)
}
