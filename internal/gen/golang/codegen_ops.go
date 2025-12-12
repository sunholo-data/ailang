// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// generateBinOp generates a Go binary operation.
func (g *Generator) generateBinOp(binop *core.BinOp) error {
	// Handle cons operator specially - it's not a simple binary op in Go
	if binop.Op == "::" {
		g.write("Cons(")
		if err := g.generateExpr(binop.Left); err != nil {
			return err
		}
		g.write(", ")
		if err := g.generateExpr(binop.Right); err != nil {
			return err
		}
		g.write(")")
		return nil
	}

	g.write("(")
	if err := g.generateExpr(binop.Left); err != nil {
		return err
	}
	g.writef(" %s ", g.mapOperator(binop.Op))
	if err := g.generateExpr(binop.Right); err != nil {
		return err
	}
	g.write(")")
	return nil
}

// generateUnOp generates a Go unary operation.
func (g *Generator) generateUnOp(unop *core.UnOp) error {
	g.writef("%s", g.mapUnaryOperator(unop.Op))
	return g.generateExpr(unop.Operand)
}

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
func (g *Generator) generateTypedRecord(rec *core.Record, recordType *RecordTypeInfo) error {
	// Generate as pointer for top-level returns (interface{} compatibility)
	g.writef("&%s{", recordType.Name)
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
				// M-DX13.2 + M-DX13.5: Field expects a record value type, but value is likely a pointer
				// M-DX14: Check if value is an ADT constructor (returns typed pointer, not interface{})
				if g.isADTConstructorExpr(value) {
					// ADT constructor returns typed pointer - just dereference, no type assertion
					g.write("*")
					if err := g.generateExpr(value); err != nil {
						return err
					}
				} else {
					// Interface value (e.g., var tmp1 interface{} = &Coord{})
					// Need to type assert then dereference: *(tmp1.(*Coord))
					g.write("*(")
					if err := g.generateExpr(value); err != nil {
						return err
					}
					g.writef(".(*%s))", goType)
				}
			} else {
				// Interface value - use type assertion
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
		if _, exists := g.adtConstructors[v.Ref.Name]; exists {
			return true
		}
	}
	if v, ok := expr.(*core.Var); ok {
		if _, exists := g.adtConstructors[v.Name]; exists {
			return true
		}
	}
	// Check for constructor with arguments (App)
	if app, ok := expr.(*core.App); ok {
		if v, ok := app.Func.(*core.VarGlobal); ok {
			if v.Ref.Module == "$adt" && strings.HasPrefix(v.Ref.Name, "make_") {
				return true
			}
			if _, exists := g.adtConstructors[v.Ref.Name]; exists {
				return true
			}
		}
		if v, ok := app.Func.(*core.Var); ok {
			if _, exists := g.adtConstructors[v.Name]; exists {
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

// generateList generates a Go slice literal.
// M-DX25.11: Uses typed slices when element type is known from CoreTypeInfo.
// M-CODEGEN-LIST: Flattens nested Let expressions to reduce IIFE nesting.
func (g *Generator) generateList(list *core.List) error {
	// M-CODEGEN-LIST: Check if any elements need flattening (contain Let expressions)
	needsFlattening := false
	for _, elem := range list.Elements {
		if containsLet(elem) {
			needsFlattening = true
			break
		}
	}

	if needsFlattening {
		// M-CODEGEN-LIST: Wrap in single IIFE that flattens all element computations
		// This reduces O(n) nesting to O(1) nesting
		return g.generateFlattenedList(list)
	}

	// M-DX26: In _impl functions, always generate []interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// M-DX25.11: Try to determine element type from CoreTypeInfo
	elemType := ""
	if !inImplFunc {
		elemType = g.getListElementType(list)
	}

	if elemType != "" && elemType != "interface{}" {
		// Generate typed slice (e.g., []int64{1, 2, 3})
		g.writef("[]%s{", elemType)
	} else {
		// Fallback to interface{} slice
		g.write("[]interface{}{")
	}

	for i, elem := range list.Elements {
		if i > 0 {
			g.write(", ")
		}
		if err := g.generateExpr(elem); err != nil {
			return err
		}
	}
	g.write("}")
	return nil
}

// containsLet checks if an expression contains a Let binding.
// M-CODEGEN-LIST: Used to detect when list elements need flattening.
func containsLet(expr core.CoreExpr) bool {
	switch e := expr.(type) {
	case *core.Let:
		return true
	case *core.LetRec:
		return true
	case *core.App:
		// Check function and all arguments
		if containsLet(e.Func) {
			return true
		}
		for _, arg := range e.Args {
			if containsLet(arg) {
				return true
			}
		}
	case *core.If:
		return containsLet(e.Cond) || containsLet(e.Then) || containsLet(e.Else)
	case *core.Match:
		if containsLet(e.Scrutinee) {
			return true
		}
		for _, arm := range e.Arms {
			if containsLet(arm.Body) {
				return true
			}
		}
	case *core.BinOp:
		return containsLet(e.Left) || containsLet(e.Right)
	case *core.UnOp:
		return containsLet(e.Operand)
	case *core.List:
		for _, elem := range e.Elements {
			if containsLet(elem) {
				return true
			}
		}
	case *core.Array:
		for _, elem := range e.Elements {
			if containsLet(elem) {
				return true
			}
		}
	case *core.Tuple:
		for _, elem := range e.Elements {
			if containsLet(elem) {
				return true
			}
		}
	case *core.Record:
		for _, value := range e.Fields {
			if containsLet(value) {
				return true
			}
		}
	case *core.RecordAccess:
		return containsLet(e.Record)
	case *core.RecordUpdate:
		if containsLet(e.Base) {
			return true
		}
		for _, value := range e.Updates {
			if containsLet(value) {
				return true
			}
		}
	}
	return false
}

// generateFlattenedList generates a list with flattened Let bindings.
// M-CODEGEN-LIST: Wraps list in single IIFE, flattens all element computations.
//
// Instead of:
//
//	func() interface{} {
//	    var tmp1 = f()
//	    return func() interface{} {
//	        var tmp2 = g()
//	        return []interface{}{tmp1, tmp2}
//	    }()
//	}()
//
// We generate:
//
//	func() interface{} {
//	    var _list_e0 = f()
//	    var _list_e1 = g()
//	    return []interface{}{_list_e0, _list_e1}
//	}()
func (g *Generator) generateFlattenedList(list *core.List) error {
	g.write("func() interface{} {\n")
	g.indent++

	// Evaluate all elements into temporary variables with flattened bindings
	varNames := make([]string, len(list.Elements))
	for i, elem := range list.Elements {
		varName := g.uniqueVarName("_list_e")
		varNames[i] = varName

		// Use flattenExprBindings to extract and emit Let bindings
		finalExpr := g.flattenExprBindings(elem)

		// Assign the final expression to our list element var
		g.writef("var %s interface{} = ", varName)
		if err := g.generateExpr(finalExpr); err != nil {
			return err
		}
		g.write("\n")
	}

	// Build the slice with flattened element references
	g.write("return []interface{}{")
	for i, varName := range varNames {
		if i > 0 {
			g.write(", ")
		}
		g.write(varName)
	}
	g.write("}\n")

	g.indent--
	g.write("}()")
	return nil
}

// flattenExprBindings extracts and emits Let bindings from an expression,
// returning the final non-Let expression.
// M-CODEGEN-LIST: Recursively flattens nested Let expressions.
func (g *Generator) flattenExprBindings(expr core.CoreExpr) core.CoreExpr {
	current := expr

	// Extract consecutive Let bindings
	for {
		let, ok := current.(*core.Let)
		if !ok {
			break
		}

		// Recursively flatten the value
		flatValue := g.flattenExprBindings(let.Value)

		// Emit this binding
		goName := ToGoVarName(let.Name)
		g.writef("var %s interface{} = ", goName)
		// Note: we call generateExpr here, which may still have some nesting
		// for complex expressions, but this is much better than before
		if err := g.generateExpr(flatValue); err != nil {
			// On error, just emit the original - fallback behavior
			if err := g.generateExpr(let.Value); err != nil {
				return nil // Still return nil to continue, error already recorded
			}
		}
		g.write("\n")

		// Move to the body for next iteration
		current = let.Body
	}

	return current
}

// uniqueVarName generates a unique variable name with the given prefix.
// M-CODEGEN-LIST: Prevents name collisions in flattened bindings.
func (g *Generator) uniqueVarName(prefix string) string {
	g.varCounter++
	return fmt.Sprintf("%s%d", prefix, g.varCounter)
}

// getListElementType extracts the Go element type for a list from CoreTypeInfo.
// M-DX25.11: Returns empty string if type is unknown or not a list type.
func (g *Generator) getListElementType(list *core.List) string {
	if g.coreTypeInfo == nil {
		return ""
	}

	nodeID := list.NodeID
	if nodeID == 0 {
		return ""
	}

	typ, ok := g.coreTypeInfo[nodeID]
	if !ok {
		return ""
	}

	// Map the type to Go and extract element type
	goType, err := g.TypeMapper.MapType(typ)
	if err != nil {
		return ""
	}

	// Check if it's a slice type and extract element type
	goTypeStr := string(goType)
	if len(goTypeStr) > 2 && goTypeStr[:2] == "[]" {
		return goTypeStr[2:]
	}

	return ""
}

// generateArray generates a Go slice literal for arrays.
// M-TYPE1: Arrays use the same Go representation as lists (slices).
func (g *Generator) generateArray(arr *core.Array) error {
	// M-DX26: In _impl functions, always generate []interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// Try to determine element type from CoreTypeInfo
	elemType := ""
	if !inImplFunc {
		elemType = g.getArrayElementType(arr)
	}

	if elemType != "" && elemType != "interface{}" {
		// Generate typed slice (e.g., []int64{1, 2, 3})
		g.writef("[]%s{", elemType)
	} else {
		// Fallback to interface{} slice
		g.write("[]interface{}{")
	}

	for i, elem := range arr.Elements {
		if i > 0 {
			g.write(", ")
		}
		if err := g.generateExpr(elem); err != nil {
			return err
		}
	}
	g.write("}")
	return nil
}

// getArrayElementType extracts the Go element type for an array from CoreTypeInfo.
// M-TYPE1: Returns empty string if type is unknown or not an array type.
func (g *Generator) getArrayElementType(arr *core.Array) string {
	if g.coreTypeInfo == nil {
		return ""
	}

	nodeID := arr.NodeID
	if nodeID == 0 {
		return ""
	}

	typ, ok := g.coreTypeInfo[nodeID]
	if !ok {
		return ""
	}

	// Map the type to Go and extract element type
	goType, err := g.TypeMapper.MapType(typ)
	if err != nil {
		return ""
	}

	// Check if it's a slice type and extract element type
	goTypeStr := string(goType)
	if len(goTypeStr) > 2 && goTypeStr[:2] == "[]" {
		return goTypeStr[2:]
	}

	return ""
}

// generateTuple generates a Go struct for tuple.
func (g *Generator) generateTuple(tuple *core.Tuple) error {
	g.write("[]interface{}{")
	for i, elem := range tuple.Elements {
		if i > 0 {
			g.write(", ")
		}
		if err := g.generateExpr(elem); err != nil {
			return err
		}
	}
	g.write("}")
	return nil
}

// generateIntrinsic generates code for intrinsic operations.
func (g *Generator) generateIntrinsic(intr *core.Intrinsic) error {
	op := g.mapIntrinsicOp(intr.Op)
	if len(intr.Args) == 1 {
		// Unary
		g.writef("(%s", op)
		if err := g.generateExpr(intr.Args[0]); err != nil {
			return err
		}
		g.write(")")
	} else if len(intr.Args) == 2 {
		// Binary
		g.write("(")
		if err := g.generateExpr(intr.Args[0]); err != nil {
			return err
		}
		g.writef(" %s ", op)
		if err := g.generateExpr(intr.Args[1]); err != nil {
			return err
		}
		g.write(")")
	}
	return nil
}

// generateDictApp generates dictionary method application.
func (g *Generator) generateDictApp(da *core.DictApp) error {
	if err := g.generateExpr(da.Dict); err != nil {
		return err
	}
	g.writef(".%s(", ToPascalCase(da.Method))
	for i, arg := range da.Args {
		if i > 0 {
			g.write(", ")
		}
		if err := g.generateExpr(arg); err != nil {
			return err
		}
	}
	g.write(")")
	return nil
}

// mapOperator maps AILANG binary operators to Go operators.
func (g *Generator) mapOperator(op string) string {
	switch op {
	case "+", "-", "*", "/", "%":
		return op
	case "==", "!=", "<", ">", "<=", ">=":
		return op
	case "&&", "||":
		return op
	case "++":
		return "+" // Works for strings
	default:
		return op
	}
}

// mapUnaryOperator maps AILANG unary operators to Go operators.
func (g *Generator) mapUnaryOperator(op string) string {
	switch op {
	case "-":
		return "-"
	case "!":
		return "!"
	default:
		return op
	}
}

// mapIntrinsicOp maps intrinsic operations to Go operators.
func (g *Generator) mapIntrinsicOp(op core.IntrinsicOp) string {
	switch op {
	case core.OpAdd:
		return "+"
	case core.OpSub:
		return "-"
	case core.OpMul:
		return "*"
	case core.OpDiv:
		return "/"
	case core.OpMod:
		return "%"
	case core.OpEq:
		return "=="
	case core.OpNe:
		return "!="
	case core.OpLt:
		return "<"
	case core.OpLe:
		return "<="
	case core.OpGt:
		return ">"
	case core.OpGe:
		return ">="
	case core.OpConcat:
		return "+"
	case core.OpAnd:
		return "&&"
	case core.OpOr:
		return "||"
	case core.OpNot:
		return "!"
	case core.OpNeg:
		return "-"
	default:
		return "?"
	}
}
