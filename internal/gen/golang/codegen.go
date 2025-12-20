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
	"github.com/sunholo/ailang/internal/types"
)

// ADTConstructorInfo holds information about an ADT constructor.
type ADTConstructorInfo struct {
	TypeName   string   // The ADT type name (e.g., "Selection")
	CtorName   string   // The constructor name (e.g., "SelectionNone")
	GoFuncName string   // The Go constructor function name (e.g., "NewSelectionSelectionNone")
	FieldCount int      // Number of fields (0 for nullary constructors)
	FieldTypes []string // Go type strings for each field (e.g., ["int64", "float64"])
	FieldNames []string // Original field names (e.g., ["x", "y"]) - empty strings for positional fields
}

// RecordTypeInfo holds information about a record type for typed struct generation.
// M-DX13: Enables generating typed struct literals instead of map[string]interface{}.
type RecordTypeInfo struct {
	Name       string            // Go struct name (e.g., "World")
	Fields     []string          // Field names in order (e.g., ["Npcs", "Tiles", "Width", "Height"])
	FieldTypes map[string]string // Field name -> Go type (e.g., "Npcs" -> "[]*NPC")
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

	// adtSliceTypes tracks ADT type names that need slice converter functions
	// M-DX12: These are generated as ConvertTo<ADT>Slice() functions (exported)
	adtSliceTypes map[string]bool

	// recordTypes maps record type names to their info for typed struct generation
	// M-DX13: Enables generating &World{...} instead of map[string]interface{}{...}
	recordTypes map[string]*RecordTypeInfo

	// skipRuntimeHelpers skips generating runtime helpers (for multi-file compilation)
	skipRuntimeHelpers bool

	// coreTypeInfo maps Core expression NodeIDs to AILANG types
	// M-DX23: Used to generate typed function signatures
	coreTypeInfo types.CoreTypeInfo

	// expectedReturnType tracks the expected return type for current function
	// M-DX24: Used to generate type assertions at return boundaries
	expectedReturnType GoType

	// matchReturnType tracks the expected return type for current match expression
	// M-DX25.5: Used to generate type assertions in match arms
	matchReturnType string

	// matchScrutineeType tracks the Go type of the current match scrutinee
	// M-DX25.7: Used to generate typed list operations instead of interface{} helpers
	matchScrutineeType string

	// matchScrutineeAILANGType tracks the AILANG type of the current match scrutinee
	// M-DX29: Used to extract type arguments from generic types like Option[ADT]
	matchScrutineeAILANGType types.Type

	// inFlatChain tracks whether we're inside a flattened if-else chain
	// M-CODEGEN-FLAT-IF-ELSE: Prevents nested chains from re-wrapping in IIFEs
	inFlatChain bool

	// funcParamTypes maps function names to their Go parameter types
	// M-DX25.10: Used for call site type assertions when calling user-defined functions
	funcParamTypes map[string][]string

	// funcReturnTypes maps function names to their Go return types
	// M-DX25.10: Used to check if function call results need type assertions
	funcReturnTypes map[string]string

	// currentFuncParams maps parameter names to their Go types for the function being generated
	// M-DX25.10: Used to check if a Var reference is a param declared as interface{}
	currentFuncParams map[string]string

	// typedLocalVars maps local variable names to their concrete Go types
	// M-DX27: Used to track variables bound from typed ADT fields
	// When a variable is bound from an ADT field extraction (e.g., s := _adt.Foo.Bar),
	// the variable has a concrete type (bool, int64, etc.) not interface{}.
	typedLocalVars map[string]string

	// currentFuncDeclaredReturn stores the declared return type name for the current function
	// M-CROSS-MODULE: Used to resolve record literal types when CoreTypeInfo has structural types
	currentFuncDeclaredReturn string

	// funcTypeOverrides stores explicit function type signatures from AST annotations.
	// M-CODEGEN-TYPED-PARAMS: Used when CoreTypeInfo has unresolved structural types
	// that need to be mapped back to their declared nominal types.
	funcTypeOverrides map[string]*FuncTypeOverride

	// needsMathImport tracks whether generated code uses math package functions
	// M-CODEGEN-STDLIB-MATH: Set to true when mapPureMathBuiltin returns a match
	needsMathImport bool

	// needsStrconvImport tracks whether generated code uses strconv package functions
	// M-CODEGEN-STDLIB-STRING: Set to true when string conversion builtins are used
	needsStrconvImport bool

	// output buffer for generated code
	buf bytes.Buffer

	// indentation level
	indent int

	// varCounter for generating unique variable names
	// M-CODEGEN-LIST: Used for flattened list element bindings
	varCounter int

	// errors encountered during generation
	errors []error

	// verifyContracts enables runtime contract checking
	// M-VERIFY: When true, generates predicate checks for requires/ensures clauses
	verifyContracts bool

	// currentFuncName tracks the function name being generated for contract error messages
	// M-VERIFY: Used to look up contracts from prog.Meta and for error reporting
	currentFuncName string

	// moduleName is the current module's short name for function namespacing
	// M-DX18: Non-exported functions are prefixed with {moduleName}__ to prevent collisions
	// when compiling multiple modules to the same Go package.
	moduleName string
}

