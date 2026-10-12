package compiler

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// BuiltinTable is the native OpBuiltinCall name table, shared with the VM
// through internal/bytecode (see bytecode.BuiltinNames).
var BuiltinTable = bytecode.BuiltinNames

// AdaptedBuiltinTable is the adapted extension of the index space (#1447).
var AdaptedBuiltinTable = bytecode.AdaptedBuiltinNames

// builtinIndex maps a builtin's IR name to its OpBuiltinCall index: native
// BuiltinTable entries first, then AdaptedBuiltinTable (#1447). B is a uint8,
// so the combined table may not exceed 256 entries.
var builtinIndex = func() map[string]uint8 {
	total := len(BuiltinTable) + len(AdaptedBuiltinTable)
	if total > 256 {
		panic(fmt.Sprintf("bytecode compiler: %d native+adapted builtins exceed the 256-entry OpBuiltinCall index space", total))
	}
	m := make(map[string]uint8, total)
	for i, name := range BuiltinTable {
		m[name] = uint8(i)
	}
	for i, name := range AdaptedBuiltinTable {
		m[name] = uint8(len(BuiltinTable) + i)
	}
	return m
}()

// HOFBuiltinTable is the OpBuiltinCallHOF name table (bytecode.HOFBuiltinNames).
var HOFBuiltinTable = bytecode.HOFBuiltinNames

var hofBuiltinIndex = func() map[string]uint8 {
	m := make(map[string]uint8, len(HOFBuiltinTable))
	for i, name := range HOFBuiltinTable {
		m[name] = uint8(i)
	}
	return m
}()

var effectBuiltinIndex = func() map[string]uint8 {
	m := make(map[string]uint8, len(bytecode.EffectBuiltinNames))
	for i, name := range bytecode.EffectBuiltinNames {
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
	if effectIdx, ok := effectBuiltinIndex[e.Name]; ok {
		fc.emit(bytecode.EncodeABC(bytecode.OpEffectCall, dst, effectIdx, uint8(n)))
		return dst, nil
	}

	// A pure registry builtin that is neither native nor adapted: say so, with
	// the reason, instead of calling it effectful (#1447).
	if spec, ok := bytecode.PureBuiltinSpec(e.Name); ok {
		return 0, fmt.Errorf("compiler: pure builtin %q has no VM implementation (%s)", e.Name, bytecode.AdaptReason(spec.Type()))
	}
	// Effectful / not-yet-wired builtins cannot be executed by the VM. Returning
	// a compile error here causes the enclosing proto to be tagged EvalOnly by
	// compiler.go Phase 2, which in turn causes the bridge to dispatch the whole
	// function through the evaluator at call time. That is the correct behavior
	// until M-BYTECODE-2E wires effectful builtins natively into the VM.
	return 0, fmt.Errorf("compiler: effectful builtin %q not yet wired (Phase 2E)", e.Name)
}
