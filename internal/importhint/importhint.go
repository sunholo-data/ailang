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

// SymbolsOf, when set, returns the symbols a given module exports (wired to
// stdlibindex.SymbolsOf). Used to suggest a close-named export when the requested
// symbol exists nowhere — e.g. `flushStdout` -> `flush`. This keeps ONE canonical
// name per operation (no duplicate aliases) while still rescuing a cold-start miss.
var SymbolsOf func(modID string) []string

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
	// The symbol exists in no module. Rescue a close cold-start miss by suggesting
	// the nearest export of the module the user aimed at (e.g. flushStdout -> flush),
	// rather than shipping a duplicate alias.
	if SymbolsOf != nil {
		if best := closestExport(symbol, SymbolsOf(modID)); best != "" {
			return fmt.Sprintf(" — did you mean '%s'? (exported by '%s')", best, modID)
		}
	}
	return ""
}

// closestExport returns the export most likely intended for `want`, or "" when no
// export is close enough (a genuinely unknown symbol gets no misleading hint).
// Signals, strongest first: a prefix relationship (flushStdout↔flush, shorter side
// ≥3 chars), then a small Levenshtein distance (catches transposition/typo misses).
func closestExport(want string, exports []string) string {
	wl := strings.ToLower(want)
	best := ""
	for _, e := range exports {
		el := strings.ToLower(e)
		if el == wl {
			continue
		}
		shorter := len(el)
		if len(wl) < shorter {
			shorter = len(wl)
		}
		if shorter >= 3 && (strings.HasPrefix(wl, el) || strings.HasPrefix(el, wl)) {
			if best == "" || lenDiff(e, want) < lenDiff(best, want) {
				best = e
			}
		}
	}
	if best != "" {
		return best
	}
	bestDist := 3 // only accept edit distance <= 2
	for _, e := range exports {
		if d := levenshtein(wl, strings.ToLower(e)); d < bestDist {
			bestDist, best = d, e
		}
	}
	return best
}

func lenDiff(a, b string) int {
	if d := len(a) - len(b); d >= 0 {
		return d
	} else {
		return -d
	}
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = prev[j] + 1
			if v := cur[j-1] + 1; v < cur[j] {
				cur[j] = v
			}
			if v := prev[j-1] + cost; v < cur[j] {
				cur[j] = v
			}
		}
		prev = cur
	}
	return prev[len(b)]
}
