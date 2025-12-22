// Package golang provides Go code generation from AILANG Core AST.
//
// M-CODEGEN-VALUE-TYPES: This file contains type analysis for value vs pointer decisions.
package golang

// IsLeafRecord checks if all field types are primitives (no nested records, ADTs, or slices).
// M-CODEGEN-VALUE-TYPES: Leaf records are candidates for value type generation.
//
// Returns true if all fields are primitive types (int64, float64, bool, string).
// Returns false if any field is:
//   - A user-defined type (e.g., *Camera, Coord)
//   - A slice (e.g., []int64, []*NPC)
//   - A map (e.g., map[string]interface{})
//   - An interface{}
//
// Note: Uses isPrimitiveGoType from codegen_ops.go.
func IsLeafRecord(fieldTypes map[string]string) bool {
	for _, goType := range fieldTypes {
		if !isPrimitiveGoType(goType) {
			return false
		}
	}
	return true
}

// AnalyzeRecordType determines the TypeCategory for a record type.
// M-CODEGEN-VALUE-TYPES: Two-step heuristic:
//  1. Eligibility: Must be a leaf record (all primitive fields)
//  2. Threshold: Field count must be ≤ threshold
//
// Returns TypeCategoryValue for small, primitive-only records.
// Returns TypeCategoryPointer for all other records.
func (g *Generator) AnalyzeRecordType(fieldTypes map[string]string) (TypeCategory, bool) {
	isLeaf := IsLeafRecord(fieldTypes)

	// Step 1: Eligibility check
	if !isLeaf {
		return TypeCategoryPointer, isLeaf
	}

	// Step 2: Threshold check
	fieldCount := len(fieldTypes)
	if fieldCount <= g.valueThreshold {
		return TypeCategoryValue, isLeaf
	}

	return TypeCategoryPointer, isLeaf
}

// RegisterRecordTypeWithAnalysis registers a record type with automatic category analysis.
// M-CODEGEN-VALUE-TYPES: Convenience method that combines registration with analysis.
func (g *Generator) RegisterRecordTypeWithAnalysis(name string, fields []string, fieldTypes map[string]string) {
	category, isLeaf := g.AnalyzeRecordType(fieldTypes)

	g.recordTypes[name] = &RecordTypeInfo{
		Name:       name,
		Fields:     fields,
		FieldTypes: fieldTypes,
		FieldCount: len(fields),
		Category:   category,
		IsLeaf:     isLeaf,
	}
}

// GoReprForType returns the Go type representation for an AILANG type name.
// M-CODEGEN-VALUE-TYPES: This is THE single source of truth for value vs pointer decisions.
// ALL codegen paths MUST use this function to determine Go type representations.
//
// Returns:
//   - goType: The Go type name (e.g., "Coord" or "World")
//   - isPointer: true if the type should be used as a pointer (*Type)
//
// Rules:
//   - Registered record types use their analyzed Category
//   - Unknown types default to pointer (safe fallback)
//   - Primitive types are never registered here (handled by isPrimitiveGoType)
func (g *Generator) GoReprForType(typeName string) (goType string, isPointer bool) {
	// Check if we have registration info for this type
	if info, ok := g.recordTypes[typeName]; ok {
		if info.Category == TypeCategoryValue {
			return info.Name, false // Value type: Coord
		}
		return info.Name, true // Pointer type: *World
	}

	// Unknown types default to pointer (safe fallback)
	// This handles ADTs, unregistered types, etc.
	return ToGoTypeName(typeName), true
}

// GoTypeStringForType returns the Go type string for an AILANG type name.
// M-CODEGEN-VALUE-TYPES: Convenience wrapper that returns the full type string.
//
// Examples:
//   - "Coord" (if value type)
//   - "*World" (if pointer type)
func (g *Generator) GoTypeStringForType(typeName string) string {
	goType, isPointer := g.GoReprForType(typeName)
	if isPointer {
		return "*" + goType
	}
	return goType
}

// IsValueType checks if a registered type is a value type.
// M-CODEGEN-VALUE-TYPES: Used for conditional codegen logic.
// Returns false for unknown types (safe fallback to pointer semantics).
func (g *Generator) IsValueType(typeName string) bool {
	if info, ok := g.recordTypes[typeName]; ok {
		return info.Category == TypeCategoryValue
	}
	return false
}

// IsPointerType checks if a registered type is a pointer type.
// M-CODEGEN-VALUE-TYPES: Inverse of IsValueType for convenience.
// Returns true for unknown types (safe fallback).
func (g *Generator) IsPointerType(typeName string) bool {
	return !g.IsValueType(typeName)
}
