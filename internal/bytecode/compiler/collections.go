package compiler

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// recordTypeInfo stores per-record-type metadata: the canonical
// alphabetically-sorted field-name list. Used for both record literals
// (to compute the local-constant-index sequence after MAKE_RECORD) and
// field accesses (to translate field name → sorted index).
//
// ambiguous marks a name declared more than once with different field sets
// (two modules each defining `Point`): the compiler cannot tell which one a
// receiver is, so accesses through that name resolve by name at runtime.
type recordTypeInfo struct {
	sortedFields []string
	ambiguous    bool
}

// registerRecordType records a declared record's sorted field list. A name
// declared again with a different field set (two modules each defining
// `Point`) becomes ambiguous: no static slot is ever taken from it.
func registerRecordType(recordTypes map[string]recordTypeInfo, name string, sortedFields []string) {
	if prev, ok := recordTypes[name]; ok {
		if prev.ambiguous || !slices.Equal(prev.sortedFields, sortedFields) {
			recordTypes[name] = recordTypeInfo{ambiguous: true}
		}
		return
	}
	recordTypes[name] = recordTypeInfo{sortedFields: sortedFields}
}

// resolveRecordAliases registers each alias (`type P = V`) that names a
// record type — directly or through a chain of aliases — under the alias's
// own name. A cycle, a dangling target or a name already declared as
// something else leaves the alias unregistered, so accesses through it
// resolve by name at runtime.
func resolveRecordAliases(recordTypes map[string]recordTypeInfo, aliases map[string]string) {
	for alias := range aliases {
		target := aliases[alias]
		seen := map[string]bool{alias: true}
		for {
			if _, isAlias := aliases[target]; !isAlias || seen[target] {
				break
			}
			seen[target] = true
			target = aliases[target]
		}
		info, ok := recordTypes[target]
		if !ok {
			continue
		}
		if _, clash := recordTypes[alias]; clash {
			recordTypes[alias] = recordTypeInfo{ambiguous: true}
			continue
		}
		recordTypes[alias] = info
	}
}

// fieldIndex returns the index of name in the sorted field list, or -1.
func (r recordTypeInfo) fieldIndex(name string) int {
	for i, f := range r.sortedFields {
		if f == name {
			return i
		}
	}
	return -1
}

// adtTypeInfo stores per-ADT metadata: the source-order tag → integer ordinal
// mapping. Tag ordinals are stable per type and shared between constructors
// and switch cases.
type adtTypeInfo struct {
	tagOrdinal map[string]int // tag name → ordinal
	tagFields  map[string]int // tag name → field arity
}

// --- ListLit / TupleLit / Cons ----------------------------------------------

func (fc *funcCompiler) compileListLit(e stmt.ListLit) (uint8, error) {
	n := len(e.Elems)
	if n == 0 {
		// Empty list — allocate a fresh register and emit MAKE_LIST with count=0.
		// MAKE_LIST reads from registers [B, B+C); for C=0 we don't need any
		// source registers, so B can be 0.
		dst, err := fc.regs.allocTemp()
		if err != nil {
			return 0, err
		}
		fc.emit(bytecode.EncodeABC(bytecode.OpMakeList, dst, 0, 0))
		return dst, nil
	}
	base, err := fc.regs.allocContig(n)
	if err != nil {
		return 0, err
	}
	for i, el := range e.Elems {
		if err := fc.compileExprIntoSlot(el, base+uint8(i)); err != nil {
			return 0, err
		}
	}
	// Free elem regs first so the dest can reuse one.
	fc.regs.freeContig(base, n)
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpMakeList, dst, base, uint8(n)))
	return dst, nil
}

func (fc *funcCompiler) compileTupleLit(e stmt.TupleLit) (uint8, error) {
	n := len(e.Elems)
	if n == 0 {
		return 0, fmt.Errorf("compiler: empty tuple literal")
	}
	base, err := fc.regs.allocContig(n)
	if err != nil {
		return 0, err
	}
	for i, el := range e.Elems {
		if err := fc.compileExprIntoSlot(el, base+uint8(i)); err != nil {
			return 0, err
		}
	}
	fc.regs.freeContig(base, n)
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpMakeTuple, dst, base, uint8(n)))
	return dst, nil
}

