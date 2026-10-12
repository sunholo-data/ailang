package format

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"strings"
	"testing"
)

func TestMultilineSourceLayout(t *testing.T) {
	for _, expr := range []string{
		"\"<html>\n  <body>hello</body>\n</html>\"",
		"[\"first\n  second\", \"one\\nline\"]",
		"{html: \"first\n\\\"quoted\\\" \\\\ tail\"}",
		"if true then \"yes\n  raw\" else \"no\\nraw\"",
		"\"first\n  ${x} tail\nend\"",
		"\"first\n  ${\"nested }\"} tail\"",
		"(\"first\n  raw\", \"other\n  raw\")",
		"match x { \"first\n  raw\" => \"match\", _ => \"other\" }",
	} {
		t.Run(expr, func(t *testing.T) {
			src := []byte("module multiline\npure func f(x: string) = " + expr + "\n")
			p := parser.New(lexer.New(string(src), "multiline.ail"))
			prog := p.Parse()
			if len(p.Errors()) > 0 {
				t.Fatalf("parse: %v", p.Errors())
			}
			out, err := SourceWithComments(prog, src, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), "\n") {
				t.Fatalf("flattened: %s", out)
			}
			// Every raw literal's complete spelling must survive, including escapes.
			env, _ := NewEnvelope(src)
			for _, sp := range env.stringSpans() {
				raw := env.src[sp.start:sp.end]
				if strings.Contains(raw, "\n") && !strings.Contains(string(out), raw) {
					t.Fatalf("lost %q in %s", raw, out)
				}
			}
			rp := parser.New(lexer.New(string(out), "multiline.ail"))
			again := rp.Parse()
			if len(rp.Errors()) > 0 {
				t.Fatalf("reparse: %v", rp.Errors())
			}
			// The AST-only printer is a source-independent semantic control.
			before, e1 := Source(prog, Options{})
			after, e2 := Source(again, Options{})
			if e1 != nil || e2 != nil || string(before) != string(after) {
				t.Fatalf("payload changed: %s / %s (%v, %v)", before, after, e1, e2)
			}
			twice, err := SourceWithComments(again, out, Options{})
			if err != nil || string(twice) != string(out) {
				t.Fatalf("not idempotent: %s (%v)", twice, err)
			}
		})
	}
}

func TestMultilineAttachmentRejectsMismatchedPayload(t *testing.T) {
	src := []byte("pure func f() = \"original\n  raw\"\n")
	p := parser.New(lexer.New(string(src), "multiline.ail"))
	prog := p.Parse()
	block := prog.File.Funcs[0].Body.(*ast.Block)
	block.Exprs[0].(*ast.Literal).Value = "replacement\nvalue"
	out, err := SourceWithComments(prog, src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "original") || !strings.Contains(string(out), `"replacement\nvalue"`) {
		t.Fatalf("unsafe source attachment: %s", out)
	}
}

func TestEscapedOneLineStringStaysOneLine(t *testing.T) {
	src := []byte("pure func f() = \"escaped\\n  text\"\n")
	p := parser.New(lexer.New(string(src), "multiline.ail"))
	prog := p.Parse()
	out, err := SourceWithComments(prog, src, Options{})
	if err != nil || !strings.Contains(string(out), `"escaped\n  text"`) {
		t.Fatalf("changed one-line layout: %s (%v)", out, err)
	}
}
