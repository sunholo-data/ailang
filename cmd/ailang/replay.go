package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ailtrace "github.com/sunholo/ailang/internal/trace"
)

func replayCommand() {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output comparison result as JSON")
	fileOverride := fs.String("file", "", "Override source file path (default: resolve from module name)")
	capsOverride := fs.String("caps", "", "Override capabilities (default: from trace)")
	entryOverride := fs.String("entry", "", "Override entry function (default: main)")
	quiet := fs.Bool("quiet", false, "Suppress progress messages")
	_ = fs.Parse(os.Args[2:])

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Usage: ailang replay [flags] <trace.jsonl>")
		fmt.Fprintln(os.Stderr, "\nRe-executes a program and compares the trace against a recorded baseline.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
		os.Exit(2)
	}

	traceFile := fs.Arg(0)

	// Read baseline trace
	f, err := os.Open(traceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(2)
	}
	defer f.Close()

	baseline, err := ailtrace.ReadJSONL(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: reading trace: %v\n", red("Error"), err)
		os.Exit(2)
	}

	if len(baseline) == 0 {
		fmt.Fprintf(os.Stderr, "%s: trace file is empty\n", red("Error"))
		os.Exit(2)
	}

	// Extract metadata from baseline
	moduleName, caps := ailtrace.TraceMetadata(baseline)

	// Resolve source file
	sourceFile := *fileOverride
	if sourceFile == "" {
		sourceFile = resolveSourceFile(moduleName, traceFile)
	}
	if sourceFile == "" {
		fmt.Fprintf(os.Stderr, "%s: cannot resolve source file for module %q\n", red("Error"), moduleName)
		fmt.Fprintln(os.Stderr, "  Use --file to specify the source file path")
		os.Exit(2)
	}

	// Resolve capabilities
	capsStr := *capsOverride
	if capsStr == "" && len(caps) > 0 {
		capsStr = strings.Join(caps, ",")
	}

	// Resolve entry
	entry := *entryOverride
	if entry == "" {
		entry = "main"
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Replaying: %s\n", sourceFile)
		if moduleName != "" {
			fmt.Fprintf(os.Stderr, "Module: %s\n", moduleName)
		}
		if capsStr != "" {
			fmt.Fprintf(os.Stderr, "Capabilities: %s\n", capsStr)
		}
		fmt.Fprintf(os.Stderr, "Baseline events: %d\n", len(baseline))
	}

	// Re-execute with trace collection
	replayEvents, err := rerunWithTrace(sourceFile, capsStr, entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: replay execution: %v\n", red("Error"), err)
		os.Exit(2)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Replay events: %d\n", len(replayEvents))
	}

	// Compare traces
	result := ailtrace.CompareTraces(baseline, replayEvents)

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		printCompareResult(result, *quiet)
	}

	if !result.Match {
		os.Exit(1)
	}
}

// resolveSourceFile finds the .ail source file from a module name.
// Tries: module_name.ail relative to CWD, then relative to trace file.
func resolveSourceFile(moduleName, traceFile string) string {
	if moduleName == "" {
		return ""
	}

	// Try module name as file path relative to CWD
	candidate := moduleName + ".ail"
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	// Try relative to trace file directory
	traceDir := filepath.Dir(traceFile)
	candidate = filepath.Join(traceDir, moduleName+".ail")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	return ""
}

// rerunWithTrace executes the program with --emit-trace jsonl and returns the events.
func rerunWithTrace(sourceFile, caps, entry string) ([]ailtrace.TraceEvent, error) {
	// Build command args
	args := []string{"run", "--emit-trace", "jsonl", "--quiet"}
	if caps != "" {
		args = append(args, "--caps", caps)
	}
	if entry != "" {
		args = append(args, "--entry", entry)
	}
	args = append(args, sourceFile)

	// Find the ailang binary (use the same one running this command)
	binary, err := os.Executable()
	if err != nil {
		binary = "ailang" // fallback to PATH
	}

	cmd := exec.Command(binary, args...)
	cmd.Stderr = os.Stderr // Show any warnings/errors

	// Capture stdout (JSONL output)
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ailang run failed (exit %d)", exitErr.ExitCode())
		}
		return nil, err
	}

	// Parse JSONL from stdout
	return ailtrace.ReadJSONL(strings.NewReader(string(stdout)))
}

// printCompareResult prints a human-readable comparison result.
func printCompareResult(result ailtrace.CompareResult, quiet bool) {
	if result.Match {
		fmt.Fprintf(os.Stderr, "\n%s REPLAY MATCHES (%d/%d events identical)\n",
			green("✓"), result.BaselineN, result.BaselineN)
		return
	}

	fmt.Fprintf(os.Stderr, "\n%s REPLAY MISMATCH (%d differences in %d events)\n\n",
		red("✗"), len(result.Mismatches), result.BaselineN)

	for _, mm := range result.Mismatches {
		fmt.Fprintf(os.Stderr, "  Event #%d: %s differs\n", mm.Index, yellow(mm.Field))
		fmt.Fprintf(os.Stderr, "    Expected: %s\n", mm.Expected)
		fmt.Fprintf(os.Stderr, "    Actual:   %s\n", mm.Actual)
		if mm.Context != "" {
			fmt.Fprintf(os.Stderr, "    Context:  %s\n", mm.Context)
		}
		fmt.Fprintln(os.Stderr)
	}
}
