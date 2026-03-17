// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeStdlibHelpers writes stdlib function implementations needed by generated code.
// M-CODEGEN-STDLIB-BUILTINS: These map AILANG stdlib functions to Go implementations.
// The generated code calls these as PascalCase functions (e.g., Trim, Split, Map).
func (g *Generator) writeRuntimeStdlibHelpers() {
	g.writeStdlibStringHelpers()
	g.writeStdlibListHelpers()
	g.writeStdlibJsonHelpers()
}

// writeStdlibStringHelpers writes std/string function implementations.
func (g *Generator) writeStdlibStringHelpers() {
	g.writef("// ============================================================================\n")
	g.writef("// std/string runtime helpers\n")
	g.writef("// M-CODEGEN-STDLIB-BUILTINS: Generated from AILANG stdlib function signatures.\n")
	g.writef("// ============================================================================\n\n")

	// Trim — strings.TrimSpace
	g.writef("// Trim removes leading and trailing whitespace.\n")
	g.writef("func Trim(s interface{}) interface{} {\n")
	g.indent++
	g.writef("return strings.TrimSpace(s.(string))\n")
	g.indent--
	g.writef("}\n\n")

	// ToUpper
	g.writef("// ToUpper converts string to uppercase.\n")
	g.writef("func ToUpper(s interface{}) interface{} {\n")
	g.indent++
	g.writef("return strings.ToUpper(s.(string))\n")
	g.indent--
	g.writef("}\n\n")

	// ToLower
	g.writef("// ToLower converts string to lowercase.\n")
	g.writef("func ToLower(s interface{}) interface{} {\n")
	g.indent++
	g.writef("return strings.ToLower(s.(string))\n")
	g.indent--
	g.writef("}\n\n")

	// Contains
	g.writef("// Contains checks if string contains substring.\n")
	g.writef("func Contains(hay interface{}, needle interface{}) interface{} {\n")
	g.indent++
	g.writef("return strings.Contains(hay.(string), needle.(string))\n")
	g.indent--
	g.writef("}\n\n")

	// StartsWith
	g.writef("// StartsWith checks if string starts with prefix.\n")
	g.writef("func StartsWith(s interface{}, prefix interface{}) interface{} {\n")
	g.indent++
	g.writef("return strings.HasPrefix(s.(string), prefix.(string))\n")
	g.indent--
	g.writef("}\n\n")

	// EndsWith
	g.writef("// EndsWith checks if string ends with suffix.\n")
	g.writef("func EndsWith(s interface{}, suffix interface{}) interface{} {\n")
	g.indent++
	g.writef("return strings.HasSuffix(s.(string), suffix.(string))\n")
	g.indent--
	g.writef("}\n\n")

	// Find — returns int index (-1 if not found)
	g.writef("// Find returns index of substring (-1 if not found).\n")
	g.writef("func Find(hay interface{}, needle interface{}) interface{} {\n")
	g.indent++
	g.writef("return int64(strings.Index(hay.(string), needle.(string)))\n")
	g.indent--
	g.writef("}\n\n")

	// Compare
	g.writef("// Compare compares two strings lexicographically.\n")
	g.writef("func Compare(a interface{}, b interface{}) interface{} {\n")
	g.indent++
	g.writef("return int64(strings.Compare(a.(string), b.(string)))\n")
	g.indent--
	g.writef("}\n\n")

	// Split — returns []interface{} of strings
	g.writef("// Split splits string by delimiter, returns list of strings.\n")
	g.writef("func Split(s interface{}, delimiter interface{}) interface{} {\n")
	g.indent++
	g.writef("parts := strings.Split(s.(string), delimiter.(string))\n")
	g.writef("result := make([]interface{}, len(parts))\n")
	g.writef("for i, p := range parts {\n")
	g.indent++
	g.writef("result[i] = p\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Substring — s[start:end] with bounds clamping
	g.writef("// Substring extracts a substring from start to end index.\n")
	g.writef("func Substring(s interface{}, start interface{}, end interface{}) interface{} {\n")
	g.indent++
	g.writef("str := s.(string)\n")
	g.writef("st := int(toInt64(start))\n")
	g.writef("en := int(toInt64(end))\n")
	g.writef("if st < 0 { st = 0 }\n")
	g.writef("if en > len(str) { en = len(str) }\n")
	g.writef("if st > en { return \"\" }\n")
	g.writef("return str[st:en]\n")
	g.indent--
	g.writef("}\n\n")

	// Join — note: AILANG order is join(delimiter, xs), Go is Join(xs, delimiter)
	g.writef("// Join joins a list of strings with a delimiter.\n")
	g.writef("// AILANG: join(delimiter, xs). Go runtime takes same order.\n")
	g.writef("func Join(delimiter interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("strs := make([]string, len(list))\n")
	g.writef("for i, v := range list {\n")
	g.indent++
	g.writef("strs[i] = v.(string)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return strings.Join(strs, delimiter.(string))\n")
	g.indent--
	g.writef("}\n\n")

	// Chars — splits string into individual characters
	g.writef("// Chars splits string into list of single-character strings.\n")
	g.writef("func Chars(s interface{}) interface{} {\n")
	g.indent++
	g.writef("str := s.(string)\n")
	g.writef("result := make([]interface{}, 0, len(str))\n")
	g.writef("for _, r := range str {\n")
	g.indent++
	g.writef("result = append(result, string(r))\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Words — splits string on whitespace
	g.writef("// Words splits string into words (whitespace-separated).\n")
	g.writef("func Words(s interface{}) interface{} {\n")
	g.indent++
	g.writef("parts := strings.Fields(s.(string))\n")
	g.writef("result := make([]interface{}, len(parts))\n")
	g.writef("for i, p := range parts {\n")
	g.indent++
	g.writef("result[i] = p\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Repeat
	g.writef("// Repeat repeats a string n times.\n")
	g.writef("func Repeat(s interface{}, n interface{}) interface{} {\n")
	g.indent++
	g.writef("return strings.Repeat(s.(string), int(toInt64(n)))\n")
	g.indent--
	g.writef("}\n\n")

	// IntToStr
	g.writef("// IntToStr converts int to string.\n")
	g.writef("func IntToStr(n interface{}) interface{} {\n")
	g.indent++
	g.writef("return fmt.Sprintf(\"%%d\", toInt64(n))\n")
	g.indent--
	g.writef("}\n\n")

	// FloatToStr
	g.writef("// FloatToStr converts float to string.\n")
	g.writef("func FloatToStr(f interface{}) interface{} {\n")
	g.indent++
	g.writef("return fmt.Sprintf(\"%%g\", f.(float64))\n")
	g.indent--
	g.writef("}\n\n")

	// StringToInt — returns Option[int] (Some(n) or None)
	g.writef("// StringToInt parses string to int, returns Option (Some/None).\n")
	g.writef("func StringToInt(s interface{}) interface{} {\n")
	g.indent++
	g.writef("str := strings.TrimSpace(s.(string))\n")
	g.writef("n, err := strconv.ParseInt(str, 10, 64)\n")
	g.writef("if err != nil { return NewOptionNone() }\n")
	g.writef("return NewOptionSome(n)\n")
	g.indent--
	g.writef("}\n\n")

	// StringToFloat — returns Option[float]
	g.writef("// StringToFloat parses string to float, returns Option (Some/None).\n")
	g.writef("func StringToFloat(s interface{}) interface{} {\n")
	g.indent++
	g.writef("str := strings.TrimSpace(s.(string))\n")
	g.writef("f, err := strconv.ParseFloat(str, 64)\n")
	g.writef("if err != nil { return NewOptionNone() }\n")
	g.writef("return NewOptionSome(f)\n")
	g.indent--
	g.writef("}\n\n")

	// SplitAny
	g.writef("// SplitAny splits string on any of the given delimiters.\n")
	g.writef("func SplitAny(s interface{}, delimiters interface{}) interface{} {\n")
	g.indent++
	g.writef("str := s.(string)\n")
	g.writef("delims := toSlice(delimiters)\n")
	g.writef("// Build a FieldsFunc that splits on any delimiter\n")
	g.writef("delimSet := make(map[rune]bool)\n")
	g.writef("for _, d := range delims {\n")
	g.indent++
	g.writef("for _, r := range d.(string) {\n")
	g.indent++
	g.writef("delimSet[r] = true\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("parts := strings.FieldsFunc(str, func(r rune) bool { return delimSet[r] })\n")
	g.writef("result := make([]interface{}, len(parts))\n")
	g.writef("for i, p := range parts {\n")
	g.indent++
	g.writef("result[i] = p\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")
}

// writeStdlibListHelpers writes std/list function implementations.
func (g *Generator) writeStdlibListHelpers() {
	g.writef("// ============================================================================\n")
	g.writef("// std/list runtime helpers\n")
	g.writef("// M-CODEGEN-STDLIB-BUILTINS: Higher-order function implementations.\n")
	g.writef("// ============================================================================\n\n")

	// toSlice helper (may already exist — check and skip if so)
	g.writef("// toSlice converts interface{} to []interface{} slice.\n")
	g.writef("func toSlice(v interface{}) []interface{} {\n")
	g.indent++
	g.writef("if v == nil { return nil }\n")
	g.writef("if s, ok := v.([]interface{}); ok { return s }\n")
	g.writef("return nil\n")
	g.indent--
	g.writef("}\n\n")

	// Map
	g.writef("// Map applies function to each element of a list.\n")
	g.writef("func Map(f interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("result := make([]interface{}, len(list))\n")
	g.writef("for i, x := range list {\n")
	g.indent++
	g.writef("result[i] = CallFunc(f, x)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Filter
	g.writef("// Filter keeps elements where predicate returns true.\n")
	g.writef("func Filter(p interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("var result []interface{}\n")
	g.writef("for _, x := range list {\n")
	g.indent++
	g.writef("if CallFunc(p, x).(bool) {\n")
	g.indent++
	g.writef("result = append(result, x)\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("if result == nil { result = []interface{}{} }\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Foldl
	g.writef("// Foldl left-folds a list with an accumulator.\n")
	g.writef("func Foldl(f interface{}, acc interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("result := acc\n")
	g.writef("for _, x := range list {\n")
	g.indent++
	g.writef("result = CallFunc(f, result, x)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Foldr
	g.writef("// Foldr right-folds a list with an accumulator.\n")
	g.writef("func Foldr(f interface{}, acc interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("result := acc\n")
	g.writef("for i := len(list) - 1; i >= 0; i-- {\n")
	g.indent++
	g.writef("result = CallFunc(f, list[i], result)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Reverse
	g.writef("// Reverse reverses a list.\n")
	g.writef("func Reverse(xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("n := len(list)\n")
	g.writef("result := make([]interface{}, n)\n")
	g.writef("for i, v := range list {\n")
	g.indent++
	g.writef("result[n-1-i] = v\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Take
	g.writef("// Take returns the first n elements of a list.\n")
	g.writef("func Take(n interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("count := int(toInt64(n))\n")
	g.writef("if count > len(list) { count = len(list) }\n")
	g.writef("if count < 0 { count = 0 }\n")
	g.writef("result := make([]interface{}, count)\n")
	g.writef("copy(result, list[:count])\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Drop
	g.writef("// Drop removes the first n elements of a list.\n")
	g.writef("func Drop(n interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("count := int(toInt64(n))\n")
	g.writef("if count > len(list) { count = len(list) }\n")
	g.writef("if count < 0 { count = 0 }\n")
	g.writef("result := make([]interface{}, len(list)-count)\n")
	g.writef("copy(result, list[count:])\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Any
	g.writef("// Any returns true if predicate holds for any element.\n")
	g.writef("func Any(p interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("for _, x := range list {\n")
	g.indent++
	g.writef("if CallFunc(p, x).(bool) { return true }\n")
	g.indent--
	g.writef("}\n")
	g.writef("return false\n")
	g.indent--
	g.writef("}\n\n")

	// SortBy
	g.writef("// SortBy sorts a list using a comparison function.\n")
	g.writef("func SortBy(cmp interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("result := make([]interface{}, len(list))\n")
	g.writef("copy(result, list)\n")
	g.writef("sort.Slice(result, func(i, j int) bool {\n")
	g.indent++
	g.writef("return toInt64(CallFunc(cmp, result[i], result[j])) < 0\n")
	g.indent--
	g.writef("})\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// FlatMap
	g.writef("// FlatMap maps then flattens one level.\n")
	g.writef("func FlatMap(f interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("var result []interface{}\n")
	g.writef("for _, x := range list {\n")
	g.indent++
	g.writef("inner := toSlice(CallFunc(f, x))\n")
	g.writef("result = append(result, inner...)\n")
	g.indent--
	g.writef("}\n")
	g.writef("if result == nil { result = []interface{}{} }\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// Zip
	g.writef("// Zip pairs elements from two lists into tuples.\n")
	g.writef("func Zip(xs interface{}, ys interface{}) interface{} {\n")
	g.indent++
	g.writef("listX := toSlice(xs)\n")
	g.writef("listY := toSlice(ys)\n")
	g.writef("n := len(listX)\n")
	g.writef("if len(listY) < n { n = len(listY) }\n")
	g.writef("result := make([]interface{}, n)\n")
	g.writef("for i := 0; i < n; i++ {\n")
	g.indent++
	g.writef("result[i] = []interface{}{listX[i], listY[i]}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// FindIndex
	g.writef("// FindIndex returns Some(index) of first matching element, or None.\n")
	g.writef("func FindIndex(p interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("for i, x := range list {\n")
	g.indent++
	g.writef("if CallFunc(p, x).(bool) {\n")
	g.indent++
	g.writef("return NewOptionSome(int64(i))\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")
}

// writeStdlibJsonHelpers writes std/json function implementations.
func (g *Generator) writeStdlibJsonHelpers() {
	g.writef("// ============================================================================\n")
	g.writef("// std/json runtime helpers\n")
	g.writef("// M-CODEGEN-STDLIB-BUILTINS: JSON constructor and accessor helpers.\n")
	g.writef("// ============================================================================\n\n")

	// Js — creates JString variant
	g.writef("// Js creates a JString Json value.\n")
	g.writef("func Js(s interface{}) interface{} {\n")
	g.indent++
	g.writef("return NewJsonJString(s)\n")
	g.indent--
	g.writef("}\n\n")

	// Jn — creates JNull
	g.writef("// Jn creates a JNull Json value.\n")
	g.writef("func Jn() interface{} {\n")
	g.indent++
	g.writef("return NewJsonJNull()\n")
	g.indent--
	g.writef("}\n\n")

	// Jb — creates JBool
	g.writef("// Jb creates a JBool Json value.\n")
	g.writef("func Jb(b interface{}) interface{} {\n")
	g.indent++
	g.writef("return NewJsonJBool(b)\n")
	g.indent--
	g.writef("}\n\n")

	// Jnum — creates JNumber
	g.writef("// Jnum creates a JNumber Json value.\n")
	g.writef("func Jnum(x interface{}) interface{} {\n")
	g.indent++
	g.writef("return NewJsonJNumber(x)\n")
	g.indent--
	g.writef("}\n\n")

	// Ja — creates JArray
	g.writef("// Ja creates a JArray Json value.\n")
	g.writef("func Ja(xs interface{}) interface{} {\n")
	g.indent++
	g.writef("return NewJsonJArray(xs)\n")
	g.indent--
	g.writef("}\n\n")

	// Jo — creates JObject
	g.writef("// Jo creates a JObject Json value.\n")
	g.writef("func Jo(kvs interface{}) interface{} {\n")
	g.indent++
	g.writef("return NewJsonJObject(kvs)\n")
	g.indent--
	g.writef("}\n\n")

	// Kv — creates a {key: string, value: Json} record
	g.writef("// Kv creates a key-value pair for JSON objects.\n")
	g.writef("func Kv(k interface{}, v interface{}) interface{} {\n")
	g.indent++
	g.writef("return map[string]interface{}{\"key\": k, \"value\": v}\n")
	g.indent--
	g.writef("}\n\n")

	// Decode — parses JSON string into Json ADT, returns Result[Json, string]
	g.writef("// Decode parses a JSON string into a Json ADT value.\n")
	g.writef("// Returns Result: Ok(Json) or Err(errorMessage).\n")
	g.writef("func Decode(s interface{}) interface{} {\n")
	g.indent++
	g.writef("// Placeholder: JSON decode requires encoding/json integration.\n")
	g.writef("// For now, return Err indicating decode is not yet available in compiled mode.\n")
	g.writef("return NewResultErr(\"JSON decode not yet available in compiled Go mode\")\n")
	g.indent--
	g.writef("}\n\n")

	// Encode — serializes Json ADT to string
	g.writef("// Encode serializes a Json ADT value to a JSON string.\n")
	g.writef("func Encode(obj interface{}) interface{} {\n")
	g.indent++
	g.writef("// Placeholder: JSON encode requires walking the Json ADT.\n")
	g.writef("return \"{}\"\n")
	g.indent--
	g.writef("}\n\n")

	// AsString — extracts string from JString, returns Option[string]
	g.writef("// AsString extracts string from JString Json value.\n")
	g.writef("func AsString(j interface{}) interface{} {\n")
	g.indent++
	g.writef("json := j.(*Json)\n")
	g.writef("if json.Kind == JsonKindJString {\n")
	g.indent++
	g.writef("return NewOptionSome(json.JString.Field0)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// AsArray — extracts array from JArray, returns Option[List[Json]]
	g.writef("// AsArray extracts array from JArray Json value.\n")
	g.writef("func AsArray(j interface{}) interface{} {\n")
	g.indent++
	g.writef("json := j.(*Json)\n")
	g.writef("if json.Kind == JsonKindJArray {\n")
	g.indent++
	g.writef("return NewOptionSome(json.JArray.Field0)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// Get — looks up key in JObject, returns Option[Json]
	g.writef("// Get looks up a key in a JObject Json value.\n")
	g.writef("func JsonGet(obj interface{}, key interface{}) interface{} {\n")
	g.indent++
	g.writef("json := obj.(*Json)\n")
	g.writef("if json.Kind != JsonKindJObject { return NewOptionNone() }\n")
	g.writef("k := key.(string)\n")
	g.writef("kvs := toSlice(json.JObject.Field0)\n")
	g.writef("for _, kv := range kvs {\n")
	g.indent++
	g.writef("rec := kv.(map[string]interface{})\n")
	g.writef("if rec[\"Key\"] == k || rec[\"key\"] == k {\n")
	g.indent++
	g.writef("val := rec[\"Value\"]\n")
	g.writef("if val == nil { val = rec[\"value\"] }\n")
	g.writef("return NewOptionSome(val)\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// GetString — looks up key and extracts string, returns Option[string]
	g.writef("// GetString looks up key in JObject and extracts string value.\n")
	g.writef("func GetString(obj interface{}, key interface{}) interface{} {\n")
	g.indent++
	g.writef("opt := JsonGet(obj, key)\n")
	g.writef("if IsNone(opt).(bool) { return NewOptionNone() }\n")
	g.writef("return AsString(OptionGetOrElse(opt, nil))\n")
	g.indent--
	g.writef("}\n\n")

	// GetInt — looks up key and extracts int, returns Option[int]
	g.writef("// GetInt looks up key in JObject and extracts int value.\n")
	g.writef("func GetInt(obj interface{}, key interface{}) interface{} {\n")
	g.indent++
	g.writef("opt := JsonGet(obj, key)\n")
	g.writef("if IsNone(opt).(bool) { return NewOptionNone() }\n")
	g.writef("val := OptionGetOrElse(opt, nil)\n")
	g.writef("json := val.(*Json)\n")
	g.writef("if json.Kind == JsonKindJNumber {\n")
	g.indent++
	g.writef("return NewOptionSome(int64(json.JNumber.Field0.(float64)))\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// IsNone / IsSome for Option
	g.writef("// IsNone checks if an Option is None.\n")
	g.writef("func IsNone(opt interface{}) interface{} {\n")
	g.indent++
	g.writef("o := opt.(*Option)\n")
	g.writef("return o.Kind == OptionKindNone\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// IsSome checks if an Option is Some.\n")
	g.writef("func IsSome(opt interface{}) interface{} {\n")
	g.indent++
	g.writef("o := opt.(*Option)\n")
	g.writef("return o.Kind == OptionKindSome\n")
	g.indent--
	g.writef("}\n\n")

	// OptionGetOrElse
	g.writef("// OptionGetOrElse extracts value from Some, or returns default.\n")
	g.writef("func OptionGetOrElse(opt interface{}, defaultVal interface{}) interface{} {\n")
	g.indent++
	g.writef("o := opt.(*Option)\n")
	g.writef("if o.Kind == OptionKindSome { return o.Some.Field0 }\n")
	g.writef("return defaultVal\n")
	g.indent--
	g.writef("}\n\n")

	// GetArray — looks up key and extracts array, returns Option[List[Json]]
	g.writef("// GetArray looks up key in JObject and extracts array value.\n")
	g.writef("func GetArray(obj interface{}, key interface{}) interface{} {\n")
	g.indent++
	g.writef("opt := JsonGet(obj, key)\n")
	g.writef("if IsNone(opt).(bool) { return NewOptionNone() }\n")
	g.writef("return AsArray(OptionGetOrElse(opt, nil))\n")
	g.indent--
	g.writef("}\n\n")

	// JsonHas — checks if key exists in JObject
	g.writef("// JsonHas checks if a key exists in a JObject.\n")
	g.writef("func JsonHas(obj interface{}, key interface{}) interface{} {\n")
	g.indent++
	g.writef("opt := JsonGet(obj, key)\n")
	g.writef("return IsSome(opt)\n")
	g.indent--
	g.writef("}\n\n")

	// JsonGetOr — get with default
	g.writef("// JsonGetOr looks up key in JObject, returns default if not found.\n")
	g.writef("func JsonGetOr(obj interface{}, key interface{}, defaultVal interface{}) interface{} {\n")
	g.indent++
	g.writef("opt := JsonGet(obj, key)\n")
	g.writef("return OptionGetOrElse(opt, defaultVal)\n")
	g.indent--
	g.writef("}\n\n")

	// JsonKeys — get all keys from JObject
	g.writef("// JsonKeys returns all keys from a JObject.\n")
	g.writef("func JsonKeys(obj interface{}) interface{} {\n")
	g.indent++
	g.writef("json := obj.(*Json)\n")
	g.writef("if json.Kind != JsonKindJObject { return []interface{}{} }\n")
	g.writef("kvs := toSlice(json.JObject.Field0)\n")
	g.writef("result := make([]interface{}, len(kvs))\n")
	g.writef("for i, kv := range kvs {\n")
	g.indent++
	g.writef("rec := kv.(map[string]interface{})\n")
	g.writef("k := rec[\"Key\"]\n")
	g.writef("if k == nil { k = rec[\"key\"] }\n")
	g.writef("result[i] = k\n")
	g.indent--
	g.writef("}\n")
	g.writef("return result\n")
	g.indent--
	g.writef("}\n\n")

	// IsOk / IsErr for Result
	g.writef("// IsOk checks if a Result is Ok.\n")
	g.writef("func IsOk(r interface{}) interface{} {\n")
	g.indent++
	g.writef("res := r.(*Result)\n")
	g.writef("return res.Kind == ResultKindOk\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// IsErr checks if a Result is Err.\n")
	g.writef("func IsErr(r interface{}) interface{} {\n")
	g.indent++
	g.writef("res := r.(*Result)\n")
	g.writef("return res.Kind == ResultKindErr\n")
	g.indent--
	g.writef("}\n\n")

	// GetBool — looks up key and extracts bool, returns Option[bool]
	g.writef("// GetBool looks up key in JObject and extracts bool value.\n")
	g.writef("func GetBool(obj interface{}, key interface{}) interface{} {\n")
	g.indent++
	g.writef("opt := JsonGet(obj, key)\n")
	g.writef("if IsNone(opt).(bool) { return NewOptionNone() }\n")
	g.writef("val := OptionGetOrElse(opt, nil)\n")
	g.writef("json := val.(*Json)\n")
	g.writef("if json.Kind == JsonKindJBool {\n")
	g.indent++
	g.writef("return NewOptionSome(json.JBool.Field0)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// AsNumber — extracts number from JNumber, returns Option[float]
	g.writef("// AsNumber extracts number from JNumber Json value.\n")
	g.writef("func AsNumber(j interface{}) interface{} {\n")
	g.indent++
	g.writef("json := j.(*Json)\n")
	g.writef("if json.Kind == JsonKindJNumber {\n")
	g.indent++
	g.writef("return NewOptionSome(json.JNumber.Field0)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// AsBool — extracts bool from JBool, returns Option[bool]
	g.writef("// AsBool extracts bool from JBool Json value.\n")
	g.writef("func AsBool(j interface{}) interface{} {\n")
	g.indent++
	g.writef("json := j.(*Json)\n")
	g.writef("if json.Kind == JsonKindJBool {\n")
	g.indent++
	g.writef("return NewOptionSome(json.JBool.Field0)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// AsObject — extracts object entries, returns Option[List[{key, value}]]
	g.writef("// AsObject extracts entries from JObject Json value.\n")
	g.writef("func AsObject(j interface{}) interface{} {\n")
	g.indent++
	g.writef("json := j.(*Json)\n")
	g.writef("if json.Kind == JsonKindJObject {\n")
	g.indent++
	g.writef("return NewOptionSome(json.JObject.Field0)\n")
	g.indent--
	g.writef("}\n")
	g.writef("return NewOptionNone()\n")
	g.indent--
	g.writef("}\n\n")

	// Effect function placeholders — these need handler implementations
	// GetEnvOr — Env effect
	g.writef("// GetEnvOr gets environment variable with default. Requires Env handler.\n")
	g.writef("func GetEnvOr(key interface{}, defaultVal interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"GetEnvOr: Env effect not available in compiled mode - provide an Env handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// GetArgs — Env effect
	g.writef("// GetArgs returns command line arguments. Requires Env handler.\n")
	g.writef("func GetArgs() interface{} {\n")
	g.indent++
	g.writef("panic(\"GetArgs: Env effect not available in compiled mode - provide an Env handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// GetEnv — Env effect
	g.writef("// GetEnv gets environment variable. Requires Env handler.\n")
	g.writef("func GetEnv(key interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"GetEnv: Env effect not available in compiled mode - provide an Env handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// ReadFile — FS effect
	g.writef("// ReadFile reads a file as string. Requires FS handler.\n")
	g.writef("func ReadFile(path interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"ReadFile: FS effect not available in compiled mode - provide an FS handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// WriteFile — FS effect
	g.writef("// WriteFile writes data to a file. Requires FS handler.\n")
	g.writef("func WriteFile(path interface{}, data interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"WriteFile: FS effect not available in compiled mode - provide an FS handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// ReadFileBytes — FS effect
	g.writef("// ReadFileBytes reads file as bytes. Requires FS handler implementation.\n")
	g.writef("func ReadFileBytes(path interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"ReadFileBytes: FS effect not available in compiled mode - provide an FS handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// CallJsonSimple — AI effect
	g.writef("// CallJsonSimple calls AI with JSON response. Requires AI handler implementation.\n")
	g.writef("func CallJsonSimple(args ...interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"CallJsonSimple: AI effect not available in compiled mode - provide an AI handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// XML accessor stubs — these operate on the generated XmlNode ADT struct
	// Full implementations need the XML ADT struct shape which varies per project.
	// For now, they panic with clear messages.

	g.writef("// XmlFindAll finds all descendant elements matching tag name.\n")
	g.writef("func XmlFindAll(node interface{}, tag interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"XmlFindAll: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// XmlFindFirst finds first descendant element matching tag name.\n")
	g.writef("func XmlFindFirst(node interface{}, tag interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"XmlFindFirst: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// XmlGetText extracts text content from a node.\n")
	g.writef("func XmlGetText(node interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"XmlGetText: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// GetAttr gets attribute value by name from an XML node.\n")
	g.writef("func GetAttr(node interface{}, name interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"GetAttr: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// XmlGetChildren gets child nodes of an XML element.\n")
	g.writef("func XmlGetChildren(node interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"XmlGetChildren: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// XmlGetTag gets tag name of an XML element.\n")
	g.writef("func XmlGetTag(node interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"XmlGetTag: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// FindAllTexts finds all matching elements and extracts their text.\n")
	g.writef("func FindAllTexts(node interface{}, tag interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"FindAllTexts: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// FindAllAttrs finds all matching elements and extracts an attribute.\n")
	g.writef("func FindAllAttrs(node interface{}, tag interface{}, attr interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"FindAllAttrs: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("// XmlSerialize serializes an XmlNode tree to XML string.\n")
	g.writef("func XmlSerialize(node interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"XmlSerialize: XML operations not yet available in compiled mode\")\n")
	g.indent--
	g.writef("}\n\n")

	// XmlParse — XML parse effect placeholder
	g.writef("// XmlParse parses XML string. Requires XML handler implementation.\n")
	g.writef("func XmlParse(xml interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"XmlParse: XML parse not yet available in compiled mode - provide an XML handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// CallJson — AI effect with JSON response
	g.writef("// CallJson calls AI with JSON response format. Requires AI handler.\n")
	g.writef("func CallJson(args ...interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"CallJson: AI effect not available in compiled mode - provide an AI handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// JsonRepair — JSON repair
	g.writef("// JsonRepair attempts to repair malformed JSON.\n")
	g.writef("func JsonRepair(s interface{}) interface{} {\n")
	g.indent++
	g.writef("// Placeholder: returns the string unchanged\n")
	g.writef("return NewResultOk(s)\n")
	g.indent--
	g.writef("}\n\n")

	// Nth — returns element at index as Option
	g.writef("// Nth returns element at index, or None if out of bounds.\n")
	g.writef("func Nth(xs interface{}, idx interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("i := int(toInt64(idx))\n")
	g.writef("if i < 0 || i >= len(list) { return NewOptionNone() }\n")
	g.writef("return NewOptionSome(list[i])\n")
	g.indent--
	g.writef("}\n\n")

	// Println — IO effect
	g.writef("// Println prints a value to stdout.\n")
	g.writef("func Println(v interface{}) interface{} {\n")
	g.indent++
	g.writef("fmt.Println(Show(v))\n")
	g.writef("return struct{}{}\n")
	g.indent--
	g.writef("}\n\n")

	// Last — returns last element of list as Option
	g.writef("// Last returns the last element of a list, or None if empty.\n")
	g.writef("func Last(xs interface{}) interface{} {\n")
	g.indent++
	g.writef("list := toSlice(xs)\n")
	g.writef("if len(list) == 0 { return NewOptionNone() }\n")
	g.writef("return NewOptionSome(list[len(list)-1])\n")
	g.indent--
	g.writef("}\n\n")

	// MapE — effectful map (same as Map for compiled mode — effects are panics)
	g.writef("// MapE is the effectful version of Map. In compiled mode, behaves like Map.\n")
	g.writef("func MapE(f interface{}, xs interface{}) interface{} {\n")
	g.indent++
	g.writef("return Map(f, xs)\n")
	g.indent--
	g.writef("}\n\n")

	// ReadEntry — zip effect
	g.writef("// ReadEntry reads a zip entry. Requires FS/zip handler.\n")
	g.writef("func ReadEntry(args ...interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"ReadEntry: zip effect not available in compiled mode - provide a zip handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// ListEntries — zip effect
	g.writef("// ListEntries lists entries in a zip archive. Requires FS/zip handler.\n")
	g.writef("func ListEntries(args ...interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"ListEntries: zip effect not available in compiled mode - provide a zip handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// CreateArchive — zip/archive effect
	g.writef("// CreateArchive creates a zip archive. Requires FS handler.\n")
	g.writef("func CreateArchive(path interface{}, entries interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"CreateArchive: FS/zip effect not available in compiled mode - provide an FS handler\")\n")
	g.indent--
	g.writef("}\n\n")

	// Call — AI effect placeholder
	g.writef("// Call is a placeholder for AI effect function calls.\n")
	g.writef("// In compiled mode, AI effects need a handler implementation.\n")
	g.writef("func Call(args ...interface{}) interface{} {\n")
	g.indent++
	g.writef("panic(\"AI Call effect not available in compiled mode - provide an AI handler\")\n")
	g.indent--
	g.writef("}\n\n")
}
