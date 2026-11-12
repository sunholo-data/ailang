package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

// testDeprecationWarnings returns the TC_DEPRECATED_001 warnings of a test
// file, from one check-mode compile of the file as written.
//
// `ailang test` compiles named test bodies through synthesized temp modules,
// so positions from those compiles point into the temp files, and the
// pipeline warnings they produce were never printed. A deprecated name used
// only in a test block (the common case downstream) gave no notice before
// alias removal turned it into IMP010. This compile is served by the compile
// cache on a warm run. A file that does not compile here reports nothing; the
// test run itself reports the error.
func testDeprecationWarnings(filename string) []elaborate.Warning {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}
	// TransientRoot: no MOD010 note for the root (the test run never printed
	// one, and this compile must add nothing but the deprecations) and no
	// cache entry for it; its imports are still served from the cache.
	res, err := pipeline.Run(pipeline.Config{DryLink: true, RelaxModules: true, TransientRoot: true},
		pipeline.Source{Code: string(content), Filename: filename})
	if err != nil {
		return nil
	}
	var out []elaborate.Warning
	for _, w := range res.Warnings {
		if _, ok := w.(*pipeline.DeprecatedUseWarning); ok {
			out = append(out, w)
		}
	}
	return out
}

// printedTestDeprecations de-duplicates across one `ailang test` run: a module
// imported by several test files is compiled once per file, and its warnings
// would otherwise print once per importer, under both a relative and an
// absolute spelling of its path.
var printedTestDeprecations = map[string]bool{}

// printTestDeprecationWarnings writes a test file's deprecation warnings to
// stderr, so stdout (human or --json results) is unchanged. Each use site is
// printed once per run, with its path relative to the working directory when
// it lies under it.
func printTestDeprecationWarnings(filename string) {
	cwd, _ := os.Getwd()
	for _, w := range testDeprecationWarnings(filename) {
		dw := w.(*pipeline.DeprecatedUseWarning)
		abs := dw.Location.File
		if a, err := filepath.Abs(abs); err == nil {
			abs = a
		}
		if r, err := filepath.EvalSymlinks(abs); err == nil {
			abs = r
		}
		key := fmt.Sprintf("%s|%d|%d|%s|%s.%s|%v", abs, dw.Location.Line, dw.Location.Column, dw.Code(), dw.Module, dw.Name, dw.UnusedImport)
		if printedTestDeprecations[key] {
			continue
		}
		printedTestDeprecations[key] = true
		if cwd != "" {
			if c, err := filepath.EvalSymlinks(cwd); err == nil {
				cwd = c
			}
			if rel, err := filepath.Rel(cwd, abs); err == nil && !strings.HasPrefix(rel, "..") {
				dw.Location.File = rel
			} else {
				dw.Location.File = abs
			}
		}
		fmt.Fprintf(os.Stderr, "%s\n", yellow(dw.String()))
	}
}
