package smt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// RecordTypeInfo describes a record type for SMT encoding.
type RecordTypeInfo struct {
	SortName   string            // SMT-LIB sort name (e.g., "Point")
	CtorName   string            // Constructor name (e.g., "mk_Point")
	FieldNames []string          // Sorted field names
	FieldSorts map[string]string // Field name → SMT-LIB sort
}

// collectAndDeclareRecordTypes collects record types from function parameters,
// return type, body expressions, and ensures clauses.
// It populates activeRecordTypes/activeFieldSetToSort for use during encoding.
func collectAndDeclareRecordTypes(params []FunctionParam, returnSort string, returnType types.Type, body core.CoreExpr, contracts []*core.Contract, ctx *SMTContext, result *EncodeResult) {
	// Source 1: Function parameter types
	for _, p := range params {
		collectRecordType(p.Type, ctx, result)
	}
	// Source 2: Return type annotation
	if returnType != nil {
		collectRecordType(returnType, ctx, result)
	}
	// Source 3: Function body expressions (record literals)
	if body != nil {
		collectRecordTypesFromBody(body, ctx, result)
	}
	// Source 4: Ensures clause expressions
	for _, c := range contracts {
		if c.Kind == core.EnsuresKind && c.Expr != nil {
			collectRecordTypesFromBody(c.Expr, ctx, result)
		}
	}
}

// collectRecordType recursively extracts record types from an AILANG type
// and emits declare-datatype declarations. Uses DFS with cycle detection
// to handle nested records and reject self-referential types.
func collectRecordType(t types.Type, ctx *SMTContext, result *EncodeResult) {
	// Delegate to the safe version, ignoring errors (legacy behavior for non-cycle cases)
	_ = collectRecordTypeSafe(t, ctx, result, nil)
}

// collectRecordTypeSafe recursively extracts record types from an AILANG type
// and emits declare-datatype declarations in dependency order (inner before outer).
// The visiting set tracks records currently being processed to detect cycles.
// Returns an error if a self-referential or mutually recursive record is detected.
func collectRecordTypeSafe(t types.Type, ctx *SMTContext, result *EncodeResult, visiting map[string]bool) error {
	if t == nil {
		return nil
	}
	rec, ok := t.(*types.TRecord)
	if !ok {
		return nil
	}

	sortName := MapRecordSortName(rec)
	if ctx.DeclaredTypes[sortName] {
		return nil // already declared
	}

	// Initialize visiting set on first call
	if visiting == nil {
		visiting = make(map[string]bool)
	}

	// Cycle detection: if we're already visiting this sort, it's self-referential
	if visiting[sortName] {
		return fmt.Errorf("self-referential record type cycle detected: %s", sortName)
	}
	visiting[sortName] = true
	defer func() { delete(visiting, sortName) }()

	// Map all field types (may recursively discover nested record types)
	fieldSorts, err := MapRecordFields(rec)
	if err != nil {
		return nil // skip records with unencodable field types
	}

	// Recursively collect nested record types first (depth-first ensures inner before outer)
	for fieldName, fieldType := range rec.Fields {
		if err := collectRecordTypeSafe(fieldType, ctx, result, visiting); err != nil {
			return fmt.Errorf("field %q: %w", fieldName, err)
		}
	}

	// Build record type info
	fieldNames := SortedFieldNamesStr(fieldSorts)
	info := &RecordTypeInfo{
		SortName:   sortName,
		CtorName:   RecordConstructorName(sortName),
		FieldNames: fieldNames,
		FieldSorts: fieldSorts,
	}
	activeRecordTypes[sortName] = info

	// Build field-set key for lookup during encoding
	key := strings.Join(fieldNames, ",")
	activeFieldSetToSort[key] = sortName

	// Emit declaration (inner records already emitted by recursive calls above)
	decl := DeclareRecordDatatype(sortName, fieldSorts)
	result.Declarations = append(result.Declarations, decl)
	ctx.DeclaredTypes[sortName] = true

	return nil
}

