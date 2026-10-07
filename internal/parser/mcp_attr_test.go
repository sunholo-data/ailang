package parser

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
)

// MCP annotations: @mcp_title / @mcp_hints (M1 of the Parse directory
// listing) and the listed-surface set from M-SERVEAPI-DIRECTORY-READY.

// TestMCPTitleAndHintsAnnotations: @mcp_title takes exactly one string,
// @mcp_hints one or more; the parser records them verbatim (the hint
// vocabulary is checked at MCP registration, like @optional names).
func TestMCPTitleAndHintsAnnotations(t *testing.T) {
	input := `
@mcp_title("Parse document")
@mcp_hints("readOnly", "openWorld")
export func parse(x: string) -> string ! {IO} { x }`
	p := New(lexer.New(input, "test.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	fn := file.Funcs[0]
	title := fn.GetAnnotation("mcp_title")
	if title == nil || len(title.Args) != 1 || title.Args[0].(*ast.Literal).Value != "Parse document" {
		t.Errorf("@mcp_title: %+v", title)
	}
	hints := fn.GetAnnotation("mcp_hints")
	if hints == nil || len(hints.Args) != 2 || hints.Args[1].(*ast.Literal).Value != "openWorld" {
		t.Errorf("@mcp_hints: %+v", hints)
	}

	for _, bad := range []string{
		`@mcp_title("a", "b")`,
		`@mcp_title()`,
		`@mcp_hints(readOnly)`,
		`@mcp_hints("readOnly",)`,
	} {
		p := New(lexer.New(bad+"\nexport func f(x: int) -> int { x }", "test.ail"))
		p.ParseFile()
		if len(p.Errors()) == 0 {
			t.Errorf("%s: expected a parse error", bad)
		}
	}
}

// @mcp_hints() is a complete, empty declaration: a tool that writes
// additively in a closed world. It must parse with zero args.
func TestMCPHintsEmpty(t *testing.T) {
	p := New(lexer.New("@mcp_hints()\nexport func f(x: int) -> int ! {IO} { x }", "test.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	ann := file.Funcs[0].GetAnnotation("mcp_hints")
	if ann == nil || len(ann.Args) != 0 {
		t.Fatalf("@mcp_hints(): %+v", ann)
	}
}

// M-SERVEAPI-DIRECTORY-READY M1: @mcp_auth takes exactly one string,
// @mcp_secret one or more, @mcp_token_verifier and @mcp_agent_only none.
func TestMCPDirectoryAnnotations(t *testing.T) {
	input := `
@mcp_auth("oauth2")
@mcp_secret("apiKey", "token")
@mcp_agent_only
export func gated(apiKey: string, token: string) -> string ! {IO} { apiKey }

@mcp_token_verifier
export func verify(token: string) -> bool ! {Net} { true }`
	p := New(lexer.New(input, "test.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	g := file.Funcs[0]
	if a := g.GetAnnotation("mcp_auth"); a == nil || len(a.Args) != 1 || a.Args[0].(*ast.Literal).Value != "oauth2" {
		t.Errorf("@mcp_auth: %+v", a)
	}
	if a := g.GetAnnotation("mcp_secret"); a == nil || len(a.Args) != 2 {
		t.Errorf("@mcp_secret: %+v", a)
	}
	if g.GetAnnotation("mcp_agent_only") == nil {
		t.Error("@mcp_agent_only missing")
	}
	if a := file.Funcs[1].GetAnnotation("mcp_token_verifier"); a == nil || len(a.Args) != 0 {
		t.Errorf("@mcp_token_verifier: %+v", a)
	}

	for _, bad := range []string{
		`@mcp_auth()`,
		`@mcp_auth("oauth2", "noauth")`,
		`@mcp_auth(oauth2)`,
		`@mcp_secret()`,
	} {
		p := New(lexer.New(bad+"\nexport func f(x: int) -> int { x }", "test.ail"))
		p.ParseFile()
		if len(p.Errors()) == 0 {
			t.Errorf("%s: expected a parse error", bad)
		}
	}
}

// M-MCP-FILE-HANDOFF F1b: @mcp_file takes one or more param names and may be
// repeated; the names are checked against the signature at serve-api load.
func TestMCPFileAnnotation(t *testing.T) {
	input := `
@mcp_file("file", "other")
@mcp_file("third")
export func f(file: string, other: string, third: string) -> string { file }`
	p := New(lexer.New(input, "test.ail"))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	var got []string
	for _, a := range file.Funcs[0].Annotations {
		if a.Name == "mcp_file" {
			for _, arg := range a.Args {
				got = append(got, arg.(*ast.Literal).Value.(string))
			}
		}
	}
	if len(got) != 3 || got[0] != "file" || got[1] != "other" || got[2] != "third" {
		t.Errorf("@mcp_file args = %v", got)
	}
	for _, bad := range []string{`@mcp_file()`, `@mcp_file(file)`} {
		p := New(lexer.New(bad+"\nexport func f(x: int) -> int { x }", "test.ail"))
		p.ParseFile()
		if len(p.Errors()) == 0 {
			t.Errorf("%s: expected a parse error", bad)
		}
	}
}
