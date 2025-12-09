// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeListHelpers writes list manipulation functions.
func (g *Generator) writeRuntimeListHelpers() {
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
}

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

	// Helper functions for Option - call ADT constructors for type safety
	// M-CODEGEN-OPTION-RUNTIME-HELPERS: Use NewOptionSome/NewOptionNone instead of maps
	g.writef("// makeOptionSome creates a Some value.\n")
	g.writef("// Calls ADT constructor for type-safe Option values.\n")
	g.writef("func makeOptionSome(v interface{}) interface{} {\n")
	g.indent++
	g.writef("return NewOptionSome(v)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// makeOptionNone creates a None value.\n")
	g.writef("// Calls ADT constructor for type-safe Option values.\n")
	g.writef("func makeOptionNone() interface{} {\n")
	g.indent++
	g.writef("return NewOptionNone()\n")
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
