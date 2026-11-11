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
// replacement does not exist (or is the function itself): a typo would
// otherwise ship a warning that tells every caller to use a name that is not
// there.
//
// It deliberately does NOT reject a replacement that is itself deprecated.
// That fact lives in another module and does not change that module's
// interface digest, so a check here would give one verdict on a cold compile
// and another when the defining module is served from the cache — and it would
// turn a later std deprecation into a build error for an unchanged library.
// The use-site warning names the live end of such a chain instead
// (finalReplacement), and it is computed fresh on every run.
const CodeDeprecatedTarget = "TC_DEPRECATED_002"

// validateDeprecationTargets checks every @deprecated("replacement") in a
// freshly compiled module. The replacement is either an export of the same
// module ("headOr"), or a module-qualified "mod/path.name" whose module this
// module imports (so its interface is loaded) and which exports name, and it
// must not be the function itself. Everything it reads is this module's own
// source and the export lists of its imports, which are in its cache key.
func validateDeprecationTargets(file *ast.File, own *iface.Iface, lookup func(string) *iface.Iface) error {
	if file == nil || own == nil {
		return nil
	}
	resolve := func(modPath string) *iface.Iface {
		if modPath == own.Module {
			return own
		}
		if lookup != nil {
			return lookup(modPath)
		}
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
		modPath, name, qualified := splitDeprecationTarget(repl, own.Module)
		target := resolve(modPath)
		if target == nil {
			return deprecatedTargetError(fd, repl, fmt.Sprintf("module %s is not imported by %s, so %q cannot be checked; import it, or name an export of %s", modPath, own.Module, repl, own.Module), "")
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
			if hint != "" && qualified {
				hint = modPath + "." + hint // a bare hint would itself be an error here
			}
			return deprecatedTargetError(fd, repl, fmt.Sprintf("%s does not export %s", modPath, name), hint)
		}
	}
	return nil
}

// splitDeprecationTarget splits "mod/path.name" into its module and name; an
// unqualified target names an export of the defining module.
func splitDeprecationTarget(repl, ownModule string) (modPath, name string, qualified bool) {
	if i := strings.LastIndex(repl, "."); i > 0 && i < len(repl)-1 {
		return repl[:i], repl[i+1:], true
	}
	return ownModule, repl, false
}

// deprecatedTargetError renders TC_DEPRECATED_002 for humans and unwraps to a
// structured report for `check --json`, both placed at the annotation.
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
		text: fmt.Sprintf("Error %s at %s: %s\n  Fix: %s", CodeDeprecatedTarget, at.String(), msg, fix),
		rep:  errors.WrapReport(rep),
	}
}

type deprecatedTargetErr struct {
	text string
	rep  error
}

func (e *deprecatedTargetErr) Error() string { return e.text }
func (e *deprecatedTargetErr) Unwrap() error { return e.rep }
