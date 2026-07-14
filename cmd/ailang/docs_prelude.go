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
	fmt.Println("Available without import in ENTRY modules (a module with an exported `main`).")
	fmt.Println("Library modules must import these explicitly.")
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
	fmt.Println("## Implicit imports (lowest precedence, entry modules only)")
	fmt.Println()
	for _, modPath := range loader.EntryPreludeModules() {
		syms := loader.EntryPreludeSymbols(modPath)
		fmt.Printf("  %s -- %s\n", modPath, strings.Join(syms, ", "))
	}
	fmt.Println()

	fmt.Println("## Scope notes")
	fmt.Println()
	fmt.Println("  - Entry-only: only modules with an exported `main` get these; library modules must import.")
	fmt.Println("  - Lowest precedence: the implicit imports are registered first, so your own imports win.")
	fmt.Println("  - Silent shadowing: your own definitions (e.g. `type Option`) shadow the prelude, no warning.")
}