func (fc *funcCompiler) compileCons(e stmt.Cons) (uint8, error) {
	head, err := fc.compileExpr(e.Head)
	if err != nil {
		return 0, err
	}
	tail, err := fc.compileExpr(e.Tail)
	if err != nil {
		return 0, err
	}
	if !fc.isPinned(tail) {
		fc.regs.freeTemp(tail)
	}
	if !fc.isPinned(head) {
		fc.regs.freeTemp(head)
	}
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpCons, dst, head, tail))
	return dst, nil
}

// --- RecordLit / FieldAccess / RecordUpdate ---------------------------------

func (fc *funcCompiler) compileRecordLit(e stmt.RecordLit) (uint8, error) {
	n := len(e.Fields)
	if n == 0 {
		return 0, fmt.Errorf("compiler: empty record literal")
	}

	// Sort field names alphabetically and remember the source-order → sorted
	// position so we know which value goes into which slot.
	type sortedField struct {
		name  string
		value stmt.Expr
	}
	sf := make([]sortedField, n)
	for i, f := range e.Fields {
		sf[i] = sortedField{f.Name, f.Value}
	}
	sort.Slice(sf, func(i, j int) bool { return sf[i].name < sf[j].name })

	// Allocate contiguous register block for the values, then compile each
	// value into its target slot.
	base, err := fc.regs.allocContig(n)
	if err != nil {
		return 0, err
	}
	for i, f := range sf {
		if err := fc.compileExprIntoSlot(f.value, base+uint8(i)); err != nil {
			return 0, err
		}
	}

	// Pre-register the field-name string constants.
	nameConstIdx := make([]uint16, n)
	for i, f := range sf {
		idx, err := fc.addLocalConst(bytecode.NewString(f.name))
		if err != nil {
			return 0, err
		}
		nameConstIdx[i] = idx
	}

	fc.regs.freeContig(base, n)
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpMakeRecord, dst, base, uint8(n)))
	// Pseudo-LOAD_CONST instructions for each field name.
	for _, ci := range nameConstIdx {
		fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, 0, ci))
	}
	return dst, nil
}

func (fc *funcCompiler) compileFieldAccess(e stmt.FieldAccess) (uint8, error) {
	rec, err := fc.compileExpr(e.Record)
	if err != nil {
		return 0, err
	}

	// Tuple field access: lower pass emits FieldAccess with names "_0", "_1",
	// etc. when destructuring tuples. The index is read literally from the
	// suffix — no sorting because tuples are positional.
	var idx int
	if strings.HasPrefix(e.Field, "_") {
		if n, perr := strconv.Atoi(e.Field[1:]); perr == nil {
			idx = n
		} else {
			idx = fc.lookupFieldIndex(e.Field, e.KnownFields, e.RecordType)
		}
	} else {
		idx = fc.lookupFieldIndex(e.Field, e.KnownFields, e.RecordType)
	}
	if idx < 0 {
		// Static field index resolution failed — fall back to runtime name
		// lookup via the _record_get builtin. This handles row-polymorphic
		// records and anonymous records whose full field set wasn't known
		// at lower time (M-BYTECODE-MULTIMODULE M3).
		return fc.compileFieldAccessByName(rec, e.Field)
	}
	if idx > 255 {
		return 0, fmt.Errorf("compiler: field index %d exceeds 255", idx)
	}

	if !fc.isPinned(rec) {
		fc.regs.freeTemp(rec)
	}
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpGetField, dst, rec, uint8(idx)))
	return dst, nil
}

