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
	verifyContractsFlag := fs.Bool("verify-contracts", false, "M-VERIFY: Generate runtime contract checks")
	relaxModulesFlag := fs.Bool("relax-modules", false, "Relax MOD010 validation (allow module path mismatches)")
	// M-CODEGEN-VALUE-TYPES: Control value vs pointer generation for small leaf records
	valueThresholdFlag := fs.Int("value-threshold", 4, "Max fields for value-type records (0 = all pointers, v0.5.9 behavior)")

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

	// Check for filename arguments
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
		printCompileHelp()
		os.Exit(1)
	}

	// Expand directories to .ail files
	filenames := expandFilenames(fs.Args())

	// Validate flags
	if !*emitGoFlag {
		fmt.Fprintf(os.Stderr, "%s: --emit-go is required (other backends not yet supported)\n", red("Error"))
		os.Exit(1)
	}

	// Check AILANG_RELAX_MODULES environment variable
	relaxModulesEffective := *relaxModulesFlag
	if envVal := os.Getenv("AILANG_RELAX_MODULES"); envVal != "" {
		switch strings.ToLower(envVal) {
		case "1", "true", "yes":
			relaxModulesEffective = true
		}
	}

	// M-CODEGEN-VALUE-TYPES: Validate value threshold
	valueThreshold := *valueThresholdFlag
	if valueThreshold < 0 {
		fmt.Fprintf(os.Stderr, "%s: --value-threshold cannot be negative (%d), using 0 (all pointers)\n", yellow("Warning"), valueThreshold)
		valueThreshold = 0
	}

	// Accumulated data from all files
	var allTypeDecls []*ast.TypeDecl
	var allExternFuncs []*ast.FuncDecl
	var allFuncs []*ast.FuncDecl // M-CODEGEN-TYPED-PARAMS: All non-extern functions for type registration
	var allCoreDecls []pipeline.Result
	var pkgName string
	seenTypes := make(map[string]bool) // Track types to avoid duplicates

	// Process each file
	for _, filename := range filenames {
		// Read the source file
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
			os.Exit(1)
		}

		// Check file extension
		if !strings.HasSuffix(filename, ".ail") {
			fmt.Fprintf(os.Stderr, "%s: file '%s' should have .ail extension\n", yellow("Warning"), filename)
		}

		if *releaseFlag {
			fmt.Printf("%s Compiling %s (RELEASE MODE - Debug erased)\n", cyan("→"), filename)
		} else {
			fmt.Printf("%s Compiling %s\n", cyan("→"), filename)
		}

		// Run the pipeline to parse and type-check
		cfg := pipeline.Config{
			Mode:         pipeline.ModeCheck, // Parse + type-check only, no eval
			RelaxModules: relaxModulesEffective,
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

		// Determine package name from first file (or use flag)
		if pkgName == "" {
			pkgName = *packageNameFlag
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
		}

		// Extract type declarations from AST (current module)
		if file != nil {
			for _, decl := range file.Decls {
				if td, ok := decl.(*ast.TypeDecl); ok {
					if !seenTypes[td.Name] {
						allTypeDecls = append(allTypeDecls, td)
						seenTypes[td.Name] = true
					}
				}
			}
			// Extract extern functions from Funcs
			for _, fn := range file.Funcs {
				if fn.IsExtern {
					allExternFuncs = append(allExternFuncs, fn)
				} else {
					// M-CODEGEN-TYPED-PARAMS: Collect non-extern functions for type registration
					allFuncs = append(allFuncs, fn)
				}
			}
		}

		// Extract type declarations from imported modules (cross-module ADT support)
		currentModPath := ""
		if file != nil && file.Module != nil {
			currentModPath = file.Module.Path
		}
		for _, mod := range result.Modules {
			if mod.File != nil {
				if mod.File.Module != nil && mod.File.Module.Path == currentModPath {
					continue
				}
				for _, decl := range mod.File.Decls {
					if td, ok := decl.(*ast.TypeDecl); ok {
						if !seenTypes[td.Name] {
							allTypeDecls = append(allTypeDecls, td)
							seenTypes[td.Name] = true
						}
					}
				}
			}
		}

		// Store result for later function generation
		allCoreDecls = append(allCoreDecls, result)
	}

	fmt.Printf("%s Package name: %s\n", cyan("→"), pkgName)
	fmt.Printf("%s Processed %d file(s)\n", cyan("→"), len(filenames))

	// Create output directory
	// M-CODEGEN-OUTPUT-PATH: Don't create nested directory if --out already matches package name
	outDir := *outFlag
	if filepath.Base(*outFlag) != pkgName {
		outDir = filepath.Join(*outFlag, pkgName)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot create output directory '%s': %v\n", red("Error"), outDir, err)
		os.Exit(1)
	}

	// M-DX12: Track ADT types used in list fields for converter generation
	var adtSliceTypes map[string]bool

	// Generate types from ADTs (accumulated from all files)
	if len(allTypeDecls) > 0 {
		fmt.Printf("%s Generating types (%d type declarations)\n", cyan("→"), len(allTypeDecls))
		adtGen := gen.NewADTGenerator(pkgName)
		// M-CODEGEN-VALUE-TYPES: Set value threshold for struct field generation
		adtGen.SetValueThreshold(valueThreshold)
		typesCode, err := adtGen.GenerateTypeDecls(allTypeDecls)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: type generation failed: %v\n", red("Error"), err)
			os.Exit(1)
		}

		// M-DX12: Capture ADT slice types for converter generation
		adtSliceTypes = adtGen.GetADTSliceTypes()

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

	// Generate effect handlers (includes Debug, Rand, Clock, FS, Net, Env, AI)
	fmt.Printf("%s Generating effect handlers\n", cyan("→"))
	effectsGen := gen.NewEffectsGenerator(pkgName)
	handlers := []gen.EffectHandler{
		gen.DefaultDebugHandler(),
		gen.DefaultRandHandler(),
		gen.DefaultClockHandler(),
		gen.DefaultFSHandler(),
		gen.DefaultNetHandler(),
		gen.DefaultEnvHandler(),
		gen.DefaultAIHandler(),
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
	if len(allExternFuncs) > 0 {
		fmt.Printf("%s Generating extern stubs (%d extern functions)\n", cyan("→"), len(allExternFuncs))
		stubsCode := generateExternStubs(pkgName, allExternFuncs)

		stubsFile := filepath.Join(outDir, "extern_stubs.go")
		if err := os.WriteFile(stubsFile, stubsCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write stubs file: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Printf("%s Generated %s\n", green("✓"), stubsFile)
	}

	// Generate functions from Core AST (accumulated from all files)
	// Note: Function generation is experimental and may produce incomplete code
	codeGen := gen.New(pkgName)

	// M-CODEGEN-VALUE-TYPES: Set value threshold for record literal generation
	codeGen.SetValueThreshold(valueThreshold)

	// M-VERIFY: Enable runtime contract checking if flag is set
	if *verifyContractsFlag {
		codeGen.SetVerifyContracts(true)
		fmt.Printf("%s Contract verification enabled (runtime checks will be generated)\n", cyan("→"))
	}

	// Register ADT constructors (with field types and names for proper codegen)
	// M-DX13: Also register record types for typed struct literal generation
	for _, td := range allTypeDecls {
		if adt, ok := td.Definition.(*ast.AlgebraicType); ok {
			for _, ctor := range adt.Constructors {
				fieldTypes := extractFieldTypes(ctor.Fields)
				fieldNames := extractFieldNames(ctor.Fields)
				codeGen.RegisterADTConstructorFull(td.Name, ctor.Name, fieldTypes, fieldNames)
			}
		} else if rec, ok := td.Definition.(*ast.RecordType); ok {
			// M-DX13: Register record type for typed struct literal generation
			// M-CODEGEN-VALUE-TYPES: Use analysis to determine value vs pointer category
			fields, fieldTypes := extractRecordTypeInfo(rec)
			codeGen.RegisterRecordTypeWithAnalysis(capitalize(td.Name), fields, fieldTypes)
		}
	}

	// M-DX12: Register ADT slice types for converter generation
	if len(adtSliceTypes) > 0 {
		codeGen.RegisterADTSliceTypes(adtSliceTypes)
	}

	// M-CODEGEN-TYPED-PARAMS: Register function types from AST to prevent cross-module contamination
	// This ensures declared types (e.g., ArrivalState) are used instead of inferred structural types
	for _, fn := range allFuncs {
		paramTypes := make([]gen.GoType, 0)
		for _, p := range fn.Params {
			// Skip implicit unit params (zero-arg functions)
			if p.Name == "_" && p.Type != nil && p.Type.String() == "()" {
				continue
			}
			if p.Type != nil {
				paramTypes = append(paramTypes, gen.GoType(ailangTypeToGo(p.Type)))
			} else {
				paramTypes = append(paramTypes, gen.GoType("interface{}"))
			}
		}
		var returnType gen.GoType = "interface{}"
		if fn.ReturnType != nil {
			returnType = gen.GoType(ailangTypeToGo(fn.ReturnType))
		}
		codeGen.RegisterFunctionType(fn.Name, paramTypes, returnType)
	}

	// Count total declarations across all files
	totalDecls := 0
	for _, res := range allCoreDecls {
		if res.Artifacts.Core != nil {
			totalDecls += len(res.Artifacts.Core.Decls)
		}
	}

	if totalDecls > 0 {
		fmt.Printf("%s Generating functions (%d declarations from %d files)\n", cyan("→"), totalDecls, len(allCoreDecls))

		// For multi-file compilation, generate runtime helpers separately
		fmt.Printf("%s Generating runtime helpers (shared)\n", cyan("→"))
		runtimeCode, err := codeGen.GenerateRuntime()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: runtime generation failed: %v\n", red("Error"), err)
			os.Exit(1)
		}
		runtimeFile := filepath.Join(outDir, "runtime.go")
		if err := os.WriteFile(runtimeFile, runtimeCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write runtime file: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Printf("%s Generated %s\n", green("✓"), runtimeFile)

		// Skip runtime helpers in subsequent code generation
		codeGen.SetSkipRuntimeHelpers(true)

		// Generate code for each file's Core program into separate output files
		for i, res := range allCoreDecls {
			coreProg := res.Artifacts.Core
			if coreProg == nil || len(coreProg.Decls) == 0 {
				continue
			}

			// M-DX23: Set CoreTypeInfo for typed function signatures
			if res.Artifacts.CoreTI != nil {
				codeGen.SetCoreTypeInfo(res.Artifacts.CoreTI)
			}

			// M-DX18: Set module name for function namespacing
			// Non-exported functions will be prefixed with {moduleName}__ to prevent collisions
			if res.Artifacts.AST != nil && res.Artifacts.AST.Module != nil && res.Artifacts.AST.Module.Path != "" {
				parts := strings.Split(res.Artifacts.AST.Module.Path, "/")
				moduleName := sanitizeModuleName(parts[len(parts)-1])
				codeGen.SetModuleName(moduleName)
			} else {
				// Fallback: use source filename without extension
				baseName := filepath.Base(filenames[i])
				moduleName := sanitizeModuleName(strings.TrimSuffix(baseName, ".ail"))
				codeGen.SetModuleName(moduleName)
			}

			funcsCode, err := codeGen.Generate(coreProg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: function generation failed for file %d: %v\n", yellow("Warning"), i+1, err)
				continue
			}

			// Derive output filename from source filename
			sourceFile := filenames[i]
			baseName := filepath.Base(sourceFile)
			goFileName := strings.TrimSuffix(baseName, ".ail") + ".go"
			goFilePath := filepath.Join(outDir, goFileName)

			if err := os.WriteFile(goFilePath, funcsCode, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "%s: cannot write functions file: %v\n", red("Error"), err)
				os.Exit(1)
			}
			fmt.Printf("%s Generated %s\n", green("✓"), goFilePath)
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

// sanitizeModuleName converts a module name to a valid Go identifier prefix.
// M-DX18: Used to namespace non-exported functions when compiling multiple modules
// to the same Go package. Unlike package names, module names preserve case since
// they're used as function name prefixes, not Go package names.
func sanitizeModuleName(name string) string {
	// Replace invalid characters with underscores
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		if r == '_' {
			return r
		}
		if r == '-' {
			return '_'
		}
		return -1 // remove
	}, name)

	// Ensure doesn't start with number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "mod" + result
	}

	if result == "" {
		result = "module"
	}

	return result
}

