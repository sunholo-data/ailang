package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/bytecode/compiler"
	"github.com/sunholo-data/ailang/internal/gen/lower"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

// disasmCommand handles `ailang disasm <file.ail>` — runs the file through
// pipeline → lower → bytecode compiler and prints a human-readable
// disassembly of the resulting BytecodeImage.
//
// This is a debugging tool for the M-BYTECODE-VM Phase 2D path. It exists
// so that compiler/VM bugs can be inspected directly without going through
// a test harness.
func disasmCommand() {
	fs := flag.NewFlagSet("disasm", flag.ExitOnError)
	relaxModulesFlag := fs.Bool("relax-modules", false, "Relax MOD010 module path validation")
	helpFlag := fs.Bool("help", false, "Show help for disasm command")
	fs.BoolVar(helpFlag, "h", false, "Show help for disasm command")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *helpFlag || fs.NArg() < 1 {
		fmt.Println("Usage: ailang disasm [options] <file.ail>")
		fmt.Println()
		fmt.Println("Compiles a .ail file through the bytecode pipeline and prints a")
		fmt.Println("human-readable disassembly of the resulting BytecodeImage.")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --relax-modules    Relax MOD010 validation (allow module path mismatches)")
		if *helpFlag {
			return
		}
		os.Exit(1)
	}

	filename := fs.Arg(0)
	img, err := compileBytecodeFromFile(filename, *relaxModulesFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Print(bytecode.Disassemble(img))
}

// compileBytecodeFromFile is the shared "AILANG source → BytecodeImage"
// helper used by `disasm`. It mirrors tests/golden/bytecode/golden_test.go::
// tryCompileAILFile so the CLI and the parity gate stay in lockstep.
func compileBytecodeFromFile(filename string, relaxModules bool) (*bytecode.BytecodeImage, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", filename, err)
	}
	res, err := pipeline.Run(pipeline.Config{
		Mode:         pipeline.ModeCheck,
		RelaxModules: relaxModules,
	}, pipeline.Source{Filename: filename, Code: string(data)})
	if err != nil {
		return nil, fmt.Errorf("pipeline: %w", err)
	}
	if len(res.Errors) > 0 {
		var msgs []string
		for _, e := range res.Errors {
			msgs = append(msgs, fmt.Sprintf("%v", e))
		}
		return nil, fmt.Errorf("pipeline errors:\n  %s", strings.Join(msgs, "\n  "))
	}
	return compileBytecodeFromResult(res, "disasm")
}

// compileBytecodeFromResult lowers an already-typechecked pipeline result and
// compiles it to a BytecodeImage. This is the shared back-half of the bytecode
// compile pipeline used by both `ailang disasm` (which runs the pipeline
// itself) and `ailang run --bytecode` (which reuses the pipeline result that
// the evaluator path already produced, avoiding a double compile).
//
// pkgName is the package label stamped onto the resulting stmt.Program; it
// affects only diagnostics and disassembly headers, not name resolution.
func compileBytecodeFromResult(res pipeline.Result, pkgName string) (*bytecode.BytecodeImage, error) {
	if res.Artifacts.Core == nil || res.Artifacts.AST == nil {
		return nil, fmt.Errorf("compile from result: missing AST/Core artifacts")
	}
	prog := &stmt.Program{Package: pkgName}

	// M-BYTECODE-MULTIMODULE M1: if the pipeline loaded additional modules
	// (imports from the entry file), lower every reachable module and merge
	// the results into a single stmt.Program. Each module's FuncDecls are
	// tagged with its module path so the compiler can canonicalize funcIdx
	// keys and cross-module GlobalRefs resolve without bridging.
	//
	// Ordering: modules are lowered in sorted path order so the resulting
	// image is deterministic across runs. TypeDecls from every module are
	// deduplicated by name (later modules lose the race — single-source-of
	// -truth per name is the contract).
	seenTypes := map[string]bool{}

	if len(res.Modules) > 0 {
		// Multi-module mode. Each LoadedModule already carries Core, CoreTI,
		// and the surface AST. The entry module is included in res.Modules,
		// so we do NOT additionally lower res.Artifacts.Core here (that would
		// double-register its FuncDecls and fail funcIdx canonicalization).
		modIDs := make([]string, 0, len(res.Modules))
		for id := range res.Modules {
			modIDs = append(modIDs, id)
		}
		sort.Strings(modIDs)

		for _, modID := range modIDs {
			mod := res.Modules[modID]
			if mod == nil || mod.Core == nil || mod.File == nil {
				continue
			}
			// TypeDecls: walk the module's AST and register each unique type
			// declaration. Tag with the module name so cross-module ADT/record
			// lookups (M3) can find the right entry.
			for _, decl := range mod.File.Decls {
				td, ok := decl.(*ast.TypeDecl)
				if !ok {
					continue
				}
				canonicalType := modID + "." + td.Name
				if seenTypes[canonicalType] {
					continue
				}
				seenTypes[canonicalType] = true
				lowered := lower.LowerTypeDecl(td)
				// Preserve the bare name for in-module lookups; M3 will add
				// module-tagged lookups on top of this.
				prog.TypeDecls = append(prog.TypeDecls, lowered)
			}

			// CoreTI is stored as interface{} in LoadedModule to avoid an
			// import cycle; the pipeline always puts a types.CoreTypeInfo
			// there, so the assertion is safe.
			cti, _ := mod.CoreTI.(types.CoreTypeInfo)
			modProg, err := lower.LowerProgram(mod.Core, cti, mod.File, pkgName)
			if err != nil {
				return nil, fmt.Errorf("lower module %s: %w", modID, err)
			}
			// Tag every FuncDecl with its source module so the compiler
			// keys funcIdx by the canonical "module.name" form.
			for i := range modProg.FuncDecls {
				modProg.FuncDecls[i].Module = modID
			}
			prog.FuncDecls = append(prog.FuncDecls, modProg.FuncDecls...)
		}
		return compiler.Compile(prog)
	}

	// Single-file mode: no imported modules, lower the entry Core directly.
	for _, decl := range res.Artifacts.AST.Decls {
		td, ok := decl.(*ast.TypeDecl)
		if !ok || seenTypes[td.Name] {
			continue
		}
		seenTypes[td.Name] = true
		prog.TypeDecls = append(prog.TypeDecls, lower.LowerTypeDecl(td))
	}
	fileProg, err := lower.LowerProgram(res.Artifacts.Core, res.Artifacts.CoreTI, res.Artifacts.AST, pkgName)
	if err != nil {
		return nil, fmt.Errorf("lower: %w", err)
	}
	prog.FuncDecls = append(prog.FuncDecls, fileProg.FuncDecls...)
	return compiler.Compile(prog)
}
