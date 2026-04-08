package vm

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo/ailang/internal/bytecode"
)

// BuiltinFunc is a pure-builtin handler. It receives the argument slice
// (read directly from the caller's register window) and returns a result
// value or an error.
type BuiltinFunc func(args []bytecode.Value) (bytecode.Value, error)

// BuiltinTable is the VM-side dispatch table for OpBuiltinCall. The order
// MUST match compiler.BuiltinTable — both lists are validated at startup
// by validateBuiltinTables (called from the VM constructor in tests).
//
// Phase 2C scope: only the pure builtins reachable from the golden corpus
// (`tests/golden/codegen/`). All other builtins lower to OpBuiltinTrap and
// will be wired through the evaluator in Phase 2E.
var BuiltinTable = []BuiltinFunc{
	builtinShow,         // _show
	builtinLen,          // _len
	builtinListGet,      // _list_get
	builtinListTail,     // _list_tail
	builtinConcatString, // _concat_String
}

// builtinShow returns a string representation of any value. Matches the
// evaluator's `_show` semantics (see internal/builtins/show.go:showValue)
// for every Value shape the VM can produce.
//
// M-BYTECODE-MULTIMODULE M1 exposed a pre-existing gap: compound shapes
// (List/Tuple/Record/ADT) fell through to a "<TagName>" fallback. This
// was masked before M1 because stdlib call sites were never lowered to
// the VM path and went through the eval bridge instead. With M1 lowering
// all reachable modules, this builtin must fully mirror the evaluator.
func builtinShow(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("_show: expected 1 arg, got %d", len(args))
	}
	return bytecode.NewString(showValue(args[0])), nil
}

// showValue is the recursive counterpart of internal/builtins/show.go:showValue
// for bytecode.Value. Strings are rendered without quotes to match the
// evaluator's user-facing `show` semantics (identity on strings).
func showValue(v bytecode.Value) string {
	switch v.Tag {
	case bytecode.TagInt:
		return strconv.FormatInt(v.Int, 10)
	case bytecode.TagFloat:
		// Match evaluator's show: use 'f' so the decimal point is visible,
		// and pad whole-number floats to `N.0` (e.g. 5 → "5.0"). See
		// internal/builtins/show.go:showValue float case.
		f := v.Flt
		if f != f { // NaN
			return "NaN"
		}
		if f > 0 && f*2 == f { // +Inf
			return "Inf"
		}
		if f < 0 && f*2 == f { // -Inf
			return "-Inf"
		}
		s := strconv.FormatFloat(f, 'f', -1, 64)
		if !strings.Contains(s, ".") {
			s += ".0"
		}
		return s
	case bytecode.TagBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case bytecode.TagString:
		return v.AsString()
	case bytecode.TagUnit:
		return "()"
	case bytecode.TagList:
		elems := v.AsList()
		if len(elems) == 0 {
			return "[]"
		}
		parts := make([]string, len(elems))
		for i, e := range elems {
			parts[i] = showValue(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case bytecode.TagTuple:
		elems := v.AsTuple()
		parts := make([]string, len(elems))
		for i, e := range elems {
			parts[i] = showValue(e)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case bytecode.TagRecord:
		fields := v.AsRecord()
		if len(fields) == 0 {
			return "{}"
		}
		// Sort field names for deterministic output, matching eval.
		names := make([]string, len(fields))
		vals := make(map[string]bytecode.Value, len(fields))
		for i, f := range fields {
			names[i] = f.Name
			vals[f.Name] = f.Value
		}
		sort.Strings(names)
		parts := make([]string, len(names))
		for i, n := range names {
			parts[i] = n + ": " + showValue(vals[n])
		}
		return "{" + strings.Join(parts, ", ") + "}"
	case bytecode.TagADT:
		// ADT tag is a per-type ordinal, not a constructor name — mapping
		// back to a name requires the compiler's type table (§4.3), which
		// the VM does not currently carry. Defer to Value.String() for the
		// low-fidelity `<adt#N …>` rendering. Fixing this cleanly is M3
		// scope (cross-module ADT/record merging).
		return v.String()
	default:
		return fmt.Sprintf("<%s>", v.Tag)
	}
}

// builtinLen returns the length of a list, tuple, string, or record.
func builtinLen(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("_len: expected 1 arg, got %d", len(args))
	}
	v := args[0]
	switch v.Tag {
	case bytecode.TagList:
		return bytecode.NewInt(int64(len(v.AsList()))), nil
	case bytecode.TagTuple:
		return bytecode.NewInt(int64(len(v.AsTuple()))), nil
	case bytecode.TagString:
		return bytecode.NewInt(int64(len(v.AsString()))), nil
	default:
		return bytecode.Value{}, fmt.Errorf("_len: unsupported tag %s", v.Tag)
	}
}

// builtinListGet returns the element at index i in a list.
func builtinListGet(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("_list_get: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("_list_get: arg 0 must be list, got %s", args[0].Tag)
	}
	if args[1].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("_list_get: arg 1 must be int, got %s", args[1].Tag)
	}
	elems := args[0].AsList()
	i := int(args[1].Int)
	if i < 0 || i >= len(elems) {
		return bytecode.Value{}, fmt.Errorf("_list_get: index %d out of range [0,%d)", i, len(elems))
	}
	return elems[i], nil
}

// builtinConcatString concatenates two strings. The lower pass intercepts
// stdlib calls to `$builtin.concat_String` and routes them through this
// dispatch entry; see internal/gen/lower/expr.go:lowerApp.
func builtinConcatString(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("_concat_String: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("_concat_String: arg 0 must be string, got %s", args[0].Tag)
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("_concat_String: arg 1 must be string, got %s", args[1].Tag)
	}
	return bytecode.NewString(args[0].AsString() + args[1].AsString()), nil
}

// builtinListTail returns the suffix of a list starting at index n.
// Used by the lower pass to bind `tail` in `head :: tail` patterns.
func builtinListTail(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("_list_tail: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("_list_tail: arg 0 must be list, got %s", args[0].Tag)
	}
	if args[1].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("_list_tail: arg 1 must be int, got %s", args[1].Tag)
	}
	elems := args[0].AsList()
	n := int(args[1].Int)
	if n < 0 {
		n = 0
	}
	if n > len(elems) {
		n = len(elems)
	}
	tail := make([]bytecode.Value, len(elems)-n)
	copy(tail, elems[n:])
	return bytecode.NewList(tail), nil
}