// collectRecordTypesFromBody walks a Core AST expression to discover record
// types from core.Record literal nodes. This supplements parameter-based
// discovery by finding records constructed in function bodies and contract clauses.
func collectRecordTypesFromBody(expr core.CoreExpr, ctx *SMTContext, result *EncodeResult) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.Record:
		// Found a record literal — try to build a TRecord from field expressions
		if rec := inferRecordTypeFromLiteral(e); rec != nil {
			collectRecordType(rec, ctx, result)
		}
		// Also walk field value expressions for nested records
		for _, v := range e.Fields {
			collectRecordTypesFromBody(v, ctx, result)
		}
	case *core.Let:
		collectRecordTypesFromBody(e.Value, ctx, result)
		collectRecordTypesFromBody(e.Body, ctx, result)
	case *core.LetRec:
		for _, b := range e.Bindings {
			collectRecordTypesFromBody(b.Value, ctx, result)
		}
		collectRecordTypesFromBody(e.Body, ctx, result)
	case *core.If:
		collectRecordTypesFromBody(e.Cond, ctx, result)
		collectRecordTypesFromBody(e.Then, ctx, result)
		collectRecordTypesFromBody(e.Else, ctx, result)
	case *core.Match:
		collectRecordTypesFromBody(e.Scrutinee, ctx, result)
		for _, arm := range e.Arms {
			collectRecordTypesFromBody(arm.Body, ctx, result)
			if arm.Guard != nil {
				collectRecordTypesFromBody(arm.Guard, ctx, result)
			}
		}
	case *core.App:
		collectRecordTypesFromBody(e.Func, ctx, result)
		for _, arg := range e.Args {
			collectRecordTypesFromBody(arg, ctx, result)
		}
	case *core.Lambda:
		collectRecordTypesFromBody(e.Body, ctx, result)
	case *core.BinOp:
		collectRecordTypesFromBody(e.Left, ctx, result)
		collectRecordTypesFromBody(e.Right, ctx, result)
	case *core.UnOp:
		collectRecordTypesFromBody(e.Operand, ctx, result)
	case *core.Intrinsic:
		for _, arg := range e.Args {
			collectRecordTypesFromBody(arg, ctx, result)
		}
	case *core.RecordAccess:
		collectRecordTypesFromBody(e.Record, ctx, result)
	case *core.RecordUpdate:
		collectRecordTypesFromBody(e.Base, ctx, result)
		for _, v := range e.Updates {
			collectRecordTypesFromBody(v, ctx, result)
		}
	case *core.List:
		for _, elem := range e.Elements {
			collectRecordTypesFromBody(elem, ctx, result)
		}
	case *core.Tuple:
		for _, elem := range e.Elements {
			collectRecordTypesFromBody(elem, ctx, result)
		}
	case *core.DictApp:
		collectRecordTypesFromBody(e.Dict, ctx, result)
		for _, arg := range e.Args {
			collectRecordTypesFromBody(arg, ctx, result)
		}
	case *core.DictAbs:
		collectRecordTypesFromBody(e.Body, ctx, result)
	}
	// Lit, Var, VarGlobal, DictRef — no sub-expressions to walk
}

// inferRecordTypeFromLiteral tries to build a types.TRecord from a core.Record
// literal by inferring field types from the field value expressions.
// Returns nil if any field type cannot be inferred.
func inferRecordTypeFromLiteral(rec *core.Record) *types.TRecord {
	fields := make(map[string]types.Type, len(rec.Fields))
	for name, expr := range rec.Fields {
		t := inferTypeFromExpr(expr)
		if t == nil {
			return nil // can't infer — skip this record
		}
		fields[name] = t
	}
	return &types.TRecord{Fields: fields}
}

// inferTypeFromExpr infers a types.Type from a Core expression.
// Handles literals, variables with known sorts, and simple expressions.
func inferTypeFromExpr(expr core.CoreExpr) types.Type {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *core.Lit:
		switch e.Value.(type) {
		case int, int64:
			return &types.TCon{Name: "int"}
		case float64:
			return &types.TCon{Name: "float"}
		case bool:
			return &types.TCon{Name: "bool"}
		case string:
			return &types.TCon{Name: "string"}
		}
	case *core.Var:
		// Check if this variable has a known sort from activeRecordTypes context
		// Variables used in record fields typically have types from function params
		return nil // conservative — let other discovery paths handle
	case *core.BinOp:
		// Arithmetic/comparison ops — infer from operands
		return inferTypeFromExpr(e.Left)
	case *core.App:
		// For builtin calls, we can sometimes infer return type
		return nil // conservative
	case *core.RecordAccess:
		// Field access — type depends on the field, hard to infer without context
		return nil
	}
	return nil
}

// encodeRecord encodes a record construction expression.
// Record{Fields: {x: 5, y: 10}} → (mk_Point 5 10)
func encodeRecord(rec *core.Record) (string, error) {
	info := lookupRecordByFields(rec.Fields)
	if info == nil {
		return "", fmt.Errorf("record construction: unknown record type with fields %v (not declared in function signature)", fieldNamesFromExprMap(rec.Fields))
	}

	// Encode field values in sorted order
	var args []string
	for _, fieldName := range info.FieldNames {
		fieldExpr, ok := rec.Fields[fieldName]
		if !ok {
			return "", fmt.Errorf("record construction: missing field %q", fieldName)
		}
		encoded, err := EncodeExpr(fieldExpr)
		if err != nil {
			return "", fmt.Errorf("record field %q: %w", fieldName, err)
		}
		args = append(args, encoded)
	}

	return fmt.Sprintf("(%s %s)", info.CtorName, strings.Join(args, " ")), nil
}

