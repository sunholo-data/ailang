package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/sunholo-data/ailang/internal/check"
	"github.com/sunholo-data/ailang/internal/pkg"
	ailangTesting "github.com/sunholo-data/ailang/internal/testing"
)

// checkTestPackage uses the same discovery and analyses as check --package.
// Progress goes to the preamble stream; structured diagnostics stay in the suite.
func checkTestPackage(dir string, manifest *pkg.PackageManifest, progress io.Writer) *ailangTesting.SourceCheckResult {
	result := &ailangTesting.SourceCheckResult{Errors: []string{}, Warnings: []string{}}
	sources, orphans, err := check.DiscoverPackageSources(dir)
	if err != nil {
		result.Failed = 1
		result.Errors = append(result.Errors, fmt.Sprintf("failed to discover package sources: %v", err))
		return result
	}
	for _, file := range orphans {
		rel, err := filepath.Rel(dir, file)
		if err != nil {
			rel = file
		}
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s (no module declaration — orphan)", rel))
	}
	// Standalone check returns without export validation when no sources exist.
	if len(sources) == 0 {
		return result
	}
	report := check.CheckPackageFiles(dir, manifest, sources, check.PackageOptions{
		OnFile: func(fr check.FileResult) {
			if fr.Status == check.FilePassed {
				fmt.Fprintf(progress, "  ✓ %s\n", fr.Rel)
			} else {
				fmt.Fprintf(progress, "  ✗ %s\n", fr.Rel)
			}
		},
	})
	result.Passed, result.Failed = report.Passed, report.Failed
	result.Errors = append(result.Errors, report.Errors...)
	result.Warnings = append(result.Warnings, report.Warnings...)
	// Export-drift warnings originate in a map walk; keep the public report stable.
	sort.Strings(result.Warnings)
	return result
}

// reportPackageTestSuite emits exactly one report, including pre-pass failures
// and empty package suites, without redefining generic empty-suite success.
func reportPackageTestSuite(result *ailangTesting.SuiteResult, format string, colors bool) {
	outputFormat := ailangTesting.FormatHuman
	if format == "json" {
		outputFormat = ailangTesting.FormatJSON
	}
	if err := ailangTesting.NewReporter(outputFormat, os.Stdout, colors).Report(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to generate report: %v\n", err)
		os.Exit(1)
	}
}
