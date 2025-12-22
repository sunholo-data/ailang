// Package golang provides Go code generation from AILANG Core AST.
//
// This file contains record-related code generation:
// - Record literals (typed structs and map[string]interface{})
// - Record field access and updates
// - Helper functions for record type handling
package golang

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// generateRecord generates a Go struct literal.
// M-DX13: Uses typed struct when record type is known, otherwise map[string]interface{}.
// M-DX26 FIX: Records MUST be typed even in _impl functions because type assertions
// work on the actual runtime type. If _impl returns map[string]interface{} but the
// wrapper asserts (*World), it will panic. The actual value must be *World.
// M-CROSS-MODULE: Check declared return type and CoreTypeInfo to prevent type contamination.
func (g *Generator) generateRecord(rec *core.Record) error {
	// M-CROSS-MODULE: First check if we have a declared function return type
	// This ensures the correct nominal type is used when multiple records share fields
	if g.currentFuncDeclaredReturn != "" {
		if info, exists := g.recordTypes[g.currentFuncDeclaredReturn]; exists {
			// Verify the fields match
			if len(info.Fields) == len(rec.Fields) {
				match := true
				for _, goFieldName := range info.Fields {
					ailangName := toLowerFirst(goFieldName)
					if _, ok := rec.Fields[ailangName]; !ok {
						match = false
						break
					}
				}
				if match {
					return g.generateTypedRecord(rec, info)
				}
			}
		}
	}

	// M-CROSS-MODULE: Try to get the type name from CoreTypeInfo
	// This preserves nominal type identity when set during unification
	if g.coreTypeInfo != nil {
		if recType, ok := g.coreTypeInfo[rec.NodeID]; ok {
			if tRec, ok := recType.(*types.TRecord); ok && tRec.TypeName != "" {
				// Use the nominal type name from unification
				goTypeName := ToGoTypeName(tRec.TypeName)
				if info, exists := g.recordTypes[goTypeName]; exists {
					return g.generateTypedRecord(rec, info)
				}
			} else if os.Getenv("DEBUG_CODEGEN") == "1" {
				// Debug: TypeName lookup failed
				fieldList := make([]string, 0, len(rec.Fields))
				for f := range rec.Fields {
					fieldList = append(fieldList, f)
				}
				if tRec, ok := recType.(*types.TRecord); ok {
					fmt.Fprintf(os.Stderr, "[DEBUG_CODEGEN] TypeName empty for record: fields=%v, TypeName=%q (should be set by unification!)\n",
						fieldList, tRec.TypeName)
				} else {
					fmt.Fprintf(os.Stderr, "[DEBUG_CODEGEN] CoreTypeInfo has wrong type: fields=%v, type=%T\n",
						fieldList, recType)
				}
			}
		}
	}

	// Build set of field names to look up matching record type
	fieldNames := make(map[string]bool, len(rec.Fields))
	for name := range rec.Fields {
		fieldNames[name] = true
	}

	// Check if we have a known record type matching these fields
	// M-DX26 FIX: Always use typed records - interface{} return type just means
	// the signature is interface{}, not that the actual values must be untyped.
	// WARNING: This is ambiguous when multiple records have same fields - use checks above
	recordType := g.GetRecordTypeByFields(fieldNames)
	if recordType != nil {
		if os.Getenv("DEBUG_CODEGEN") == "1" {
			fieldList := make([]string, 0, len(fieldNames))
			for f := range fieldNames {
				fieldList = append(fieldList, f)
			}
			fmt.Fprintf(os.Stderr, "[DEBUG_CODEGEN] GetRecordTypeByFields match: fields=%v, found=%q (may be wrong if multiple types share fields!)\n",
				fieldList, recordType.Name)
		}
		return g.generateTypedRecord(rec, recordType)
	}

	// M-CODEGEN-RECORD-TYPENAME-PRESERVATION: Fallback to untyped map
	// WARNING: If you reach this fallback, it likely means:
	// 1. TRecord.TypeName was lost during substitution (check unification_substitution.go)
	// 2. Record type wasn't registered via RegisterRecordType
	// 3. Field name mismatch between record literal and type definition
	// Debug: DEBUG_CODEGEN=1 to see warnings when fallback triggers
	if os.Getenv("DEBUG_CODEGEN") == "1" {
		fieldList := make([]string, 0, len(fieldNames))
		for f := range fieldNames {
			fieldList = append(fieldList, f)
		}
		var typeInfo string
		if g.coreTypeInfo != nil {
			if t := g.coreTypeInfo[rec.NodeID]; t != nil {
				typeInfo = fmt.Sprintf("type=%T", t)
				if tRec, ok := t.(*types.TRecord); ok {
					typeInfo += fmt.Sprintf(", TypeName=%q", tRec.TypeName)
				}
			} else {
				typeInfo = "no entry for NodeID"
			}
		} else {
			typeInfo = "coreTypeInfo is nil"
		}
		fmt.Fprintf(os.Stderr, "[DEBUG_CODEGEN] Record fallback to map[string]interface{}: fields=%v, declaredReturn=%q, %s\n",
			fieldList, g.currentFuncDeclaredReturn, typeInfo)
	}
	g.write("map[string]interface{}{")
	first := true
	for name, value := range rec.Fields {
		if !first {
			g.write(", ")
		}
		first = false
		g.writef("%q: ", name)
		if err := g.generateExpr(value); err != nil {
			return err
		}
	}
	g.write("}")
	return nil
}

