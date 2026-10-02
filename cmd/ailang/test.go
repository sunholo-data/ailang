package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/pkg"
	ailangTesting "github.com/sunholo-data/ailang/internal/testing"
)

// parseTestArgs parses `ailang test` flags wherever they appear and returns
// every positional path, in order. Go's flag package stops at the first
// positional argument, so `ailang test a.ail b.ail --json` used to run only
// a.ail and silently drop both b.ail and --json (stapledons_godot report).
// Arguments after a literal "--" are paths even if they look like flags.
func parseTestArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var paths []string
	rest := args
	for {
		if err := fs.Parse(rest); err != nil {
			return nil, err
		}
		remaining := fs.Args()
		consumed := len(rest) - len(remaining)
		if consumed > 0 && rest[consumed-1] == "--" {
			return append(paths, remaining...), nil
		}
		if len(remaining) == 0 {
			return paths, nil
		}
		paths = append(paths, remaining[0])
		rest = remaining[1:]
	}
}

// collectTestFiles walks each path and returns every .ail file to test, in
// argument order, each file once. Leftover named-test body copies from an
// older `ailang test` that was interrupted (#1502) are skipped and reported:
// they duplicate a real test module, and re-running one doubled the suite.
func collectTestFiles(paths []string) ([]string, error) {
	var files, debris []string
	seen := make(map[string]bool)
	for _, root := range paths {
		err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(p, ".ail") {
				return nil
			}
			if pkg.IsNamedTestBodyFile(filepath.Base(p)) {
				debris = append(debris, p)
				return nil
			}
			key := filepath.Clean(p)
			if !seen[key] {
				seen[key] = true
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	warnNamedTestDebris(debris)
	return files, nil
}

// warnNamedTestDebris names leftover `_namedtest_body_*.ail` files on stderr.
func warnNamedTestDebris(debris []string) {
	if len(debris) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "%s skipping %d leftover named-test body file(s) from an interrupted older `ailang test` run — delete them, they are not part of the package:\n",
		yellow("⚠"), len(debris))
	for _, d := range debris {
		fmt.Fprintf(os.Stderr, "    %s\n", d)
	}
}

// runTestsV2 executes all tests in the given paths (files or directories)
// and reports one aggregate result; any failing file fails the run.
func runTestsV2(paths []string, formatStr string, colorEnabled bool, allowSkips bool, cfg ailangTesting.TestConfig) {
	preambleWriter := os.Stdout
	if formatStr == "json" {
		preambleWriter = os.Stderr
	}
	label := strings.Join(paths, ", ")
	fmt.Fprintf(preambleWriter, "%s Running tests in %s\n", cyan("→"), label)

	testFiles, err := collectTestFiles(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if len(testFiles) == 0 {
		fmt.Printf("%s No .ail files found in %s\n", yellow("⚠"), label)
		os.Exit(0)
	}

	// Aggregate results across all files
	aggregateResults := ailangTesting.NewSuiteResult("All Tests")
	aggregateResults.SetSeedMetadata(cfg)

	// Run tests for each file
	for _, file := range testFiles {
		fileResult := runTestFile(file, cfg)
		if fileResult != nil {
			// Merge results into aggregate
			aggregateResults.Tests = append(aggregateResults.Tests, fileResult.Tests...)
			aggregateResults.Properties = append(aggregateResults.Properties, fileResult.Properties...)
			aggregateResults.TotalTests += fileResult.TotalTests
			aggregateResults.PassedTests += fileResult.PassedTests
			aggregateResults.FailedTests += fileResult.FailedTests
			aggregateResults.SkippedTests += fileResult.SkippedTests
			aggregateResults.VacuousSkips += fileResult.VacuousSkips
			aggregateResults.TotalDuration += fileResult.TotalDuration
			aggregateResults.Engine.Merge(fileResult.Engine)
		}
	}
	reportTestEngine(aggregateResults, cfg)

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

	// Exit with appropriate code.
	// --allow-skips: treat an all-skipped suite as success (exit 0).
	// Default: all-skipped exits 1 per M-NAMED-TEST-BLOCKS spec.
	succeeded := aggregateResults.Success() || (allowSkips && aggregateResults.SuccessAllowingSkips())
	if !succeeded {
		os.Exit(1)
	}
}

// runTestFile runs all tests in a single file.
// Returns nil if the file has no tests.
func runTestFile(filename string, cfg ailangTesting.TestConfig) *ailangTesting.SuiteResult {
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
	result, err := ailangTesting.RunTestsFromFileWithConfig(filename, file, cfg)
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
func runPackageTests(dir string, formatStr string, colorEnabled bool, allowSkips bool, cfg ailangTesting.TestConfig) {
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
	var debris []string
	err = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(p, ".ail") {
			return nil
		}
		if pkg.IsNamedTestBodyFile(filepath.Base(p)) {
			debris = append(debris, p)
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

	warnNamedTestDebris(debris)

	// Print package info
	preambleWriter := os.Stdout
	if formatStr == "json" {
		preambleWriter = os.Stderr
	}
	fmt.Fprintf(preambleWriter, "%s Package %s\n", cyan("→"), bold(manifest.Package.Name))
	fmt.Fprintf(preambleWriter, "  %s source modules, %s test files\n",
		cyan(fmt.Sprintf("%d", len(sourceFiles))),
		cyan(fmt.Sprintf("%d", len(testFiles))))

	if len(testFiles) == 0 {
		fmt.Printf("%s No *_test.ail files found in package %s\n", yellow("⚠"), manifest.Package.Name)
		os.Exit(0)
	}

	fmt.Fprintln(preambleWriter)

	// Aggregate results across all test files
	aggregateResults := ailangTesting.NewSuiteResult(manifest.Package.Name)
	aggregateResults.SetSeedMetadata(cfg)

	for _, file := range testFiles {
		fileResult := runTestFile(file, cfg)
		if fileResult != nil {
			aggregateResults.Tests = append(aggregateResults.Tests, fileResult.Tests...)
			aggregateResults.Properties = append(aggregateResults.Properties, fileResult.Properties...)
			aggregateResults.TotalTests += fileResult.TotalTests
			aggregateResults.PassedTests += fileResult.PassedTests
			aggregateResults.FailedTests += fileResult.FailedTests
			aggregateResults.SkippedTests += fileResult.SkippedTests
			aggregateResults.VacuousSkips += fileResult.VacuousSkips
			aggregateResults.TotalDuration += fileResult.TotalDuration
			aggregateResults.Engine.Merge(fileResult.Engine)
		}
	}
	reportTestEngine(aggregateResults, cfg)

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

	// Exit with appropriate code.
	// --allow-skips: treat an all-skipped suite as success (exit 0).
	succeeded := aggregateResults.Success() || (allowSkips && aggregateResults.SuccessAllowingSkips())
	if !succeeded {
		os.Exit(1)
	}
}

// reportTestEngine names, on stderr, where named-test bodies ran under
// --bytecode, so an evaluator fallback is never silent. Stdout (including
// --json) is identical across engines.
func reportTestEngine(res *ailangTesting.SuiteResult, cfg ailangTesting.TestConfig) {
	if !cfg.Bytecode {
		return
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", cyan("→"), res.Engine.Summary())
}

// printTestHelp shows help for the test command.
func printTestHelp() {
	fmt.Println("Usage: ailang test [options] [path...]")
	fmt.Println()
	fmt.Println("Run tests in AILANG files.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --json             Machine-readable JSON output")
	fmt.Println("  --format <format>  Output format: human (default), json (legacy spelling of --json)")
	fmt.Println("  --no-color         Disable colored output")
	fmt.Println("  --package          Package mode: discover *_test.ail files via ailang.toml")
	fmt.Println("  --allow-skips      Exit 0 even if all tests were skipped (default: exit 1)")
	fmt.Println("  --seed N           Master seed for property generation (signed int64; replayable)")
	fmt.Println("  --random-seed      Read one master seed from crypto/rand and report it for replay")
	fmt.Println("  --max-recursion-depth N  Evaluator call-depth limit for test bodies (default 10000, as ailang run)")
	fmt.Println("  --bytecode         Run named-test bodies on the bytecode VM where they compile;")
	fmt.Println("                     others fall back to the evaluator (inline tests and properties")
	fmt.Println("                     always use the evaluator). Outcomes and seeds are identical.")
	fmt.Println("  --strict-bytecode  As --bytecode, but a body the VM cannot run fails its test")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang test                    # Run all tests in current directory")
	fmt.Println("  ailang test file.ail           # Run tests in specific file")
	fmt.Println("  ailang test a.ail b.ail        # Run tests in several files (one summary)")
	fmt.Println("  ailang test --json .           # JSON output for CI")
	fmt.Println("  ailang test --package .        # Run all *_test.ail in package")
	fmt.Println("  ailang test --package ../lib   # Run tests in another package")
	fmt.Println()
	fmt.Println("Test Syntax:")
	fmt.Println("  test \"name\" = expression       # Unit test (must return true)")
	fmt.Println("  property \"name\" (x: int) = ... # Property test (QuickCheck-style)")
}
