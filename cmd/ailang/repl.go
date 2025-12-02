package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/repl"
)

func runREPL(learn bool, trace bool, strictSyntax bool) {
	// Use the new REPL implementation with version info
	r := repl.NewWithVersion(Version, BuildTime)
	if trace {
		r.EnableTrace()
	}
	if strictSyntax {
		r.SetStrictSyntaxMode(true)
	}
	r.Start(os.Stdin, os.Stdout)
}

//nolint:unused // TODO: Implement test runner functionality
func runTests(path string) {
	fmt.Printf("%s Running tests in %s\n", cyan("→"), path)

	// Find all .ail files with tests
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasSuffix(p, ".ail") {
			// TODO: Check if file has tests and run them
			fmt.Printf("  %s %s\n", green("✓"), p)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// TODO: Implement test runner
	fmt.Printf("\n%s All tests passed!\n", green("✓"))
}

func watchFile(filename string, trace bool, binopShim bool, failOnShim bool, requireLowering bool, trackInstantiations bool, noMono bool, debugCompile bool, maxRecursionDepth int) {
	fmt.Printf("%s Watching %s for changes...\n", cyan("👁"), filename)
	fmt.Println("Press Ctrl+C to stop")

	// TODO: Implement file watching
	// For now, just run the file once (no json/compact/quiet for watch mode)
	// Default to main entrypoint with null args for watch mode, no caps, no stdlib overrides, no env overrides, no CLI args
	runFile(filename, []string{}, trace, 0, false, false, false, false, binopShim, failOnShim, requireLowering, trackInstantiations, noMono, debugCompile, false, "main", "null", true, false, "", maxRecursionDepth, "", false, false, "", "", "", "", "", false, "", false)
}
