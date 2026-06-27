// Package importhint builds the actionable suffix appended to IMP010 "symbol not exported by"
// errors, steering an agent away from the two import mistakes that loop it: importing a builtin
// that is already auto-imported, or importing a real symbol from the wrong module.
//
// It is shared by the loader and the linker — the two IMP010 producers — so the hint is identical
// on both paths (the loader path fires first during module loading; the linker path during link).
// A leaf package: it carries no dependency on the stdlib filesystem scan (that arrives via the
// injected Locator), so neither internal/loader nor internal/link gains one. M-AGENT-STUCK-FIXES M2.
package importhint

import (
	"fmt"
	"strings"
)

// autoImportedBuiltins are values bound in every module without an import. Importing one
// (e.g. `import std/string (show)`) is itself the bug behind an IMP010 loop a benchmark agent
// spun on. Verified empirically: of the prelude class methods only `show` is exposed as a bare
// callable — the rest (eq/lt/add/...) are operator-dispatched, and print/println live in std/io.
var autoImportedBuiltins = map[string]bool{"show": true}

// Locator, when set, returns the stdlib module paths that export a symbol. Wired to
// stdlibindex.Modules from the CLI (cmd/ailang/diagnostics_wiring.go) so this package keeps no
// filesystem-scan dependency — mirrors types.ImportSuggester.
var Locator func(string) []string

// IMP010 returns the suffix to append to an IMP010 "not exported by" message, or "" when neither
// import mistake applies (a genuinely unknown symbol gets no misleading hint).
func IMP010(symbol, modID string) string {
	if autoImportedBuiltins[symbol] {
		return fmt.Sprintf(" — '%s' is a builtin available in every module; remove it from the import list", symbol)
	}
	if Locator != nil {
		var others []string
		for _, m := range Locator(symbol) {
			if m != modID {
				others = append(others, m)
			}
		}
		if len(others) > 0 {
			return fmt.Sprintf(" — '%s' is exported by %s; import it from there, not '%s'", symbol, strings.Join(others, ", "), modID)
		}
	}
	return ""
}
