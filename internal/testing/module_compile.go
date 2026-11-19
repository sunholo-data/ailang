package testing

import (
	"os"
	"regexp"
	"strconv"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

// A module that does not compile cannot run any of its tests: every inline
// row, named test and property compiles the module first. Reporting that as
// one failure per test ("failed to extract function binding: …", six times
// for six tests) buried the one error that mattered, and since #2248 a
// contract type error (TC_CONTRACT_001) is such a module error. So it is
// reported once, as itself, in place of every test — with no extra compile:
//
//   - when the named-test batch failed at a position in the module's own
//     source, or in a dependency's (none inside a test entry), that error
//     is the module's;
//   - when there is no batch, the file's inline rows share one memoized
//     compile of the module (memo key "binding", ExtractFunctionBinding),
//     which is run first and its error reported.
//
// Module-less files are left alone: their inline rows compile a different
// text per function (effectful functions stripped).

// moduleErrorOutsideEntries reports whether a batch compile error points only
// into the stripped module (lines 1..baseLines of the batched source) or into
// a dependency's file, never into an appended test entry: then the module or
// one of its imports, not a test, fails to compile.
func moduleErrorOutsideEntries(msg string, baseLines int) bool {
	matches := batchTempPos.FindAllStringSubmatch(msg, -1)
	for _, m := range matches {
		line, err := strconv.Atoi(m[1])
		if err != nil || line < 1 || line > baseLines {
			return false
		}
	}
	if len(matches) > 0 {
		return true
	}
	// No position in the batched file: a dependency's error names its own
	// file. An entry's effect error carries no position at all.
	return anySourcePos.MatchString(msg)
}

// anySourcePos matches a "file.ail:line" position anywhere in a message.
var anySourcePos = regexp.MustCompile(`\.ail:\d+`)

// inlineModuleConfig is the pipeline configuration of the shared inline-row
// compile of a module file (ExtractFunctionBinding's module case).
func (e *Executor) inlineModuleConfig() pipeline.Config {
	return pipeline.Config{
		SkipTestRowValidation: true,
		Mode:                  pipeline.ModeEval,
		RelaxModules:          true,
		GlobalResolver:        e.globalResolver,
	}
}

// moduleCompileFailure returns the module's own compile error (positions in
// the user's file), or "" when it compiles or this check does not apply.
func (e *Executor) moduleCompileFailure(hasInline bool) string {
	if e.sourceFile == nil || e.sourceFile.Module == nil {
		return ""
	}
	if e.batchFailure != nil {
		return e.batchModuleError
	}
	if e.batch != nil || !hasInline {
		return ""
	}
	src, err := os.ReadFile(e.modulePath)
	if err != nil {
		return "" // each test reports the unreadable source itself
	}
	if _, err := e.runInlineCompile("binding", e.inlineModuleConfig(),
		pipeline.Source{Code: string(src), Filename: e.modulePath}); err != nil {
		return err.Error()
	}
	return ""
}

// moduleCompileResult is the single failure that stands for every test of a
// module that does not compile.
func moduleCompileResult(file *ast.File, modulePath, msg string) TestResult {
	return TestResult{
		Name:     "module " + file.Module.Path + " does not compile",
		Status:   StatusFail,
		Error:    "no test ran: " + msg,
		Location: modulePath,
	}
}

func hasInlineRows(suite *TestSuite) bool {
	for _, tc := range suite.Tests {
		if tc.IsInline {
			return true
		}
	}
	return false
}
