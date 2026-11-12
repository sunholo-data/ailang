package parser

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
)

// @deprecated("replacement"[, "note"]) and @experimental
// (M-V1-STDLIB-SURFACE-POLISH §6.1).
func TestDeprecatedAndExperimentalAnnotations(t *testing.T) {
	input := `module lib

export pure func headOr[a](xs: [a], d: a) -> a = d

@deprecated("headOr")
export pure func head_or[a](xs: [a], d: a) -> a = headOr(xs, d)

@deprecated("contains", "arguments swap")
export pure func member(x: int, xs: [int]) -> bool = false

@experimental
export func repair(s: string) -> string = s

@verify(depth: 2)
@experimental
@deprecated("g")
pure func both(x: int) -> int = x
`
	p := New(lexer.New(input, "lib.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	got := map[string]string{}
	for _, fd := range file.Funcs {
		repl, note, dep := fd.Deprecation()
		got[fd.Name] = repl + "|" + note + "|" + map[bool]string{true: "dep", false: ""}[dep] + "|" + map[bool]string{true: "exp", false: ""}[fd.IsExperimental()]
	}
	want := map[string]string{
		"headOr":  "|||",
		"head_or": "headOr||dep|",
		"member":  "contains|arguments swap|dep|",
		"repair":  "|||exp",
		"both":    "g||dep|exp",
	}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("%s: got %q, want %q", name, got[name], w)
		}
	}
}

func TestDeprecatedAnnotationArgumentErrors(t *testing.T) {
	for name, ann := range map[string]string{
		"no args":      `@deprecated`,
		"empty list":   `@deprecated()`,
		"three args":   `@deprecated("a", "b", "c")`,
		"not string":   `@deprecated(headOr)`,
		"int arg":      `@deprecated(1)`,
		"empty name":   `@deprecated("")`,
		"blank name":   `@deprecated("  ")`,
		"exp has arg":  `@experimental("x")`,
		"exp has unit": `@experimental()`,
	} {
		t.Run(name, func(t *testing.T) {
			p := New(lexer.New("module lib\n\n"+ann+"\nexport func f(x: int) -> int = x\n", "lib.ail"))
			_ = p.ParseFile()
			if len(p.Errors()) == 0 {
				t.Fatalf("%s: accepted", ann)
			}
			if name == "exp has arg" || name == "exp has unit" {
				// The right message, first: not "must be followed by a function".
				if first := p.Errors()[0].Error(); !strings.Contains(first, "PAR_EXPERIMENTAL_ARG") || !strings.Contains(first, "takes no arguments") {
					t.Errorf("@experimental(\"x\"): first error %q, want PAR_EXPERIMENTAL_ARG", first)
				}
				for _, err := range p.Errors() {
					if strings.Contains(err.Error(), "must be followed by a function") {
						t.Errorf("misleading follow-on error: %v", err)
					}
				}
				return
			}
			named := false
			for _, err := range p.Errors() {
				if strings.Contains(err.Error(), "PAR_DEPRECATED_ARG") {
					named = true
				}
			}
			if !named {
				t.Errorf("%s: no PAR_DEPRECATED_ARG among %v", ann, p.Errors())
			}
		})
	}
}
