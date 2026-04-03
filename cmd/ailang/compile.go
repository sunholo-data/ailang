package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
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
	// M-CODEGEN-COMPILE-GATE: Verify generated Go code compiles
	noVerifyGoFlag := fs.Bool("no-verify-go", false, "Skip go build verification of generated code")
	// M-CODEGEN-IR: Statement IR pipeline (new architecture)
	emitGoV2Flag := fs.Bool("emit-go-v2", false, "Use Statement IR pipeline (new codegen architecture)")

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
	if !*emitGoFlag && !*emitGoV2Flag {
		fmt.Fprintf(os.Stderr, "%s: --emit-go or --emit-go-v2 is required\n", red("Error"))
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
			ReleaseMode:  *releaseFlag,
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

	// M-CODEGEN-IR: Statement IR pipeline (v2)
	if *emitGoV2Flag {
		compileV2(allCoreDecls, allTypeDecls, filenames, pkgName, outDir, !*noVerifyGoFlag, relaxModulesEffective)
		return
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

	// M-CODEGEN-DICTIONARIES M4: Clear derived Eq types from previous compilation
	gen.ClearDerivedEqTypes()

	// Register ADT constructors (with field types and names for proper codegen)
	// M-DX13: Also register record types for typed struct literal generation
	// M-CODEGEN-DICTIONARIES M4: Register ADT types with deriving (Eq)
	for _, td := range allTypeDecls {
		if adt, ok := td.Definition.(*ast.AlgebraicType); ok {
			// Extract constructor names for derived Eq
			var ctorNames []string
			for _, ctor := range adt.Constructors {
				fieldTypes := extractFieldTypes(ctor.Fields)
				fieldNames := extractFieldNames(ctor.Fields)
				codeGen.RegisterADTConstructorFull(td.Name, ctor.Name, fieldTypes, fieldNames)
				ctorNames = append(ctorNames, ctor.Name)
			}
			// M-CODEGEN-DICTIONARIES M4: Register for derived Eq if needed
			for _, deriving := range td.Deriving {
				if deriving == ast.DeriveEq {
					codeGen.RegisterDerivedEqType(capitalize(td.Name), ctorNames)
					break
				}
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

	// M-CODEGEN-VALUE-TYPES: Build set of value record names for ailangTypeToGo
	// This must match the analysis in RegisterRecordTypeWithAnalysis
	valueRecordNames := make(map[string]bool)
	for _, td := range allTypeDecls {
		if rec, ok := td.Definition.(*ast.RecordType); ok {
			// Same analysis as RegisterRecordTypeWithAnalysis
			_, fieldTypes := extractRecordTypeInfo(rec)
			isLeaf := gen.IsLeafRecord(fieldTypes)
			if isLeaf && len(rec.Fields) <= valueThreshold {
				valueRecordNames[capitalize(td.Name)] = true
			}
		}
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
				paramTypes = append(paramTypes, gen.GoType(ailangTypeToGoWithValueRecords(p.Type, valueRecordNames)))
			} else {
				paramTypes = append(paramTypes, gen.GoType("interface{}"))
			}
		}
		var returnType gen.GoType = "interface{}"
		if fn.ReturnType != nil {
			returnType = gen.GoType(ailangTypeToGoWithValueRecords(fn.ReturnType, valueRecordNames))
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

		// M-CODEGEN-SUSTAINABILITY: Generate module files FIRST so that registry
		// helpers (Intersect, Union, etc.) are accumulated during codegen.
		// Runtime is generated AFTER to include all needed registry helpers.
		codeGen.SetSkipRuntimeHelpers(true)

		// Generate code for each file's Core program into separate output files
		for i, res := range allCoreDecls {
			coreProg := res.Artifacts.Core
			if coreProg == nil || len(coreProg.Decls) == 0 {
				continue
			}

			// M-CODEGEN-MULTIMOD: Reset per-module name caches to prevent cross-module
			// function name collisions. Preserves shared type registrations (ADTs, records).
			codeGen.ResetPerModuleState()

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
				// M-CODEGEN-STDLIB-BUILTINS: Still write the raw (unformatted) code on error.
				// The code may be partially valid — writing it allows other modules to reference
				// functions that were successfully generated before the error.
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

		// M-CODEGEN-SUSTAINABILITY: Generate runtime AFTER module files so that
		// registry-only helpers (Intersect, Union, etc.) accumulated during module
		// codegen are included in runtime.go.
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

		// M-CODEGEN-DICTIONARIES: Generate type class dictionaries
		fmt.Printf("%s Generating type class dictionaries\n", cyan("→"))
		dictCode, err := codeGen.GenerateDictionaries()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: dictionary generation failed: %v\n", red("Error"), err)
			os.Exit(1)
		}
		dictFile := filepath.Join(outDir, "dictionaries.go")
		if err := os.WriteFile(dictFile, dictCode, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write dictionaries file: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Printf("%s Generated %s\n", green("✓"), dictFile)
	}

	// M-CODEGEN-COMPILE-GATE: Verify generated Go code compiles
	if !*noVerifyGoFlag {
		verifyGoCompilation(outDir, pkgName)
	}

	fmt.Printf("\n%s Compilation complete!\n", green("✓"))
	fmt.Printf("  Output: %s/\n", outDir)
	fmt.Printf("\n  %s Build commands:\n", cyan("→"))
	fmt.Printf("    Debug mode:   cd %s && go build\n", *outFlag)
	fmt.Printf("    Release mode: cd %s && go build -tags release\n", *outFlag)
}

// verifyGoCompilation runs `go build` on generated code to catch codegen errors early.
// M-CODEGEN-COMPILE-GATE: This is the single highest-leverage quality gate — it turns
// every silent codegen failure into a loud error at the right time.
func verifyGoCompilation(outDir, pkgName string) {
	// Check if go binary is available
	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Printf("%s Skipping Go verification (go binary not in PATH)\n", yellow("⚠"))
		return
	}
	_ = goPath

	fmt.Printf("\n%s Verifying generated Go code compiles...\n", cyan("→"))

	// Create temporary go.mod if one doesn't exist
	goModPath := filepath.Join(outDir, "go.mod")
	createdGoMod := false
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		modContent := fmt.Sprintf("module %s\n\ngo 1.21\n", pkgName)
		if err := os.WriteFile(goModPath, []byte(modContent), 0644); err != nil {
			fmt.Printf("%s Cannot create temporary go.mod: %v\n", yellow("⚠"), err)
			return
		}
		createdGoMod = true
		defer func() {
			if createdGoMod {
				os.Remove(goModPath)
			}
		}()
	}

	// Run go build
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = outDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Parse and display Go compiler errors as codegen diagnostics
		fmt.Printf("\n%s Generated Go code has compilation errors:\n", red("✗"))
		lines := strings.Split(string(output), "\n")
		errorCount := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Format: ./file.go:line:col: error message
			if strings.Contains(line, ": ") {
				fmt.Printf("  %s codegen: %s\n", red("→"), line)
				errorCount++
			}
		}
		if errorCount > 0 {
			fmt.Printf("\n  %s %d codegen error(s). Fix the codegen or use --no-verify-go to skip.\n", red("✗"), errorCount)
		}
		// Clean up temporary go.mod before exit
		if createdGoMod {
			os.Remove(goModPath)
		}
		os.Exit(1)
	}

	// Clean up temporary go.mod
	if createdGoMod {
		os.Remove(goModPath)
	}

	fmt.Printf("%s Generated Go code compiles successfully\n", green("✓"))
}
