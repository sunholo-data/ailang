package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

func TestReservedParameterRecovery(t *testing.T) {
	for _, bad := range []string{"with", "recv", "send", "recv: (int, int) -> int", "recv: {x: int, y: (int, int) -> int}", "recv: [int]", "with: int, recv: int"} {
		t.Run(bad, func(t *testing.T) {
			src := fmt.Sprintf("module probe\nfunc f(a: int, %s, b: int) -> int = a + b\nfunc next() -> int = 42", bad)
			p := New(lexer.New(src, "probe.ail"))
			file := p.ParseFile()
			want := 1
			if strings.HasPrefix(bad, "with:") {
				want = 2
			}
			if len(p.Errors()) != want {
				t.Fatalf("errors: %v", p.Errors())
			}
			for _, e := range p.Errors() {
				pe := e.(*ParserError)
				if pe.Code != "PAR_RESERVED_KEYWORD" || pe.Pos.Line != 2 {
					t.Fatalf("wrong diagnostic: %v", e)
				}
			}
			if len(file.Funcs) != 2 {
				t.Fatalf("lost declaration: %+v", file)
			}
			f := file.Funcs[0]
			if len(f.Params) != 2 || f.Params[0].Name != "a" || f.Params[1].Name != "b" || f.Body == nil || file.Funcs[1].Name != "next" {
				t.Fatalf("corrupted AST: %+v", f)
			}
		})
	}
}

func TestReservedParameterAllKeywords(t *testing.T) {
	words := strings.Fields("func pure let letrec in if then else match with type class instance module import export extern forall exists test tests property properties assert spawn parallel select channel send recv timeout as deriving requires ensures invariant true false not and or")
	if len(words) != 41 {
		t.Fatal("keyword coverage changed")
	}
	for _, word := range words {
		t.Run(word, func(t *testing.T) {
			p := New(lexer.New("module probe\nfunc f("+word+": int) -> int = 42", "probe.ail"))
			p.Parse()
			if lexer.LookupIdentContextual(word) == lexer.IDENT {
				if len(p.Errors()) != 0 {
					t.Fatal(p.Errors())
				}
				return
			}
			if len(p.Errors()) != 1 {
				t.Fatalf("errors: %v", p.Errors())
			}
			e := p.Errors()[0].(*ParserError)
			if e.Code != "PAR_RESERVED_KEYWORD" || e.Pos.Line != 2 || e.Pos.Column != 8 || e.NearToken.Literal != word {
				t.Fatalf("wrong error: %v", e)
			}
		})
	}
}

func TestReservedParameterBoundaries(t *testing.T) {
	for _, src := range []string{
		"func f(with: int, b: int) -> int = b",
		"func f(a: int, recv: int) -> int = a",
		"func f(with) -> int = 42",
		"func f() -> int = 42",
		"func f() -> int = (func(a: int, recv: int, b: int) => a + b)(1, 2)",
		"func f(recv: (int, int)",
	} {
		t.Run(src, func(t *testing.T) {
			p := New(lexer.New("module probe\n"+src, "probe.ail"))
			file := p.ParseFile()
			for _, f := range file.Funcs {
				if f == nil {
					continue
				}
				for _, param := range f.Params {
					if param.Name == "" {
						t.Fatal("fabricated parameter")
					}
				}
			}
			if strings.Contains(src, "recv") || strings.Contains(src, "with") {
				if len(p.Errors()) == 0 {
					t.Fatal("missing error")
				}
			} else if len(p.Errors()) != 0 {
				t.Fatal(p.Errors())
			}
		})
	}
	p := New(lexer.New("module probe\nfunc f(recv: int) -> int = 42\nfunc broken() -> int = )", "probe.ail"))
	p.Parse()
	if len(p.Errors()) < 2 || p.Errors()[0].(*ParserError).Code != "PAR_RESERVED_KEYWORD" {
		t.Fatalf("independent error lost: %v", p.Errors())
	}
}

func TestReservedRecordField(t *testing.T) {
	for _, word := range []string{"with", "recv", "42"} {
		p := New(lexer.New(word+": int", "probe.ail"))
		p.parseRecordFieldDef()
		e := p.Errors()[0].(*ParserError)
		if e.Code != "PAR_FIELD_NAME_EXPECTED" {
			t.Fatal(e)
		}
		if word == "42" {
			if e.Message != "expected field name" {
				t.Fatal(e)
			}
		} else if !strings.Contains(e.Message, "reserved keyword '"+word+"'") || len(e.Suggestions) == 0 {
			t.Fatal(e)
		}
	}
}