// generateTypedRecord generates a typed Go struct literal.
// M-DX13: Generates &World{Npcs: ..., Width: ...} instead of map[string]interface{}.
// M-DX13.2: Generates pointer (&) only when used as return value, not for nested records.
// M-CODEGEN-VALUE-TYPES: Generates Type{} for leaf records (value), &Type{} for others (pointer).
func (g *Generator) generateTypedRecord(rec *core.Record, recordType *RecordTypeInfo) error {
	// M-CODEGEN-VALUE-TYPES: Use Category to decide value vs pointer representation
	if recordType.Category == TypeCategoryValue {
		// Leaf record with few fields - generate as value
		g.writef("%s{", recordType.Name)
	} else {
		// Non-leaf, recursive, or large record - generate as pointer
		g.writef("&%s{", recordType.Name)
	}
	first := true
	for _, goFieldName := range recordType.Fields {
		// Convert Go field name (PascalCase) to AILANG field name (camelCase)
		ailangName := toLowerFirst(goFieldName)
		value, ok := rec.Fields[ailangName]
		if !ok {
			continue // Field not present in literal (will be zero value)
		}

		if !first {
			g.write(", ")
		}
		first = false
		g.writef("%s: ", goFieldName)

		// Get the expected Go type for this field
		goType := recordType.FieldTypes[goFieldName]

		// M-DX15: Check if value is an empty list and generate typed empty slice
		if list, isList := value.(*core.List); isList && len(list.Elements) == 0 && strings.HasPrefix(goType, "[]") {
			// Empty list - generate typed empty slice (e.g., []int64{} not []interface{}{})
			g.writef("%s{}", goType)
		} else if sliceConv := g.getSliceConversion(goType); sliceConv != "" {
			// Check if we need a slice converter for ADT types
			g.writef("%s(", sliceConv)
			if err := g.generateExpr(value); err != nil {
				return err
			}
			g.write(")")
		} else if goType != "" && goType != "interface{}" {
			// M-DX13.1: For literals, use type conversion; for interface values, use assertion
			if lit, isLit := value.(*core.Lit); isLit {
				// M-CODEGEN-V2.M4: Only wrap literals if their native type differs from expected.
				// generateLit already adds int64()/float64()/etc.
				litType := g.getLitGoType(lit)
				if isPrimitiveGoType(goType) && litType != goType {
					// Type conversion needed (e.g., int to float)
					g.writef("%s(", goType)
					if err := g.generateExpr(lit); err != nil {
						return err
					}
					g.write(")")
				} else {
					// Same type or non-primitive - generate directly (already typed)
					if err := g.generateExpr(lit); err != nil {
						return err
					}
				}
			} else if nestedRec, isRec := value.(*core.Record); isRec {
				// M-DX13.2: Nested record - check if field expects pointer or value
				if strings.HasPrefix(goType, "*") {
					// Field expects pointer - generate &Struct{}
					// M-CODEGEN-NESTED-RECORD-TYPE: Use expected field type, not GetRecordTypeByFields
					// This fixes Vec3/SystemPos confusion when multiple types have same structure
					expectedTypeName := strings.TrimPrefix(goType, "*")
					if expectedType, exists := g.recordTypes[expectedTypeName]; exists {
						if err := g.generateTypedRecord(nestedRec, expectedType); err != nil {
							return err
						}
					} else {
						// Fallback to default generation (will use GetRecordTypeByFields)
						if err := g.generateExpr(nestedRec); err != nil {
							return err
						}
					}
				} else {
					// Field expects value - generate Struct{} (dereference if needed)
					// Look up the nested record type
					nestedFieldNames := make(map[string]bool, len(nestedRec.Fields))
					for name := range nestedRec.Fields {
						nestedFieldNames[name] = true
					}
					if nestedType := g.GetRecordTypeByFields(nestedFieldNames); nestedType != nil {
						// Generate as value struct (no &)
						if err := g.generateTypedRecordValue(nestedRec, nestedType); err != nil {
							return err
						}
					} else {
						// Fallback: generate and dereference
						g.write("*")
						if err := g.generateExpr(nestedRec); err != nil {
							return err
						}
					}
				}
			} else if isRecordValueType(goType) && !strings.HasPrefix(goType, "*") {
				// M-DX13.2 + M-DX13.5: Field expects a record value type
				// M-DX14: Check if value is an ADT constructor (returns typed pointer, not interface{})
				// M-CODEGEN-VALUE-TYPES: Check if this record is a value type (stored as value in interface{})
				recordInfo, isValueRecord := g.recordTypes[goType]
				if isValueRecord && recordInfo.Category == TypeCategoryValue {
					// M-CODEGEN-VALUE-TYPES: Value record from interface{} may be value OR pointer
					// Use AsTypeName helper to handle both cases
					if g.exprProducesInterface(value) && !g.isADTConstructorExpr(value) {
						g.markValueTypeConverterNeeded(goType)
						g.writef("As%s(", goType)
						if err := g.generateExpr(value); err != nil {
							return err
						}
						g.write(")")
					} else {
						// Non-interface value or ADT constructor - generate directly
						if err := g.generateExpr(value); err != nil {
							return err
						}
					}
				} else if g.isADTConstructorExpr(value) {
					// ADT constructor returns typed pointer - just dereference, no type assertion
					g.write("*")
					if err := g.generateExpr(value); err != nil {
						return err
					}
				} else {
					// Non-value-type from interface{} - use pointer assertion + dereference
					g.write("*(")
					if err := g.generateExpr(value); err != nil {
						return err
					}
					g.writef(".(*%s))", goType)
				}
			} else {
				// Interface value - use type assertion
				// M-CODEGEN-VALUE-TYPES: Check if this is a value-type record field
				// FieldGet ALWAYS returns pointers for struct fields (uses f.Addr().Interface())
				// This applies even for values accessed via Let bindings (ANF form)
				// For value-type records, we need: *value.(*SystemPos) to dereference
				checkType := strings.TrimPrefix(goType, "*")
				isValueTypeField := false
				if recordInfo, ok := g.recordTypes[checkType]; ok && recordInfo.Category == TypeCategoryValue {
					isValueTypeField = true
				}

				if isValueTypeField && g.exprProducesInterface(value) && !g.isADTConstructorExpr(value) {
					// M-CODEGEN-VALUE-TYPES: Value-type field from interface{} may be value OR pointer
					// FieldGet returns *Type, direct construction returns Type
					// Use AsTypeName helper to handle both cases
					g.markValueTypeConverterNeeded(checkType)
					g.writef("As%s(", checkType)
					if err := g.generateExpr(value); err != nil {
						return err
					}
					g.write(")")
				} else {
					if err := g.generateExpr(value); err != nil {
						return err
					}
					// M-CODEGEN-POINTER-RETURN-TYPES: Add type assertion for all typed fields
					// Both primitives (int64, string) and user-defined pointers (*ArrivalPhase)
					// need type assertions when value is interface{}
					// BUT: ADT constructors already return typed pointers, not interface{}
					if g.exprProducesInterface(value) && !g.isADTConstructorExpr(value) {
						g.writef(".(%s)", goType)
					}
				}
			}
		} else {
			if err := g.generateExpr(value); err != nil {
				return err
			}
		}
	}
	g.write("}")
	return nil
}

