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

// ClosureCaller is the minimal interface a HOF builtin needs to invoke
// closure arguments. Implemented by *VM. Keeps HOF builtins decoupled
// from VM internals.
type ClosureCaller interface {
	CallClosure(closure bytecode.Value, args []bytecode.Value) (bytecode.Value, error)
}

// HOFBuiltinFunc is a higher-order builtin that can call VM closures.
// Used by OpBuiltinCallHOF dispatch.
type HOFBuiltinFunc func(caller ClosureCaller, args []bytecode.Value) (bytecode.Value, error)

// HOFBuiltinTable is the VM-side dispatch table for OpBuiltinCallHOF.
// The order MUST match compiler.HOFBuiltinTable — both lists are
// validated at startup by validateBuiltinTables.
var HOFBuiltinTable = []HOFBuiltinFunc{
	hofBuiltinListMap,          // __list_map
	hofBuiltinListFilter,       // __list_filter
	hofBuiltinListFoldl,        // __list_foldl
	hofBuiltinStrFoldChars,     // __str_foldChars
	hofBuiltinStrFoldSlices,    // __str_foldSlices
	hofBuiltinStrMapSlicesJoin, // __str_mapSlicesJoin
}

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
	builtinRecordGet,    // _record_get
	builtinNotBool,      // _not_Bool
	builtinIntToFloat,   // _intToFloat
	builtinListLength,   // __list_length
	builtinConcatList,   // _concat_List
	// M-BYTECODE-STDLIB-BUILTINS M1: string builtins
	builtinStrLen,           // __str_len
	builtinStrCompare,       // __str_compare
	builtinStrEq,            // __str_eq
	builtinStrFind,          // __str_find
	builtinStrSlice,         // __str_slice
	builtinStrTrim,          // __str_trim
	builtinStrUpper,         // __str_upper
	builtinStrLower,         // __str_lower
	builtinStrSplit,         // __str_split
	builtinStrChars,         // __str_chars
	builtinStrStartsWith,    // __str_startsWith
	builtinStrEndsWith,      // __str_endsWith
	builtinStrJoin,          // __str_join
	builtinStrWords,         // __str_words
	builtinStrSplitAny,      // __str_splitAny
	builtinStrReplace,       // __str_replace
	builtinStrReplaceMany,   // __str_replaceMany
	builtinStrStartsWithIC,  // __str_startsWithIC
	builtinStrCharAt,        // __str_charAt
	builtinStrCharCode,      // __str_charCode
	builtinStrDecodeQP,      // __str_decodeQP
	builtinEscapeXml,        // __escapeXml
	builtinStringIntToStr,   // __string_intToStr
	builtinStringFloatToStr, // __string_floatToStr
	builtinStringToInt,      // __stringToInt
	builtinStringToFloat,    // __stringToFloat
	// M-BYTECODE-STDLIB-BUILTINS M2: math + conversion builtins
	builtinMathSin,      // __math_sin
	builtinMathCos,      // __math_cos
	builtinMathTan,      // __math_tan
	builtinMathAsin,     // __math_asin
	builtinMathAcos,     // __math_acos
	builtinMathAtan,     // __math_atan
	builtinMathAtan2,    // __math_atan2
	builtinMathSqrt,     // __math_sqrt
	builtinMathPow,      // __math_pow
	builtinMathExp,      // __math_exp
	builtinMathLog,      // __math_log
	builtinMathLog10,    // __math_log10
	builtinMathFloor,    // __math_floor
	builtinMathCeil,     // __math_ceil
	builtinMathRound,    // __math_round
	builtinMathAbsFloat, // __math_abs_Float
	builtinMathAbsInt,   // __math_abs_Int
	builtinMathPI,       // __math_PI
	builtinMathE,        // __math_E
	builtinFloatToInt,   // _floatToInt
	builtinModInt,       // _mod_Int
	builtinFloatToInt2,  // __float_to_int
	builtinIntToFloat2,  // __int_to_float
	// M-BYTECODE-STDLIB-BUILTINS M3: list builtins
	builtinListNth,        // __list_nth
	builtinListMember,     // __list_member
	builtinListDedup,      // __list_dedup
	builtinListDifference, // __list_difference
	builtinListIntersect,  // __list_intersect
	builtinListUnion,      // __list_union
}

// builtinRecordGet returns the value of the named field in a record. Used as
// a fallback for FieldAccess when the bytecode compiler could not resolve
// the field's static index at compile time (e.g. row-polymorphic records
// whose full field set wasn't known). The record carries its field names
// at runtime, so a linear scan gives us the answer.
//
// Added for M-BYTECODE-MULTIMODULE M3.
func builtinRecordGet(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("_record_get: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagRecord {
		return bytecode.Value{}, fmt.Errorf("_record_get: arg 0 must be record, got %s", args[0].Tag)
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("_record_get: arg 1 must be string, got %s", args[1].Tag)
	}
	name := args[1].AsString()
	for _, f := range args[0].AsRecord() {
		if f.Name == name {
			return f.Value, nil
		}
	}
	return bytecode.Value{}, fmt.Errorf("_record_get: field %q not found", name)
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

// builtinNotBool returns the boolean negation of its argument.
func builtinNotBool(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("_not_Bool: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagBool {
		return bytecode.Value{}, fmt.Errorf("_not_Bool: arg must be bool, got %s", args[0].Tag)
	}
	return bytecode.NewBool(!args[0].Bool), nil
}

// builtinIntToFloat converts an integer to a float.
func builtinIntToFloat(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("_intToFloat: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagInt {
		return bytecode.Value{}, fmt.Errorf("_intToFloat: arg must be int, got %s", args[0].Tag)
	}
	return bytecode.NewFloat(float64(args[0].Int)), nil
}

// builtinListLength returns the length of a list as an integer.
// This is the stdlib alias `__list_length`; distinct from `_len` which
// also handles tuples and strings.
func builtinListLength(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 1 {
		return bytecode.Value{}, fmt.Errorf("__list_length: expected 1 arg, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("__list_length: arg must be list, got %s", args[0].Tag)
	}
	return bytecode.NewInt(int64(len(args[0].AsList()))), nil
}

// builtinConcatList concatenates two lists into a new list.
func builtinConcatList(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 2 {
		return bytecode.Value{}, fmt.Errorf("_concat_List: expected 2 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("_concat_List: arg 0 must be list, got %s", args[0].Tag)
	}
	if args[1].Tag != bytecode.TagList {
		return bytecode.Value{}, fmt.Errorf("_concat_List: arg 1 must be list, got %s", args[1].Tag)
	}
	a := args[0].AsList()
	b := args[1].AsList()
	result := make([]bytecode.Value, len(a)+len(b))
	copy(result, a)
	copy(result[len(a):], b)
	return bytecode.NewList(result), nil
}
