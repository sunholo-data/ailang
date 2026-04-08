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
// helper used by `disasm` and (in M2) by `run --bytecode`. It mirrors
// tests/golden/bytecode/golden_test.go::tryCompileAILFile so the CLI and
// the parity gate stay in lockstep.
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
	prog := &stmt.Program{Package: "disasm"}
	seenTypes := map[string]bool{}
	if res.Artifacts.AST != nil {
		for _, decl := range res.Artifacts.AST.Decls {
			td, ok := decl.(*ast.TypeDecl)
			if !ok || seenTypes[td.Name] {
				continue
			}
			seenTypes[td.Name] = true
			prog.TypeDecls = append(prog.TypeDecls, lower.LowerTypeDecl(td))
		}
	}
	fileProg, err := lower.LowerProgram(res.Artifacts.Core, res.Artifacts.CoreTI, res.Artifacts.AST, "disasm")
	if err != nil {
		return nil, fmt.Errorf("lower: %w", err)
	}
	prog.FuncDecls = append(prog.FuncDecls, fileProg.FuncDecls...)
	return compiler.Compile(prog)
}
