package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sunholo-data/ailang/internal/astedit"
)

// runAstEdit implements `ailang ast-edit replace --file <f> --decl <name> [--new <txt>] [--in-place]`.
//
// Span-anchored semantic edit (M-AILANG-NATIVE-HARNESS #8): replace exactly ONE top-level
// declaration by its parsed source span, preserving the rest of the file byte-for-byte. This is
// the anti-thrash edit primitive — instead of rewriting a 500-line file to change one function
// (the measured agent failure mode on long tasks), the agent names the decl and splices new text
// into just that range. New text comes from --new (a file) or stdin. The result prints to stdout,
// or overwrites the file with --in-place. Re-run `ailang check` afterwards: this splices source,
// it does NOT validate semantics.
func runAstEdit() {
	args := flag.Args() // ["ast-edit", "replace", ...]
	if len(args) < 2 || args[1] != "replace" {
		fmt.Fprintf(os.Stderr, "%s: usage: ailang ast-edit replace --file <f> --decl <name> [--new <txt-file>] [--in-place]\n", red("Error"))
		fmt.Fprintln(os.Stderr, "Replaces one top-level declaration by its parsed span; the rest of the file is preserved exactly.")
		os.Exit(1)
	}
	fs := flag.NewFlagSet("ast-edit replace", flag.ExitOnError)
	file := fs.String("file", "", "AILANG source file to edit")
	decl := fs.String("decl", "", "name of the top-level declaration to replace")
	newFile := fs.String("new", "", "file containing the replacement decl text (default: read stdin)")
	inPlace := fs.Bool("in-place", false, "overwrite the file instead of printing the result to stdout")
	_ = fs.Parse(args[2:])

	if *file == "" || *decl == "" {
		fmt.Fprintf(os.Stderr, "%s: --file and --decl are required\n", red("Error"))
		os.Exit(1)
	}
	src, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: read %s: %v\n", red("Error"), *file, err)
		os.Exit(1)
	}
	var newText []byte
	if *newFile != "" {
		newText, err = os.ReadFile(*newFile)
	} else {
		newText, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: read replacement text: %v\n", red("Error"), err)
		os.Exit(1)
	}
	out, err := astedit.ReplaceDecl(string(src), *file, *decl, string(newText))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if *inPlace {
		if err := os.WriteFile(*file, []byte(out), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: write %s: %v\n", red("Error"), *file, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "%s replaced %q in %s\n", green("✓"), *decl, *file)
	} else {
		fmt.Print(out)
	}
}