// generateTypedRecordValue generates a typed Go struct literal as a value (no pointer).
// M-DX13.2: Used for nested records where the field expects a value, not a pointer.
func (g *Generator) generateTypedRecordValue(rec *core.Record, recordType *RecordTypeInfo) error {
	g.writef("%s{", recordType.Name) // No & - value type
	first := true
	for _, goFieldName := range recordType.Fields {
		ailangName := toLowerFirst(goFieldName)
		value, ok := rec.Fields[ailangName]
		if !ok {
			continue
		}

		if !first {
			g.write(", ")
		}
		first = false
		g.writef("%s: ", goFieldName)

		goType := recordType.FieldTypes[goFieldName]

		if sliceConv := g.getSliceConversion(goType); sliceConv != "" {
			g.writef("%s(", sliceConv)
			if err := g.generateExpr(value); err != nil {
				return err
			}
			g.write(")")
		} else if lit, isLit := value.(*core.Lit); isLit && isPrimitiveGoType(goType) {
			// M-CODEGEN-V2.M4: Only wrap literals if their native type differs from expected.
			litType := g.getLitGoType(lit)
			if litType != goType {
				g.writef("%s(", goType)
				if err := g.generateExpr(lit); err != nil {
					return err
				}
				g.write(")")
			} else {
				// Same type - generate directly (already typed)
				if err := g.generateExpr(lit); err != nil {
					return err
				}
			}
		} else {
			if err := g.generateExpr(value); err != nil {
				return err
			}
			if isPrimitiveGoType(goType) {
				g.writef(".(%s)", goType)
			}
		}
	}
	g.write("}")
	return nil
}

