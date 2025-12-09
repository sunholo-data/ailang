package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/pipeline"
)

func checkFile(filename string, strictSyntax bool, relaxModules bool, timeout string, debugCompile bool) {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Parse timeout if specified
	var timeoutDuration time.Duration
	if timeout != "" {
		var err error
		timeoutDuration, err = time.ParseDuration(timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: invalid timeout duration '%s': %v\n", red("Error"), timeout, err)
			fmt.Println("Examples: 30s, 2m, 1m30s")
			os.Exit(1)
		}
	}

	// Type check
	fmt.Printf("%s Type checking %s...\n", cyan("→"), filename)

	// Effect check
	fmt.Printf("%s Effect checking...\n", cyan("→"))

	// Check AILANG_RELAX_MODULES environment variable
	relaxModulesEffective := relaxModules
	if envVal := os.Getenv("AILANG_RELAX_MODULES"); envVal != "" {
		switch strings.ToLower(envVal) {
		case "1", "true", "yes":
			relaxModulesEffective = true
		}
	}

	// Use unified pipeline in dry-run mode (no evaluation)
	cfg := pipeline.Config{
		DryLink:          true, // Don't evaluate, just check
		StrictSyntaxMode: strictSyntax,
		RelaxModules:     relaxModulesEffective,
		DebugCompile:     debugCompile,
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	// Run with timeout if specified
	if timeoutDuration > 0 {
		runCheckWithTimeout(cfg, src, timeoutDuration, filename)
	} else {
		runCheck(cfg, src)
	}
}

// runCheck executes the pipeline without timeout
func runCheck(cfg pipeline.Config, src pipeline.Source) {
	result, err := pipeline.Run(cfg, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Check for any errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
		}
		os.Exit(1)
	}

	// Print phase timing breakdown if debug-compile enabled
	if cfg.DebugCompile && len(result.PhaseTimings) > 0 {
		printPhaseTimings(result.PhaseTimings)
	}

	fmt.Printf("\n%s No errors found!\n", green("✓"))
}

// runCheckWithTimeout executes the pipeline with a watchdog timer
func runCheckWithTimeout(cfg pipeline.Config, src pipeline.Source, timeout time.Duration, filename string) {
	type checkResult struct {
		result pipeline.Result
		err    error
	}

	done := make(chan checkResult, 1)

	go func() {
		result, err := pipeline.Run(cfg, src)
		done <- checkResult{result, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), r.err)
			os.Exit(1)
		}
		if len(r.result.Errors) > 0 {
			for _, e := range r.result.Errors {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
			}
			os.Exit(1)
		}
		// Print phase timing breakdown if debug-compile enabled
		if cfg.DebugCompile && len(r.result.PhaseTimings) > 0 {
			printPhaseTimings(r.result.PhaseTimings)
		}
		fmt.Printf("\n%s No errors found!\n", green("✓"))

	case <-time.After(timeout):
		fmt.Fprintf(os.Stderr, "\n%s Compilation timed out after %s\n", red("TIMEOUT"), timeout)
		fmt.Fprintf(os.Stderr, "File: %s\n\n", filename)

		// Dump goroutine stacks for debugging
		buf := make([]byte, 1<<20) // 1MB buffer
		n := runtime.Stack(buf, true)
		fmt.Fprintf(os.Stderr, "Stack dump (all goroutines):\n%s\n", buf[:n])

		fmt.Fprintf(os.Stderr, "\n%s This may indicate cyclic types. Try: ailang debug cycles %s\n",
			yellow("Hint:"), filename)
		os.Exit(124) // Standard timeout exit code
	}
}

// printPhaseTimings outputs a formatted breakdown of compilation phase timings
func printPhaseTimings(timings map[string]int64) {
	fmt.Printf("\n%s Compilation Phase Timings:\n", cyan("⏱"))

	// Define phase order and human-readable names
	phaseOrder := []string{
		"load", "topo", "parse", "elaborate", "typecheck",
		"dict_elaboration", "monomorphization", "lower",
		"dict_elab", "anf_verify", "link", "compile", "evaluate",
	}
	phaseNames := map[string]string{
		"load":             "Loading",
		"topo":             "Topo Sort",
		"parse":            "Parsing",
		"elaborate":        "Elaboration",
		"typecheck":        "Type Checking",
		"dict_elaboration": "Dict Elaboration",
		"monomorphization": "Monomorphization",
		"lower":            "Lowering",
		"dict_elab":        "Dict Linking",
		"anf_verify":       "ANF Verify",
		"link":             "Linking",
		"compile":          "Compile",
		"evaluate":         "Evaluate",
	}

	// Calculate total and find slowest phase
	var total int64
	var slowestPhase string
	var slowestTime int64
	for _, ms := range timings {
		total += ms
		if ms > slowestTime {
			slowestTime = ms
		}
	}

	// Print phases in order
	for _, phase := range phaseOrder {
		if ms, ok := timings[phase]; ok {
			name := phaseNames[phase]
			if name == "" {
				name = phase
			}
			marker := ""
			if ms == slowestTime && ms > 100 {
				marker = yellow(" ← Slowest")
				slowestPhase = name
			} else if ms > 100 {
				marker = yellow(" ⚠")
			}
			fmt.Printf("  %-18s %6dms%s\n", name+":", ms, marker)
		}
	}

	// Print any phases not in our order list
	var extra []string
	for phase := range timings {
		found := false
		for _, p := range phaseOrder {
			if p == phase {
				found = true
				break
			}
		}
		if !found {
			extra = append(extra, phase)
		}
	}
	sort.Strings(extra)
	for _, phase := range extra {
		ms := timings[phase]
		fmt.Printf("  %-18s %6dms\n", phase+":", ms)
	}

	fmt.Printf("  %s\n", strings.Repeat("-", 28))
	fmt.Printf("  %-18s %6dms\n", "Total:", total)

	// Add warning for slow phases
	if slowestPhase != "" && slowestTime > 100 {
		fmt.Printf("\n%s %s took %dms (threshold: 100ms)\n",
			yellow("Warning:"), slowestPhase, slowestTime)
		fmt.Println("  Consider checking for complex recursive types.")
	}
}

func outputInterface(modulePath string) {
	// Read the file
	filename := modulePath
	if !strings.HasSuffix(filename, ".ail") {
		// Try to resolve as module path
		filename = strings.ReplaceAll(modulePath, "/", string(filepath.Separator)) + ".ail"
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Type check and build interface
	cfg := pipeline.Config{
		DryLink: true, // Don't evaluate, just check
	}
	src := pipeline.Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	}

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Check for errors
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
		}
		os.Exit(1)
	}

	// Get the interface
	if result.Interface == nil {
		fmt.Fprintf(os.Stderr, "%s: no interface generated for module\n", red("Error"))
		os.Exit(1)
	}

	// Output normalized JSON
	jsonBytes, err := result.Interface.ToNormalizedJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to serialize interface: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Println(string(jsonBytes))
}

func exportTraining() {
	fmt.Printf("%s Exporting training data...\n", cyan("→"))

	// TODO: Implement training data export
	fmt.Printf("  Analyzing execution traces...\n")
	fmt.Printf("  Filtering high-quality traces (score > 0.8)...\n")
	fmt.Printf("  Formatting for fine-tuning...\n")

	fmt.Printf("\n%s Exported 0 training examples to training_data.jsonl\n", green("✓"))
}
