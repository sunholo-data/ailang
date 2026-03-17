// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeSliceConverters writes slice type conversion functions.
func (g *Generator) writeRuntimeSliceConverters() {
	// Slice conversion helpers
	g.writef("// ConvertToInt64Slice converts []interface{} to []int64.\n")
	g.writef("func ConvertToInt64Slice(v interface{}) []int64 {\n")
	g.indent++
	g.writef("if v == nil {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("slice, ok := v.([]interface{})\n")
	g.writef("if !ok {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make([]int64, len(slice))\n")
	g.writef("for i, elem := range slice {\n")
	g.indent++
	g.writef("result[i] = toInt64(elem)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// ConvertToStringSlice converts []interface{} to []string.\n")
	g.writef("func ConvertToStringSlice(v interface{}) []string {\n")
	g.indent++
	g.writef("if v == nil {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("slice, ok := v.([]interface{})\n")
	g.writef("if !ok {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make([]string, len(slice))\n")
	g.writef("for i, elem := range slice {\n")
	g.indent++
	g.writef("if s, ok := elem.(string); ok {\n")
	g.indent++
	g.writef("result[i] = s\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// ConvertToRecordSlice converts []interface{} to []map[string]interface{}.\n")
	g.writef("func ConvertToRecordSlice(v interface{}) []map[string]interface{} {\n")
	g.indent++
	g.writef("if v == nil {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("slice, ok := v.([]interface{})\n")
	g.writef("if !ok {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make([]map[string]interface{}, len(slice))\n")
	g.writef("for i, elem := range slice {\n")
	g.indent++
	g.writef("if m, ok := elem.(map[string]interface{}); ok {\n")
	g.indent++
	g.writef("result[i] = m\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// M-CODEGEN-BOOL-SLICE: Bool slice converter
	g.writef("// ConvertToBoolSlice converts []interface{} to []bool.\n")
	g.writef("func ConvertToBoolSlice(v interface{}) []bool {\n")
	g.indent++
	g.writef("if v == nil {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("// Passthrough if already []bool\n")
	g.writef("if bs, ok := v.([]bool); ok {\n")
	g.indent++
	g.writef("return bs\n")
	g.indent--
	g.writef("}\n")
	g.writef("slice, ok := v.([]interface{})\n")
	g.writef("if !ok {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make([]bool, len(slice))\n")
	g.writef("for i, elem := range slice {\n")
	g.indent++
	g.writef("if b, ok := elem.(bool); ok {\n")
	g.indent++
	g.writef("result[i] = b\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// M-CODEGEN-UNIFIED-SLICE: Float64 slice converter
	g.writef("// ConvertToFloat64Slice converts []interface{} to []float64.\n")
	g.writef("func ConvertToFloat64Slice(v interface{}) []float64 {\n")
	g.indent++
	g.writef("if v == nil {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("// Passthrough if already []float64\n")
	g.writef("if fs, ok := v.([]float64); ok {\n")
	g.indent++
	g.writef("return fs\n")
	g.indent--
	g.writef("}\n")
	g.writef("slice, ok := v.([]interface{})\n")
	g.writef("if !ok {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make([]float64, len(slice))\n")
	g.writef("for i, elem := range slice {\n")
	g.indent++
	g.writef("if f, ok := elem.(float64); ok {\n")
	g.indent++
	g.writef("result[i] = f\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")
}

// writeADTSliceConverters generates type-safe slice conversion functions for ADT types.
// M-DX12: These enable [ADT] fields to be typed slices in generated Go structs.
// M-DX22: Now generates converters for ALL ADT types (not just those in slice fields).
func (g *Generator) writeADTSliceConverters() {
	// M-DX22: Collect ALL ADT types - both explicitly registered and from constructors
	allTypes := make(map[string]bool)

	// Include explicitly registered slice types (M-DX12)
	for typeName := range g.adtSliceTypes {
		allTypes[typeName] = true
	}

	// M-DX22: Include all ADT types from registered constructors
	for _, info := range g.adtConstructors {
		allTypes[info.TypeName] = true
	}

	// Sort for deterministic output
	var sortedTypes []string
	for typeName := range allTypes {
		sortedTypes = append(sortedTypes, typeName)
	}
	// Sort alphabetically
	for i := 0; i < len(sortedTypes); i++ {
		for j := i + 1; j < len(sortedTypes); j++ {
			if sortedTypes[i] > sortedTypes[j] {
				sortedTypes[i], sortedTypes[j] = sortedTypes[j], sortedTypes[i]
			}
		}
	}

	for _, typeName := range sortedTypes {
		goTypeName := ToGoTypeName(typeName)
		// M-BUGFIX: Export converters so external packages can use them
		funcName := "ConvertTo" + goTypeName + "Slice"

		// M-CODEGEN-VALUE-TYPES: Check if this is a value-type record
		// adtSliceTypes may include record types that appear in list fields
		recordInfo := g.recordTypes[goTypeName]
		isValueType := recordInfo != nil && recordInfo.Category == TypeCategoryValue

		if isValueType {
			// Value type: generate []Type and e.(Type)
			g.writef("// %s converts []interface{} to []%s.\n", funcName, goTypeName)
			g.writef("// M-DX12: Fail-fast - panics on type mismatch (compiler bug detection).\n")
			g.writef("func %s(v interface{}) []%s {\n", funcName, goTypeName)
		} else {
			// Pointer type: generate []*Type and e.(*Type)
			g.writef("// %s converts []interface{} to []*%s.\n", funcName, goTypeName)
			g.writef("// M-DX12: Fail-fast - panics on type mismatch (compiler bug detection).\n")
			g.writef("func %s(v interface{}) []*%s {\n", funcName, goTypeName)
		}
		g.indent++

		// Handle nil
		g.writef("if v == nil {\n")
		g.indent++
		g.writef("return nil\n")
		g.indent--
		g.writef("}\n")

		// Assert to []interface{}
		g.writef("src, ok := v.([]interface{})\n")
		g.writef("if !ok {\n")
		g.indent++
		g.writef("panic(fmt.Sprintf(\"%s: expected []interface{}, got %%T\", v))\n", funcName)
		g.indent--
		g.writef("}\n")

		// Handle empty slice (return empty, not nil)
		g.writef("if len(src) == 0 {\n")
		g.indent++
		if isValueType {
			g.writef("return []%s{}\n", goTypeName)
		} else {
			g.writef("return []*%s{}\n", goTypeName)
		}
		g.indent--
		g.writef("}\n")

		// Convert elements
		if isValueType {
			g.writef("out := make([]%s, len(src))\n", goTypeName)
			g.writef("for i, e := range src {\n")
			g.indent++
			g.writef("if elem, ok := e.(%s); ok {\n", goTypeName)
			g.indent++
			g.writef("out[i] = elem\n")
			g.indent--
			g.writef("} else if ptr, ok := e.(*%s); ok {\n", goTypeName)
			g.indent++
			g.writef("out[i] = *ptr\n")
			g.indent--
			g.writef("} else {\n")
			g.indent++
			g.writef("panic(fmt.Sprintf(\"%s: element %%d: expected %s or *%s, got %%T\", i, e))\n", funcName, goTypeName, goTypeName)
			g.indent--
			g.writef("}\n")
			g.indent--
			g.writef("}\n")
			g.writef("return out\n")
		} else {
			g.writef("out := make([]*%s, len(src))\n", goTypeName)
			g.writef("for i, e := range src {\n")
			g.indent++
			g.writef("elem, ok := e.(*%s)\n", goTypeName)
			g.writef("if !ok {\n")
			g.indent++
			g.writef("panic(fmt.Sprintf(\"%s: element %%d: expected *%s, got %%T\", i, e))\n", funcName, goTypeName)
			g.indent--
			g.writef("}\n")
			g.writef("out[i] = elem\n")
			g.indent--
			g.writef("}\n")
			g.writef("return out\n")
		}

		g.indent--
		g.writef("}\n\n")
	}
}

// writeRecordSliceConverters generates type-safe slice conversion functions for record types.
// M-CODEGEN-UNIFIED-SLICE: These enable [Record] return types to be typed slices.
// IMPORTANT: Skip types already handled by writeADTSliceConverters to avoid duplicates.
func (g *Generator) writeRecordSliceConverters() {
	// Build set of ADT types to skip (they're handled by writeADTSliceConverters)
	// Use ToGoTypeName for consistent comparison (handles case normalization)
	adtTypes := make(map[string]bool)
	for typeName := range g.adtSliceTypes {
		adtTypes[ToGoTypeName(typeName)] = true
	}
	for _, info := range g.adtConstructors {
		adtTypes[ToGoTypeName(info.TypeName)] = true
	}

	// Sort for deterministic output
	var sortedTypes []string
	for typeName := range g.recordTypes {
		// Skip types already handled by ADT converters
		// recordTypes keys are already in Go format (capitalized)
		if adtTypes[typeName] {
			continue
		}
		sortedTypes = append(sortedTypes, typeName)
	}
	// Sort alphabetically
	for i := 0; i < len(sortedTypes); i++ {
		for j := i + 1; j < len(sortedTypes); j++ {
			if sortedTypes[i] > sortedTypes[j] {
				sortedTypes[i], sortedTypes[j] = sortedTypes[j], sortedTypes[i]
			}
		}
	}

	for _, typeName := range sortedTypes {
		goTypeName := ToGoTypeName(typeName)
		funcName := "ConvertTo" + goTypeName + "Slice"

		// M-CODEGEN-VALUE-TYPES: Check if this is a value type
		recordInfo := g.recordTypes[typeName]
		isValueType := recordInfo != nil && recordInfo.Category == TypeCategoryValue

		if isValueType {
			g.writef("// %s converts []interface{} to []%s.\n", funcName, goTypeName)
			g.writef("// M-CODEGEN-UNIFIED-SLICE: Record slice converter (value type).\n")
			g.writef("func %s(v interface{}) []%s {\n", funcName, goTypeName)
		} else {
			g.writef("// %s converts []interface{} to []*%s.\n", funcName, goTypeName)
			g.writef("// M-CODEGEN-UNIFIED-SLICE: Record slice converter.\n")
			g.writef("func %s(v interface{}) []*%s {\n", funcName, goTypeName)
		}
		g.indent++

		// Handle nil
		g.writef("if v == nil {\n")
		g.indent++
		g.writef("return nil\n")
		g.indent--
		g.writef("}\n")

		// Assert to []interface{}
		g.writef("src, ok := v.([]interface{})\n")
		g.writef("if !ok {\n")
		g.indent++
		g.writef("panic(fmt.Sprintf(\"%s: expected []interface{}, got %%T\", v))\n", funcName)
		g.indent--
		g.writef("}\n")

		// Handle empty slice (return empty, not nil)
		g.writef("if len(src) == 0 {\n")
		g.indent++
		if isValueType {
			g.writef("return []%s{}\n", goTypeName)
		} else {
			g.writef("return []*%s{}\n", goTypeName)
		}
		g.indent--
		g.writef("}\n")

		// Convert elements
		if isValueType {
			g.writef("out := make([]%s, len(src))\n", goTypeName)
			g.writef("for i, e := range src {\n")
			g.indent++
			g.writef("if elem, ok := e.(%s); ok {\n", goTypeName)
			g.indent++
			g.writef("out[i] = elem\n")
			g.indent--
			g.writef("} else if ptr, ok := e.(*%s); ok {\n", goTypeName)
			g.indent++
			g.writef("out[i] = *ptr\n")
			g.indent--
			g.writef("} else {\n")
			g.indent++
			g.writef("panic(fmt.Sprintf(\"%s: element %%d: expected %s or *%s, got %%T\", i, e))\n", funcName, goTypeName, goTypeName)
			g.indent--
			g.writef("}\n")
			g.indent--
			g.writef("}\n")
			g.writef("return out\n")
		} else {
			g.writef("out := make([]*%s, len(src))\n", goTypeName)
			g.writef("for i, e := range src {\n")
			g.indent++
			g.writef("elem, ok := e.(*%s)\n", goTypeName)
			g.writef("if !ok {\n")
			g.indent++
			g.writef("panic(fmt.Sprintf(\"%s: element %%d: expected *%s, got %%T\", i, e))\n", funcName, goTypeName)
			g.indent--
			g.writef("}\n")
			g.writef("out[i] = elem\n")
			g.indent--
			g.writef("}\n")
			g.writef("return out\n")
		}

		g.indent--
		g.writef("}\n\n")
	}
}

// writeValueTypeConverters generates AsTypeName helper functions for value-type records.
// M-CODEGEN-VALUE-TYPES: These handle the dual representation in interface{}:
// - FieldGet returns pointers (*Type) due to Go reflection
// - Direct construction returns values (Type)
// The converter tries value assertion first, then pointer dereference.
//
// M-CODEGEN-VALUE-TYPES-FIX: Generate converters for ALL value-type records, not just
// the ones explicitly marked during code generation. This is needed because in multi-file
// compilation, GenerateRuntime() is called BEFORE declaration codegen populates the map.
func (g *Generator) writeValueTypeConverters() {
	// Collect all value-type records from recordTypes registry
	// M-CODEGEN-VALUE-TYPES-FIX: Use recordTypes as source of truth, not valueTypeConverters
	var sortedTypes []string
	for typeName, info := range g.recordTypes {
		if info.Category == TypeCategoryValue {
			sortedTypes = append(sortedTypes, typeName)
		}
	}
	// Sort alphabetically
	for i := 0; i < len(sortedTypes); i++ {
		for j := i + 1; j < len(sortedTypes); j++ {
			if sortedTypes[i] > sortedTypes[j] {
				sortedTypes[i], sortedTypes[j] = sortedTypes[j], sortedTypes[i]
			}
		}
	}

	for _, goTypeName := range sortedTypes {
		funcName := "As" + goTypeName

		g.writef("// %s extracts %s from interface{} that may contain value or pointer.\n", funcName, goTypeName)
		g.writef("// M-CODEGEN-VALUE-TYPES: Handles both direct construction (value) and FieldGet (pointer).\n")
		g.writef("func %s(v interface{}) %s {\n", funcName, goTypeName)
		g.indent++

		// Try value assertion first (from direct struct construction)
		g.writef("if val, ok := v.(%s); ok {\n", goTypeName)
		g.indent++
		g.writef("return val\n")
		g.indent--
		g.writef("}\n")

		// Then pointer dereference (from FieldGet)
		g.writef("if ptr, ok := v.(*%s); ok {\n", goTypeName)
		g.indent++
		g.writef("return *ptr\n")
		g.indent--
		g.writef("}\n")

		// Panic on unexpected type
		g.writef("panic(fmt.Sprintf(\"%s: expected %s or *%s, got %%T\", v))\n", funcName, goTypeName, goTypeName)

		g.indent--
		g.writef("}\n\n")
	}
}

// markValueTypeConverterNeeded registers a value-type record that needs an AsTypeName converter.
// M-CODEGEN-VALUE-TYPES: Called when generating field assignments for value-type records.
func (g *Generator) markValueTypeConverterNeeded(goTypeName string) {
	g.valueTypeConverters[goTypeName] = true
}
