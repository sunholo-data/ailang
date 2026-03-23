package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/pipeline"
	"github.com/sunholo/ailang/internal/pkg"
	"github.com/sunholo/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// checkPackageWithContext checks all source files in an AILANG package by reading
// ailang.toml, discovering source files, running each through the pipeline (which
// auto-routes files with imports through module resolution), and validating exports.
func checkPackageWithContext(dir string, strictSyntax bool, relaxModules bool, timeout string, debugCompile bool, jsonFlag bool, quietFlag bool) {
	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot resolve path '%s': %v\n", red("Error"), dir, err)
		os.Exit(1)
	}

	// Initialize telemetry
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-check-package")
	if err != nil {
		// Non-fatal: continue without telemetry
	} else {
		defer shutdownTelemetry(ctx)
	}
	// Start root span
	_, span := checkTracer.Start(telemetry.ExtractTraceContext(ctx), "ailang.check.package",
		trace.WithAttributes(
			attribute.String("package.dir", absDir),
		),
	)
	defer span.End()

	// Load manifest
	manifest, err := pkg.LoadManifest(absDir)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if jsonFlag {
			outputCheckJSON(checkJSONOutput{
				File:       absDir,
				Passed:     false,
				ErrorCount: 1,
				Errors: []checkJSONError{{
					Code:    "PKG_MANIFEST",
					Message: fmt.Sprintf("cannot load ailang.toml: %v", err),
					File:    filepath.Join(absDir, "ailang.toml"),
				}},
			})
		} else {
			fmt.Fprintf(os.Stderr, "%s: cannot load ailang.toml in '%s': %v\n", red("Error"), absDir, err)
		}
		os.Exit(1)
	}

	if !jsonFlag && !quietFlag {
		fmt.Printf("%s Checking package %s (%d exported modules)...\n\n",
			cyan("→"), manifest.Package.Name, len(manifest.Exports.Modules))
	}

	// Discover all .ail source files in the package directory
	sourceFiles, orphanFiles, err := discoverPackageSources(absDir, manifest)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		fmt.Fprintf(os.Stderr, "%s: failed to discover package sources: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if len(sourceFiles) == 0 {
		if !jsonFlag && !quietFlag {
			fmt.Printf("%s No .ail source files found in %s\n", yellow("!"), absDir)
		}
		return
	}

	// Report orphan files (files without module declarations)
	if len(orphanFiles) > 0 && !jsonFlag && !quietFlag {
		for _, f := range orphanFiles {
			rel, _ := filepath.Rel(absDir, f)
			if rel == "" {
				rel = f
			}
			fmt.Printf("  %s %s (no module declaration — orphan)\n", yellow("!"), rel)
		}
	}

	// Sort source files for deterministic order
	sort.Strings(sourceFiles)

	// Suppress warnings in JSON/quiet mode
	if jsonFlag || quietFlag {
		os.Setenv("AILANG_QUIET_WARNINGS", "1")
	}

	// Parse timeout once
	var timeoutDuration time.Duration
	if timeout != "" {
		timeoutDuration, err = time.ParseDuration(timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: invalid timeout duration '%s': %v\n", red("Error"), timeout, err)
			os.Exit(1)
		}
	}

	// Check each source file through the pipeline.
	// The pipeline auto-routes files with imports through runModuleWithContext,
	// which handles cross-module type resolution.
	var passed, failed int
	var allErrors []string
	compiledModules := make(map[string]bool)

	for _, file := range sourceFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: cannot read: %v", file, err))
			failed++
			continue
		}

		cfg := pipeline.Config{
			DryLink:          true,
			StrictSyntaxMode: strictSyntax,
			RelaxModules:     true, // Package mode: MOD010 relaxed (manifest validates module names)
			DebugCompile:     debugCompile,
		}
		src := pipeline.Source{
			Code:     string(content),
			Filename: file,
			IsREPL:   false,
		}

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
			case <-time.After(timeoutDuration):
				allErrors = append(allErrors, fmt.Sprintf("%s: timed out after %s", file, timeout))
				failed++
				continue
			}
		} else {
			result, checkErr = pipeline.Run(cfg, src)
		}

		rel, _ := filepath.Rel(absDir, file)
		if rel == "" {
			rel = file
		}

		if checkErr != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: %v", rel, checkErr))
			failed++
			if !jsonFlag && !quietFlag {
				fmt.Printf("  %s %s\n", red("✗"), rel)
			}
			continue
		}

		if len(result.Errors) > 0 {
			for _, e := range result.Errors {
				allErrors = append(allErrors, fmt.Sprintf("%s: %v", rel, e))
			}
			failed++
			if !jsonFlag && !quietFlag {
				fmt.Printf("  %s %s\n", red("✗"), rel)
			}
			continue
		}

		passed++
		if !jsonFlag && !quietFlag {
			fmt.Printf("  %s %s\n", green("✓"), rel)
		}

		// Track compiled module for export validation
		modPath := modulePathFromFile(file, absDir)
		if modPath != "" {
			compiledModules[modPath] = true
		}
	}

	// Validate exports: check that each exported module compiled successfully
	var warnings []string
	for _, exportedMod := range manifest.Exports.Modules {
		if !compiledModules[exportedMod] {
			warnings = append(warnings, fmt.Sprintf("exported module %q not found or failed to compile", exportedMod))
		}
	}

	// Report warnings for modules compiled but not in exports
	for mod := range compiledModules {
		found := false
		for _, exp := range manifest.Exports.Modules {
			if mod == exp {
				found = true
				break
			}
		}
		if !found {
			warnings = append(warnings, fmt.Sprintf("module %q compiled but not listed in [exports].modules", mod))
		}
	}

	// Output results
	if jsonFlag {
		jsonErrors := make([]checkJSONError, 0, len(allErrors)+len(warnings))
		for _, e := range allErrors {
			jsonErrors = append(jsonErrors, checkJSONError{
				Code:    "ERROR",
				Message: e,
			})
		}
		for _, w := range warnings {
			jsonErrors = append(jsonErrors, checkJSONError{
				Code:    "WARNING",
				Message: w,
			})
		}
		outputCheckJSON(checkJSONOutput{
			File:       absDir,
			Passed:     failed == 0,
			ErrorCount: len(allErrors),
			Errors:     jsonErrors,
		})
		if failed > 0 {
			os.Exit(1)
		}
		return
	}

	// Human-readable output
	if !quietFlag {
		fmt.Println()
	}

	// Print warnings
	if len(warnings) > 0 && !quietFlag {
		for _, w := range warnings {
			fmt.Printf("  %s %s\n", yellow("!"), w)
		}
		fmt.Println()
	}

	if failed == 0 {
		span.SetStatus(codes.Ok, "package check passed")
		if !quietFlag {
			fmt.Printf("%s %d files checked, all passed!\n", green("✓"), passed)
		}
	} else {
		span.SetStatus(codes.Error, "package check failed")
		if !quietFlag {
			fmt.Printf("%s %d files checked: %d passed, %d failed\n",
				red("✗"), passed+failed, passed, failed)
			fmt.Println()
			fmt.Println("Errors:")
			for _, e := range allErrors {
				fmt.Printf("  • %s\n", e)
			}
		} else {
			for _, e := range allErrors {
				fmt.Fprintf(os.Stderr, "%s\n", e)
			}
		}
		os.Exit(1)
	}
}

// discoverPackageSources finds all .ail files in a package directory and
// separates them into source files (with module declarations) and orphan files
// (without module declarations). Returns (sourceFiles, orphanFiles, error).
func discoverPackageSources(dir string, manifest *pkg.PackageManifest) ([]string, []string, error) {
	var sourceFiles, orphanFiles []string

	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip hidden directories and common non-source dirs
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "_vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".ail") {
			return nil
		}

		// Check if file has a module declaration
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // skip unreadable files
		}

		if hasModuleDeclaration(string(content)) {
			sourceFiles = append(sourceFiles, path)
		} else {
			orphanFiles = append(orphanFiles, path)
		}
		return nil
	})

	return sourceFiles, orphanFiles, walkErr
}

// hasModuleDeclaration checks if AILANG source code contains a module declaration.
func hasModuleDeclaration(code string) bool {
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			return true
		}
	}
	return false
}

// modulePathFromFile extracts the module path from a source file by reading
// its module declaration. Returns empty string if no module declaration found.
func modulePathFromFile(file string, pkgDir string) string {
	content, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}
