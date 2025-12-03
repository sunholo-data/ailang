package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	gen "github.com/sunholo/ailang/internal/gen/golang"
	"github.com/sunholo/ailang/internal/pipeline"
)

// compileCommand handles the 'ailang compile' subcommand
func compileCommand() {
	fs := flag.NewFlagSet("compile", flag.ExitOnError)

	// Output flags
	emitGoFlag := fs.Bool("emit-go", false, "Generate Go source code")
	outFlag := fs.String("out", "gen", "Output directory for generated files")
	packageNameFlag := fs.String("package-name", "", "Go package name (default: derived from module name)")
	releaseFlag := fs.Bool("release", false, "Release mode: erase Debug effect (zero-cost)")

	// Help flag
	helpFlag := fs.Bool("help", false, "Show help for compile command")
	fs.BoolVar(helpFlag, "h", false, "Show help for compile command")

	// Parse flags
	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *helpFlag {
		printCompileHelp()
		return
	}

	// Check for filename argument
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		printCompileHelp()
		os.Exit(1)
	}

	filename := fs.Arg(0)

	// Validate flags
	if !*emitGoFlag {
		fmt.Fprintf(os.Stderr, "%s: --emit-go is required (other backends not yet supported)\n", red("Error"))
		os.Exit(1)
	}

	// Read the source file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Check file extension
	if !strings.HasSuffix(filename, ".ail") {
		fmt.Fprintf(os.Stderr, "%s: file should have .ail extension\n", yellow("Warning"))
	}

	if *releaseFlag {
		fmt.Printf("%s Compiling %s (RELEASE MODE - Debug erased)\n", cyan("→"), filename)
	} else {
		fmt.Printf("%s Compiling %s\n", cyan("→"), filename)
	}

	// Run the pipeline to parse and type-check
	cfg := pipeline.Config{
		Mode: pipeline.ModeCheck, // Parse + type-check only, no eval
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: compilation failed: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(os.Stderr, "%s: errors:\n", red("Error"))
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
		os.Exit(1)
	}

	file := result.Artifacts.AST
	coreProg := result.Artifacts.Core

	// Determine package name
	pkgName := *packageNameFlag
	if pkgName == "" {
		// Derive from module name or filename
		if file != nil && file.Module != nil && file.Module.Path != "" {
			// Use last component of module path
			parts := strings.Split(file.Module.Path, "/")
			pkgName = sanitizePackageName(parts[len(parts)-1])
		} else {
			// Use filename without extension
			base := filepath.Base(filename)
			pkgName = sanitizePackageName(strings.TrimSuffix(base, ".ail"))
		}
	}

	fmt.Printf("%s Package name: %s\n", cyan("→"), pkgName)

	// Create output directory
	outDir := filepath.Join(*outFlag, pkgName)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot create output directory '%s': %v\n", red("Error"), outDir, err)
		os.Exit(1)
	}

	// Extract type declarations from AST (current module)
	var typeDecls []*ast.TypeDecl
	var externFuncs []*ast.FuncDecl
	if file != nil {
		for _, decl := range file.Decls {
			if td, ok := decl.(*ast.TypeDecl); ok {
				typeDecls = append(typeDecls, td)
			}
		}
		// Extract extern functions from Funcs
		for _, fn := range file.Funcs {
			if fn.IsExtern {
				externFuncs = append(externFuncs, fn)
			}
		}
	}

	// Extract type declarations from imported modules (cross-module ADT support)
	// Skip the current module to avoid duplicate type declarations
	currentModPath := ""
	if file != nil && file.Module != nil {
		currentModPath = file.Module.Path
	}
	for _, mod := range result.Modules {
		if mod.File != nil {
			// Skip current module (already extracted above)
			if mod.File.Module != nil && mod.File.Module.Path == currentModPath {
				continue
			}
			for _, decl := range mod.File.Decls {
				if td, ok := decl.(*ast.TypeDecl); ok {
					typeDecls = append(typeDecls, td)
				}
			}
		}
	}

	// Generate types from ADTs
	if len(typeDecls) > 0 {
		fmt.Printf("%s Generating types (%d type declarations)\n", cyan("→"), len(typeDecls))
		adtGen := gen.NewADTGenerator(pkgName)
		typesCode, err := adtGen.GenerateTypeDecls(typeDecls)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: type generation failed: %v\n", red("Error"), err)
			os.Exit(1)
		}

		typesFile := filepath.Join(outDir, "types.go")
		if err := os.WriteFile(typesFile, typesCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write types file: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Printf("%s Generated %s\n", green("✓"), typesFile)
	}

	// Generate Debug effect types with build tags for debug/release modes
	fmt.Printf("%s Generating Debug effect types (debug and release modes)\n", cyan("→"))
	debugGen := gen.NewDebugGenerator(pkgName)

	// Generate debug mode file (full implementation)
	debugCodeDebug, err := debugGen.GenerateDebugTypesDebug()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: debug types (debug) generation failed: %v\n", red("Error"), err)
		os.Exit(1)
	}
	debugFileDebug := filepath.Join(outDir, "debug_types_debug.go")
	if err := os.WriteFile(debugFileDebug, debugCodeDebug, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot write debug types file: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Generated %s\n", green("✓"), debugFileDebug)

	// Generate release mode file (no-op implementation)
	debugCodeRelease, err := debugGen.GenerateDebugTypesRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: debug types (release) generation failed: %v\n", red("Error"), err)
		os.Exit(1)
	}
	debugFileRelease := filepath.Join(outDir, "debug_types_release.go")
	if err := os.WriteFile(debugFileRelease, debugCodeRelease, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot write debug types file: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Generated %s\n", green("✓"), debugFileRelease)

	// Generate effect handlers (includes Debug, Rand, Clock)
	fmt.Printf("%s Generating effect handlers\n", cyan("→"))
	effectsGen := gen.NewEffectsGenerator(pkgName)
	handlers := []gen.EffectHandler{
		gen.DefaultDebugHandler(),
		gen.DefaultRandHandler(),
		gen.DefaultClockHandler(),
	}
	handlersCode, err := effectsGen.GenerateHandlers(handlers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: handlers generation failed: %v\n", red("Error"), err)
		os.Exit(1)
	}

	handlersFile := filepath.Join(outDir, "handlers.go")
	if err := os.WriteFile(handlersFile, handlersCode, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot write handlers file: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Generated %s\n", green("✓"), handlersFile)

	// Generate extern function stubs
	if len(externFuncs) > 0 {
		fmt.Printf("%s Generating extern stubs (%d extern functions)\n", cyan("→"), len(externFuncs))
		stubsCode := generateExternStubs(pkgName, externFuncs)

		stubsFile := filepath.Join(outDir, "extern_stubs.go")
		if err := os.WriteFile(stubsFile, stubsCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write stubs file: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Printf("%s Generated %s\n", green("✓"), stubsFile)
	}

	// Generate functions from Core AST
	// Note: Function generation is experimental and may produce incomplete code
	if coreProg != nil && len(coreProg.Decls) > 0 {
		fmt.Printf("%s Generating functions (%d declarations)\n", cyan("→"), len(coreProg.Decls))
		codeGen := gen.New(pkgName)

		// Register ADT constructors from current module (with field types for proper type assertions)
		for _, td := range typeDecls {
			if adt, ok := td.Definition.(*ast.AlgebraicType); ok {
				for _, ctor := range adt.Constructors {
					fieldTypes := extractFieldTypes(ctor.Fields)
					codeGen.RegisterADTConstructorWithTypes(td.Name, ctor.Name, fieldTypes)
				}
			}
		}

		// Register ADT constructors from imported modules (cross-module ADT support)
		for _, mod := range result.Modules {
			if mod.File != nil {
				for _, decl := range mod.File.Decls {
					if td, ok := decl.(*ast.TypeDecl); ok {
						if adt, ok := td.Definition.(*ast.AlgebraicType); ok {
							for _, ctor := range adt.Constructors {
								fieldTypes := extractFieldTypes(ctor.Fields)
								codeGen.RegisterADTConstructorWithTypes(td.Name, ctor.Name, fieldTypes)
							}
						}
					}
				}
			}
		}

		funcsCode, err := codeGen.Generate(coreProg)
		if err != nil {
			// Function generation is experimental - warn but continue
			fmt.Fprintf(os.Stderr, "%s: function generation skipped (experimental): %v\n", yellow("Warning"), err)
			fmt.Fprintf(os.Stderr, "  Type generation succeeded - use generated types in your Go code\n")
		} else {
			funcsFile := filepath.Join(outDir, "funcs.go")
			if err := os.WriteFile(funcsFile, funcsCode, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "%s: cannot write functions file: %v\n", red("Error"), err)
				os.Exit(1)
			}
			fmt.Printf("%s Generated %s\n", green("✓"), funcsFile)
		}
	}

	fmt.Printf("\n%s Compilation complete!\n", green("✓"))
	fmt.Printf("  Output: %s/\n", outDir)
	fmt.Printf("\n  %s Build commands:\n", cyan("→"))
	fmt.Printf("    Debug mode:   cd %s && go build\n", *outFlag)
	fmt.Printf("    Release mode: cd %s && go build -tags release\n", *outFlag)
}

// sanitizePackageName converts a string to a valid Go package name
func sanitizePackageName(name string) string {
	// Replace invalid characters
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A') // lowercase
		}
		if r == '_' || r == '-' {
			return '_'
		}
		return -1 // remove
	}, name)

	// Ensure doesn't start with number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "pkg" + result
	}

	if result == "" {
		result = "generated"
	}

	return result
}

