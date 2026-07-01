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
