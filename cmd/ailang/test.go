package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/pkg"
	ailangTesting "github.com/sunholo/ailang/internal/testing"
)

// runTestsV2 executes all tests in the given path.
// Replaces the stub implementation in main.go.
func runTestsV2(path string, formatStr string, colorEnabled bool) {
	fmt.Printf("%s Running tests in %s\n", cyan("→"), path)

	// Collect all test files
	var testFiles []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(p, ".ail") {
			testFiles = append(testFiles, p)
		}

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if len(testFiles) == 0 {
		fmt.Printf("%s No .ail files found in %s\n", yellow("⚠"), path)
		os.Exit(0)
	}

	// Aggregate results across all files
	aggregateResults := ailangTesting.NewSuiteResult("All Tests")

	// Run tests for each file
	for _, file := range testFiles {
		fileResult := runTestFile(file)
		if fileResult != nil {
			// Merge results into aggregate
			aggregateResults.Tests = append(aggregateResults.Tests, fileResult.Tests...)
			aggregateResults.Properties = append(aggregateResults.Properties, fileResult.Properties...)
			aggregateResults.TotalTests += fileResult.TotalTests
			aggregateResults.PassedTests += fileResult.PassedTests
			aggregateResults.FailedTests += fileResult.FailedTests
			aggregateResults.SkippedTests += fileResult.SkippedTests
			aggregateResults.TotalDuration += fileResult.TotalDuration
		}
	}

	// Convert format string to OutputFormat
	var format ailangTesting.OutputFormat
	if formatStr == "json" {
		format = ailangTesting.FormatJSON
	} else {
		format = ailangTesting.FormatHuman
	}

	// Report results
	reporter := ailangTesting.NewReporter(format, os.Stdout, colorEnabled)
	if err := reporter.Report(aggregateResults); err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to generate report: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Exit with appropriate code
	if !aggregateResults.Success() {
		os.Exit(1)
	}
}

// runTestFile runs all tests in a single file.
// Returns nil if the file has no tests.
func runTestFile(filename string) *ailangTesting.SuiteResult {
	// Read source
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to read %s: %v\n", red("Error"), filename, err)
		return nil
	}

	// Parse the file
	lex := lexer.New(string(source), filename)
	p := parser.New(lex)
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		// File has syntax errors
		result := ailangTesting.NewSuiteResult(filename)
		result.FailedTests = 1
		result.TotalTests = 1
		// Create a failed test result for the parse error
		result.Tests = append(result.Tests, ailangTesting.TestResult{
			Name:   "parse",
			Status: ailangTesting.StatusFail,
		})
		return result
	}

	// Use the built-in test runner
	result, err := ailangTesting.RunTestsFromFile(filename, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to run tests in %s: %v\n", yellow("⚠"), filename, err)
		return nil
	}

	// Skip files with no tests
	if result.TotalTests == 0 {
		return nil
	}

	return result
}

// runPackageTests discovers and runs *_test.ail files in a package directory.
// It reads ailang.toml for package metadata, finds test files by convention,
// and runs them using the existing test framework.
func runPackageTests(dir string, formatStr string, colorEnabled bool) {
	// Load package manifest
	manifest, err := pkg.LoadManifest(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to load ailang.toml in %s: %v\n", red("Error"), dir, err)
		fmt.Fprintf(os.Stderr, "Hint: --package requires an ailang.toml in the target directory.\n")
		os.Exit(1)
	}

	// Discover test files (*_test.ail)
	var testFiles []string
	var sourceFiles []string
	err = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(p, ".ail") {
			return nil
		}
		if strings.HasSuffix(p, "_test.ail") {
			testFiles = append(testFiles, p)
		} else {
			sourceFiles = append(sourceFiles, p)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Print package info
	fmt.Printf("%s Package %s\n", cyan("→"), bold(manifest.Package.Name))
	fmt.Printf("  %s source modules, %s test files\n",
		cyan(fmt.Sprintf("%d", len(sourceFiles))),
		cyan(fmt.Sprintf("%d", len(testFiles))))

	if len(testFiles) == 0 {
		fmt.Printf("%s No *_test.ail files found in package %s\n", yellow("⚠"), manifest.Package.Name)
		os.Exit(0)
	}

	fmt.Println()

	// Aggregate results across all test files
	aggregateResults := ailangTesting.NewSuiteResult(manifest.Package.Name)

	for _, file := range testFiles {
		fileResult := runTestFile(file)
		if fileResult != nil {
			aggregateResults.Tests = append(aggregateResults.Tests, fileResult.Tests...)
			aggregateResults.Properties = append(aggregateResults.Properties, fileResult.Properties...)
			aggregateResults.TotalTests += fileResult.TotalTests
			aggregateResults.PassedTests += fileResult.PassedTests
			aggregateResults.FailedTests += fileResult.FailedTests
			aggregateResults.SkippedTests += fileResult.SkippedTests
			aggregateResults.TotalDuration += fileResult.TotalDuration
		}
	}

	// Convert format string to OutputFormat
	var format ailangTesting.OutputFormat
	if formatStr == "json" {
		format = ailangTesting.FormatJSON
	} else {
		format = ailangTesting.FormatHuman
	}

	// Report results
	reporter := ailangTesting.NewReporter(format, os.Stdout, colorEnabled)
	if err := reporter.Report(aggregateResults); err != nil {
		fmt.Fprintf(os.Stderr, "%s: Failed to generate report: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Exit with appropriate code
	if !aggregateResults.Success() {
		os.Exit(1)
	}
}

// printTestHelp shows help for the test command.
func printTestHelp() {
	fmt.Println("Usage: ailang test [options] [path]")
	fmt.Println()
	fmt.Println("Run tests in AILANG files.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --format <format>  Output format: human (default), json")
	fmt.Println("  --no-color         Disable colored output")
	fmt.Println("  --package          Package mode: discover *_test.ail files via ailang.toml")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang test                    # Run all tests in current directory")
	fmt.Println("  ailang test file.ail           # Run tests in specific file")
	fmt.Println("  ailang test --format json .    # JSON output for CI")
	fmt.Println("  ailang test --package .        # Run all *_test.ail in package")
	fmt.Println("  ailang test --package ../lib   # Run tests in another package")
	fmt.Println()
	fmt.Println("Test Syntax:")
	fmt.Println("  test \"name\" = expression       # Unit test (must return true)")
	fmt.Println("  property \"name\" (x: int) = ... # Property test (QuickCheck-style)")
}