// FuncTypeOverride stores explicit function type signatures from AST annotations.
// M-CODEGEN-TYPED-PARAMS: Used to override inferred structural types with declared nominal types.
type FuncTypeOverride struct {
	ParamTypes []GoType
	ReturnType GoType
}

// SetSkipRuntimeHelpers controls whether runtime helpers are included in output.
// Use this when compiling multiple files to avoid duplicate declarations.
func (g *Generator) SetSkipRuntimeHelpers(skip bool) {
	g.skipRuntimeHelpers = skip
}

// RegisterFunctionType registers an explicit function type signature from AST.
// M-CODEGEN-TYPED-PARAMS: This ensures declared types (e.g., ArrivalState) are used
// instead of inferred structural types (TRecord{...}) which can cause cross-module contamination.
func (g *Generator) RegisterFunctionType(name string, paramTypes []GoType, returnType GoType) {
	if g.funcTypeOverrides == nil {
		g.funcTypeOverrides = make(map[string]*FuncTypeOverride)
	}
	g.funcTypeOverrides[name] = &FuncTypeOverride{
		ParamTypes: paramTypes,
		ReturnType: returnType,
	}
}

// SetCoreTypeInfo provides type information for generating typed function signatures.
// M-DX23: When set, functions will be generated with concrete parameter and return types
// instead of interface{}.
func (g *Generator) SetCoreTypeInfo(cti types.CoreTypeInfo) {
	g.coreTypeInfo = cti
}

// SetVerifyContracts enables runtime contract checking.
// M-VERIFY: When enabled, generates predicate checks for requires/ensures clauses.
func (g *Generator) SetVerifyContracts(enabled bool) {
	g.verifyContracts = enabled
}

// SetModuleName sets the module name for function namespacing.
// M-DX18: Non-exported functions will be prefixed with {moduleName}__ to prevent
// collisions when multiple modules are compiled to the same Go package.
// The moduleName should be the last component of the module path (e.g., "solar_demo"
// from "sim/solar_demo").
func (g *Generator) SetModuleName(name string) {
	g.moduleName = name
}

// GetModuleName returns the current module name used for function namespacing.
// M-DX18: Used for debugging and testing.
func (g *Generator) GetModuleName() string {
	return g.moduleName
}

