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
