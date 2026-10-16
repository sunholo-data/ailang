package testing

import (
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"reflect"
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

func TestStripParserRangesLexicalContexts(t *testing.T) {
	for name, expr := range map[string]string{
		"opening":       `"{" == "{"`,
		"closing":       `"}" == "}"`,
		"balanced":      `"{}" == "{}"`,
		"json":          `"\"promoted\":{\"kind\"" == "\"promoted\":{\"kind\""`,
		"interpolation": `"${"{"}" == "{"`,
		"character":     `'{' == '{'`,
		"comment":       "true -- } { {",
	} {
		t.Run(name, func(t *testing.T) {
			source := "module lexical\npure func before() -> int = 1\ntest \"literal\" {\n " + expr +
				"\n}\npure func between() -> int = 2\nproperty \"literal\" {\n forall(n: int) => " + expr +
				"\n}\npure func after() -> int = 3\n"
			p := parser.New(lexer.New(source, "lexical.ail"))
			file := p.ParseFile()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse: %v", p.Errors())
			}
			want := "module lexical\npure func before() -> int = 1\npure func between() -> int = 2\npure func after() -> int = 3\n"
			for _, strip := range []func(string, *ast.File) (string, []int){
				new(Executor).stripTestBlocks,
				func(s string, f *ast.File) (string, []int) { return new(Executor).stripWithLineMap(s, f, nil) },
			} {
				got, lines := strip(source, file)
				if got != want || !reflect.DeepEqual(lines, []int{1, 2, 6, 10}) {
					t.Fatalf("source = %q, lines = %v; want %q, [1 2 6 10]", got, lines, want)
				}
			}
		})
	}
}

// A recorded end must be authoritative even when the supplied source differs.
func TestStripUsesRecordedEnd(t *testing.T) {
	for _, decl := range []ast.Node{
		&ast.TestDecl{Pos: ast.Pos{Line: 2}, End: ast.Pos{Line: 4}},
		&ast.PropertyDecl{Pos: ast.Pos{Line: 2}, End: ast.Pos{Line: 4}},
	} {
		got := testAndPropertySkipRanges([]string{"module x", "no opening brace", "body", "end", "keep"}, &ast.File{Decls: []ast.Node{decl}})
		if !reflect.DeepEqual(got, []sourceLineRange{{start: 2, end: 4}}) {
			t.Fatalf("ranges = %v", got)
		}
	}
}

func TestStripBraceFixtures(t *testing.T) {
	for _, name := range []string{"named_test_brace_string", "named_test_closebrace_string", "named_test_interp_brace"} {
		t.Run(name, func(t *testing.T) {
			_, source, file := parseStripFixture(t, name+".ail")
			got, lines := new(Executor).stripTestBlocks(source, file)
			want := "module " + name + "\nimport std/string (contains)\n"
			if got != want || !reflect.DeepEqual(lines, []int{1, 2}) {
				t.Fatalf("source = %q, lines = %v", got, lines)
			}
		})
	}
}
