package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/errors"
)

// externDocs is the page the EXT004 message names: it lists `extern` as a
// reserved word whose declarations are rejected.
const externDocs = "https://ailang.sunholo.com/docs/reference/reserved-keywords"

// externFuncError renders EXT004 for humans ("Error EXT004: … Fix: …") and
// unwraps to a structured report, so `check --json` and the LSP place it at the
// declaration (the MOD015 shape).
type externFuncError struct {
	text string
	rep  error
}

func (e *externFuncError) Error() string { return e.text }
func (e *externFuncError) Unwrap() error { return e.rep }

// checkNoExternFuncs rejects the first `extern func` declaration in file
// (#2390). Extern functions were implemented only in Go emitted by `ailang
// compile --emit-go`, which was removed (#2389): neither the evaluator nor the
// bytecode VM can run one. The declaration itself is the error, whether or not
// anything calls it, so it is raised before any function is collected: an
// importer of a module that declares one fails while loading that module.
func checkNoExternFuncs(file *ast.File) error {
	for _, fn := range file.Funcs {
		if !fn.IsExtern {
			continue
		}
		at := fn.Pos
		if at.File == "" {
			at.File = file.Path
		}
		msg := fmt.Sprintf("extern function '%s' cannot be declared: extern functions were implemented only by the "+
			"Go emitter (`ailang compile --emit-go`), which was removed, and no engine (the evaluator or the "+
			"bytecode VM) can run them", fn.Name)
		fix := fmt.Sprintf("write '%s' as an ordinary AILANG func, or call a std function or an effect that does "+
			"the work (see %s)", fn.Name, externDocs)
		rep := &errors.Report{
			Schema:  "ailang.error/v1",
			Code:    errors.EXT004,
			Phase:   "elaborate",
			Message: msg,
			Span:    &ast.Span{Start: at, End: at},
			Data:    map[string]any{"name": fn.Name},
			Fix:     &errors.Fix{Suggestion: fix, Confidence: 0.9},
		}
		return &externFuncError{
			text: fmt.Sprintf("Error %s at %s:%d:%d: %s\n  Fix: %s", errors.EXT004, at.File, at.Line, at.Column, msg, fix),
			rep:  errors.WrapReport(rep),
		}
	}
	return nil
}
