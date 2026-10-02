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

// resolveTagOrdinal maps (typeName, tag) to the tag's declaration ordinal.
// An empty or unknown typeName falls back to inferADTFromTags, which scans
// ADTs in declaration order (never Go map order — ailang#1355). A tag no
// registered ADT declares is a loud compile error, which tags the function
// EvalOnly rather than guessing.
func (fc *funcCompiler) resolveTagOrdinal(typeName, tag string) (int, error) {
	if info, ok := fc.adtTypes[typeName]; ok {
		if ord, ok := info.tagOrdinal[tag]; ok {
			return ord, nil
		}
		return 0, fmt.Errorf("compiler: unknown tag %s.%s in tag check", typeName, tag)
	}
	if _, info, ok := fc.inferADTFromTags([]string{tag}); ok {
		return info.tagOrdinal[tag], nil
	}
	return 0, fmt.Errorf("compiler: no ADT declares constructor %q (tag check, type %q)", tag, typeName)
}