func printCompileHelp() {
	fmt.Println(`Usage: ailang compile [options] <file.ail>

Compile AILANG source to other languages.

Options:
  --emit-go              Generate Go source code (required)
  --out <dir>            Output directory (default: "gen")
  --package-name <name>  Go package name (default: derived from module)
  --release              Mark this as a release build (info only)
  -h, --help             Show this help message

Examples:
  # Generate Go code from world.ail
  ailang compile --emit-go world.ail

  # Specify output directory and package name
  ailang compile --emit-go --out gen/ --package-name game world.ail

Output Structure:
  <out>/<package>/
  ├── types.go              # Generated ADT types
  ├── debug_types_debug.go  # Debug effect (full implementation, //go:build !release)
  ├── debug_types_release.go# Debug effect (no-ops, //go:build release)
  ├── handlers.go           # Effect handler interfaces (Debug, Rand, Clock)
  ├── extern_stubs.go       # Stubs for extern functions (implement these)
  └── funcs.go              # Generated functions (experimental)

Build Commands:
  Debug mode (default):  cd <out> && go build
  Release mode (no-op):  cd <out> && go build -tags release

The Debug effect uses Go build tags for zero-cost release builds.
In release mode, all Debug operations (Log, Assert) are no-ops.`)
}

// generateExternStubs generates Go stub code for extern functions
func generateExternStubs(pkgName string, funcs []*ast.FuncDecl) []byte {
	var buf strings.Builder

	buf.WriteString("// Code generated by ailang. DO NOT EDIT.\n")
	buf.WriteString("// Extern function stubs - implement these in your Go code.\n\n")
	buf.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	buf.WriteString("// TODO: Implement these extern functions.\n")
	buf.WriteString("// Copy the signatures below and add your implementation.\n")
	buf.WriteString("//\n")
	buf.WriteString("// Type mapping (AILANG -> Go):\n")
	buf.WriteString("//   int    -> int64\n")
	buf.WriteString("//   float  -> float64\n")
	buf.WriteString("//   string -> string\n")
	buf.WriteString("//   bool   -> bool\n")
	buf.WriteString("//   [T]    -> []T (slice)\n")
	buf.WriteString("//   { field: T } -> *TypeName (pointer to generated struct)\n")
	buf.WriteString("//\n\n")

	for _, fn := range funcs {
		// Generate doc comment with AILANG signature
		buf.WriteString(fmt.Sprintf("// %s is an extern function declared in AILANG.\n", fn.Name))
		buf.WriteString("//\n")
		buf.WriteString("// AILANG signature:\n")
		buf.WriteString(fmt.Sprintf("//   extern func %s(", fn.Name))
		for i, param := range fn.Params {
			if param.Name == "_" && param.Type.String() == "()" {
				continue
			}
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%s: %s", param.Name, param.Type.String()))
		}
		buf.WriteString(")")
		if fn.ReturnType != nil {
			buf.WriteString(fmt.Sprintf(" -> %s", fn.ReturnType.String()))
		}
		buf.WriteString("\n//\n")
		buf.WriteString("// Implement this function to provide the behavior.\n")
		buf.WriteString("// See docs/guides/go-interop.md for type mapping reference.\n")

		// Generate function signature
		buf.WriteString(fmt.Sprintf("func %s(", capitalize(fn.Name)))

		// Generate parameters
		for i, param := range fn.Params {
			if param.Name == "_" && param.Type.String() == "()" {
				// Skip unit parameters
				continue
			}
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("%s %s", param.Name, ailangTypeToGo(param.Type)))
		}
		buf.WriteString(")")

		// Generate return type
		if fn.ReturnType != nil {
			buf.WriteString(fmt.Sprintf(" %s", ailangTypeToGo(fn.ReturnType)))
		}

		buf.WriteString(" {\n")
		buf.WriteString("\tpanic(\"not implemented: ")
		buf.WriteString(fn.Name)
		buf.WriteString("\")\n")
		buf.WriteString("}\n\n")
	}

	return []byte(buf.String())
}