// encodeRecordAccess encodes a record field access.
// RecordAccess{Record: p, Field: "x"} → (x p)
func encodeRecordAccess(ra *core.RecordAccess) (string, error) {
	record, err := EncodeExpr(ra.Record)
	if err != nil {
		return "", fmt.Errorf("record access: %w", err)
	}
	return fmt.Sprintf("(%s %s)", ra.Field, record), nil
}

// encodeRecordUpdate encodes a functional record update.
// RecordUpdate{Base: p, Updates: {x: 20}} → (mk_Point 20 (y p))
func encodeRecordUpdate(ru *core.RecordUpdate) (string, error) {
	info := lookupRecordByFields(ru.Updates)
	if info == nil {
		// Try to find by looking at base expression type (if it's a known variable)
		info = lookupRecordForUpdate(ru)
	}
	if info == nil {
		return "", fmt.Errorf("record update: unknown record type")
	}

	base, err := EncodeExpr(ru.Base)
	if err != nil {
		return "", fmt.Errorf("record update base: %w", err)
	}

	// Build constructor args: updated fields use new values, others use accessor on base
	var args []string
	for _, fieldName := range info.FieldNames {
		if updateExpr, ok := ru.Updates[fieldName]; ok {
			encoded, err := EncodeExpr(updateExpr)
			if err != nil {
				return "", fmt.Errorf("record update field %q: %w", fieldName, err)
			}
			args = append(args, encoded)
		} else {
			// Use accessor on base: (fieldName base)
			args = append(args, fmt.Sprintf("(%s %s)", fieldName, base))
		}
	}

	return fmt.Sprintf("(%s %s)", info.CtorName, strings.Join(args, " ")), nil
}

// lookupRecordByFields finds a record type info that contains ALL the given field names.
// For construction, the fields must match exactly; for updates, they must be a subset.
func lookupRecordByFields(fields interface{}) *RecordTypeInfo {
	if activeRecordTypes == nil {
		return nil
	}

	var names []string
	switch f := fields.(type) {
	case map[string]core.CoreExpr:
		for name := range f {
			names = append(names, name)
		}
	case map[string]string:
		for name := range f {
			names = append(names, name)
		}
	default:
		return nil
	}
	sort.Strings(names)
	key := strings.Join(names, ",")

	if sortName, ok := activeFieldSetToSort[key]; ok {
		return activeRecordTypes[sortName]
	}
	return nil
}

// lookupRecordForUpdate finds the record type for an update expression.
// Tries all known record types and checks if the update fields are a subset.
func lookupRecordForUpdate(ru *core.RecordUpdate) *RecordTypeInfo {
	if activeRecordTypes == nil {
		return nil
	}
	updateFields := make(map[string]bool, len(ru.Updates))
	for name := range ru.Updates {
		updateFields[name] = true
	}
	for _, info := range activeRecordTypes {
		// Check if all update fields exist in this record type
		allPresent := true
		for name := range updateFields {
			if _, ok := info.FieldSorts[name]; !ok {
				allPresent = false
				break
			}
		}
		if allPresent {
			return info
		}
	}
	return nil
}

// fieldNamesFromExprMap extracts sorted field names from a map of expressions.
func fieldNamesFromExprMap(fields map[string]core.CoreExpr) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// encodeList encodes a list literal to SMT-LIB using Z3 sequence theory.
// [1, 2, 3] → (seq.++ (seq.unit 1) (seq.++ (seq.unit 2) (seq.unit 3)))
// [] → (as seq.empty (Seq Int))  (needs type from context)
func encodeList(list *core.List) (string, error) {
	if len(list.Elements) == 0 {
		// Empty list — need an element sort for the typed empty sequence.
		// Default to Int; the SMT solver will unify sorts as needed.
		return "(as seq.empty (Seq Int))", nil
	}

	// Single element: (seq.unit elem)
	if len(list.Elements) == 1 {
		elem, err := EncodeExpr(list.Elements[0])
		if err != nil {
			return "", fmt.Errorf("list element 0: %w", err)
		}
		return fmt.Sprintf("(seq.unit %s)", elem), nil
	}

	// Multiple elements: chain of (seq.++ (seq.unit e1) (seq.++ (seq.unit e2) ...))
	// Build right-to-left
	encoded := make([]string, len(list.Elements))
	for i, elem := range list.Elements {
		e, err := EncodeExpr(elem)
		if err != nil {
			return "", fmt.Errorf("list element %d: %w", i, err)
		}
		encoded[i] = fmt.Sprintf("(seq.unit %s)", e)
	}

	// Z3's seq.++ is variadic, so we can pass all at once
	return fmt.Sprintf("(seq.++ %s)", strings.Join(encoded, " ")), nil
}
