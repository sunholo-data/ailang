package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/importhint"
)

// CodeDeprecatedTarget is the error for an @deprecated annotation whose
// replacement does not exist: a typo would otherwise ship a warning that tells
// every caller to use a name that is not there.
const CodeDeprecatedTarget = "TC_DEPRECATED_002"

// validateDeprecationTargets checks every @deprecated("replacement") in a
// freshly compiled module. The replacement is either an export of the same
// module ("headOr"), or a module-qualified "mod/path.name" whose module this
// module imports (so its interface is loaded) and which exports name. A
// function cannot name itself.
func validateDeprecationTargets(file *ast.File, own *iface.Iface, lookup func(string) *iface.Iface) error {
	if file == nil || own == nil {
		return nil
	}
	for _, fd := range file.Funcs {
		if fd == nil {
			continue
		}
		repl, _, ok := fd.Deprecation()
		if !ok {
			continue
		}
		modPath, name := own.Module, repl
		target := own
		if i := strings.LastIndex(repl, "."); i > 0 && strings.Contains(repl[:i], "/") {
			modPath, name = repl[:i], repl[i+1:]
			target = nil
			if lookup != nil {
				target = lookup(modPath)
			}
			if target == nil {
				return deprecatedTargetError(fd, repl, fmt.Sprintf("module %s is not imported by %s, so %q cannot be checked; import it, or name an export of %s", modPath, own.Module, repl, own.Module), "")
			}
		}
		if modPath == own.Module && name == fd.Name {
			return deprecatedTargetError(fd, repl, fmt.Sprintf("%s cannot be its own replacement", fd.Name), "")
		}
		if _, exists := target.Exports[name]; !exists {
			var names []string
			for n := range target.Exports {
				names = append(names, n)
			}
			sort.Strings(names)
			hint := importhint.Closest(name, names)
			return deprecatedTargetError(fd, repl, fmt.Sprintf("%s does not export %s", modPath, name), hint)
		}
	}
	return nil
}

// deprecatedTargetError renders TC_DEPRECATED_002 for humans and unwraps to a
// structured report for `check --json`, placed at the annotation.
func deprecatedTargetError(fd *ast.FuncDecl, repl, why, didYouMean string) error {
	at := fd.Pos
	if a := fd.GetAnnotation(ast.AnnotationDeprecated); a != nil && a.Pos.Line > 0 {
		at = a.Pos
	}
	msg := fmt.Sprintf("@deprecated(%q) on %s names no replacement: %s", repl, fd.Name, why)
	fix := "name an export of this module, e.g. @deprecated(\"newName\"), or a module-qualified \"std/list.headOr\" from an imported module"
	if didYouMean != "" {
		fix = fmt.Sprintf("did you mean @deprecated(%q)?", didYouMean)
	}
	rep := &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    CodeDeprecatedTarget,
		Phase:   "link",
		Message: msg,
		Span:    &ast.Span{Start: at, End: at},
		Data:    map[string]any{"function": fd.Name, "replacement": repl},
		Fix:     &errors.Fix{Suggestion: fix, Confidence: 0.9},
	}
	return &deprecatedTargetErr{
		text: fmt.Sprintf("Error %s: %s\n  Fix: %s", CodeDeprecatedTarget, msg, fix),
		rep:  errors.WrapReport(rep),
	}
}

type deprecatedTargetErr struct {
	text string
	rep  error
}

func (e *deprecatedTargetErr) Error() string { return e.text }
func (e *deprecatedTargetErr) Unwrap() error { return e.rep }
