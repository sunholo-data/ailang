package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/check"
	"github.com/sunholo-data/ailang/internal/pkg"
	otelplatform "github.com/sunholo-data/ailang/internal/platform/otel"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// checkPackageWithContext checks all source files in an AILANG package: load
// ailang.toml, discover sources, hand them to check.CheckPackageFiles, print.
// The analyses and the per-file loop live in internal/check (M-V1-SIMPLIFY-S2 M3).
func checkPackageWithContext(dir string, strictSyntax bool, relaxModules bool, timeout string, debugCompile bool, jsonFlag bool, quietFlag bool) {
	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot resolve path '%s': %v\n", red("Error"), dir, err)
		os.Exit(1)
	}

	// Initialize telemetry
	ctx := context.Background()
	shutdownTelemetry, err := otelplatform.Init(ctx, "ailang-check-package")
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
	sourceFiles, orphanFiles, err := check.DiscoverPackageSources(absDir)
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

	report := check.CheckPackageFiles(absDir, manifest, sourceFiles, check.PackageOptions{
		StrictSyntax: strictSyntax,
		DebugCompile: debugCompile,
		Timeout:      timeoutDuration,
		TimeoutLabel: timeout,
		OnFile: func(fr check.FileResult) {
			if jsonFlag || quietFlag {
				return
			}
			switch fr.Status {
			case check.FilePassed:
				fmt.Printf("  %s %s\n", green("✓"), fr.Rel)
			case check.FileCompileError:
				fmt.Printf("  %s %s\n", red("✗"), fr.Rel)
			case check.FileInterrefError:
				fmt.Printf("  %s %s (inter-function reference errors)\n", red("✗"), fr.Rel)
			case check.FileStrictFallback:
				fmt.Printf("  %s %s (STRICT_FALLBACK_001)\n", red("✗"), fr.Rel)
			}
		},
	})
	allErrors, warnings, passed, failed := report.Errors, report.Warnings, report.Passed, report.Failed

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