// compileFieldAccessByName emits an OpBuiltinCall to _record_get with the
// record (already materialized in recReg) and the field name as a string
// constant. Used when static field index resolution fails.
func (fc *funcCompiler) compileFieldAccessByName(recReg uint8, field string) (uint8, error) {
	builtinIdx, ok := builtinIndex["_record_get"]
	if !ok {
		return 0, fmt.Errorf("compiler: _record_get builtin missing from BuiltinTable")
	}
	// Allocate a contiguous [dst, rec, name] block. We can't reuse recReg
	// directly because it may not be adjacent to the name register.
	block, err := fc.regs.allocContig(3)
	if err != nil {
		return 0, err
	}
	// Copy rec into block+1.
	fc.emit(bytecode.EncodeABC(bytecode.OpMove, block+1, recReg, 0))
	if !fc.isPinned(recReg) {
		fc.regs.freeTemp(recReg)
	}
	// Load field name constant into block+2.
	nameIdx, err := fc.addLocalConst(bytecode.NewString(field))
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, block+2, nameIdx))
	// BUILTIN_CALL dst=block, idx=_record_get, argc=2.
	fc.emit(bytecode.EncodeABC(bytecode.OpBuiltinCall, block, builtinIdx, 2))
	// Free the two arg slots; result lives in `block`.
	fc.regs.freeContig(block+1, 2)
	return block, nil
}

// compileRecordUpdateByName compiles `{base | f1: v1, ...}` whose base type
// is not statically known as a chain of _record_set builtin calls, each
// replacing (or adding) one field by name on the runtime record — the same
// operation the evaluator performs (evalCoreRecordUpdate). The alternative,
// guessing the base's shape from another registered type, rebuilt records
// with the wrong field set (ailang#1354).
func (fc *funcCompiler) compileRecordUpdateByName(e stmt.RecordUpdate) (uint8, error) {
	builtinIdx, ok := builtinIndex["_record_set"]
	if !ok {
		return 0, fmt.Errorf("compiler: _record_set builtin missing from BuiltinTable")
	}
	rec, err := fc.compileExpr(e.Base)
	if err != nil {
		return 0, err
	}
	for _, f := range e.Fields {
		// Contiguous [dst, rec, name, value] block for BUILTIN_CALL argc=3.
		block, err := fc.regs.allocContig(4)
		if err != nil {
			return 0, err
		}
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, block+1, rec, 0))
		if !fc.isPinned(rec) {
			fc.regs.freeTemp(rec)
		}
		nameIdx, err := fc.addLocalConst(bytecode.NewString(f.Name))
		if err != nil {
			return 0, err
		}
		fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, block+2, nameIdx))
		if err := fc.compileExprIntoSlot(f.Value, block+3); err != nil {
			return 0, err
		}
		fc.emit(bytecode.EncodeABC(bytecode.OpBuiltinCall, block, builtinIdx, 3))
		fc.regs.freeContig(block+1, 3)
		rec = block
	}
	return rec, nil
}

// resolveRecordFields returns the sorted field list of a receiver's record
// type, or nil when the type is not statically known. Sources, in order:
//  1. knownFields — the receiver's inferred field set from the lower pass
//     (anonymous / row-polymorphic records, M-BYTECODE-MULTIMODULE M3).
//  2. recordType — the receiver's declared type name, looked up exactly in
//     the TypeDecl table (named records such as `v: V`).
//
// There is deliberately no third source. Guessing the type from any
// registered record that merely contains the field name picked another
// type's slot whenever two records shared a field name, and did so in Go
// map order, so codegen changed from run to run (ailang#1354, #1355).
func (fc *funcCompiler) resolveRecordFields(knownFields []string, recordType string) []string {
	if len(knownFields) > 0 {
		return knownFields
	}
	if recordType != "" {
		if info, ok := fc.recordTypes[recordType]; ok && !info.ambiguous {
			return info.sortedFields
		}
	}
	return nil
}

