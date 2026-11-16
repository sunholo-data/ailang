package runner

import (
	"fmt"
	"io"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/bytecode"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

// Strict-VM coverage report (M-BYTECODE-STRICT-256-REG-CEILING Phase A).
//
// A function the bytecode compiler cannot lower is tagged EvalOnly instead of
// failing the image (M-BYTECODE-2D M3). Without a report, that tag was only
// visible as a mid-run `vm: CALL: ... is evaluator-only` trap under
// --strict-bytecode, and not at all on a non-strict --bytecode run. These
// helpers turn the tag into a BC001 diagnostic at the earliest gate.

// evalOnlyWarnLimit caps how many names the non-strict warning lists; the
// total it prints is always exact.
const evalOnlyWarnLimit = 10

// StrictBytecodeReport compiles res to a bytecode image and returns the
// evaluator-only prototypes reachable from the entry module's own functions —
// every function the file declares, called or not, plus whatever imported
// function they reach. Imported evaluator-only helpers the module never
// reaches are not reported: they cannot trap. This is the
// `ailang check --strict-bytecode` gate; a run reports from its entry only.
func StrictBytecodeReport(res pipeline.Result) ([]bytecode.EvalOnlyEntry, error) {
	img, err := compileImageRecovering(res, "check")
	if err != nil {
		return nil, err
	}
	entryModule := EntryModuleID(res)
	if entryModule == "" && len(res.Modules) > 0 {
		return nil, fmt.Errorf("internal: entry module not found among %d compiled modules", len(res.Modules))
	}
	return img.EvalOnlyReachable(moduleProtoRoots(img, entryModule)), nil
}

// compileImageRecovering is CompileBytecodeFromResult plus Validate, with the
// lower pass's remaining panics turned into a compile error.
func compileImageRecovering(res pipeline.Result, pkgName string) (img *bytecode.BytecodeImage, err error) {
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("compile panic: %v", r)
			}
		}()
		img, err = CompileBytecodeFromResult(res, pkgName)
	}()
	if err != nil {
		return nil, fmt.Errorf("bytecode compile: %w", err)
	}
	if err := img.Validate(); err != nil {
		return nil, fmt.Errorf("bytecode validate: %w", err)
	}
	return img, nil
}

// moduleProtoRoots returns the image indices of the prototypes the entry
// module declares. A single-module image ("" entry module) holds nothing else.
func moduleProtoRoots(img *bytecode.BytecodeImage, entryModule string) []int {
	prefix := entryModule + "."
	var roots []int
	for i, p := range img.Prototypes {
		if entryModule == "" || strings.HasPrefix(p.Name, prefix) {
			roots = append(roots, i)
		}
	}
	return roots
}

// BC001Report renders one evaluator-only prototype as a structured BC001
// diagnostic. The message carries the position too, because the human check
// output prints only "CODE: message".
func BC001Report(e bytecode.EvalOnlyEntry) error {
	msg := fmt.Sprintf("function %s is evaluator-only under --strict-bytecode (%s)", e.Name, e.Reason)
	rep := &ailerrors.Report{
		Schema:  "ailang.error/v1",
		Code:    ailerrors.BC001,
		Phase:   "bytecode",
		Message: msg,
		Data:    map[string]any{"function": e.Name, "reason": e.Reason},
		Fix: &ailerrors.Fix{
			Suggestion: "split or restructure the function so it compiles to bytecode, or run without --strict-bytecode (the evaluator bridge runs it)",
			Confidence: 0.5,
		},
	}
	if loc := entryLocation(e); loc != "" {
		rep.Message += " at " + loc
		rep.Span = &ast.Span{Start: ast.Pos{File: e.File, Line: e.Line}, End: ast.Pos{File: e.File, Line: e.Line}}
	}
	return ailerrors.WrapReport(rep)
}

// BC001Errors is BC001Report over a whole report, in report order.
func BC001Errors(entries []bytecode.EvalOnlyEntry) []error {
	errs := make([]error, len(entries))
	for i, e := range entries {
		errs[i] = BC001Report(e)
	}
	return errs
}

// StrictEvalOnlyError is the pre-dispatch failure of a strict run or test:
// one BC001 line per evaluator-only prototype reachable from entry.
func StrictEvalOnlyError(entry string, entries []bytecode.EvalOnlyEntry) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%d evaluator-only function(s) reachable from %s; --strict-bytecode wires no evaluator bridge:", len(entries), entry)
	for _, err := range BC001Errors(entries) {
		b.WriteString("\n  ")
		b.WriteString(err.Error())
	}
	return fmt.Errorf("%s", b.String())
}

// WarnEvalOnly prints the non-strict --bytecode notice: these functions will
// run on the evaluator through the bridge. The total is exact; at most
// evalOnlyWarnLimit names are listed. Prints nothing for an empty report.
func WarnEvalOnly(w io.Writer, entry string, entries []bytecode.EvalOnlyEntry) {
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(w, "%s bytecode: %d evaluator-only function(s) reachable from %s run on the evaluator via the bridge (--strict-bytecode rejects them as BC001):\n",
		yellow("⚠"), len(entries), entry)
	for i, e := range entries {
		if i == evalOnlyWarnLimit {
			fmt.Fprintf(w, "    ... and %d more\n", len(entries)-evalOnlyWarnLimit)
			break
		}
		loc := entryLocation(e)
		if loc != "" {
			loc = " at " + loc
		}
		fmt.Fprintf(w, "    %s%s: %s\n", e.Name, loc, e.Reason)
	}
}

func entryLocation(e bytecode.EvalOnlyEntry) string {
	switch {
	case e.File != "" && e.Line > 0:
		return fmt.Sprintf("%s:%d", e.File, e.Line)
	case e.File != "":
		return e.File
	}
	return ""
}
