package testing

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// #1821: a brace inside a string literal in a multi-line test body must not
// end (or fail to end) the stripped block early.
func TestTestAndPropertySkipRangesIgnoresBracesInStrings(t *testing.T) {
	src := strings.Join([]string{
		`module brace`,                      // 1
		`import std/string (contains)`,      // 2
		``,                                  // 3
		`test "brace in a string" {`,        // 4
		` contains("{a", "{")`,              // 5
		`}`,                                 // 6
		`test "json" {`,                     // 7
		`  contains("\"k\":{\"v\"", "}}")`,  // 8
		`}`,                                 // 9
		`-- a comment with { in it`,         // 10
		`export pure func one() -> int = 1`, // 11
		``,
	}, "\n")
	file := &ast.File{Decls: []ast.Node{
		&ast.TestDecl{Name: "brace in a string", Pos: ast.Pos{Line: 4}},
		&ast.TestDecl{Name: "json", Pos: ast.Pos{Line: 7}},
	}}
	got := testAndPropertySkipRanges(splitLines(src), file)
	want := []sourceLineRange{{start: 4, end: 6}, {start: 7, end: 9}}
	if len(got) != len(want) {
		t.Fatalf("ranges = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("range %d = %v, want %v", i, got[i], want[i])
		}
	}
}