// capitalize makes the first letter uppercase (for exported Go functions)
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// isUserDefinedGoType returns true if the Go type is a user-defined type (ADT, struct, etc.)
// rather than a primitive type. Used to determine if slice elements need interface{} wrapping.
func isUserDefinedGoType(goType string) bool {
	switch goType {
	case "int64", "float64", "bool", "string", "interface{}", "struct{}",
		"map[string]interface{}", "map[string]any", "[]interface{}":
		return false
	default:
		// Pointers to types (e.g., *Direction) are user-defined
		// Named types (e.g., Direction, NPC) are user-defined
		return true
	}
}

// extractFieldTypes extracts Go type strings from AST constructor fields
func extractFieldTypes(fields []ast.Type) []string {
	types := make([]string, len(fields))
	for i, field := range fields {
		types[i] = ailangTypeToGo(field)
	}
	return types
}

// ailangTypeToGo converts an AILANG type to a Go type string
func ailangTypeToGo(t ast.Type) string {
	switch typ := t.(type) {
	case *ast.SimpleType:
		switch typ.Name {
		case "int":
			return "int64"
		case "float":
			return "float64"
		case "string":
			return "string"
		case "bool":
			return "bool"
		case "()":
			return "struct{}"
		default:
			// Assume it's a user-defined type
			return "*" + capitalize(typ.Name)
		}
	case *ast.ListType:
		elemType := ailangTypeToGo(typ.Element)
		// For ADT/user-defined element types, use interface{} to match adt.go
		// AILANG runtime passes []interface{}, not []*ADTType
		if isUserDefinedGoType(elemType) {
			return "interface{}"
		}
		return "[]" + elemType
	case *ast.RecordType:
		return "map[string]interface{}"
	default:
		return "interface{}"
	}
}