// New creates a new Generator with the specified package name.
func New(packageName string) *Generator {
	g := &Generator{
		PackageName:       packageName,
		TypeMapper:        NewTypeMapper(),
		adtConstructors:   make(map[string]*ADTConstructorInfo),
		topLevelFuncs:     make(map[string]string),
		adtSliceTypes:     make(map[string]bool),
		recordTypes:       make(map[string]*RecordTypeInfo),
		funcParamTypes:    make(map[string][]string),
		funcReturnTypes:   make(map[string]string),
		currentFuncParams: make(map[string]string),
		typedLocalVars:    make(map[string]string),
	}
	// M-BUGFIX: Wire up record type lookup for TRecord -> named struct mapping
	g.TypeMapper.RecordTypeLookup = func(fields map[string]bool) (string, bool) {
		if info := g.GetRecordTypeByFields(fields); info != nil {
			return info.Name, true
		}
		return "", false
	}
	return g
}

// RegisterADTSliceType marks an ADT type as needing a slice converter function.
// M-DX12: This is called when encountering [ADT] in a record/struct field.
func (g *Generator) RegisterADTSliceType(typeName string) {
	g.adtSliceTypes[typeName] = true
}

// RegisterADTSliceTypes registers multiple ADT types for slice converter generation.
// M-DX12: Used to transfer types discovered during type generation to the code generator.
func (g *Generator) RegisterADTSliceTypes(types map[string]bool) {
	for typeName := range types {
		g.adtSliceTypes[typeName] = true
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
// Deprecated: Use RegisterADTConstructorFull for named field support.
func (g *Generator) RegisterADTConstructorWithTypes(typeName, ctorName string, fieldTypes []string) {
	g.RegisterADTConstructorFull(typeName, ctorName, fieldTypes, nil)
}

// RegisterADTConstructorFull registers an ADT constructor with field types and names.
// This enables proper field access in pattern matching with named fields.
func (g *Generator) RegisterADTConstructorFull(typeName, ctorName string, fieldTypes, fieldNames []string) {
	goFuncName := "New" + ToVariantStructName(typeName, ctorName)
	g.adtConstructors[ctorName] = &ADTConstructorInfo{
		TypeName:   typeName,
		CtorName:   ctorName,
		GoFuncName: goFuncName,
		FieldCount: len(fieldTypes),
		FieldTypes: fieldTypes,
		FieldNames: fieldNames,
	}
}

// RegisterRecordType registers a record type for typed struct literal generation.
// M-DX13: Enables generating &World{Field: val} instead of map[string]interface{}{...}.
// The fields slice should contain Go field names (PascalCase), and fieldTypes maps
// each Go field name to its Go type string.
func (g *Generator) RegisterRecordType(name string, fields []string, fieldTypes map[string]string) {
	g.recordTypes[name] = &RecordTypeInfo{
		Name:       name,
		Fields:     fields,
		FieldTypes: fieldTypes,
	}
}

// GetRecordTypeByFields looks up a record type by matching its field names.
// M-DX13: Used to infer the struct type from a record literal's fields.
// Returns nil if no matching record type is found.
// M-TYPENAME-NESTED-PROPAGATION: Also returns nil if multiple types match (ambiguous).
// Callers should use CoreTypeInfo TypeName when available for unambiguous selection.
func (g *Generator) GetRecordTypeByFields(fieldNames map[string]bool) *RecordTypeInfo {
	var matches []*RecordTypeInfo
	for _, info := range g.recordTypes {
		if len(info.Fields) != len(fieldNames) {
			continue
		}
		match := true
		for _, f := range info.Fields {
			// Convert Go field name (PascalCase) to AILANG field name (camelCase)
			ailangName := strings.ToLower(f[:1]) + f[1:]
			if !fieldNames[ailangName] {
				match = false
				break
			}
		}
		if match {
			matches = append(matches, info)
		}
	}
	// M-TYPENAME-NESTED-PROPAGATION: Return nil if ambiguous (multiple matches)
	// This forces callers to use more specific type information (CoreTypeInfo TypeName)
	if len(matches) == 1 {
		return matches[0]
	}
	return nil
}

// Generate produces Go source code from a Core program.
// Returns the formatted Go source code.
func (g *Generator) Generate(prog *core.Program) ([]byte, error) {
	g.buf.Reset()
	g.errors = nil
	g.indent = 0
	g.prog = prog                // Store for DeclMeta access
	g.needsMathImport = false    // Reset for each generation
	g.needsStrconvImport = false // Reset for each generation

	// M-CODEGEN-STDLIB-MATH: Two-phase generation to detect imports needed
	// Phase 1: Generate declarations to temporary buffer to detect math usage
	var declsBuf bytes.Buffer
	origBuf := g.buf
	g.buf = declsBuf

	for _, decl := range prog.Decls {
		if err := g.generateDecl(decl); err != nil {
			g.errors = append(g.errors, err)
		}
	}

	declsCode := g.buf.Bytes()
	g.buf = origBuf

	// Phase 2: Write header with correct imports, then append declarations
	g.writePackageHeader()
	g.buf.Write(declsCode)

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
		// Import fmt, reflect, strings for runtime helpers
		// M-DX16: reflect and strings needed for typed struct RecordUpdate
		// M-CODEGEN-STDLIB-MATH: math needed when using std/math functions
		// M-CODEGEN-STDLIB-STRING: strconv needed when using string conversion functions
		g.writef("import (\n")
		g.writef("\t\"fmt\"\n")
		if g.needsMathImport {
			g.writef("\t\"math\"\n")
		}
		g.writef("\t\"reflect\"\n")
		if g.needsStrconvImport {
			g.writef("\t\"strconv\"\n")
		}
		g.writef("\t\"strings\"\n")
		g.writef(")\n\n")
		g.writeRuntimeHelpers()
	} else if g.needsMathImport || g.needsStrconvImport {
		// M-CODEGEN-STDLIB-MATH/STRING: Even when skipping runtime helpers,
		// we need to add imports if the code uses stdlib functions
		g.writef("import (\n")
		if g.needsMathImport {
			g.writef("\t\"math\"\n")
		}
		if g.needsStrconvImport {
			g.writef("\t\"strconv\"\n")
		}
		g.writef(")\n\n")
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
	// M-DX16: reflect and strings needed for typed struct RecordUpdate
	g.writef("import (\n")
	g.writef("\t\"fmt\"\n")
	g.writef("\t\"reflect\"\n")
	g.writef("\t\"strings\"\n")
	g.writef(")\n\n")
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
// M-CODEGEN-UNIFIED-SLICE: Checks all type registries (primitives, ADTs, records).
func (g *Generator) getSliceConversion(goType string) string {
	switch goType {
	case "[]int64":
		return "ConvertToInt64Slice"
	case "[]float64":
		// M-CODEGEN-UNIFIED-SLICE: Float64 slice converter
		return "ConvertToFloat64Slice"
	case "[]string":
		return "ConvertToStringSlice"
	case "[]bool":
		// M-CODEGEN-BOOL-SLICE: Bool slice converter
		return "ConvertToBoolSlice"
	case "[]map[string]interface{}", "[]map[string]any":
		return "ConvertToRecordSlice"
	default:
		// M-CODEGEN-UNIFIED-SLICE: Check pointer slices against all type registries
		if strings.HasPrefix(goType, "[]*") {
			typeName := goType[3:] // Extract "Direction" from "[]*Direction"

			// Check ADT types (from adtConstructors - covers all ADTs)
			for _, info := range g.adtConstructors {
				if info.TypeName == typeName {
					return "ConvertTo" + typeName + "Slice"
				}
			}

			// Check record types
			if _, ok := g.recordTypes[typeName]; ok {
				return "ConvertTo" + typeName + "Slice"
			}

			// Legacy: check adtSliceTypes for backward compatibility
			if g.adtSliceTypes[typeName] {
				return "ConvertTo" + typeName + "Slice"
			}
		}
		return ""
	}
}
