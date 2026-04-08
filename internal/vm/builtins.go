package vm

import (
	"fmt"
	"strconv"

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
// evaluator's `_show` semantics for the value subset the corpus produces.
func builtinShow(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("_show: expected 1 arg, got %d", len(args))
	}
	v := args[0]
	switch v.Tag {
	case bytecode.TagInt:
		return bytecode.NewString(strconv.FormatInt(v.Int, 10)), nil
	case bytecode.TagFloat:
		return bytecode.NewString(strconv.FormatFloat(v.Flt, 'g', -1, 64)), nil
	case bytecode.TagBool:
		if v.Bool {
			return bytecode.NewString("true"), nil
		}
		return bytecode.NewString("false"), nil
	case bytecode.TagString:
		return v, nil
	case bytecode.TagUnit:
		return bytecode.NewString("()"), nil
	default:
		return bytecode.NewString(fmt.Sprintf("<%s>", v.Tag)), nil
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
