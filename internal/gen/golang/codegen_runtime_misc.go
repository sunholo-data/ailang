// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeMiscHelpers writes miscellaneous runtime functions (function calling, strings, effects).
func (g *Generator) writeRuntimeMiscHelpers() {
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

	// Show helper for converting values to canonical AILANG surface syntax.
	g.writef("// Show converts any value to canonical AILANG surface syntax.\n")
	g.writef("func Show(v interface{}) string {\n")
	g.indent++
	g.writef("return showValue(v, 0)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("const (\n")
	g.indent++
	g.writef("showMaxDepth = 3\n")
	g.writef("showMaxWidth = 80\n")
	g.writef("showElisionPrefix = 20\n")
	g.writef("showElisionSuffix = 20\n")
	g.indent--
	g.writef(")\n\n")

	g.writef("func showValue(v interface{}, depth int) string {\n")
	g.indent++
	g.writef("if depth > showMaxDepth { return \"...\" }\n")
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
	g.writef("if math.IsNaN(x) { return \"NaN\" }\n")
	g.writef("if math.IsInf(x, 1) { return \"Inf\" }\n")
	g.writef("if math.IsInf(x, -1) { return \"-Inf\" }\n")
	g.writef("s := strconv.FormatFloat(x, 'f', -1, 64)\n")
	g.writef("if !strings.Contains(s, \".\") { s += \".0\" }\n")
	g.writef("return s\n")
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
	g.writef("case ArrayVal:\n")
	g.indent++
	g.writef("return showSequence(reflect.ValueOf(x), depth, \"#[\", \"]\")\n")
	g.indent--
	g.writef("case Tuple:\n")
	g.indent++
	g.writef("return showSequence(reflect.ValueOf(x), depth, \"(\", \")\")\n")
	g.indent--
	g.writef("default:\n")
	g.indent++
	g.writef("return showReflect(reflect.ValueOf(x), depth)\n")
	g.indent--
	g.writef("}\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("func showReflect(v reflect.Value, depth int) string {\n")
	g.indent++
	g.writef("if depth > showMaxDepth { return \"...\" }\n")
	g.writef("if !v.IsValid() { return \"()\" }\n")
	g.writef("if v.Kind() == reflect.Interface {\n")
	g.indent++
	g.writef("if v.IsNil() { return \"()\" }\n")
	g.writef("return showValue(v.Elem().Interface(), depth)\n")
	g.indent--
	g.writef("}\n")
	g.writef("switch v.Kind() {\n")
	g.writef("case reflect.Ptr:\n")
	g.indent++
	g.writef("if v.IsNil() { return \"()\" }\n")
	g.writef("return showReflect(v.Elem(), depth)\n")
	g.indent--
	g.writef("case reflect.Slice, reflect.Array:\n")
	g.indent++
	g.writef("return showSequence(v, depth, \"[\", \"]\")\n")
	g.indent--
	g.writef("case reflect.Map:\n")
	g.indent++
	g.writef("return showMap(v, depth)\n")
	g.indent--
	g.writef("case reflect.Struct:\n")
	g.indent++
	g.writef("return showStruct(v, depth)\n")
	g.indent--
	g.writef("case reflect.Func:\n")
	g.indent++
	g.writef("return \"<function>\"\n")
	g.indent--
	g.writef("}\n")
	g.writef("return \"<unknown>\"\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("func showSequence(v reflect.Value, depth int, open, close string) string {\n")
	g.indent++
	g.writef("parts := make([]string, v.Len())\n")
	g.writef("for i := 0; i < v.Len(); i++ { parts[i] = showValue(v.Index(i).Interface(), depth+1) }\n")
	g.writef("return truncateShow(open + strings.Join(parts, \", \") + close)\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("func showMap(v reflect.Value, depth int) string {\n")
	g.indent++
	g.writef("if v.Type().Key().Kind() != reflect.String { return \"<map>\" }\n")
	g.writef("tag := v.MapIndex(reflect.ValueOf(\"_tag\"))\n")
	g.writef("if tag.IsValid() {\n")
	g.indent++
	g.writef("ctor, ok := tag.Interface().(string)\n")
	g.writef("if !ok { return \"<unknown>\" }\n")
	g.writef("keys := make([]string, 0, v.Len()-1)\n")
	g.writef("for _, key := range v.MapKeys() { if key.String() != \"_tag\" { keys = append(keys, key.String()) } }\n")
	g.writef("sort.Strings(keys)\n")
	g.writef("if len(keys) == 0 { return ctor }\n")
	g.writef("parts := make([]string, len(keys))\n")
	g.writef("for i, key := range keys { parts[i] = showValue(v.MapIndex(reflect.ValueOf(key)).Interface(), depth+1) }\n")
	g.writef("return ctor + \"(\" + strings.Join(parts, \", \") + \")\"\n")
	g.indent--
	g.writef("}\n")
	g.writef("// Go maps also back inline records. Map{...} is deliberate non-round-trippable debug notation;\n")
	g.writef("// this shared representation must retain record syntax here.\n")
	g.writef("keys := make([]string, 0, v.Len())\n")
	g.writef("for _, key := range v.MapKeys() { keys = append(keys, key.String()) }\n")
	g.writef("sort.Strings(keys)\n")
	g.writef("parts := make([]string, len(keys))\n")
	g.writef("for i, key := range keys { parts[i] = key + \": \" + showValue(v.MapIndex(reflect.ValueOf(key)).Interface(), depth+1) }\n")
	g.writef("return truncateShow(\"{\" + strings.Join(parts, \", \") + \"}\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("func showStruct(v reflect.Value, depth int) string {\n")
	g.indent++
	g.writef("if v.NumField() == 0 { return \"()\" }\n")
	g.writef("if kind := v.FieldByName(\"Kind\"); kind.IsValid() {\n")
	g.indent++
	g.writef("for i := 0; i < v.NumField(); i++ {\n")
	g.indent++
	g.writef("fieldInfo := v.Type().Field(i)\n")
	g.writef("field := v.Field(i)\n")
	g.writef("if fieldInfo.Name == \"Kind\" || field.Kind() != reflect.Ptr || field.IsNil() { continue }\n")
	g.writef("variant := field.Elem()\n")
	g.writef("parts := make([]string, variant.NumField())\n")
	g.writef("for j := 0; j < variant.NumField(); j++ { parts[j] = showValue(variant.Field(j).Interface(), depth+1) }\n")
	g.writef("if len(parts) == 0 { return fieldInfo.Name }\n")
	g.writef("return fieldInfo.Name + \"(\" + strings.Join(parts, \", \") + \")\"\n")
	g.indent--
	g.writef("}\n")
	g.writef("return \"<unknown>\"\n")
	g.indent--
	g.writef("}\n")
	g.writef("fields := make(map[string]reflect.Value, v.NumField())\n")
	g.writef("keys := make([]string, 0, v.NumField())\n")
	g.writef("for i := 0; i < v.NumField(); i++ {\n")
	g.indent++
	g.writef("fieldInfo := v.Type().Field(i)\n")
	g.writef("if fieldInfo.PkgPath != \"\" { continue }\n")
	g.writef("name := strings.ToLower(fieldInfo.Name[:1]) + fieldInfo.Name[1:]\n")
	g.writef("keys = append(keys, name)\n")
	g.writef("fields[name] = v.Field(i)\n")
	g.indent--
	g.writef("}\n")
	g.writef("sort.Strings(keys)\n")
	g.writef("parts := make([]string, len(keys))\n")
	g.writef("for i, key := range keys { parts[i] = key + \": \" + showValue(fields[key].Interface(), depth+1) }\n")
	g.writef("return truncateShow(\"{\" + strings.Join(parts, \", \") + \"}\")\n")
	g.indent--
	g.writef("}\n\n")

	g.writef("func truncateShow(s string) string {\n")
	g.indent++
	g.writef("if len(s) <= showMaxWidth { return s }\n")
	g.writef("if showElisionPrefix+showElisionSuffix+3 >= len(s) { return s }\n")
	g.writef("return s[:showElisionPrefix] + \"...\" + s[len(s)-showElisionSuffix:]\n")
	g.indent--
	g.writef("}\n\n")

	// ConcatString helper for string concatenation
	g.writef("// ConcatString concatenates two values as strings.\n")
	g.writef("func ConcatString(a, b interface{}) string {\n")
	g.indent++
	g.writef("return fmt.Sprintf(\"%%v%%v\", a, b)\n")
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
}
