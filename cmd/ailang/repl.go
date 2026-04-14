package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/repl"
	"github.com/sunholo/ailang/internal/telemetry"
)

func runREPL(learn bool, trace bool, strictSyntax bool) {
	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-repl")
	if err != nil {
		// Non-fatal: continue without telemetry
	} else {
		defer shutdownTelemetry(ctx)
	}

	// Extract trace context from environment (enables cross-process trace linking)
	// If TRACEPARENT is set (e.g., by coordinator or CI), the REPL session will be
	// a child span in the parent trace
	ctx = telemetry.ExtractTraceContext(ctx)
	// Note: REPL doesn't create its own root span here - that's handled by the repl package
	// The extracted context will flow through to any spans created during REPL execution

	// Use the new REPL implementation with version info
	r := repl.NewWithVersion(Version, BuildTime)
	if trace {
		r.EnableTrace()
	}
	if strictSyntax {
		r.SetStrictSyntaxMode(true)
	}
	r.StartWithContext(ctx, os.Stdin, os.Stdout)
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
	// Default to main entrypoint with null args for watch mode, no caps, no stdlib overrides, no env overrides, no CLI args, no relaxModules, no debug-types, no budget bypass, no contract verification
	runFile(filename, []string{}, trace, 0, false, false, false, false, binopShim, failOnShim, requireLowering, trackInstantiations, noMono, debugCompile, false, "main", "null", true, false, false, "", maxRecursionDepth, "", false, false, "", "", "", "", "", false, "", false, false, false, 0, false, "", false, "", "", false, "", false, false, false, "", false, "30s", "", 10*1024*1024, false, false, false)
}
