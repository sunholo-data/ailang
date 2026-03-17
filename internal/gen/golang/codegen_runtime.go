// Package golang provides Go code generation from AILANG Core AST.
package golang

// writeRuntimeHelpers writes utility functions needed by generated code.
// This is the main entry point that calls specialized helper generators.
func (g *Generator) writeRuntimeHelpers() {
	// Record and field operations (M-DX16, M-DX18, M-DX19, M-DX20, M-DX21)
	g.writeRuntimeRecordHelpers()

	// List operations (cons, head, tail, length, concat)
	g.writeRuntimeListHelpers()

	// Arithmetic, comparison, and type conversion functions
	g.writeRuntimeArithmeticHelpers()

	// Function calling helpers
	g.writeRuntimeMiscHelpers()

	// Slice type conversion functions
	g.writeRuntimeSliceConverters()

	// M-TYPE1: Array runtime functions
	g.writeArrayRuntimeFunctions()

	// M-DX12: Generate ADT slice converters for registered types
	g.writeADTSliceConverters()

	// M-CODEGEN-UNIFIED-SLICE: Generate record slice converters
	g.writeRecordSliceConverters()

	// M-CODEGEN-VALUE-TYPES: Generate value-type converters (AsTypeName)
	g.writeValueTypeConverters()

	// M-CODEGEN-STDLIB-BUILTINS: Stdlib function implementations (string ops, list HOFs)
	// TODO(M-CODEGEN-SUSTAINABILITY): Remove after migration to registry helpers is verified
	g.writeRuntimeStdlibHelpers()

	// M-CODEGEN-SUSTAINABILITY: Registry-generated helpers (replaces codegen_runtime_stdlib.go)
	g.writeRegistryHelpers()
}
