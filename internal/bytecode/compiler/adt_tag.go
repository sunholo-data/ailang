package compiler

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// compileADTTagEq lowers a stmt.ADTTagEq to the instruction shape one
// compileSwitch case test uses:
//
//	GET_TAG    tag, value
//	LOAD_CONST ord, #ordinal
//	EQ         dst, tag, ord
//
// The ordinal comes from the same declaration-order tag table the switch
// path and MAKE_ADT use, so no new opcode or builtin is involved. ADT values
// are never read by name — the by-name _record_get route rejects them (#1503).
func (fc *funcCompiler) compileADTTagEq(e stmt.ADTTagEq) (uint8, error) {
	ordinal, err := fc.resolveTagOrdinal(e.TypeName, e.Tag)
	if err != nil {
		return 0, err
	}
	val, err := fc.compileExpr(e.Value)
	if err != nil {
		return 0, err
	}
	tagReg, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpGetTag, tagReg, val, 0))
	if !fc.isPinned(val) {
		fc.regs.freeTemp(val)
	}

	ordConstIdx, err := fc.addLocalConst(bytecode.NewInt(int64(ordinal)))
	if err != nil {
		return 0, err
	}
	ordReg, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, ordReg, ordConstIdx))

	fc.regs.freeTemp(ordReg)
	fc.regs.freeTemp(tagReg)
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpEq, dst, tagReg, ordReg))
	return dst, nil
}

// resolveTagOrdinal maps (typeName, tag) to the tag's declaration ordinal in
// typeName, the ADT the type checker stamped on the pattern. A missing or
// unknown typeName, or a tag it does not declare, is a loud compile error
// (which tags the function EvalOnly): an ordinal is only meaningful in its
// own type, so it is never looked up by the tag alone (#2303 — two ADTs can
// declare the same tag at different ordinals).
func (fc *funcCompiler) resolveTagOrdinal(typeName, tag string) (int, error) {
	info, err := fc.adtInfo(typeName, "tag check of "+tag)
	if err != nil {
		return 0, err
	}
	if ord, ok := info.tagOrdinal[tag]; ok {
		return ord, nil
	}
	return 0, fmt.Errorf("compiler: unknown tag %s.%s in tag check", typeName, tag)
}

// adtInfo returns the registered ADT typeName for a tag test (what names the
// test, for the error).
func (fc *funcCompiler) adtInfo(typeName, what string) (adtTypeInfo, error) {
	if typeName == "" {
		return adtTypeInfo{}, fmt.Errorf("compiler: %s has no ADT: a constructor pattern reached the back end without the type the type checker resolves it to (a compiler bug: such a pattern is TC_MATCH_007)", what)
	}
	info, ok := fc.adtTypes[typeName]
	if !ok {
		return adtTypeInfo{}, fmt.Errorf("compiler: unknown ADT %q in %s", typeName, what)
	}
	return info, nil
}
