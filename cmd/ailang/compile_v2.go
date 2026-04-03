package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/gen/emitgo"
	"github.com/sunholo/ailang/internal/gen/lower"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/pipeline"
)

// compileV2 uses the Statement IR pipeline:
//
//	pipeline.Run(ModeCheck) → Core + CoreTypeInfo + AST
//	lower.LowerProgram(core, cti, ast) → stmt.Program
//	emitgo.Emit(program) → Go source code
func compileV2(
	allResults []pipeline.Result,
	allTypeDecls []*ast.TypeDecl,
	filenames []string,
	pkgName string,
	outDir string,
	verifyGo bool,
	relaxModules bool,
) {
	fmt.Printf("%s Using Statement IR pipeline (v2)\n", cyan("→"))

	// Single-file mode: lower each file independently.
	// Multi-file mode: merge all into one Program.
	prog := &stmt.Program{
		Package: pkgName,
	}

	// Phase 1: Lower type declarations from all AST files.
	seenTypes := make(map[string]bool)
	for _, res := range allResults {
		if res.Artifacts.AST == nil {
			continue
		}
		for _, decl := range res.Artifacts.AST.Decls {
			td, ok := decl.(*ast.TypeDecl)
			if !ok {
				continue
			}
			if seenTypes[td.Name] {
				continue
			}
			seenTypes[td.Name] = true
			prog.TypeDecls = append(prog.TypeDecls, lower.LowerTypeDecl(td))
		}
		// Also check imported modules for cross-module ADT support.
		for _, mod := range res.Modules {
			if mod.File == nil {
				continue
			}
			for _, decl := range mod.File.Decls {
				td, ok := decl.(*ast.TypeDecl)
				if !ok {
					continue
				}
				if seenTypes[td.Name] {
					continue
				}
				seenTypes[td.Name] = true
				prog.TypeDecls = append(prog.TypeDecls, lower.LowerTypeDecl(td))
			}
		}
	}

	// Phase 2: Lower function declarations from Core programs.
	for i, res := range allResults {
		coreProg := res.Artifacts.Core
		if coreProg == nil || len(coreProg.Decls) == 0 {
			continue
		}

		cti := res.Artifacts.CoreTI
		if cti == nil {
			fmt.Fprintf(os.Stderr, "%s: file %d has no CoreTypeInfo, skipping\n", yellow("Warning"), i+1)
			continue
		}

		// Determine module name for function namespacing.
		moduleName := ""
		if res.Artifacts.AST != nil && res.Artifacts.AST.Module != nil && res.Artifacts.AST.Module.Path != "" {
			parts := strings.Split(res.Artifacts.AST.Module.Path, "/")
			moduleName = sanitizeModuleNameV2(parts[len(parts)-1])
		} else if i < len(filenames) {
			baseName := filepath.Base(filenames[i])
			moduleName = sanitizeModuleNameV2(strings.TrimSuffix(baseName, ".ail"))
		}

		// Lower this file's Core program.
		fileProg, err := lower.LowerProgram(coreProg, cti, res.Artifacts.AST, pkgName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: lowering failed for file %d: %v\n", red("Error"), i+1, err)
			os.Exit(1)
		}

		// Tag functions with module name and merge into main program.
		for j := range fileProg.FuncDecls {
			fileProg.FuncDecls[j].Module = moduleName
		}
		prog.FuncDecls = append(prog.FuncDecls, fileProg.FuncDecls...)
		prog.Imports = append(prog.Imports, fileProg.Imports...)
	}

	// Deduplicate imports.
	prog.Imports = deduplicateImports(prog.Imports)

	// Qualify intra-module function references (VarRef → GlobalRef).
	lower.QualifyFuncRefs(prog)

	fmt.Printf("%s Lowered %d type(s), %d function(s)\n",
		cyan("→"), len(prog.TypeDecls), len(prog.FuncDecls))

	// Phase 3: Emit Go source code.
	// Write types.go if there are type declarations.
	if len(prog.TypeDecls) > 0 {
		typesOut, err := emitgo.EmitTypes(prog)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: type emission failed: %v\n", red("Error"), err)
			// Write unformatted for debugging.
			os.WriteFile(filepath.Join(outDir, "types.go"), typesOut, 0644)
			os.Exit(1)
		}
		typesFile := filepath.Join(outDir, "types.go")
		if err := os.WriteFile(typesFile, typesOut, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write %s: %v\n", red("Error"), typesFile, err)
			os.Exit(1)
		}
		fmt.Printf("%s Generated %s\n", green("✓"), typesFile)
	}

	// Write one .go file per source file (matching old codegen behavior).
	for i, res := range allResults {
		if res.Artifacts.Core == nil || len(res.Artifacts.Core.Decls) == 0 {
			continue
		}

		moduleName := ""
		if res.Artifacts.AST != nil && res.Artifacts.AST.Module != nil && res.Artifacts.AST.Module.Path != "" {
			parts := strings.Split(res.Artifacts.AST.Module.Path, "/")
			moduleName = sanitizeModuleNameV2(parts[len(parts)-1])
		} else if i < len(filenames) {
			baseName := filepath.Base(filenames[i])
			moduleName = sanitizeModuleNameV2(strings.TrimSuffix(baseName, ".ail"))
		}

		funcsOut, err := emitgo.EmitFuncs(prog, moduleName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: function emission failed for module %s: %v\n",
				yellow("Warning"), moduleName, err)
		}

		var goFileName string
		if i < len(filenames) {
			baseName := filepath.Base(filenames[i])
			goFileName = strings.TrimSuffix(baseName, ".ail") + ".go"
		} else {
			goFileName = fmt.Sprintf("module_%d.go", i)
		}
		goFilePath := filepath.Join(outDir, goFileName)

		if err := os.WriteFile(goFilePath, funcsOut, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: cannot write %s: %v\n", red("Error"), goFilePath, err)
			os.Exit(1)
		}
		fmt.Printf("%s Generated %s\n", green("✓"), goFilePath)
	}

	// Phase 4: Verify Go compilation.
	if verifyGo {
		verifyGoCompilation(outDir, pkgName)
	}

	fmt.Printf("\n%s Compilation complete! (Statement IR pipeline)\n", green("✓"))
	fmt.Printf("  Output: %s/\n", outDir)
}

func sanitizeModuleNameV2(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "/", "_"), "-", "_")
}

func deduplicateImports(imports []stmt.ImportSpec) []stmt.ImportSpec {
	seen := make(map[string]bool)
	var result []stmt.ImportSpec
	for _, imp := range imports {
		key := imp.Path
		if imp.Alias != "" {
			key = imp.Alias + " " + imp.Path
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, imp)
		}
	}
	return result
}
