package compiler

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
)

// BuiltinTable is the canonical, source-ordered table of pure builtins
// reachable from the Phase 2C golden corpus. Indices in this table become
// the B field of OpBuiltinCall and are interpreted by the VM's builtin
// dispatch table (internal/vm/builtins.go).
//
// Adding a builtin is a two-step process:
//  1. Append its name here.
//  2. Add a matching entry in vm.BuiltinTable.
//
// The two tables MUST stay in lockstep — there is a runtime sanity check
// at VM startup that the lengths agree.
var BuiltinTable = []string{
	"_show",
	"_len",
	"_list_get",
	"_list_tail",
	"_concat_String",
	"_record_get",
	"_not_Bool",
	"_intToFloat",
	"__list_length",
	"_concat_List",
	// M-BYTECODE-STDLIB-BUILTINS M1: string builtins
	"__str_len",
	"__str_compare",
	"__str_eq",
	"__str_find",
	"__str_slice",
	"__str_trim",
	"__str_upper",
	"__str_lower",
	"__str_split",
	"__str_chars",
	"__str_startsWith",
	"__str_endsWith",
	"__str_join",
	"__str_words",
	"__str_splitAny",
	"__str_replace",
	"__str_replaceMany",
	"__str_startsWithIC",
	"__str_charAt",
	"__str_charCode",
	"__str_decodeQP",
	"__escapeXml",
	"__string_intToStr",
	"__string_floatToStr",
	"__stringToInt",
	"__stringToFloat",
	// M-BYTECODE-STDLIB-BUILTINS M2: math + conversion builtins
	"__math_sin",
	"__math_cos",
	"__math_tan",
	"__math_asin",
	"__math_acos",
	"__math_atan",
	"__math_atan2",
	"__math_sqrt",
	"__math_pow",
	"__math_exp",
	"__math_log",
	"__math_log10",
	"__math_floor",
	"__math_ceil",
	"__math_round",
	"__math_abs_Float",
	"__math_abs_Int",
	"__math_PI",
	"__math_E",
	"_floatToInt",
	"_mod_Int",
	"__float_to_int",
	"__int_to_float",
	// M-BYTECODE-STDLIB-BUILTINS M3: list builtins
	"__list_nth",
	"__list_member",
	"__list_dedup",
	"__list_difference",
	"__list_intersect",
	"__list_union",
}

var builtinIndex = func() map[string]uint8 {
	m := make(map[string]uint8, len(BuiltinTable))
	for i, name := range BuiltinTable {
		m[name] = uint8(i)
	}
	return m
}()

// HOFBuiltinTable lists builtins that take closure arguments. These are
// dispatched via OpBuiltinCallHOF, which passes the VM as a ClosureCaller
// so the builtin can invoke its closure arguments.
// Order MUST match vm.HOFBuiltinTable.
var HOFBuiltinTable = []string{
	"__list_map",
	"__list_filter",
	"__list_foldl",
	"__str_foldChars",
	"__str_foldSlices",
	"__str_mapSlicesJoin",
}

var hofBuiltinIndex = func() map[string]uint8 {
	m := make(map[string]uint8, len(HOFBuiltinTable))
	for i, name := range HOFBuiltinTable {
		m[name] = uint8(i)
	}
	return m
}()

// isLowerPassDictFallback reports whether name has the shape that the
// lower pass uses when it FAILS to resolve a dictionary method. Two
// patterns:
//
//   - `_dict_<method>` — produced by lowerDictApp when the dict is a
//     polymorphic *core.Var (DictAbs param) that wasn't monomorphized.
//   - `_<Class>_<Type>_<method>` (e.g., `_Fractional_Float_mul`) — produced
//     by the lowerDictMethod fall-through when the (className, method)
//     pair has no concrete BinOp/UnOp lowering. We detect this by the
//     leading `_` followed by an UPPERCASE letter, which never occurs
//     in legitimate pure builtins (`_show`, `_len`, `_list_*`,
//     `_concat_String`).
//
// These names indicate a *lower-pass bug*, not a missing runtime feature.
// Emitting `OpBuiltinTrap` for them would defer the failure to runtime
// and mask the regression. We make them a compile error instead.
func isLowerPassDictFallback(name string) bool {
	if strings.HasPrefix(name, "_dict_") {
		return true
	}
	if len(name) >= 2 && name[0] == '_' && unicode.IsUpper(rune(name[1])) {
		return true
	}
	return false
}

// compileBuiltinCall lowers a stmt.BuiltinCall into either OpBuiltinCall
// (for builtins listed in BuiltinTable) or OpBuiltinTrap (for any other
// effectful/unsupported builtin — these will be wired through the
// evaluator in Phase 2E).
func (fc *funcCompiler) compileBuiltinCall(e stmt.BuiltinCall) (uint8, error) {
	idx, isPure := builtinIndex[e.Name]
	if !isPure && isLowerPassDictFallback(e.Name) {
		return 0, fmt.Errorf(
			"compiler: builtin %q is a lower-pass dict-resolution fallback, "+
				"not a real builtin. This indicates that internal/gen/lower "+
				"failed to resolve a typeclass method to a concrete operation. "+
				"Add a branch to lowerDictMethod (or fix the upstream class "+
				"resolution) instead of expecting this to trap at runtime",
			e.Name)
	}
	n := len(e.Args)
	if n > 255 {
		return 0, fmt.Errorf("compiler: builtin %s has %d args, exceeds 255", e.Name, n)
	}

	// Allocate dst followed by n contiguous arg registers (matches OpMakeADT
	// layout: VM reads args from R[A+1..A+C]).
	block, err := fc.regs.allocContig(n + 1)
	if err != nil {
		return 0, err
	}
	dst := block
	for i, arg := range e.Args {
		if err := fc.compileExprIntoSlot(arg, dst+uint8(i+1)); err != nil {
			return 0, err
		}
	}
	if n > 0 {
		fc.regs.freeContig(dst+1, n)
	}
	if isPure {
		fc.emit(bytecode.EncodeABC(bytecode.OpBuiltinCall, dst, idx, uint8(n)))
		return dst, nil
	}
	// Check if it's a HOF builtin (takes closure arguments).
	if hofIdx, isHOF := hofBuiltinIndex[e.Name]; isHOF {
		fc.emit(bytecode.EncodeABC(bytecode.OpBuiltinCallHOF, dst, hofIdx, uint8(n)))
		return dst, nil
	}
	// Effectful / not-yet-wired builtins cannot be executed by the VM. Returning
	// a compile error here causes the enclosing proto to be tagged EvalOnly by
	// compiler.go Phase 2, which in turn causes the bridge to dispatch the whole
	// function through the evaluator at call time. That is the correct behavior
	// until M-BYTECODE-2E wires effectful builtins natively into the VM.
	return 0, fmt.Errorf("compiler: effectful builtin %q not yet wired (Phase 2E)", e.Name)
}
