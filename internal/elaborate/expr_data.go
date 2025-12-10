package elaborate

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
)

// Data structure expression normalization: records, lists, arrays, tuples

// normalizeRecord handles record construction
func (e *Elaborator) normalizeRecord(rec *ast.Record) (core.CoreExpr, error) {
	fields := make(map[string]core.CoreExpr)
	var allBindings []binding

	for i, field := range rec.Fields {
		value := field.Value
		name := field.Name
		atomic, binds, err := e.normalizeToAtomic(value)
		if err != nil {
			return nil, err
		}
		fields[name] = atomic
		allBindings = append(allBindings, binds...)
		_ = i // use i to avoid warning
	}

	result := &core.Record{
		CoreNode: e.makeNode(rec.Position()),
		Fields:   fields,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeRecordAccess handles field access
// Also handles qualified module access (e.g., List.map for import std/list as List)
func (e *Elaborator) normalizeRecordAccess(acc *ast.RecordAccess) (core.CoreExpr, error) {
	// Check for module alias qualified access (e.g., List.map)
	if ident, ok := acc.Record.(*ast.Identifier); ok {
		qualifiedName := fmt.Sprintf("%s.%s", ident.Name, acc.Field)
		if ref, ok := e.globalEnv[qualifiedName]; ok {
			// This is a qualified module access, resolve to global reference
			return &core.VarGlobal{
				CoreNode: e.makeNode(acc.Position()),
				Ref:      ref,
			}, nil
		}
	}

	// Standard record field access
	record, binds, err := e.normalizeToAtomic(acc.Record)
	if err != nil {
		return nil, err
	}

	result := &core.RecordAccess{
		CoreNode: e.makeNode(acc.Position()),
		Record:   record,
		Field:    acc.Field,
	}

	return e.wrapWithBindings(result, binds), nil
}

// normalizeRecordUpdate handles record update: {base | field: value, ...}
// Desugars to Core RecordUpdate node, which will be handled during type checking
// The type checker needs to know all fields to properly desugar this
func (e *Elaborator) normalizeRecordUpdate(upd *ast.RecordUpdate) (core.CoreExpr, error) {
	// Normalize base record
	base, baseBinds, err := e.normalizeToAtomic(upd.Base)
	if err != nil {
		return nil, err
	}

	// Normalize updated fields
	updates := make(map[string]core.CoreExpr)
	var allBindings []binding
	allBindings = append(allBindings, baseBinds...)

	for _, field := range upd.Fields {
		value, binds, err := e.normalizeToAtomic(field.Value)
		if err != nil {
			return nil, err
		}
		updates[field.Name] = value
		allBindings = append(allBindings, binds...)
	}

	// Create Core RecordUpdate node
	result := &core.RecordUpdate{
		CoreNode: e.makeNode(upd.Position()),
		Base:     base,
		Updates:  updates,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeList handles list construction
func (e *Elaborator) normalizeList(list *ast.List) (core.CoreExpr, error) {
	var elements []core.CoreExpr
	var allBindings []binding

	for _, elem := range list.Elements {
		atomic, binds, err := e.normalizeToAtomic(elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, atomic)
		allBindings = append(allBindings, binds...)
	}

	result := &core.List{
		CoreNode: e.makeNode(list.Position()),
		Elements: elements,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeArray handles array construction
func (e *Elaborator) normalizeArray(arr *ast.Array) (core.CoreExpr, error) {
	var elements []core.CoreExpr
	var allBindings []binding

	for _, elem := range arr.Elements {
		atomic, binds, err := e.normalizeToAtomic(elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, atomic)
		allBindings = append(allBindings, binds...)
	}

	result := &core.Array{
		CoreNode: e.makeNode(arr.Position()),
		Elements: elements,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeTuple handles tuple construction
func (e *Elaborator) normalizeTuple(tuple *ast.Tuple) (core.CoreExpr, error) {
	var elements []core.CoreExpr
	var allBindings []binding

	for _, elem := range tuple.Elements {
		atomic, binds, err := e.normalizeToAtomic(elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, atomic)
		allBindings = append(allBindings, binds...)
	}

	result := &core.Tuple{
		CoreNode: e.makeNode(tuple.Position()),
		Elements: elements,
	}

	return e.wrapWithBindings(result, allBindings), nil
}
