// Command astdump parses a single .ail file and prints a deep, deterministic dump
// of the resulting AST (or the parse errors). It exists to support the
// M-SYNTAX-AI-FORGIVING corpus AST-diff fuzz gate (internal/parser/corpus_astdiff_test.go),
// which builds this command from both the pre-change (old) parser and the current
// (new) parser and diffs the output over the whole .ail corpus.
//
// Usage: astdump <file.ail>
//
// Output contract (stable, line-based):
//
//	ERRORS\n<one error per line>   — if the file does not parse cleanly
//	AST\n<spew dump>               — if it parses cleanly
package main

import (
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: astdump <file.ail>")
		os.Exit(2)
	}
	path := os.Args[1]
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(2)
	}

	p := parser.New(lexer.New(string(src), path))
	prog := p.Parse()

	if errs := p.Errors(); len(errs) > 0 {
		fmt.Println("ERRORS")
		for _, e := range errs {
			fmt.Println(e.Error())
		}
		return
	}

	cfg := &spew.ConfigState{
		Indent:                  "  ",
		DisablePointerAddresses: true,
		DisableCapacities:       true,
		DisableMethods:          true, // dump full struct, not Stringer output
		SortKeys:                true,
		SpewKeys:                true,
	}
	fmt.Println("AST")
	fmt.Print(cfg.Sdump(prog))
}
