package vm

import (
	"fmt"
	"strconv"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
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
// The order MUST match bytecode.HOFBuiltinNames — both lists are
// validated at package init by validateBuiltinTables (builtins_adapted.go).
var HOFBuiltinTable = []HOFBuiltinFunc{
	hofBuiltinListMap,          // __list_map
	hofBuiltinListFilter,       // __list_filter
	hofBuiltinListFoldl,        // __list_foldl
	hofBuiltinStrFoldChars,     // __str_foldChars
	hofBuiltinStrFoldSlices,    // __str_foldSlices
	hofBuiltinStrMapSlicesJoin, // __str_mapSlicesJoin
	hofBuiltinXmlParseFold,     // __xml_parseFold
	hofBuiltinListSortBy,       // __list_sortBy
	hofBuiltinListFlatMap,      // __list_flatMap
}

// BuiltinTable is the VM-side dispatch table for OpBuiltinCall. The order
// MUST match bytecode.BuiltinNames — both lists are validated at startup
// by validateBuiltinTables at package init (builtins_adapted.go).
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
	// M-BYTECODE-PURE-EFFECTS M1: JSON builtins
	builtinJsonEncode, // __json_encode
	builtinJsonDecode, // __json_decode
	builtinJsonRepair, // __json_repair
	// M-BYTECODE-XML-BUILTINS: XML builtins
	builtinXmlElement,           // __xmlElement
	builtinXmlText,              // __xmlText
	builtinXmlComment,           // __xmlComment
	builtinXmlGetText,           // __xml_getText
	builtinXmlGetTag,            // __xml_getTag
	builtinXmlSerialize,         // __xml_serialize
	builtinXmlSerializeWithDecl, // __xml_serializeWithDecl
	builtinXmlParse,             // __xml_parse
	builtinXmlParseElements,     // __xml_parseElements
	builtinXmlParseWithLimit,    // __xml_parseWithLimit
	builtinXmlFindAll,           // __xml_findAll
	builtinXmlFindFirst,         // __xml_findFirst
	builtinXmlGetAttr,           // __xml_getAttr
	builtinXmlGetChildren,       // __xml_getChildren
	builtinXmlFindAllTexts,      // __xml_findAllTexts
	builtinXmlFindAllAttrs,      // __xml_findAllAttrs
	// ailang#1354/#1355: by-name record update for bases of unknown type
	builtinRecordSet, // _record_set
	// M-VM-PURE-BUILTIN-COVERAGE M3 (#1447): polymorphic list ops
	builtinListReverse,  // __list_reverse
	builtinListTake,     // __list_take
	builtinListDrop,     // __list_drop
	builtinListZip,      // __list_zip
	builtinListContains, // __list_contains
	builtinListHead,     // __list_head
	builtinListExtract,  // __list_extract
	// polymorphic std/array ops (stapledons_godot 2026-10-01)
	builtinArrayEmpty,      // __array_empty
	builtinArrayMake,       // __array_make
	builtinArrayGet,        // __array_get
	builtinArrayUnsafeGet,  // __array_unsafe_get
	builtinArrayLength,     // __array_length
	builtinArraySet,        // __array_set
	builtinArrayFromList,   // __array_from_list
	builtinArrayToList,     // __array_to_list
	builtinArrayAppend,     // __array_append
	builtinArrayUpdateMany, // __array_update_many
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

// builtinRecordSet returns a copy of a record with the named field set to a
// new value, adding the field when absent — the evaluator's record-update
// semantics (evalCoreRecordUpdate). The compiler emits it for `{r | f: v}`
// when r's static type is unknown, instead of guessing r's shape from some
// other registered type (ailang#1354, #1355).
func builtinRecordSet(args []bytecode.Value) (bytecode.Value, error) {
	if len(args) != 3 {
		return bytecode.Value{}, fmt.Errorf("_record_set: expected 3 args, got %d", len(args))
	}
	if args[0].Tag != bytecode.TagRecord {
		return bytecode.Value{}, fmt.Errorf("_record_set: arg 0 must be record, got %s", args[0].Tag)
	}
	if args[1].Tag != bytecode.TagString {
		return bytecode.Value{}, fmt.Errorf("_record_set: arg 1 must be string, got %s", args[1].Tag)
	}
	name := args[1].AsString()
	old := args[0].AsRecord()
	fields := make([]bytecode.RecordField, 0, len(old)+1)
	replaced := false
	for _, f := range old {
		if f.Name == name {
			f.Value = args[2]
			replaced = true
		}
		fields = append(fields, f)
	}
	if !replaced {
		fields = append(fields, bytecode.RecordField{Name: name, Value: args[2]})
	}
	return bytecode.NewRecord(fields), nil
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

// showValue renders a VM value through the evaluator's show renderer
// (eval.RenderShow), so `show` is byte-identical on both engines (#1453).
func showValue(v bytecode.Value) string {
	return eval.RenderShow(v, inspectVMShow)
}

func vmItems(vs []bytecode.Value) []any {
	out := make([]any, len(vs))
	for i, x := range vs {
		out[i] = x
	}
	return out
}

// inspectVMShow describes a VM value for eval.RenderShow.
func inspectVMShow(x any) eval.ShowNode {
	v := x.(bytecode.Value)
	switch v.Tag {
	case bytecode.TagInt:
		return eval.ShowNode{Text: strconv.FormatInt(v.Int, 10)}
	case bytecode.TagFloat:
		return eval.ShowNode{Kind: eval.ShowFloat, Float: v.Flt}
	case bytecode.TagBool:
		return eval.ShowNode{Text: strconv.FormatBool(v.Bool)}
	case bytecode.TagString:
		return eval.ShowNode{Text: v.AsString()}
	case bytecode.TagUnit:
		return eval.ShowNode{Text: "()"}
	case bytecode.TagList:
		return eval.ShowNode{Kind: eval.ShowList, Items: vmItems(v.AsList())}
	case bytecode.TagArray:
		a := v.AsArray()
		items := make([]any, a.Len())
		for i := range items {
			items[i] = a.At(i)
		}
		return eval.ShowNode{Kind: eval.ShowArray, Items: items}
	case bytecode.TagTuple:
		return eval.ShowNode{Kind: eval.ShowTuple, Items: vmItems(v.AsTuple())}
	case bytecode.TagRecord:
		fields := v.AsRecord()
		names := make([]string, len(fields))
		items := make([]any, len(fields))
		for i, f := range fields {
			names[i], items[i] = f.Name, f.Value
		}
		return eval.ShowNode{Kind: eval.ShowRecord, Names: names, Items: items}
	case bytecode.TagBytes:
		b := v.AsBytes()
		return eval.ShowNode{Text: (&eval.BytesValue{Value: b.B, Filename: b.Filename, MimeType: b.MimeType}).String()}
	case bytecode.TagADT:
		a := v.AsADT()
		if a.Ctor == "" {
			// Every ADT the VM builds carries its name; an unnamed one is a
			// construction-site bug, so show it as such rather than guess.
			return eval.ShowNode{Text: v.String()}
		}
		return eval.ShowNode{Kind: eval.ShowCtor, Text: a.Ctor, Items: vmItems(a.Fields)}
	case bytecode.TagClosure:
		return eval.ShowNode{Text: "<function>"}
	}
	return eval.ShowNode{Text: fmt.Sprintf("<%s>", v.Tag)}
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
