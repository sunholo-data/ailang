package elaborate

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/importhint"
)

// moduleAliasPath reports whether name is a module alias (`import std/list as
// L` binds every export as "L.<name>" in the global environment) and, if so,
// the module it names.
func (e *Elaborator) moduleAliasPath(name string) (string, bool) {
	prefix := name + "."
	for key, ref := range e.globalEnv {
		if strings.HasPrefix(key, prefix) && ref.Module != "" {
			return ref.Module, true
		}
	}
	return "", false
}

// qualifiedMissingMember is IMP010 for `Alias.member` when the aliased module
// does not export member. It reads like the import-list IMP010 and carries the
// same hint, so a renamed std function names its replacement whether it is
// imported by name or reached through a module alias
// (M-V1-STDLIB-SURFACE-POLISH §6.3).
func qualifiedMissingMember(alias, member, modPath string, at ast.Pos) error {
	hint := importhint.IMP010(member, modPath)
	msg := fmt.Sprintf("symbol '%s' not exported by '%s' (qualified access %s.%s)%s", member, modPath, alias, member, hint)
	fix := fmt.Sprintf("Check exports in %s or import an existing symbol", modPath)
	if hint != "" {
		fix = strings.TrimPrefix(hint, " — ")
	}
	rep := &errors.Report{
		Schema:  "ailang.error/v1",
		Code:    errors.IMP010,
		Phase:   "elaborate",
		Message: msg,
		Span:    &ast.Span{Start: at, End: at},
		Data:    map[string]any{"symbol": member, "module_id": modPath, "alias": alias},
		Fix:     &errors.Fix{Suggestion: fix, Confidence: 0.9},
	}
	return &importCollisionError{
		text: fmt.Sprintf("Error %s: %s\n  Fix: %s", errors.IMP010, msg, fix),
		rep:  errors.WrapReport(rep),
	}
}
