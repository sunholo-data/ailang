// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeHelpers writes utility functions needed by generated code.
func (g *Generator) writeRuntimeHelpers() {
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

	g.writef("// Cons prepends an element to a list (cons operator).\n")
	g.writef("// AILANG: head :: tail\n")
	g.writef("func Cons(head interface{}, tail interface{}) []interface{} {\n")
	g.indent++
	g.writef("if tail == nil {\n")
	g.indent++
	g.writef("return []interface{}{head}\n")
	g.indent--
	g.writef("}\n")
	g.writef("list, ok := tail.([]interface{})\n")
	g.writef("if !ok {\n")
	g.indent++
	g.writef("return []interface{}{head}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return append([]interface{}{head}, list...)\n")
	g.indent--
	g.writef("}\n\n")

	// Integer arithmetic helpers (from dictionary resolution)
	g.writef("// AddInt adds two integers.\n")
	g.writef("func AddInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) + toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// SubInt subtracts two integers.\n")
	g.writef("func SubInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) - toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// MulInt multiplies two integers.\n")
	g.writef("func MulInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) * toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// DivInt divides two integers.\n")
	g.writef("func DivInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) / toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// ModInt returns integer modulo.\n")
	g.writef("func ModInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) %% toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Float arithmetic helpers
	g.writef("// AddFloat adds two floats.\n")
	g.writef("func AddFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) + toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// SubFloat subtracts two floats.\n")
	g.writef("func SubFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) - toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// MulFloat multiplies two floats.\n")
	g.writef("func MulFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) * toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// DivFloat divides two floats.\n")
	g.writef("func DivFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) / toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Type conversion helpers
	g.writef("// IntToFloat converts int to float.\n")
	g.writef("func IntToFloat(a interface{}) interface{} {\n")
	g.indent++
	g.writef("return float64(toInt64(a))\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// FloatToInt converts float to int.\n")
	g.writef("func FloatToInt(a interface{}) interface{} {\n")
	g.indent++
	g.writef("return int64(toFloat64(a))\n")
	g.indent--
	g.writef("}\n\n")

	// Comparison helpers
	g.writef("// EqInt compares two integers for equality.\n")
	g.writef("func EqInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) == toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// LtInt compares two integers (less than).\n")
	g.writef("func LtInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) < toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// LeInt compares two integers (less or equal).\n")
	g.writef("func LeInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) <= toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GtInt compares two integers (greater than).\n")
	g.writef("func GtInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) > toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GeInt compares two integers (greater or equal).\n")
	g.writef("func GeInt(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toInt64(a) >= toInt64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Negation helper
	g.writef("// NegInt negates an integer.\n")
	g.writef("func NegInt(a interface{}) interface{} {\n")
	g.indent++
	g.writef("return -toInt64(a)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// NegFloat negates a float.\n")
	g.writef("func NegFloat(a interface{}) interface{} {\n")
	g.indent++
	g.writef("return -toFloat64(a)\n")
	g.indent--
	g.writef("}\n\n")

	// M-BUGFIX: Float comparison helpers (missing in previous versions)
	g.writef("// LtFloat compares two floats (less than).\n")
	g.writef("func LtFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) < toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// LeFloat compares two floats (less or equal).\n")
	g.writef("func LeFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) <= toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GtFloat compares two floats (greater than).\n")
	g.writef("func GtFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) > toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GeFloat compares two floats (greater or equal).\n")
	g.writef("func GeFloat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return toFloat64(a) >= toFloat64(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Type conversion utility functions
	g.writef("// toInt64 converts interface{} to int64.\n")
	g.writef("func toInt64(v interface{}) int64 {\n")
	g.indent++
	g.writef("switch x := v.(type) {\n")
	g.writef("case int64:\n")
	g.indent++
	g.writef("return x\n")
	g.indent--
	g.writef("case int:\n")
	g.indent++
	g.writef("return int64(x)\n")
	g.indent--
	g.writef("case float64:\n")
	g.indent++
	g.writef("return int64(x)\n")
	g.indent--
	g.writef("default:\n")
	g.indent++
	g.writef("return 0\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// toFloat64 converts interface{} to float64.\n")
	g.writef("func toFloat64(v interface{}) float64 {\n")
	g.indent++
	g.writef("switch x := v.(type) {\n")
	g.writef("case float64:\n")
	g.indent++
	g.writef("return x\n")
	g.indent--
	g.writef("case int64:\n")
	g.indent++
	g.writef("return float64(x)\n")
	g.indent--
	g.writef("case int:\n")
	g.indent++
	g.writef("return float64(x)\n")
	g.indent--
	g.writef("default:\n")
	g.indent++
	g.writef("return 0\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n\n")

	// CallFunc helper for calling interface{} values as functions
	g.writef("// CallFunc calls an interface{} as a function with the given arguments.\n")
	g.writef("// Used for lambdas stored in variables.\n")
	g.writef("func CallFunc(f interface{}, args ...interface{}) interface{} {\n")
	g.indent++
	g.writef("switch fn := f.(type) {\n")
	g.writef("case func() interface{}:\n")
	g.indent++
	g.writef("return fn()\n")
	g.indent--
	g.writef("case func(interface{}) interface{}:\n")
	g.indent++
	g.writef("if len(args) >= 1 {\n")
	g.indent++
	g.writef("return fn(args[0])\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("case func(interface{}, interface{}) interface{}:\n")
	g.indent++
	g.writef("if len(args) >= 2 {\n")
	g.indent++
	g.writef("return fn(args[0], args[1])\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("case func(interface{}, interface{}, interface{}) interface{}:\n")
	g.indent++
	g.writef("if len(args) >= 3 {\n")
	g.indent++
	g.writef("return fn(args[0], args[1], args[2])\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("case func(...interface{}) interface{}:\n")
	g.indent++
	g.writef("return fn(args...)\n")
	g.indent--
	g.writef("}\n")
	g.writef("panic(\"CallFunc: unsupported function type\")\n")
	g.indent--
	g.writef("}\n\n")

	// ListHead helper for list destructuring
	// M-CODEGEN-TYPED-SLICES: Use reflection for typed slices (e.g., []*Tile)
	g.writef("// ListHead returns the first element of a list.\n")
	g.writef("// Handles both []interface{} and typed slices via reflection.\n")
	g.writef("func ListHead(list interface{}) interface{} {\n")
	g.indent++
	g.writef("// Fast path for []interface{}\n")
	g.writef("if l, ok := list.([]interface{}); ok && len(l) > 0 {\n")
	g.indent++
	g.writef("return l[0]\n")
	g.indent--
	g.writef("}\n")
	g.writef("// Reflection path for typed slices (e.g., []*Tile)\n")
	g.writef("v := reflect.ValueOf(list)\n")
	g.writef("if v.Kind() == reflect.Slice && v.Len() > 0 {\n")
	g.indent++
	g.writef("return v.Index(0).Interface()\n")
	g.indent--
	g.writef("}\n")
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n\n")

	// ListTail helper for list destructuring
	// M-CODEGEN-TYPED-SLICES: Use reflection for typed slices (e.g., []*Tile)
	g.writef("// ListTail returns all but the first element of a list.\n")
	g.writef("// Handles both []interface{} and typed slices via reflection.\n")
	g.writef("func ListTail(list interface{}) interface{} {\n")
	g.indent++
	g.writef("// Fast path for []interface{}\n")
	g.writef("if l, ok := list.([]interface{}); ok && len(l) > 0 {\n")
	g.indent++
	g.writef("return l[1:]\n")
	g.indent--
	g.writef("}\n")
	g.writef("// Reflection path for typed slices (e.g., []*Tile)\n")
	g.writef("v := reflect.ValueOf(list)\n")
	g.writef("if v.Kind() == reflect.Slice && v.Len() > 0 {\n")
	g.indent++
	g.writef("return v.Slice(1, v.Len()).Interface()\n")
	g.indent--
	g.writef("}\n")
	g.writef("return []interface{}{}\n")
	g.indent--
	g.writef("}\n\n")

	// ListLen helper for list length check
	// M-CODEGEN-TYPED-SLICES: Use reflection for typed slices (e.g., []*Tile)
	g.writef("// ListLen returns the length of a list.\n")
	g.writef("// Handles both []interface{} and typed slices via reflection.\n")
	g.writef("func ListLen(list interface{}) int {\n")
	g.indent++
	g.writef("// Fast path for []interface{}\n")
	g.writef("if l, ok := list.([]interface{}); ok {\n")
	g.indent++
	g.writef("return len(l)\n")
	g.indent--
	g.writef("}\n")
	g.writef("// Reflection path for typed slices (e.g., []*Tile)\n")
	g.writef("v := reflect.ValueOf(list)\n")
	g.writef("if v.Kind() == reflect.Slice {\n")
	g.indent++
	g.writef("return v.Len()\n")
	g.indent--
	g.writef("}\n")
	g.writef("return 0\n")
	g.indent--
	g.writef("}\n\n")

	// Show helper for converting values to strings
	g.writef("// Show converts any value to its string representation.\n")
	g.writef("func Show(v interface{}) string {\n")
	g.indent++
	g.writef("switch x := v.(type) {\n")
	g.writef("case string:\n")
	g.indent++
	g.writef("return x\n")
	g.indent--
	g.writef("case int64:\n")
	g.indent++
	g.writef("return fmt.Sprintf(\"%%d\", x)\n")
	g.indent--
	g.writef("case int:\n")
	g.indent++
	g.writef("return fmt.Sprintf(\"%%d\", x)\n")
	g.indent--
	g.writef("case float64:\n")
	g.indent++
	g.writef("return fmt.Sprintf(\"%%g\", x)\n")
	g.indent--
	g.writef("case bool:\n")
	g.indent++
	g.writef("if x { return \"true\" }\n")
	g.writef("return \"false\"\n")
	g.indent--
	g.writef("case nil:\n")
	g.indent++
	g.writef("return \"()\"\n")
	g.indent--
	g.writef("default:\n")
	g.indent++
	g.writef("return fmt.Sprintf(\"%%v\", x)\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n\n")

	// ConcatString helper for string concatenation
	g.writef("// ConcatString concatenates two values as strings.\n")
	g.writef("func ConcatString(a, b interface{}) string {\n")
	g.indent++
	g.writef("return fmt.Sprintf(\"%%v%%v\", a, b)\n")
	g.indent--
	g.writef("}\n\n")

	// Concat helper for list concatenation (++ operator)
	// M-CODEGEN-LIST-CONCAT: Handles []interface{} concatenation
	g.writef("// Concat concatenates two lists (++ operator).\n")
	g.writef("func Concat(a, b interface{}) interface{} {\n")
	g.indent++
	g.writef("if a == nil {\n")
	g.indent++
	g.writef("return b\n")
	g.indent--
	g.writef("}\n")
	g.writef("if b == nil {\n")
	g.indent++
	g.writef("return a\n")
	g.indent--
	g.writef("}\n")
	g.writef("sliceA, okA := a.([]interface{})\n")
	g.writef("sliceB, okB := b.([]interface{})\n")
	g.writef("if !okA || !okB {\n")
	g.indent++
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make([]interface{}, 0, len(sliceA)+len(sliceB))\n")
	g.writef("result = append(result, sliceA...)\n")
	g.writef("result = append(result, sliceB...)\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Log helper for IO effect
	g.writef("// Log prints a message and returns unit.\n")
	g.writef("func Log(msg interface{}) interface{} {\n")
	g.indent++
	g.writef("fmt.Println(msg)\n")
	g.writef("return struct{}{}\n")
	g.indent--
	g.writef("}\n\n")

	// Debug helper for Debug effect
	g.writef("// Debug prints a debug message and returns unit.\n")
	g.writef("func Debug(msg interface{}) interface{} {\n")
	g.indent++
	g.writef("fmt.Printf(\"[DEBUG] %%v\\n\", msg)\n")
	g.writef("return struct{}{}\n")
	g.indent--
	g.writef("}\n\n")

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

	// M-TYPE1: Array runtime functions
	g.writeArrayRuntimeFunctions()

	// M-DX12: Generate ADT slice converters for registered types
	g.writeADTSliceConverters()
}

// writeArrayRuntimeFunctions generates Go implementations for AILANG array operations.
// M-TYPE1: These are used when compiling std/array module functions to Go.
func (g *Generator) writeArrayRuntimeFunctions() {
	// FromList - creates array from list (both are []interface{} in Go)
	g.writef("// FromList creates an array from a list.\n")
	g.writef("// In Go, both arrays and lists are []interface{}.\n")
	g.writef("func FromList(xs interface{}) interface{} {\n")
	g.indent++
	g.writef("if xs == nil {\n")
	g.indent++
	g.writef("return []interface{}{}\n")
	g.indent--
	g.writef("}\n")
	g.writef("if list, ok := xs.([]interface{}); ok {\n")
	g.indent++
	g.writef("// Return a copy to preserve immutability\n")
	g.writef("result := make([]interface{}, len(list))\n")
	g.writef("copy(result, list)\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n")
	g.writef("return []interface{}{}\n")
	g.indent--
	g.writef("}\n\n")

	// ToList - converts array back to list (identity in Go)
	g.writef("// ToList converts an array to a list.\n")
	g.writef("// In Go, both are []interface{}, so this is essentially identity.\n")
	g.writef("func ToList(arr interface{}) interface{} {\n")
	g.indent++
	g.writef("if arr == nil {\n")
	g.indent++
	g.writef("return []interface{}{}\n")
	g.indent--
	g.writef("}\n")
	g.writef("if slice, ok := arr.([]interface{}); ok {\n")
	g.indent++
	g.writef("result := make([]interface{}, len(slice))\n")
	g.writef("copy(result, slice)\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n")
	g.writef("return []interface{}{}\n")
	g.indent--
	g.writef("}\n\n")

	// Length - returns array length
	g.writef("// Length returns the length of an array.\n")
	g.writef("func Length(arr interface{}) interface{} {\n")
	g.indent++
	g.writef("if arr == nil {\n")
	g.indent++
	g.writef("return int64(0)\n")
	g.indent--
	g.writef("}\n")
	g.writef("if slice, ok := arr.([]interface{}); ok {\n")
	g.indent++
	g.writef("return int64(len(slice))\n")
	g.indent--
	g.writef("}\n")
	g.writef("return int64(0)\n")
	g.indent--
	g.writef("}\n\n")

	// Get - gets element at index (panics if out of bounds)
	g.writef("// Get returns the element at the given index.\n")
	g.writef("// Panics if index is out of bounds.\n")
	g.writef("func Get(arr interface{}, idx interface{}) interface{} {\n")
	g.indent++
	g.writef("i := toInt64(idx)\n")
	g.writef("if slice, ok := arr.([]interface{}); ok {\n")
	g.indent++
	g.writef("if i < 0 || i >= int64(len(slice)) {\n")
	g.indent++
	g.writef("panic(fmt.Sprintf(\"array index out of bounds: %%d (length %%d)\", i, len(slice)))\n")
	g.indent--
	g.writef("}\n")
	g.writef("return slice[i]\n")
	g.indent--
	g.writef("}\n")
	g.writef("panic(\"Get: not an array\")\n")
	g.indent--
	g.writef("}\n\n")

	// GetOpt - safe get that returns Option[a]
	g.writef("// GetOpt safely returns the element at index, or None if out of bounds.\n")
	g.writef("// Returns Some(element) or None. Uses makeOptionSome/makeOptionNone helpers\n")
	g.writef("// which delegate to typed constructors if Option ADT is present.\n")
	g.writef("func GetOpt(arr interface{}, idx interface{}) interface{} {\n")
	g.indent++
	g.writef("i := toInt64(idx)\n")
	g.writef("if i < 0 {\n")
	g.indent++
	g.writef("return makeOptionNone()\n")
	g.indent--
	g.writef("}\n")
	g.writef("if slice, ok := arr.([]interface{}); ok {\n")
	g.indent++
	g.writef("if i >= int64(len(slice)) {\n")
	g.indent++
	g.writef("return makeOptionNone()\n")
	g.indent--
	g.writef("}\n")
	g.writef("return makeOptionSome(slice[i])\n")
	g.indent--
	g.writef("}\n")
	g.writef("return makeOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// Helper functions for Option - use map representation as fallback
	g.writef("// makeOptionSome creates a Some value.\n")
	g.writef("// Uses map representation for runtime compatibility.\n")
	g.writef("func makeOptionSome(v interface{}) interface{} {\n")
	g.indent++
	g.writef("return map[string]interface{}{\"tag\": \"Some\", \"value\": v}\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// makeOptionNone creates a None value.\n")
	g.writef("// Uses map representation for runtime compatibility.\n")
	g.writef("func makeOptionNone() interface{} {\n")
	g.indent++
	g.writef("return map[string]interface{}{\"tag\": \"None\"}\n")
	g.indent--
	g.writef("}\n\n")

	// UnsafeGet - unchecked access (no bounds check)
	g.writef("// UnsafeGet returns element at index without bounds checking.\n")
	g.writef("// Caller must ensure index is valid.\n")
	g.writef("func UnsafeGet(arr interface{}, idx interface{}) interface{} {\n")
	g.indent++
	g.writef("i := toInt64(idx)\n")
	g.writef("if slice, ok := arr.([]interface{}); ok {\n")
	g.indent++
	g.writef("return slice[i]\n")
	g.indent--
	g.writef("}\n")
	g.writef("panic(\"UnsafeGet: not an array\")\n")
	g.indent--
	g.writef("}\n\n")

	// Set - returns new array with element updated (immutable)
	g.writef("// Set returns a new array with the element at index updated.\n")
	g.writef("// Preserves immutability by creating a copy.\n")
	g.writef("func Set(arr interface{}, idx interface{}, val interface{}) interface{} {\n")
	g.indent++
	g.writef("i := toInt64(idx)\n")
	g.writef("if slice, ok := arr.([]interface{}); ok {\n")
	g.indent++
	g.writef("if i < 0 || i >= int64(len(slice)) {\n")
	g.indent++
	g.writef("panic(fmt.Sprintf(\"array index out of bounds: %%d (length %%d)\", i, len(slice)))\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make([]interface{}, len(slice))\n")
	g.writef("copy(result, slice)\n")
	g.writef("result[i] = val\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n")
	g.writef("panic(\"Set: not an array\")\n")
	g.indent--
	g.writef("}\n\n")

	// Make - creates array with default value
	g.writef("// Make creates an array of given size with all elements set to default.\n")
	g.writef("func Make(size interface{}, defaultVal interface{}) interface{} {\n")
	g.indent++
	g.writef("n := toInt64(size)\n")
	g.writef("if n < 0 {\n")
	g.indent++
	g.writef("n = 0\n")
	g.indent--
	g.writef("}\n")
	g.writef("result := make([]interface{}, n)\n")
	g.writef("for i := range result {\n")
	g.indent++
	g.writef("result[i] = defaultVal\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Note: NewOptionSome and NewOptionNone are generated by the ADT generator
	// when Option type is present in the module. GetOpt uses these functions.
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

		g.writef("// %s converts []interface{} to []*%s.\n", funcName, goTypeName)
		g.writef("// M-DX12: Fail-fast - panics on type mismatch (compiler bug detection).\n")
		g.writef("func %s(v interface{}) []*%s {\n", funcName, goTypeName)
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
		g.writef("return []*%s{}\n", goTypeName)
		g.indent--
		g.writef("}\n")

		// Convert elements
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

		g.indent--
		g.writef("}\n\n")
	}
}