// toLowerFirst converts PascalCase to camelCase (first letter lowercase).
func toLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]+'a'-'A') + s[1:]
}

// isPrimitiveGoType returns true if the Go type is a primitive that needs type assertion.
func isPrimitiveGoType(goType string) bool {
	switch goType {
	case "int64", "float64", "bool", "string":
		return true
	default:
		return false
	}
}

// isRecordValueType returns true if the Go type is a record/struct value type (not a pointer).
// M-DX13.2: Used to detect when we need to dereference a pointer to get a value.
func isRecordValueType(goType string) bool {
	if goType == "" || goType == "interface{}" {
		return false
	}
	if isPrimitiveGoType(goType) {
		return false
	}
	if strings.HasPrefix(goType, "*") {
		return false // It's a pointer type
	}
	if strings.HasPrefix(goType, "[]") {
		return false // It's a slice type
	}
	if strings.HasPrefix(goType, "map[") {
		return false // It's a map type
	}
	// Assume it's a struct/record type (e.g., Coord, NPC)
	return true
}

// isADTConstructorExpr returns true if the expression is an ADT constructor.
// M-DX14: ADT constructors return typed pointers (*Type), not interface{}.
// This handles both nullary constructors (Var/VarGlobal) and constructors with args (App).
func (g *Generator) isADTConstructorExpr(expr core.CoreExpr) bool {
	// Check for nullary constructor (Var or VarGlobal)
	if v, ok := expr.(*core.VarGlobal); ok {
		// Check $adt.make_TypeName_CtorName pattern
		if v.Ref.Module == "$adt" && strings.HasPrefix(v.Ref.Name, "make_") {
			return true
		}
		// Check direct constructor name
		// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
		if _, ok := g.LookupADTConstructor("", v.Ref.Name); ok {
			return true
		}
	}
	if v, ok := expr.(*core.Var); ok {
		// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
		if _, ok := g.LookupADTConstructor("", v.Name); ok {
			return true
		}
	}
	// Check for constructor with arguments (App)
	if app, ok := expr.(*core.App); ok {
		if v, ok := app.Func.(*core.VarGlobal); ok {
			if v.Ref.Module == "$adt" && strings.HasPrefix(v.Ref.Name, "make_") {
				return true
			}
			// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
			if _, ok := g.LookupADTConstructor("", v.Ref.Name); ok {
				return true
			}
		}
		if v, ok := app.Func.(*core.Var); ok {
			// M-DX22: Use LookupADTConstructor for backwards-compatible lookup
			if _, ok := g.LookupADTConstructor("", v.Name); ok {
				return true
			}
		}
	}
	return false
}

// generateRecordAccess generates field access.
// M-DX18: Uses FieldGet helper to handle both maps and typed structs.
func (g *Generator) generateRecordAccess(ra *core.RecordAccess) error {
	g.writef("FieldGet(")
	if err := g.generateExpr(ra.Record); err != nil {
		return err
	}
	g.writef(", %q)", ra.Field)
	return nil
}

// generateRecordUpdate generates Go code for a record update expression.
// AILANG: { base | field1: val1, field2: val2 }
// Go: RecordUpdate(base, map[string]interface{}{"field1": val1, ...})
func (g *Generator) generateRecordUpdate(ru *core.RecordUpdate) error {
	g.write("RecordUpdate(")
	if err := g.generateExpr(ru.Base); err != nil {
		return err
	}
	g.write(", map[string]interface{}{")

	first := true
	for field, val := range ru.Updates {
		if !first {
			g.write(", ")
		}
		first = false
		g.writef("%q: ", field)
		if err := g.generateExpr(val); err != nil {
			return err
		}
	}

	g.write("})")
	return nil
}
