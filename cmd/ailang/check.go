package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// checkTracer is the OpenTelemetry tracer for check command instrumentation.
var checkTracer = otel.Tracer("ailang.check")

// checkJSONOutput represents the JSON output format for ailang check --json
type checkJSONOutput struct {
	File       string           `json:"file"`
	Passed     bool             `json:"passed"`
	ErrorCount int              `json:"error_count"`
	Errors     []checkJSONError `json:"errors"`
}

// checkJSONError represents a single error in JSON check output
type checkJSONError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	File       string `json:"file,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// outputCheckJSON writes a checkJSONOutput to stdout
func outputCheckJSON(out checkJSONOutput) {
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// errorToCheckJSONError converts an error to a structured JSON error entry
func errorToCheckJSONError(err error, filename string) checkJSONError {
	if rep, ok := ailerrors.AsReport(err); ok {
		entry := checkJSONError{
			Code:    rep.Code,
			Message: rep.Message,
		}
		if rep.Span != nil {
			entry.File = rep.Span.Start.File
			entry.Line = rep.Span.Start.Line
			entry.Column = rep.Span.Start.Column
		}
		if rep.Fix != nil {
			entry.Suggestion = rep.Fix.Suggestion
		}
		return entry
	}
	// Fallback for non-Report errors
	return checkJSONError{
		Code:    "ERROR",
		Message: err.Error(),
		File:    filename,
	}
}

func checkFile(filename string, strictSyntax bool, relaxModules bool, timeout string, debugCompile bool, jsonFlag bool, quietFlag bool) {
	// Initialize telemetry (traces exported if GOOGLE_CLOUD_PROJECT or OTEL_EXPORTER_OTLP_ENDPOINT set)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-check")
	if err != nil {
		// Non-fatal: continue without telemetry
	} else {
		defer shutdownTelemetry(ctx)
	}

	// Extract parent trace context from environment (if running as subprocess)
	// This enables distributed tracing when ailang is spawned by coordinator/executors
	ctx = telemetry.ExtractTraceContext(ctx)

	// Extract correlation IDs for span attributes (fallback linking)
	corrTaskID, sessionID := telemetry.ExtractCorrelationIDs()

	// Generate task ID for this check command
	checkTaskID := fmt.Sprintf("check_%d", time.Now().UnixNano())

	// Inherit parent task from environment if set
	// This enables automatic hierarchy linking when ailang exec spawns ailang check
	parentTaskID := os.Getenv("AILANG_PARENT_TASK_ID")

	// If no parent task, use generic root marker for analytics
	// This ensures all checks appear in Observatory hierarchy views
	if parentTaskID == "" {
		parentTaskID = "root"
	}

	// Export current task ID so child processes inherit the hierarchy
	os.Setenv("AILANG_PARENT_TASK_ID", checkTaskID)

	// Parse timeout for span attribute
	var timeoutMs int64
	if timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			timeoutMs = d.Milliseconds()
		}
	}

	// Start root span for check operation
	ctx, span := checkTracer.Start(ctx, "ailang.check",
		trace.WithAttributes(
			attribute.String("file.path", filename),
			attribute.Int64("timeout_ms", timeoutMs),
			attribute.String("exec.task_id", checkTaskID),
			attribute.String("exec.parent_task_id", parentTaskID),
		),
	)
	defer span.End()

	// Add correlation IDs as span attributes (if present)
	if corrTaskID != "" {
		span.SetAttributes(attribute.String("ailang.task_id", corrTaskID))
	}
	if sessionID != "" {
		span.SetAttributes(attribute.String("ailang.session_id", sessionID))
	}

	// Check if path is a directory
	info, err := os.Stat(filename)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		fmt.Fprintf(os.Stderr, "%s: cannot access '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	if info.IsDir() {
		span.SetAttributes(attribute.Bool("is_directory", true))
		checkDirectoryWithContext(ctx, filename, strictSyntax, relaxModules, timeout, debugCompile, jsonFlag, quietFlag)
		return
	}

	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		fmt.Fprintf(os.Stderr, "%s: cannot read file '%s': %v\n", red("Error"), filename, err)
		os.Exit(1)
	}

	// Parse timeout if specified
	var timeoutDuration time.Duration
	if timeout != "" {
		var err error
		timeoutDuration, err = time.ParseDuration(timeout)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "invalid timeout")
			fmt.Fprintf(os.Stderr, "%s: invalid timeout duration '%s': %v\n", red("Error"), timeout, err)
			fmt.Println("Examples: 30s, 2m, 1m30s")
			os.Exit(1)
		}
	}

	// Suppress warnings in JSON/quiet mode so they don't pollute output
	if jsonFlag || quietFlag {
		os.Setenv("AILANG_QUIET_WARNINGS", "1")
	}

	// Type check / Effect check progress lines (suppressed in json/quiet mode)
	if !jsonFlag && !quietFlag {
		fmt.Printf("%s Type checking %s...\n", cyan("→"), filename)
		fmt.Printf("%s Effect checking...\n", cyan("→"))
	}

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
		runCheckWithTimeoutAndContext(ctx, cfg, src, timeoutDuration, filename, jsonFlag, quietFlag)
	} else {
		runCheckWithContext(ctx, cfg, src, filename, jsonFlag, quietFlag)
	}
}

// runCheckWithContext executes the pipeline without timeout, with telemetry context
func runCheckWithContext(ctx context.Context, cfg pipeline.Config, src pipeline.Source, filename string, jsonFlag bool, quietFlag bool) {
	// Start result span
	_, resultSpan := checkTracer.Start(ctx, "check.result")
	defer resultSpan.End()

	result, err := pipeline.Run(cfg, src)
	if err != nil {
		resultSpan.SetAttributes(
			attribute.Bool("passed", false),
			attribute.Int("errors.count", 1),
		)
		resultSpan.RecordError(err)
		resultSpan.SetStatus(codes.Error, err.Error())
		if jsonFlag {
			outputCheckJSON(checkJSONOutput{
				File:       filename,
				Passed:     false,
				ErrorCount: 1,
				Errors:     []checkJSONError{errorToCheckJSONError(err, filename)},
			})
		} else {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		}
		os.Exit(1)
	}

	// Check for any errors
	if len(result.Errors) > 0 {
		resultSpan.SetAttributes(
			attribute.Bool("passed", false),
			attribute.Int("errors.count", len(result.Errors)),
		)
		resultSpan.SetStatus(codes.Error, "check failed")
		if jsonFlag {
			jsonErrors := make([]checkJSONError, len(result.Errors))
			for i, e := range result.Errors {
				jsonErrors[i] = errorToCheckJSONError(e, filename)
			}
			outputCheckJSON(checkJSONOutput{
				File:       filename,
				Passed:     false,
				ErrorCount: len(result.Errors),
				Errors:     jsonErrors,
			})
		} else {
			for _, e := range result.Errors {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
			}
		}
		os.Exit(1)
	}

	// Print phase timing breakdown if debug-compile enabled
	if cfg.DebugCompile && len(result.PhaseTimings) > 0 {
		printPhaseTimings(result.PhaseTimings)
	}

	resultSpan.SetAttributes(
		attribute.Bool("passed", true),
		attribute.Int("errors.count", 0),
	)
	resultSpan.SetStatus(codes.Ok, "check passed")
	if jsonFlag {
		outputCheckJSON(checkJSONOutput{
			File:       filename,
			Passed:     true,
			ErrorCount: 0,
			Errors:     []checkJSONError{},
		})
	} else if !quietFlag {
		fmt.Printf("\n%s No errors found!\n", green("✓"))
	}
}

// runCheckWithTimeoutAndContext executes the pipeline with a watchdog timer and telemetry
func runCheckWithTimeoutAndContext(ctx context.Context, cfg pipeline.Config, src pipeline.Source, timeout time.Duration, filename string, jsonFlag bool, quietFlag bool) {
	// Start result span
	_, resultSpan := checkTracer.Start(ctx, "check.result",
		trace.WithAttributes(
			attribute.Int64("timeout_ms", timeout.Milliseconds()),
		),
	)
	defer resultSpan.End()

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
			resultSpan.SetAttributes(
				attribute.Bool("passed", false),
				attribute.Int("errors.count", 1),
			)
			resultSpan.RecordError(r.err)
			resultSpan.SetStatus(codes.Error, r.err.Error())
			if jsonFlag {
				outputCheckJSON(checkJSONOutput{
					File:       filename,
					Passed:     false,
					ErrorCount: 1,
					Errors:     []checkJSONError{errorToCheckJSONError(r.err, filename)},
				})
			} else {
				fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), r.err)
			}
			os.Exit(1)
		}
		if len(r.result.Errors) > 0 {
			resultSpan.SetAttributes(
				attribute.Bool("passed", false),
				attribute.Int("errors.count", len(r.result.Errors)),
			)
			resultSpan.SetStatus(codes.Error, "check failed")
			if jsonFlag {
				jsonErrors := make([]checkJSONError, len(r.result.Errors))
				for i, e := range r.result.Errors {
					jsonErrors[i] = errorToCheckJSONError(e, filename)
				}
				outputCheckJSON(checkJSONOutput{
					File:       filename,
					Passed:     false,
					ErrorCount: len(r.result.Errors),
					Errors:     jsonErrors,
				})
			} else {
				for _, e := range r.result.Errors {
					fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), e)
				}
			}
			os.Exit(1)
		}
		// Print phase timing breakdown if debug-compile enabled
		if cfg.DebugCompile && len(r.result.PhaseTimings) > 0 {
			printPhaseTimings(r.result.PhaseTimings)
		}
		resultSpan.SetAttributes(
			attribute.Bool("passed", true),
			attribute.Int("errors.count", 0),
		)
		resultSpan.SetStatus(codes.Ok, "check passed")
		if jsonFlag {
			outputCheckJSON(checkJSONOutput{
				File:       filename,
				Passed:     true,
				ErrorCount: 0,
				Errors:     []checkJSONError{},
			})
		} else if !quietFlag {
			fmt.Printf("\n%s No errors found!\n", green("✓"))
		}

	case <-time.After(timeout):
		resultSpan.SetAttributes(
			attribute.Bool("passed", false),
			attribute.Bool("timed_out", true),
		)
		resultSpan.SetStatus(codes.Error, "timeout")
		if jsonFlag {
			outputCheckJSON(checkJSONOutput{
				File:       filename,
				Passed:     false,
				ErrorCount: 1,
				Errors: []checkJSONError{{
					Code:    "TIMEOUT",
					Message: fmt.Sprintf("compilation timed out after %s", timeout),
					File:    filename,
				}},
			})
		} else {
			fmt.Fprintf(os.Stderr, "\n%s Compilation timed out after %s\n", red("TIMEOUT"), timeout)
			fmt.Fprintf(os.Stderr, "File: %s\n\n", filename)

			// Dump goroutine stacks for debugging
			buf := make([]byte, 1<<20) // 1MB buffer
			n := runtime.Stack(buf, true)
			fmt.Fprintf(os.Stderr, "Stack dump (all goroutines):\n%s\n", buf[:n])

			fmt.Fprintf(os.Stderr, "\n%s This may indicate cyclic types. Try: ailang debug cycles %s\n",
				yellow("Hint:"), filename)
		}
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

// exportTraining is defined in export_training.go

// checkDirectoryWithContext recursively checks all .ail files with telemetry
func checkDirectoryWithContext(ctx context.Context, dir string, strictSyntax bool, relaxModules bool, timeout string, debugCompile bool, jsonFlag bool, quietFlag bool) {
	var files []string

	// Walk directory to find all .ail files
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ail") {
			files = append(files, path)
		}
		return nil
	})

	if walkErr != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot walk directory '%s': %v\n", red("Error"), dir, walkErr)
		os.Exit(1)
	}

	// Handle empty directory
	if len(files) == 0 {
		if !jsonFlag && !quietFlag {
			fmt.Printf("%s No .ail files found in %s\n", yellow("⚠"), dir)
		}
		return
	}

	// Sort files for deterministic ordering
	sort.Strings(files)

	if !jsonFlag && !quietFlag {
		fmt.Printf("%s Checking %d .ail files in %s...\n\n", cyan("→"), len(files), dir)
	}

	// Track results
	var passed, failed int
	var errors []string

	// Parse timeout once if specified
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

	// Check AILANG_RELAX_MODULES environment variable
	relaxModulesEffective := relaxModules
	if envVal := os.Getenv("AILANG_RELAX_MODULES"); envVal != "" {
		switch strings.ToLower(envVal) {
		case "1", "true", "yes":
			relaxModulesEffective = true
		}
	}

	// Check each file
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: cannot read: %v", file, err))
			failed++
			continue
		}

		cfg := pipeline.Config{
			DryLink:          true,
			StrictSyntaxMode: strictSyntax,
			RelaxModules:     relaxModulesEffective,
			DebugCompile:     debugCompile,
		}
		src := pipeline.Source{
			Code:     string(content),
			Filename: file,
			IsREPL:   false,
		}

		// Run check (with or without timeout)
		var result pipeline.Result
		var checkErr error

		if timeoutDuration > 0 {
			done := make(chan struct{})
			go func() {
				result, checkErr = pipeline.Run(cfg, src)
				close(done)
			}()

			select {
			case <-done:
				// Check completed
			case <-time.After(timeoutDuration):
				errors = append(errors, fmt.Sprintf("%s: timed out after %s", file, timeout))
				failed++
				continue
			}
		} else {
			result, checkErr = pipeline.Run(cfg, src)
		}

		if checkErr != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", file, checkErr))
			failed++
			continue
		}

		if len(result.Errors) > 0 {
			for _, e := range result.Errors {
				errors = append(errors, fmt.Sprintf("%s: %v", file, e))
			}
			failed++
			if !jsonFlag && !quietFlag {
				fmt.Printf("  %s %s\n", red("✗"), file)
			}
		} else {
			passed++
			if !jsonFlag && !quietFlag {
				fmt.Printf("  %s %s\n", green("✓"), file)
			}
		}
	}

	// Print summary
	if jsonFlag {
		// JSON output for directory check
		jsonErrors := make([]checkJSONError, 0, len(errors))
		for _, e := range errors {
			jsonErrors = append(jsonErrors, checkJSONError{
				Code:    "ERROR",
				Message: e,
			})
		}
		outputCheckJSON(checkJSONOutput{
			File:       dir,
			Passed:     failed == 0,
			ErrorCount: len(errors),
			Errors:     jsonErrors,
		})
		if failed > 0 {
			os.Exit(1)
		}
	} else {
		if !quietFlag {
			fmt.Println()
		}
		if failed == 0 {
			if !quietFlag {
				fmt.Printf("%s %d files checked, all passed!\n", green("✓"), passed)
			}
		} else {
			if !quietFlag {
				fmt.Printf("%s %d files checked: %d passed, %d failed\n", red("✗"), passed+failed, passed, failed)
				fmt.Println()
				fmt.Println("Errors:")
				for _, e := range errors {
					fmt.Printf("  • %s\n", e)
				}
			} else {
				// Even in quiet mode, print errors to stderr
				for _, e := range errors {
					fmt.Fprintf(os.Stderr, "%s\n", e)
				}
			}
			os.Exit(1)
		}
	}
}
