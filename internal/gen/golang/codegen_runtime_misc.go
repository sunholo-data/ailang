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
	g.writef("case func(interface{}, interface{}, interface{}, interface{}) interface{}:\n")
	g.indent++
	g.writef("if len(args) >= 4 {\n")
	g.indent++
	g.writef("return fn(args[0], args[1], args[2], args[3])\n")
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

	// Show helper: the `show` spec (M-V1-SHOW-TOTAL), ported from
	// internal/eval/show.go — total (no width or depth limit), top-level
	// strings bare, nested strings quoted by internal/strutil's quoting rule,
	// copied in by emittedQuoteRuntime (which needs unicode/utf8, imported by
	// the runtime header). Always emitted: Println/Print/ToString helpers call
	// Show, so nearly every program reaches it; the quoting rule's NFC tables
	// add about 95 KB of source / 110 KB of binary to every compiled program.
	g.write(emittedShowRuntime)
	g.write(emittedQuoteRuntime)

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