func TestReservedSuggestionsReference(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "..", "docs", "docs", "reference", "reserved-keywords.md"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join("..", "lexer", "token.go"))
	if err != nil {
		t.Fatal(err)
	}
	entries := regexp.MustCompile(`(?m)^\s*"(\w+)":`).FindAllSubmatch(source, -1)
	if len(entries) != 41 {
		t.Fatalf("keyword map changed: %d", len(entries))
	}
	for _, entry := range entries {
		word := string(entry[1])
		if strings.Count(string(page), "| `"+word+"` |") != 1 {
			t.Errorf("missing or repeated keyword row: %s", word)
		}
	}
	for word, alternatives := range map[string][]string{
		"with": {"using", "given", "w"}, "recv": {"tx", "rx", "deadline", "expires"},
		"send": {"tx", "rx", "deadline", "expires"}, "timeout": {"tx", "rx", "deadline", "expires"},
		"channel": {"chan", "fork", "concurrent", "pick"}, "spawn": {"chan", "fork", "concurrent", "pick"},
		"parallel": {"chan", "fork", "concurrent", "pick"}, "select": {"chan", "fork", "concurrent", "pick"},
		"assert": {"check", "verify"},
	} {
		e := reservedKeywordError(lexer.Token{Type: lexer.LookupIdent(word), Literal: word}, "PAR_RESERVED_KEYWORD")
		suggestions := strings.Join(e.Suggestions, " ")
		for _, alternative := range alternatives {
			if !strings.Contains(suggestions, alternative) || !strings.Contains(string(page), "`"+alternative+"`") {
				t.Errorf("%s: missing shared alternative %s", word, alternative)
			}
		}
	}
}

func TestReservedValidTraffic(t *testing.T) {
	for _, body := range []string{
		`func double(x: int) -> int = x * 2`,
		`func f(case: int) -> int = let other = case in other`,
		`func case(x: int) -> int = x
func f() -> int = case(1)`,
		`func f() -> int = let case = {case: 42} in case.case`,
		`func f() -> int = (\a b. a + b)(1, 2)`,
		`type Person = {name: string, age: int}
func f() -> Person = {name: "A", age: 1}`,
	} {
		p := New(lexer.New("module probe\n"+body, "probe.ail"))
		p.ParseFile()
		if len(p.Errors()) != 0 {
			t.Errorf("%s: %v", body, p.Errors())
		}
	}
}

func TestReservedBindingBaselines(t *testing.T) {
	// Same-source HEAD baseline: local has six errors, function name has seven.
	for _, fixture := range []struct {
		source  string
		maximum int
	}{
		{"func f() -> int = { let recv = 5; 42 }", 6},
		{"func recv() -> int = 42", 7},
	} {
		p := New(lexer.New("module probe\n"+fixture.source, "probe.ail"))
		p.ParseFile()
		if len(p.Errors()) == 0 || len(p.Errors()) > fixture.maximum {
			t.Fatalf("baseline exceeded: %v", p.Errors())
		}
		e := p.Errors()[0].(*ParserError)
		if e.Code != "PAR_RESERVED_KEYWORD" || !strings.Contains(strings.Join(e.Suggestions, " "), "rx") {
			t.Fatal(e)
		}
	}
}

func TestReservedDialectAndImportGuards(t *testing.T) {
	p := New(lexer.New("module probe\nfunc f(x: int) -> int = match x with { _ => 42 }", "probe.ail"))
	p.ParseFile()
	count := 0
	for _, e := range p.Errors() {
		if pe, ok := e.(*ParserError); ok && pe.Code == "PAR019" {
			count++
			if pe.Message != "'match ... with' is not valid AILANG syntax (ML/Haskell pattern detected)" || strings.Join(pe.Suggestions, "\n") != "Use: match expr { pattern => body, ... }\nAILANG uses braces for match arms, not 'with'" {
				t.Fatal(pe)
			}
		}
	}
	if count != 1 {
		t.Fatalf("PAR019 count: %d", count)
	}
	p = New(lexer.New("module probe\nimport std/list as recv", "probe.ail"))
	p.ParseFile()
	if len(p.Errors()) != 1 || p.Errors()[0].(*ParserError).Code != "IMP001" {
		t.Fatal(p.Errors())
	}
}
