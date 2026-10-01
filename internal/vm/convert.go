package vm

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Value conversion between the VM and the evaluator. Used by the hybrid
// evaluator bridge (internal/runner) and by adapted pure builtins
// (builtins_adapted.go, #1447) — one implementation for both.

// BytecodeToEval converts a VM value to an evaluator value. Tier-1 only.
// Returns an error for shapes the bridge does not yet handle (Closure, ADT).
func BytecodeToEval(v bytecode.Value) (eval.Value, error) {
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
			ev, err := BytecodeToEval(e)
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
			ev, err := BytecodeToEval(e)
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
			ev, err := BytecodeToEval(f.Value)
			if err != nil {
				return nil, fmt.Errorf("record field %q: %w", f.Name, err)
			}
			dst[f.Name] = ev
		}
		return &eval.RecordValue{Fields: dst}, nil
	case bytecode.TagArray:
		a := v.AsArray()
		if a.Floats != nil {
			// Shared, not copied: both sides treat the slice as immutable.
			return eval.NewFloatArray(a.Floats), nil
		}
		dst := make([]eval.Value, len(a.Elems))
		for i, e := range a.Elems {
			ev, err := BytecodeToEval(e)
			if err != nil {
				return nil, fmt.Errorf("array[%d]: %w", i, err)
			}
			dst[i] = ev
		}
		return eval.NewArray(dst), nil
	case bytecode.TagBytes:
		b := v.AsBytes()
		return &eval.BytesValue{Value: b.B, Filename: b.Filename, MimeType: b.MimeType}, nil
	case bytecode.TagADT:
		// A VM ADT carries only a constructor ordinal, not its type, so it
		// cannot be named on the evaluator side (M-BYTECODE-2E scope).
		return nil, fmt.Errorf("bridge: ADT values not yet supported (M-BYTECODE-2E scope)")
	case bytecode.TagClosure:
		return nil, fmt.Errorf("bridge: closure values not yet supported (M-BYTECODE-2E scope)")
	}
	return nil, fmt.Errorf("bridge: unknown bytecode tag %d", v.Tag)
}

// EvalToBytecode converts an evaluator value to a VM value. Mirror of
// BytecodeToEval. Returns an error for shapes the bridge does not yet
// handle (functions, builtins, tagged ADTs, etc.).
func EvalToBytecode(v eval.Value) (bytecode.Value, error) {
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
			bv, err := EvalToBytecode(e)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("list[%d]: %w", i, err)
			}
			dst[i] = bv
		}
		return bytecode.NewList(dst), nil
	case *eval.TupleValue:
		dst := make([]bytecode.Value, len(ev.Elements))
		for i, e := range ev.Elements {
			bv, err := EvalToBytecode(e)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("tuple[%d]: %w", i, err)
			}
			dst[i] = bv
		}
		return bytecode.NewTuple(dst), nil
	case *eval.ArrayValue:
		if xs, packed := ev.Floats(); packed {
			// O(1): the packed store crosses the bridge without a copy.
			return bytecode.NewFloatArray(xs), nil
		}
		src := ev.Elements()
		dst := make([]bytecode.Value, len(src))
		for i, e := range src {
			bv, err := EvalToBytecode(e)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("array[%d]: %w", i, err)
			}
			dst[i] = bv
		}
		return bytecode.NewArray(dst), nil
	case *eval.RecordValue:
		fields := make([]bytecode.RecordField, 0, len(ev.Fields))
		for name, val := range ev.Fields {
			bv, err := EvalToBytecode(val)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("record field %q: %w", name, err)
			}
			fields = append(fields, bytecode.RecordField{Name: name, Value: bv})
		}
		// NewRecord sorts alphabetically — record-iteration order from
		// map[string]Value is non-deterministic, but the constructor handles it.
		return bytecode.NewRecord(fields), nil
	case *eval.BytesValue:
		return bytecode.NewBytes(&bytecode.BytesObj{B: ev.Value, Filename: ev.Filename, MimeType: ev.MimeType}), nil
	case *eval.TaggedValue:
		// Option and Result from an evaluator call (e.g. a codec returning
		// Result[Array[float], string]) map to the VM's fixed ordinals. Other
		// ADTs need the compiler's type table (M-BYTECODE-2E scope).
		tag, ok := bytecode.StdADTTag(ev.ModulePath, ev.TypeName, ev.CtorName)
		if !ok {
			return bytecode.Value{}, fmt.Errorf("bridge: TaggedValue (%s.%s) not yet supported (M-BYTECODE-2E scope)", ev.TypeName, ev.CtorName)
		}
		fields := make([]bytecode.Value, len(ev.Fields))
		for i, f := range ev.Fields {
			bv, err := EvalToBytecode(f)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("%s field %d: %w", ev.CtorName, i, err)
			}
			fields[i] = bv
		}
		return bytecode.NewADT(tag, fields), nil
	case *eval.FunctionValue, *eval.BuiltinFunction:
		return bytecode.Value{}, fmt.Errorf("bridge: function values not yet supported (M-BYTECODE-2E scope)")
	}
	return bytecode.Value{}, fmt.Errorf("bridge: unsupported eval value type %T", v)
}