// lookupFieldIndex resolves a field access to its alphabetical index in the
// receiver's sorted field list, or -1 when the receiver's type is not
// statically known — the caller then reads the field by name at runtime.
func (fc *funcCompiler) lookupFieldIndex(name string, knownFields []string, recordType string) int {
	for i, f := range fc.resolveRecordFields(knownFields, recordType) {
		if f == name {
			return i
		}
	}
	return -1
}

func (fc *funcCompiler) compileRecordUpdate(e stmt.RecordUpdate) (uint8, error) {
	// Strategy: build a fresh record from the base's fields, overriding the
	// supplied ones. We do this by:
	//   1. Evaluating the base record into baseReg.
	//   2. For each field in the type's sorted order: if it's in the update
	//      set, evaluate the new value; otherwise emit GET_FIELD baseReg.
	//   3. MAKE_RECORD with all values in a contiguous block.
	//
	// When the base's type is not statically known, update by name at
	// runtime instead (compileRecordUpdateByName).
	info := recordTypeInfo{sortedFields: fc.resolveRecordFields(e.KnownFields, e.RecordType)}
	if len(info.sortedFields) == 0 {
		return fc.compileRecordUpdateByName(e)
	}
	for _, f := range e.Fields {
		if info.fieldIndex(f.Name) < 0 {
			// The update adds a field the static type lacks; only the
			// by-name path builds that record the way the evaluator does.
			return fc.compileRecordUpdateByName(e)
		}
	}

	overrides := make(map[string]stmt.Expr, len(e.Fields))
	for _, f := range e.Fields {
		overrides[f.Name] = f.Value
	}

	baseReg, err := fc.compileExpr(e.Base)
	if err != nil {
		return 0, err
	}

	n := len(info.sortedFields)
	base, err := fc.regs.allocContig(n)
	if err != nil {
		return 0, err
	}
	for i, name := range info.sortedFields {
		if val, ok := overrides[name]; ok {
			if err := fc.compileExprIntoSlot(val, base+uint8(i)); err != nil {
				return 0, err
			}
		} else {
			fc.emit(bytecode.EncodeABC(bytecode.OpGetField, base+uint8(i), baseReg, uint8(i)))
		}
	}
	if !fc.isPinned(baseReg) {
		fc.regs.freeTemp(baseReg)
	}

	nameConstIdx := make([]uint16, n)
	for i, name := range info.sortedFields {
		idx, err := fc.addLocalConst(bytecode.NewString(name))
		if err != nil {
			return 0, err
		}
		nameConstIdx[i] = idx
	}
	fc.regs.freeContig(base, n)
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABC(bytecode.OpMakeRecord, dst, base, uint8(n)))
	for _, ci := range nameConstIdx {
		fc.emit(bytecode.EncodeABx(bytecode.OpLoadConst, 0, ci))
	}
	return dst, nil
}

// --- ADT constructor --------------------------------------------------------

func (fc *funcCompiler) compileADTConstructor(e stmt.ADTConstructor) (uint8, error) {
	info, ok := fc.adtTypes[e.TypeName]
	if !ok {
		return 0, fmt.Errorf("compiler: unknown ADT type %q", e.TypeName)
	}
	tag, ok := info.tagOrdinal[e.Tag]
	if !ok {
		return 0, fmt.Errorf("compiler: unknown ADT tag %s.%s", e.TypeName, e.Tag)
	}
	if tag > 255 {
		return 0, fmt.Errorf("compiler: ADT tag ordinal %d exceeds 255", tag)
	}
	n := len(e.Args)
	if n > 255 {
		return 0, fmt.Errorf("compiler: ADT constructor with %d fields exceeds 255", n)
	}

	// MAKE_ADT layout: A=dst, B=tag, C=count, fields read from R[A+1..A+C].
	// So we need to allocate dst followed by n contiguous regs for fields.
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
	fc.emit(bytecode.EncodeABC(bytecode.OpMakeADT, dst, uint8(tag), uint8(n)))
	// Free the field regs but keep dst alive.
	if n > 0 {
		fc.regs.freeContig(dst+1, n)
	}
	return dst, nil
}
