package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/bytecode/compiler"
	"github.com/sunholo/ailang/internal/gen/lower"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/pipeline"
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
	seenTypes := map[string]bool{}
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
