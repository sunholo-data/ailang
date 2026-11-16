package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

// preludeDocEntry is one line of the prelude documentation surface, rendered
// from a LIVE mechanism (not a hand-copied table). Used both by
// `ailang docs prelude` and by `ailang docs --all-functions` (prelude.* lines).
type preludeDocEntry struct {
	name      string
	signature string // rendered signature WITHOUT the leading name, e.g. "(string) -> () ! {IO}"
	doc       string
	source    string // provenance note, e.g. "pipeline.InjectPrelude"
}

// showEntry is the one prelude binding that lives in NEITHER the loader
// implicit-import mechanism NOR pipeline.InjectPrelude: `show` is a builtin
// callable bare in every module. It is the single explicit entry, guarded by a
// forward compile-probe fixture (a `show()` entry program compiles import-free).
var showEntry = preludeDocEntry{
	name:      "show",
	signature: "(a) -> string",
	doc:       "Render any value as a string (builtin, no import needed)",
	source:    "builtin",
}

// preludeDocEntries renders the prelude surface from the live mechanisms:
//  1. pipeline.InjectPrelude (injected type bindings, e.g. println) via PreludeSurface(),
//  2. the `show` builtin (explicit, probe-guarded),
//
// in a stable order. The implicit std/option / std/result imports are NOT
// per-function entries here (they are whole modules — see renderPreludeDocs's
// implicit-imports section); they appear in --all-functions under their own
// std/option.* / std/result.* module lines already.
func preludeDocEntries() []preludeDocEntry {
	var entries []preludeDocEntry

	// Source: pipeline.InjectPrelude injected bindings (e.g. println).
	for _, b := range pipeline.PreludeSurface() {
		entries = append(entries, preludeDocEntry{
			name:      b.Name,
			signature: schemeSignatureString(b.Scheme),
			doc:       "Available without import in entry modules",
			source:    "pipeline.InjectPrelude",
		})
	}

	// Source: show builtin (not in either mechanism).
	entries = append(entries, showEntry)

	// Stable order for deterministic, diffable output.
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	return entries
}

// schemeSignatureString renders a *types.Scheme as a display signature. It uses
// the type's own String() (the same renderer the compiler/errors use), then
// normalizes the effect-arrow spacing so it matches the AST-rendered stdlib
// lines (`(string) -> () ! {IO}`).
func schemeSignatureString(s *types.Scheme) string {
	if s == nil || s.Type == nil {
		return "?"
	}
	return s.Type.String()
}

// renderPreludeDocs prints the `ailang docs prelude` page, rendered from the
// live mechanisms (loader implicit-import accessors + pipeline.InjectPrelude
// enumeration + the `show` builtin). No duplicated table: adding or removing a
// binding via a mechanism changes this page automatically.
func renderPreludeDocs() {
	fmt.Println("# prelude")
	fmt.Println("println is available without import in ENTRY modules only (a module with an")
	fmt.Println("exported `main`); library modules import it (`import std/io (println)`). show is")
	fmt.Println("a builtin, available everywhere.")
	fmt.Println("The six implicit names (Option, Some, None, Result, Ok, Err) are available")
	fmt.Println("without import in EVERY module outside std/.")
	fmt.Println()

	fmt.Println("## Functions")
	fmt.Println()
	for _, e := range preludeDocEntries() {
		sig := strings.TrimPrefix(e.signature, e.name)
		fmt.Printf("  %s : %s\n", e.name, strings.TrimSpace(sig))
		if e.doc != "" {
			fmt.Printf("    %s\n", e.doc)
		}
	}
	fmt.Println()

	// Implicit whole-module imports (std/option, std/result) — rendered from the
	// loader's single source of truth, so a module/symbol added there appears
	// here automatically.
	fmt.Println("## Implicit imports (lowest precedence, every module outside std/)")
	fmt.Println()
	for _, modPath := range loader.PreludeModules() {
		syms := loader.PreludeSymbols(modPath)
		fmt.Printf("  %s -- %s\n", modPath, strings.Join(syms, ", "))
	}
	fmt.Println()

	fmt.Println("## Scope notes")
	fmt.Println()
	fmt.Println("  - Exactly six names, every module: library modules get them too (since #2371); std/")
	fmt.Println("    modules do not, nor does a file of top-level expressions. Helpers (map, getOrElse, ...)")
	fmt.Println("    still need an import, and println stays entry-only.")
	fmt.Println("  - Lowest precedence: the implicit imports are registered first, so your own imports win.")
	fmt.Println("  - Silent shadowing: your own definitions (e.g. `type Option`, or a constructor named")
	fmt.Println("    `Some`) drop that prelude module, no warning; an import from std/option or std/result")
	fmt.Println("    replaces its implicit import.")
}
