package main

import (
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/stdlibindex"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-AGENT-ERGONOMICS: wire stdlib import suggestions into diagnostics. Kept in the CLI (not
// internal/types or internal/importhint) so those packages stay free of a stdlib-scan dependency.
func init() {
	// "undefined variable" type errors -> suggest the missing import.
	types.ImportSuggester = stdlibindex.Modules
	// IMP010 "not exported by" link/loader errors -> auto-import / wrong-module hint
	// (M-AGENT-STUCK-FIXES M2). importhint is shared by both IMP010 producers.
	importhint.Locator = stdlibindex.Modules
	// ...and a close-named-export suggestion when the symbol exists nowhere
	// (e.g. flushStdout -> flush) — one canonical name, rescued via the error.
	importhint.SymbolsOf = stdlibindex.SymbolsOf
	// Unknown-stdlib-MODULE recovery: a mistyped module name (import std/time)
	// gets a "did you mean: std/clock?" + available-module list appended to the
	// resolver's not-found error (M-DX-AI-DISCOVERY M3).
	importhint.ModuleLocator = stdlibindex.AllModules
}
