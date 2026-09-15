package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runner"
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
	return runner.CompileBytecodeFromResult(res, "disasm")
}
