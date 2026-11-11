package pipeline

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
)

// TCBuiltinShadow codes a module-level definition named like a builtin that
// the builtin silently replaces.
const TCBuiltinShadow = "TC_BUILTIN_SHADOW_001"

// shadowCheckedBuiltins are the builtin names a module-level definition may
// not take. A builtin wins over a module-level definition of the same name
// (the general rule, unchanged here), so a program that defined its own
// `unreachable` before the builtin existed (M-V1-MATCH-EXHAUSTIVENESS) would
// keep compiling and call the builtin instead — a silent change of meaning.
// The name is new in this release, so it alone is refused; the older names
// keep their behaviour.
var shadowCheckedBuiltins = map[string]bool{"unreachable": true}

// checkBuiltinShadow refuses a module-level function or binding with one of
// shadowCheckedBuiltins' names.
func checkBuiltinShadow(file *ast.File) error {
	if file == nil {
		return nil
	}
	for _, fn := range file.Funcs {
		if fn != nil && shadowCheckedBuiltins[fn.Name] {
			return builtinShadowError(fn.Name, fn.Pos)
		}
	}
	for _, st := range file.Statements {
		if l, ok := st.(*ast.Let); ok && shadowCheckedBuiltins[l.Name] {
			return builtinShadowError(l.Name, l.Pos)
		}
	}
	return nil
}

func builtinShadowError(name string, pos ast.Pos) error {
	msg := fmt.Sprintf("`%s` is a builtin; a module-level definition of that name would be silently replaced by it at every call", name)
	return fmt.Errorf("%s: %w", pos, &ailerrors.ReportError{Rep: &ailerrors.Report{
		Schema:  "ailang.error/v1",
		Code:    TCBuiltinShadow,
		Phase:   "typecheck",
		Message: msg,
		Span:    &ast.Span{Start: pos, End: pos},
		Fix:     &ailerrors.Fix{Suggestion: fmt.Sprintf("rename this definition (and its callers), e.g. %s_", name), Confidence: 0.9},
	}})
}
