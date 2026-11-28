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

// VMTrapError is a coded VM runtime trap (e.g., RT_EQ_001). When wrapped by
// VM.errWrap, VMError.Error() prefixes "vm: " once without doubling.
type VMTrapError struct {
	Code    string
	Message string
}

func (e *VMTrapError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewVMError creates a coded runtime trap error for VM instructions.
func NewVMError(code, msg string) error {
	return &VMTrapError{Code: code, Message: msg}
}

func errClosureEq() error {
	return NewVMError("RT_EQ_001", "cannot compare function/closure values with == or !=")
}

// hasClosure reports whether v is or contains a TagClosure value.
// Primitives and strings/bytes return false in O(1).
func hasClosure(v bytecode.Value) bool {
	switch v.Tag {
	case bytecode.TagClosure:
		return true
	case bytecode.TagList:
		for _, el := range v.Obj.(*bytecode.ListObj).Elems {
			if hasClosure(el) {
				return true
			}
		}
	case bytecode.TagTuple:
		for _, el := range v.Obj.(*bytecode.TupleObj).Elems {
			if hasClosure(el) {
				return true
			}
		}
	case bytecode.TagArray:
		a := v.AsArray()
		if a.Floats != nil {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if hasClosure(a.At(i)) {
				return true
			}
		}
	case bytecode.TagRecord:
		for _, f := range v.Obj.(*bytecode.RecordObj).Fields {
			if hasClosure(f.Value) {
				return true
			}
		}
	case bytecode.TagADT:
		for _, f := range v.Obj.(*bytecode.ADTObj).Fields {
			if hasClosure(f) {
				return true
			}
		}
	case bytecode.TagMap:
		for _, entry := range v.AsMap().Entries {
			if hasClosure(entry.Key) || hasClosure(entry.Value) {
				return true
			}
		}
	}
	return false
}

// runtimeEqChecked is the equality used by OpEq (and OpNeq): IEEE float
// semantics (NaN != NaN) and a hard RT_EQ_001 runtime trap whenever either
// operand is or contains a closure (SOUND-1). bytecode.Value.Equal remains
// untouched for constant-pool deduplication.
func runtimeEqChecked(lhs, rhs bytecode.Value) (bool, error) {
	if lhs.Tag == bytecode.TagClosure || rhs.Tag == bytecode.TagClosure {
		return false, errClosureEq()
	}
	if lhs.Tag != rhs.Tag {
		if hasClosure(lhs) || hasClosure(rhs) {
			return false, errClosureEq()
		}
		return false, nil
	}
	switch lhs.Tag {
	case bytecode.TagFloat:
		return types.FloatEq(lhs.Flt, rhs.Flt), nil
	case bytecode.TagInt:
		return lhs.Int == rhs.Int, nil
	case bytecode.TagBool:
		return lhs.Bool == rhs.Bool, nil
	case bytecode.TagUnit:
		return true, nil
	case bytecode.TagString:
		return lhs.AsString() == rhs.AsString(), nil
	case bytecode.TagBytes:
		return string(lhs.AsBytes().B) == string(rhs.AsBytes().B), nil
	case bytecode.TagList:
		return runtimeEqAllChecked(lhs.Obj.(*bytecode.ListObj).Elems, rhs.Obj.(*bytecode.ListObj).Elems)
	case bytecode.TagTuple:
		return runtimeEqAllChecked(lhs.Obj.(*bytecode.TupleObj).Elems, rhs.Obj.(*bytecode.TupleObj).Elems)
	case bytecode.TagMap:
		m1, m2 := lhs.AsMap(), rhs.AsMap()
		if len(m1.Entries) != len(m2.Entries) {
			if hasClosure(lhs) || hasClosure(rhs) {
				return false, errClosureEq()
			}
			return false, nil
		}
		for k, v := range m1.Entries {
			o, ok := m2.Entries[k]
			if !ok {
				if hasClosure(lhs) || hasClosure(rhs) {
					return false, errClosureEq()
				}
				return false, nil
			}
			eq, err := runtimeEqChecked(v.Value, o.Value)
			if err != nil {
				return false, err
			}
			if !eq {
				if hasClosure(lhs) || hasClosure(rhs) {
					return false, errClosureEq()
				}
				return false, nil
			}
		}
		return true, nil
	case bytecode.TagArray:
		a, b := lhs.AsArray(), rhs.AsArray()
		if a.Len() != b.Len() {
			if hasClosure(lhs) || hasClosure(rhs) {
				return false, errClosureEq()
			}
			return false, nil
		}
		for i := 0; i < a.Len(); i++ {
			eq, err := runtimeEqChecked(a.At(i), b.At(i))
			if err != nil {
				return false, err
			}
			if !eq {
				for j := i + 1; j < a.Len(); j++ {
					if hasClosure(a.At(j)) || hasClosure(b.At(j)) {
						return false, errClosureEq()
					}
				}
				return false, nil
			}
		}
		return true, nil
	case bytecode.TagRecord:
		a, b := lhs.Obj.(*bytecode.RecordObj).Fields, rhs.Obj.(*bytecode.RecordObj).Fields
		if len(a) != len(b) {
			if hasClosure(lhs) || hasClosure(rhs) {
				return false, errClosureEq()
			}
			return false, nil
		}
		for i := range a {
			if a[i].Name != b[i].Name {
				if hasClosure(lhs) || hasClosure(rhs) {
					return false, errClosureEq()
				}
				return false, nil
			}
			eq, err := runtimeEqChecked(a[i].Value, b[i].Value)
			if err != nil {
				return false, err
			}
			if !eq {
				for j := i + 1; j < len(a); j++ {
					if hasClosure(a[j].Value) || hasClosure(b[j].Value) {
						return false, errClosureEq()
					}
				}
				return false, nil
			}
		}
		return true, nil
	case bytecode.TagADT:
		a, b := lhs.Obj.(*bytecode.ADTObj), rhs.Obj.(*bytecode.ADTObj)
		if a.Tag != b.Tag {
			if hasClosure(lhs) || hasClosure(rhs) {
				return false, errClosureEq()
			}
			return false, nil
		}
		return runtimeEqAllChecked(a.Fields, b.Fields)
	}
	return lhs.Equal(rhs), nil
}

func runtimeEqAllChecked(a, b []bytecode.Value) (bool, error) {
	if len(a) != len(b) {
		for _, v := range a {
			if hasClosure(v) {
				return false, errClosureEq()
			}
		}
		for _, v := range b {
			if hasClosure(v) {
				return false, errClosureEq()
			}
		}
		return false, nil
	}
	for i := range a {
		eq, err := runtimeEqChecked(a[i], b[i])
		if err != nil {
			return false, err
		}
		if !eq {
			for j := i + 1; j < len(a); j++ {
				if hasClosure(a[j]) || hasClosure(b[j]) {
					return false, errClosureEq()
				}
			}
			return false, nil
		}
	}
	return true, nil
}

// runtimeEq is a convenience boolean wrapper over runtimeEqChecked for internal
// test assertions on non-closure values.
func runtimeEq(lhs, rhs bytecode.Value) bool {
	eq, err := runtimeEqChecked(lhs, rhs)
	return err == nil && eq
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