func printCompileHelp() {
	fmt.Println(`Usage: ailang compile [options] <file.ail|directory> [...]

Compile AILANG source to Go. Supports files, directories, or both.

Options:
  --emit-go              Generate Go source code (required)
  --out <dir>            Output directory (default: "gen")
  --package-name <name>  Go package name (default: derived from first module)
  --release              Mark this as a release build (info only)
  --verify-contracts     Generate runtime contract checks (M-VERIFY Phase 0.5)
  --value-threshold <n>  Max fields for value-type records (default: 4, 0=all pointers)
  -h, --help             Show this help message

Examples:
  # Compile all .ail files in a directory
  ailang compile --emit-go sim/

  # Compile specific files
  ailang compile --emit-go world.ail npc_ai.ail

  # Mix directories and files
  ailang compile --emit-go sim/ extra.ail

  # Specify output directory and package name
  ailang compile --emit-go --out . --package-name sim_gen sim/

Output Structure:
  <out>/<package>/
  ├── types.go              # Generated ADT types (merged from all files)
  ├── debug_types_debug.go  # Debug effect (//go:build !release)
  ├── debug_types_release.go# Debug effect no-ops (//go:build release)
  ├── handlers.go           # Effect handler interfaces
  ├── runtime.go            # Shared runtime helpers
  ├── extern_stubs.go       # Stubs for extern functions (if any)
  ├── world.go              # Functions from world.ail
  ├── npc_ai.go             # Functions from npc_ai.ail
  └── step.go               # Functions from step.ail (one per source file)

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
func extractFieldTypes(fields []*ast.ConstructorField) []string {
	types := make([]string, len(fields))
	for i, field := range fields {
		types[i] = ailangTypeToGo(field.Type)
	}
	return types
}

// extractFieldNames extracts field names from AST constructor fields
// Returns empty strings for positional fields (no name specified)
func extractFieldNames(fields []*ast.ConstructorField) []string {
	names := make([]string, len(fields))
	for i, field := range fields {
		names[i] = field.Name // Empty string if positional
	}
	return names
}

// extractRecordTypeInfo extracts field information from a record type definition.
// M-DX13: Returns Go field names (PascalCase) and a map of field name -> Go type.
func extractRecordTypeInfo(rec *ast.RecordType) ([]string, map[string]string) {
	fields := make([]string, len(rec.Fields))
	fieldTypes := make(map[string]string, len(rec.Fields))
	for i, field := range rec.Fields {
		goFieldName := capitalize(field.Name)
		fields[i] = goFieldName
		fieldTypes[goFieldName] = ailangTypeToGo(field.Type)
	}
	return fields, fieldTypes
}

// expandFilenames expands directory arguments into .ail files.
// If an argument is a directory, all .ail files in it are included.
// Files are sorted alphabetically for deterministic compilation order.
func expandFilenames(args []string) []string {
	var result []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			// Keep the argument as-is (will fail later with proper error)
			result = append(result, arg)
			continue
		}

		if info.IsDir() {
			// Expand directory to all .ail files
			entries, err := os.ReadDir(arg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: cannot read directory '%s': %v\n", yellow("Warning"), arg, err)
				continue
			}

			var ailFiles []string
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".ail") {
					ailFiles = append(ailFiles, filepath.Join(arg, entry.Name()))
				}
			}

			// Sort for deterministic order
			// (os.ReadDir returns sorted entries, but be explicit)
			result = append(result, ailFiles...)

			if len(ailFiles) > 0 {
				fmt.Printf("%s Found %d .ail files in %s/\n", cyan("→"), len(ailFiles), arg)
			} else {
				fmt.Fprintf(os.Stderr, "%s: no .ail files found in directory '%s'\n", yellow("Warning"), arg)
			}
		} else {
			// Regular file
			result = append(result, arg)
		}
	}
	return result
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
			// M-CODEGEN-TYPE-ASSERTIONS: User-defined types are pointers in function signatures
			// This matches generateTypedRecord which generates &World{...} for record literals
			// Records/ADTs are always pointers when passed through interface{} boundaries
			return "*" + capitalize(typ.Name)
		}
	case *ast.ListType:
		elemType := ailangTypeToGo(typ.Element)
		// M-CODEGEN-POINTER-RETURN-TYPES: User-defined SimpleTypes now return *TypeName
		// So elemType is already "*CrewPosition" for user-defined types
		// Just concatenate "[]" + elemType - no additional "*" needed
		if strings.HasPrefix(elemType, "*") {
			// Already a pointer type (e.g., *CrewPosition) - just make it a slice
			return "[]" + elemType // []*CrewPosition
		}
		if isUserDefinedGoType(elemType) {
			// Fallback for non-pointer user types (shouldn't happen with current logic)
			return "[]*" + elemType
		}
		return "[]" + elemType
	case *ast.ArrayType:
		// M-TYPE1: Arrays use the same Go representation as lists (slices)
		elemType := ailangTypeToGo(typ.Element)
		// M-CODEGEN-POINTER-RETURN-TYPES: Same logic as ListType
		if strings.HasPrefix(elemType, "*") {
			return "[]" + elemType // []*TypeName
		}
		if isUserDefinedGoType(elemType) {
			return "[]*" + elemType
		}
		return "[]" + elemType
	case *ast.RecordType:
		return "map[string]interface{}"
	case *ast.TypeApp:
		// M-CODEGEN-OPTION-TYPE-ASSERT: Handle generic type applications like Option[Color]
		// ADTs (including Option) are always pointers in Go
		// The type argument is erased - Option[Color] and Option[int] both become *Option
		return "*" + capitalize(typ.Constructor)
	default:
		return "interface{}"
	}
}
