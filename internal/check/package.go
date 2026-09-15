package check

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// FileStatus classifies the outcome of checking one package source file.
type FileStatus int

const (
	// FilePassed: the file compiled and every analysis was clean.
	FilePassed FileStatus = iota
	// FileReadError: the file could not be read.
	FileReadError
	// FileTimeout: pipeline.Run did not finish within PackageOptions.Timeout.
	FileTimeout
	// FileCompileError: pipeline.Run returned an error or reported errors.
	FileCompileError
	// FileInterrefError: the M-PKG-INTERREF unresolved-reference analysis failed.
	FileInterrefError
	// FileStrictFallback: the STRICT_FALLBACK_001 publish-boundary check failed.
	FileStrictFallback
)

// FileResult is the outcome of checking one source file of a package.
type FileResult struct {
	// Path is the absolute file path; Rel is relative to the package dir
	// (falls back to Path when no relative form exists).
	Path   string
	Rel    string
	Status FileStatus
	// Errors are the messages appended to the package error list for this
	// file, already prefixed with the file (Rel or Path) as the CLI prints them.
	Errors []string
}

// PackageOptions configures CheckPackageFiles.
type PackageOptions struct {
	StrictSyntax bool
	DebugCompile bool
	// Timeout bounds each file's pipeline.Run; zero means unbounded.
	Timeout time.Duration
	// TimeoutLabel is the spelling used in the "timed out after %s" message;
	// defaults to Timeout.String().
	TimeoutLabel string
	// OnFile, when set, is called after each file is checked, in sorted
	// path order — the CLI's per-file progress line.
	OnFile func(FileResult)
}

// PackageReport is the outcome of CheckPackageFiles.
type PackageReport struct {
	Passed int
	Failed int
	// Errors lists every failure message in file order.
	Errors []string
	// Warnings lists manifest/export drift: exported modules that did not
	// compile, and compiled modules missing from [exports].modules.
	Warnings []string
	Files    []FileResult
}

// CheckPackageFiles checks every source file of an AILANG package through the
// pipeline (which auto-routes files with imports through module resolution),
// runs the package-boundary analyses on each, and validates the manifest's
// exports against what compiled. sourceFiles are checked in sorted order.
func CheckPackageFiles(absDir string, manifest *pkg.PackageManifest, sourceFiles []string, opts PackageOptions) PackageReport {
	sort.Strings(sourceFiles)

	timeoutLabel := opts.TimeoutLabel
	if timeoutLabel == "" && opts.Timeout > 0 {
		timeoutLabel = opts.Timeout.String()
	}

	var report PackageReport
	compiledModules := make(map[string]bool)

	record := func(fr FileResult) {
		report.Files = append(report.Files, fr)
		report.Errors = append(report.Errors, fr.Errors...)
		if fr.Status == FilePassed {
			report.Passed++
		} else {
			report.Failed++
		}
		if opts.OnFile != nil {
			opts.OnFile(fr)
		}
	}

	for _, file := range sourceFiles {
		rel, _ := filepath.Rel(absDir, file)
		if rel == "" {
			rel = file
		}
		fr := FileResult{Path: file, Rel: rel}

		content, err := os.ReadFile(file)
		if err != nil {
			fr.Status = FileReadError
			fr.Errors = []string{fmt.Sprintf("%s: cannot read: %v", file, err)}
			record(fr)
			continue
		}

		cfg := pipeline.Config{
			DryLink:          true,
			StrictSyntaxMode: opts.StrictSyntax,
			RelaxModules:     true, // Package mode: MOD010 relaxed (manifest validates module names)
			DebugCompile:     opts.DebugCompile,
		}
		src := pipeline.Source{
			Code:     string(content),
			Filename: file,
			IsREPL:   false,
		}

		var result pipeline.Result
		var checkErr error

		if opts.Timeout > 0 {
			done := make(chan struct{})
			go func() {
				result, checkErr = pipeline.Run(cfg, src)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(opts.Timeout):
				fr.Status = FileTimeout
				fr.Errors = []string{fmt.Sprintf("%s: timed out after %s", file, timeoutLabel)}
				record(fr)
				continue
			}
		} else {
			result, checkErr = pipeline.Run(cfg, src)
		}

		if checkErr != nil {
			fr.Status = FileCompileError
			fr.Errors = []string{fmt.Sprintf("%s: %v", rel, checkErr)}
			record(fr)
			continue
		}

		if len(result.Errors) > 0 {
			fr.Status = FileCompileError
			for _, e := range result.Errors {
				fr.Errors = append(fr.Errors, fmt.Sprintf("%s: %v", rel, e))
			}
			record(fr)
			continue
		}

		// M-PKG-INTERREF: Check that module-level function references resolve.
		// The pipeline type-checks successfully, but the resolver evaluates Let
		// bindings sequentially. If function B references function A via core.Var,
		// A must be defined in an earlier Let/LetRec. Catch violations here so
		// check --package doesn't give false confidence.
		if interrefWarns := InterFunctionRefs(result); len(interrefWarns) > 0 {
			fr.Status = FileInterrefError
			for _, w := range interrefWarns {
				fr.Errors = append(fr.Errors, fmt.Sprintf("%s: %s", rel, w))
			}
			record(fr)
			continue
		}

		// M-CHECK-STRICT-FALLBACKS: at the publish boundary an empty/default
		// `Ok(...)` in a Result-returning function is a HARD ERROR (exit 1) —
		// fail loudly, don't nag. The same finding is a non-blocking warning in
		// plain `ailang check`. Suppress per-function with @allow_empty_ok.
		if sfErrs := StrictFallbacks(result); len(sfErrs) > 0 {
			fr.Status = FileStrictFallback
			for _, e := range sfErrs {
				fr.Errors = append(fr.Errors, fmt.Sprintf("%s: %s", rel, e))
			}
			record(fr)
			continue
		}

		fr.Status = FilePassed
		record(fr)

		// Track compiled module for export validation
		modPath := ModulePathFromFile(file)
		if modPath != "" {
			compiledModules[modPath] = true
		}
	}

	// Validate exports: check that each exported module compiled successfully
	for _, exportedMod := range manifest.Exports.Modules {
		if !compiledModules[exportedMod] {
			report.Warnings = append(report.Warnings, fmt.Sprintf("exported module %q not found or failed to compile", exportedMod))
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
			report.Warnings = append(report.Warnings, fmt.Sprintf("module %q compiled but not listed in [exports].modules", mod))
		}
	}

	return report
}

// DiscoverPackageSources finds all .ail files in a package directory and
// separates them into source files (with module declarations) and orphan files
// (without module declarations). Returns (sourceFiles, orphanFiles, error).
func DiscoverPackageSources(dir string) ([]string, []string, error) {
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

		if HasModuleDeclaration(string(content)) {
			sourceFiles = append(sourceFiles, path)
		} else {
			orphanFiles = append(orphanFiles, path)
		}
		return nil
	})

	return sourceFiles, orphanFiles, walkErr
}

// HasModuleDeclaration checks if AILANG source code contains a module declaration.
func HasModuleDeclaration(code string) bool {
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			return true
		}
	}
	return false
}

// ModulePathFromFile extracts the module path from a source file by reading
// its module declaration. Returns empty string if no module declaration found.
func ModulePathFromFile(file string) string {
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
