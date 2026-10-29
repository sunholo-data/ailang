package vm

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/types"
)

// Arithmetic, equality and ordering helpers for the dispatch loop.

// arith dispatches a binary arithmetic operation by tag. Both operands must
// have the same numeric tag (no implicit int↔float coercion in Phase 2B).
func arith(op bytecode.OpCode, lhs, rhs bytecode.Value) (bytecode.Value, error) {
	if lhs.Tag != rhs.Tag {
		return bytecode.Value{}, fmt.Errorf("arith %s: type mismatch %s vs %s", op, lhs.Tag, rhs.Tag)
	}
	switch lhs.Tag {
	case bytecode.TagInt:
		l, r := lhs.Int, rhs.Int
		switch op {
		case bytecode.OpAdd:
			return bytecode.NewInt(l + r), nil
		case bytecode.OpSub:
			return bytecode.NewInt(l - r), nil
		case bytecode.OpMul:
			return bytecode.NewInt(l * r), nil
		case bytecode.OpDiv:
			if err := ailerrors.CheckIntDivisor(ailerrors.OpDivision, r); err != nil {
				return bytecode.Value{}, err
			}
			return bytecode.NewInt(l / r), nil
		case bytecode.OpMod:
			if err := ailerrors.CheckIntDivisor(ailerrors.OpModulo, r); err != nil {
				return bytecode.Value{}, err
			}
			return bytecode.NewInt(l % r), nil
		}
	case bytecode.TagFloat:
		l, r := lhs.Flt, rhs.Flt
		switch op {
		case bytecode.OpAdd:
			return bytecode.NewFloat(l + r), nil
		case bytecode.OpSub:
			return bytecode.NewFloat(l - r), nil
		case bytecode.OpMul:
			return bytecode.NewFloat(l * r), nil
		case bytecode.OpDiv:
			return bytecode.NewFloat(l / r), nil
		case bytecode.OpMod:
			return bytecode.Value{}, fmt.Errorf("MOD on Float not supported")
		}
	}
	return bytecode.Value{}, fmt.Errorf("arith %s on %s not supported", op, lhs.Tag)
}

// runtimeEq is the equality used by OpEq: types.FloatEq (IEEE, NaN != NaN)
// for floats wherever they sit, bare or nested in a list, tuple, record or
// ADT (M-FLOAT-EQ-ONE-SEMANTICS, #1274). It used to fall through to
// Value.Equal for composites, whose NaN == NaN is for constant-pool dedup,
// so [nan] == [nan] was true on the VM while nan == nan was false.
// Cycle-safety: values are finite trees built at runtime.
func runtimeEq(lhs, rhs bytecode.Value) bool {
	if lhs.Tag != rhs.Tag {
		return false
	}
	switch lhs.Tag {
	case bytecode.TagFloat:
		return types.FloatEq(lhs.Flt, rhs.Flt)
	case bytecode.TagInt:
		return lhs.Int == rhs.Int
	case bytecode.TagBool:
		return lhs.Bool == rhs.Bool
	case bytecode.TagUnit:
		return true
	case bytecode.TagString:
		return lhs.AsString() == rhs.AsString()
	case bytecode.TagList:
		return runtimeEqAll(lhs.Obj.(*bytecode.ListObj).Elems, rhs.Obj.(*bytecode.ListObj).Elems)
	case bytecode.TagTuple:
		return runtimeEqAll(lhs.Obj.(*bytecode.TupleObj).Elems, rhs.Obj.(*bytecode.TupleObj).Elems)
	case bytecode.TagMap:
		return lhs.AsMap().Equal(rhs.AsMap(), runtimeEq)
	case bytecode.TagBytes:
		return string(lhs.AsBytes().B) == string(rhs.AsBytes().B)
	case bytecode.TagArray:
		a, b := lhs.AsArray(), rhs.AsArray()
		if a.Len() != b.Len() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !runtimeEq(a.At(i), b.At(i)) {
				return false
			}
		}
		return true
	case bytecode.TagRecord:
		a, b := lhs.Obj.(*bytecode.RecordObj).Fields, rhs.Obj.(*bytecode.RecordObj).Fields
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i].Name != b[i].Name || !runtimeEq(a[i].Value, b[i].Value) {
				return false
			}
		}
		return true
	case bytecode.TagADT:
		a, b := lhs.Obj.(*bytecode.ADTObj), rhs.Obj.(*bytecode.ADTObj)
		return a.Tag == b.Tag && runtimeEqAll(a.Fields, b.Fields)
	}
	// Closures: the type checker rejects Eq on functions; keep the old answer.
	return lhs.Equal(rhs)
}

func runtimeEqAll(a, b []bytecode.Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !runtimeEq(a[i], b[i]) {
			return false
		}
	}
	return true
}

// compare implements LT and LE for ordered numeric and string types.
func compare(op bytecode.OpCode, lhs, rhs bytecode.Value) (bool, error) {
	if lhs.Tag != rhs.Tag {
		return false, fmt.Errorf("compare %s: type mismatch %s vs %s", op, lhs.Tag, rhs.Tag)
	}
	switch lhs.Tag {
	case bytecode.TagInt:
		switch op {
		case bytecode.OpLt:
			return lhs.Int < rhs.Int, nil
		case bytecode.OpLe:
			return lhs.Int <= rhs.Int, nil
		}
	case bytecode.TagFloat:
		switch op {
		case bytecode.OpLt:
			return types.FloatLt(lhs.Flt, rhs.Flt), nil
		case bytecode.OpLe:
			return types.FloatLte(lhs.Flt, rhs.Flt), nil
		}
	case bytecode.TagString:
		switch op {
		case bytecode.OpLt:
			return lhs.AsString() < rhs.AsString(), nil
		case bytecode.OpLe:
			return lhs.AsString() <= rhs.AsString(), nil
		}
	}
	return false, fmt.Errorf("compare %s on %s not supported", op, lhs.Tag)
}
