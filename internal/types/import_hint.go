package types

import (
	"fmt"
	"strings"
)

// ImportSuggester, if set, maps an undefined symbol to the std modules that export it
// (M-AGENT-ERGONOMICS). The CLI wires this to internal/stdlibindex; it stays nil in pure
// library use, so internal/types takes on no new dependency and no behavior change.
var ImportSuggester func(name string) []string

// importHint returns a trailing "— add `import std/M (name)`" clause for an undefined-variable
// error when name is a known stdlib export, else "". The dominant agent slip on multi-file tasks
// is using a stdlib function without importing it; this turns a bare error into a one-step fix.
func importHint(name string) string {
	if ImportSuggester == nil {
		return ""
	}
	mods := ImportSuggester(name)
	if len(mods) == 0 {
		return ""
	}
	if len(mods) == 1 {
		return fmt.Sprintf(" — add `import %s (%s)`", mods[0], name)
	}
	// Multiple exporters (e.g. `length` in list/string/array/bytes): stay neutral, the model
	// picks by the value's type rather than us guessing a primary.
	return fmt.Sprintf(" — `%s` is exported by %s; add the matching import", name, strings.Join(mods, ", "))
}

// localResolutionHint returns a truth-telling residual clause for the
// "resolution diverges by syntactic position" family (#323 → #327 → #366): when
// an "undefined variable: X" is raised for X that IS a module-level function of
// the current module, the bare message misleads (the function exists). The known
// members of this family are now fixed — expression positions by #327
// (m-record-update-local-resolution) and the module-let/letrec DECL class by #366
// (this work, m-module-let-func-resolution). This clause is the residual safety
// net for any not-yet-discovered position: it names the truth and the VERIFIED
// workaround (declare X as a `func`, which resolves in every position), and it
// does NOT cite the closed #327 as a live bug. Returns "" when X is not a known
// local function, so genuine undefined-variable errors are untouched.
func (tc *CoreTypeChecker) localResolutionHint(name string) string {
	if tc == nil || tc.moduleFuncNames == nil {
		return ""
	}
	if !tc.moduleFuncNames[name] {
		return ""
	}
	return fmt.Sprintf(" (%s is defined in this module but not resolvable in this position — please report as #366; workaround: declare %s as a `func`)", name, name)
}

// collisionHint returns a note when an applied callee is a bare name exported by more
// than one stdlib module (e.g. `length` in std/list AND std/string). `from` is the
// module the callee resolved to ("" if unknown). Returns "" when there's no ambiguity.
// Attached to a failing application constraint's Path, so it only surfaces when the
// call fails to type-check — turning the opaque "cannot unify list[a] with string"
// into an actionable alias hint (Felix ticket fb_942b7f).
func collisionHint(name, from string) string {
	if ImportSuggester == nil {
		return ""
	}
	mods := ImportSuggester(name)
	if len(mods) < 2 {
		return ""
	}
	var others []string
	for _, m := range mods {
		if m != from {
			others = append(others, m)
		}
	}
	if len(others) == 0 {
		return ""
	}
	return fmt.Sprintf("note: `%s` is also exported by %s — if you meant one of those, import it with an alias, e.g. `import %s (%s as %sAlt)`",
		name, strings.Join(others, ", "), others[0], name, name)
}
