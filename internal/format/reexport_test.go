package format

import "testing"

// `export import` lines (M-PKG-PARALLEL-AGENTS C1) print exactly as written:
// canonical text is a fixed point, and the AST round-trips.
func TestReexportImport_RoundTrips(t *testing.T) {
	src := "module lib/facade\n" +
		"import std/io (println)\n" +
		"export import lib/color (Color, Red as Crimson)\n" +
		"export import pkg/demo/core/types/document (Block, Heading) @deprecated(\"pkg/demo/core/types/document\")\n\n" +
		"export func hello() -> string = \"hi\"\n"
	if got := formatSrc(t, src); got != src {
		t.Fatalf("not a fixed point:\n--- got\n%s--- want\n%s", got, src)
	}
	assertIdempotentAndRoundTrips(t, src, "facade.ail")
}
