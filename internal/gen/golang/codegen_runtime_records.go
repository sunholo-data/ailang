// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeRecordHelpers writes record and field manipulation functions.
// M-DX16: RecordUpdate preserves typed structs using reflection.
// M-DX18: FieldGet handles both maps and typed structs.
func (g *Generator) writeRuntimeRecordHelpers() {
	// M-DX16: RecordUpdate must preserve typed structs, not convert to map
	g.writef("// RecordUpdate creates a new record with specified fields updated.\n")
	g.writef("// AILANG: { base | field1: val1, field2: val2 }\n")
	g.writef("// M-DX16: Preserves typed structs using reflection.\n")
	g.writef("func RecordUpdate(base interface{}, updates map[string]interface{}) interface{} {\n")
	g.indent++

	// Handle map case (original behavior)
	g.writef("// Handle map[string]interface{} (original behavior)\n")
	g.writef("if baseMap, ok := base.(map[string]interface{}); ok {\n")
	g.indent++
	g.writef("result := make(map[string]interface{}, len(baseMap)+len(updates))\n")
	g.writef("for k, v := range baseMap {\n")
	g.indent++
	g.writef("result[k] = v\n")
	g.indent--
	g.writef("}\n")
	g.writef("for k, v := range updates {\n")
	g.indent++
	g.writef("result[k] = v\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Handle typed structs using reflection
	g.writef("// Handle typed structs using reflection\n")
	g.writef("baseVal := reflect.ValueOf(base)\n")
	g.writef("if baseVal.Kind() == reflect.Ptr && baseVal.Elem().Kind() == reflect.Struct {\n")
	g.indent++

	// Create a new instance of the same type
	g.writef("// Create a new instance and copy all fields\n")
	g.writef("newPtr := reflect.New(baseVal.Elem().Type())\n")
	g.writef("newVal := newPtr.Elem()\n")
	g.writef("oldVal := baseVal.Elem()\n\n")

	// Copy all fields from base
	g.writef("// Copy all fields from base\n")
	g.writef("for i := 0; i < oldVal.NumField(); i++ {\n")
	g.indent++
	g.writef("if newVal.Field(i).CanSet() {\n")
	g.indent++
	g.writef("newVal.Field(i).Set(oldVal.Field(i))\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n\n")

	// Apply updates
	g.writef("// Apply updates (field names are lowercase in AILANG, PascalCase in Go)\n")
	g.writef("for fieldName, val := range updates {\n")
	g.indent++
	g.writef("// Convert field name to PascalCase\n")
	g.writef("goFieldName := strings.ToUpper(fieldName[:1]) + fieldName[1:]\n")
	g.writef("field := newVal.FieldByName(goFieldName)\n")
	g.writef("if field.IsValid() && field.CanSet() {\n")
	g.indent++
	g.writef("// Handle type conversion\n")
	g.writef("valReflect := reflect.ValueOf(val)\n")
	g.writef("if valReflect.Type().AssignableTo(field.Type()) {\n")
	g.indent++
	g.writef("field.Set(valReflect)\n")
	g.indent--

	// M-DX20: Handle pointer to value conversion
	g.writef("} else if valReflect.Kind() == reflect.Ptr && field.Kind() != reflect.Ptr {\n")
	g.indent++
	g.writef("// M-DX20: Dereference pointer if field expects value type\n")
	g.writef("if valReflect.Elem().Type().AssignableTo(field.Type()) {\n")
	g.indent++
	g.writef("field.Set(valReflect.Elem())\n")
	g.indent--
	g.writef("}\n")
	g.indent--

	// M-DX19: Handle slice conversion via reflection
	g.writef("} else if valReflect.Kind() == reflect.Slice && field.Kind() == reflect.Slice {\n")
	g.indent++
	g.writef("// M-DX19: Convert []interface{} to typed slice via reflection\n")
	g.writef("elemType := field.Type().Elem()\n")
	g.writef("newSlice := reflect.MakeSlice(field.Type(), valReflect.Len(), valReflect.Len())\n")
	g.writef("for i := 0; i < valReflect.Len(); i++ {\n")
	g.indent++
	g.writef("elem := valReflect.Index(i)\n")
	g.writef("// Handle interface{} elements\n")
	g.writef("if elem.Kind() == reflect.Interface {\n")
	g.indent++
	g.writef("elem = elem.Elem()\n")
	g.indent--
	g.writef("}\n")
	g.writef("if elem.Type().AssignableTo(elemType) {\n")
	g.indent++
	g.writef("newSlice.Index(i).Set(elem)\n")
	g.indent--
	g.writef("} else if elem.Type().ConvertibleTo(elemType) {\n")
	g.indent++
	g.writef("newSlice.Index(i).Set(elem.Convert(elemType))\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("field.Set(newSlice)\n")
	g.indent--

	g.writef("} else if valReflect.Type().ConvertibleTo(field.Type()) {\n")
	g.indent++
	g.writef("field.Set(valReflect.Convert(field.Type()))\n")
	g.indent--
	g.writef("} else {\n")
	g.indent++
	g.writef("// Try to set directly for interface{} values\n")
	g.writef("field.Set(valReflect)\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return newPtr.Interface()\n")
	g.indent--
	g.writef("}\n\n")

	// Fallback: return updates as map
	g.writef("// Fallback: create map from updates\n")
	g.writef("return updates\n")
	g.indent--
	g.writef("}\n\n")

	// M-DX18: FieldGet helper for accessing fields on maps or typed structs
	g.writef("// FieldGet retrieves a field from a record (map or typed struct).\n")
	g.writef("// M-DX18: Handles both map[string]interface{} and typed structs.\n")
	g.writef("func FieldGet(record interface{}, field string) interface{} {\n")
	g.indent++

	// Handle map case
	g.writef("// Handle map[string]interface{}\n")
	g.writef("if m, ok := record.(map[string]interface{}); ok {\n")
	g.indent++
	g.writef("return m[field]\n")
	g.indent--
	g.writef("}\n\n")

	// Handle typed struct using reflection
	g.writef("// Handle typed struct using reflection\n")
	g.writef("val := reflect.ValueOf(record)\n")
	g.writef("if val.Kind() == reflect.Ptr {\n")
	g.indent++
	g.writef("val = val.Elem()\n")
	g.indent--
	g.writef("}\n")
	g.writef("if val.Kind() == reflect.Struct {\n")
	g.indent++
	g.writef("// Convert field name to PascalCase (AILANG lowercase -> Go PascalCase)\n")
	g.writef("goField := strings.ToUpper(field[:1]) + field[1:]\n")
	g.writef("f := val.FieldByName(goField)\n")
	g.writef("if f.IsValid() {\n")
	g.indent++
	g.writef("// M-DX21: Return pointer for struct-typed fields (AILANG expects *Struct)\n")
	g.writef("if f.Kind() == reflect.Struct && f.CanAddr() {\n")
	g.indent++
	g.writef("return f.Addr().Interface()\n")
	g.indent--
	g.writef("}\n")
	g.writef("return f.Interface()\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n\n")
}
