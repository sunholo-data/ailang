// Package golang provides Go code generation from AILANG Core AST.
//
// This package implements Phase 1 (M-GAME-A) of the game support enablement:
// - ADT → Go struct generation with discriminator pattern
// - Pure function → Go function generation
// - Export keyword support
//
// Generated code uses discriminator structs for sum types (not interfaces)
// to ensure cache-friendly, branch-predictable layouts for game engines.
//
// Code is split across multiple files for maintainability:
// - codegen.go: Core types, initialization, and main generation loop
// - codegen_runtime.go: Runtime helper functions (Cons, RecordUpdate, arithmetic)
// - codegen_decl.go: Declaration generation (functions, variables)
// - codegen_expr.go: Expression generation (literals, lambdas, applications)
// - codegen_match.go: Pattern matching generation
// - codegen_ops.go: Binary/unary operations, records, lists, tuples
package golang

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// ADTConstructorInfo holds information about an ADT constructor.
type ADTConstructorInfo struct {
	TypeName   string   // The ADT type name (e.g., "Selection")
	CtorName   string   // The constructor name (e.g., "SelectionNone")
	GoFuncName string   // The Go constructor function name (e.g., "NewSelectionSelectionNone")
	FieldCount int      // Number of fields (0 for nullary constructors)
	FieldTypes []string // Go type strings for each field (e.g., ["int64", "float64"])
}

// Generator produces Go source code from AILANG Core AST.
type Generator struct {
	// PackageName is the Go package name for generated code
	PackageName string

	// TypeMapper handles AILANG → Go type conversions
	TypeMapper *TypeMapper

	// prog is the current program being generated (for accessing DeclMeta)
	prog *core.Program

	// adtConstructors maps constructor names to their info
	adtConstructors map[string]*ADTConstructorInfo

	// topLevelFuncs maps original function names to their Go names
	topLevelFuncs map[string]string

	// skipRuntimeHelpers skips generating runtime helpers (for multi-file compilation)
	skipRuntimeHelpers bool

	// output buffer for generated code
	buf bytes.Buffer

	// indentation level
	indent int

	// errors encountered during generation
	errors []error
}

// SetSkipRuntimeHelpers controls whether runtime helpers are included in output.
// Use this when compiling multiple files to avoid duplicate declarations.
func (g *Generator) SetSkipRuntimeHelpers(skip bool) {
	g.skipRuntimeHelpers = skip
}

// New creates a new Generator with the specified package name.
func New(packageName string) *Generator {
	return &Generator{
		PackageName:     packageName,
		TypeMapper:      NewTypeMapper(),
		adtConstructors: make(map[string]*ADTConstructorInfo),
		topLevelFuncs:   make(map[string]string),
	}
}

// RegisterADTConstructor registers an ADT constructor for proper code generation.
// This enables VarGlobal references to ADT constructors to generate the correct
// Go constructor function calls (e.g., NewSelectionSelectionNone() instead of SelectionNone).
// Deprecated: Use RegisterADTConstructorWithTypes for proper type assertions.
func (g *Generator) RegisterADTConstructor(typeName, ctorName string, fieldCount int) {
	goFuncName := "New" + ToVariantStructName(typeName, ctorName)
	g.adtConstructors[ctorName] = &ADTConstructorInfo{
		TypeName:   typeName,
		CtorName:   ctorName,
		GoFuncName: goFuncName,
		FieldCount: fieldCount,
		FieldTypes: nil, // No type info - will use interface{} without assertions
	}
}

// RegisterADTConstructorWithTypes registers an ADT constructor with field type information.
// This enables proper type assertions when calling constructors from generated code.
func (g *Generator) RegisterADTConstructorWithTypes(typeName, ctorName string, fieldTypes []string) {
	goFuncName := "New" + ToVariantStructName(typeName, ctorName)
	g.adtConstructors[ctorName] = &ADTConstructorInfo{
		TypeName:   typeName,
		CtorName:   ctorName,
		GoFuncName: goFuncName,
		FieldCount: len(fieldTypes),
		FieldTypes: fieldTypes,
	}
}

// Generate produces Go source code from a Core program.
// Returns the formatted Go source code.
func (g *Generator) Generate(prog *core.Program) ([]byte, error) {
	g.buf.Reset()
	g.errors = nil
	g.indent = 0
	g.prog = prog // Store for DeclMeta access

	// Write package header
	g.writePackageHeader()

	// Generate code from declarations
	// Core programs store declarations as a flat list with metadata
	for _, decl := range prog.Decls {
		if err := g.generateDecl(decl); err != nil {
			g.errors = append(g.errors, err)
		}
	}

	if len(g.errors) > 0 {
		return nil, fmt.Errorf("generation errors: %v", g.errors)
	}

	// Format the generated code
	return g.formatOutput()
}

// GenerateToWriter writes generated Go code to the provided writer.
func (g *Generator) GenerateToWriter(prog *core.Program, w io.Writer) error {
	code, err := g.Generate(prog)
	if err != nil {
		return err
	}
	_, err = w.Write(code)
	return err
}

// writePackageHeader writes the package declaration and runtime helpers.
func (g *Generator) writePackageHeader() {
	g.writef("// Code generated by ailang. DO NOT EDIT.\n")
	g.writef("package %s\n\n", g.PackageName)

	// Write runtime helpers (unless skipped for multi-file compilation)
	if !g.skipRuntimeHelpers {
		// Import fmt for Show, Log, Debug, ConcatString helpers
		g.writef("import \"fmt\"\n\n")
		g.writeRuntimeHelpers()
	}
}

// GenerateRuntime produces just the runtime helpers as a standalone Go file.
// Use this to generate a separate runtime.go when compiling multiple files.
func (g *Generator) GenerateRuntime() ([]byte, error) {
	g.buf.Reset()
	g.indent = 0

	g.writef("// Code generated by ailang. DO NOT EDIT.\n")
	g.writef("// Runtime helpers for AILANG generated code.\n")
	g.writef("package %s\n\n", g.PackageName)
	g.writef("import \"fmt\"\n\n")
	g.writeRuntimeHelpers()

	return g.formatOutput()
}

// Helper methods for output

func (g *Generator) write(s string) {
	g.buf.WriteString(strings.Repeat("\t", g.indent))
	g.buf.WriteString(s)
}

func (g *Generator) writef(format string, args ...interface{}) {
	g.buf.WriteString(strings.Repeat("\t", g.indent))
	fmt.Fprintf(&g.buf, format, args...)
}

func (g *Generator) formatOutput() ([]byte, error) {
	formatted, err := format.Source(g.buf.Bytes())
	if err != nil {
		// Return unformatted code with error for debugging
		return g.buf.Bytes(), fmt.Errorf("format error (raw code attached): %w", err)
	}
	return formatted, nil
}

// getSliceConversion returns the runtime conversion function name for a slice type.
// Returns empty string if not a slice type or if conversion is not supported.
// Only primitive slice types (int64, string, records) are converted.
// ADT slices (e.g., []*Direction) must be handled without conversion.
func (g *Generator) getSliceConversion(goType string) string {
	switch goType {
	case "[]int64":
		return "ConvertToInt64Slice"
	case "[]string":
		return "ConvertToStringSlice"
	case "[]map[string]interface{}", "[]map[string]any":
		return "ConvertToRecordSlice"
	default:
		// Don't convert ADT slices or other complex types
		// They stay as []interface{} and callers handle type assertions
		return ""
	}
}
