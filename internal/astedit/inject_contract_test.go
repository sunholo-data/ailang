package astedit

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// reparseClean returns true if src parses with zero errors (the injected output must stay valid).
func reparseClean(t *testing.T, src string) bool {
	t.Helper()
	p := parser.New(lexer.New(src, "inj_test.ail"))
	p.Parse()
	if errs := p.Errors(); len(errs) != 0 {
		t.Logf("parse errors:\n%v\n--- source ---\n%s", errs, src)
		return false
	}
	return true
}

func TestInjectContract_EquationForm(t *testing.T) {
	src := "module test/inj\nexport func f(x: int) -> int ! {} = x + 1\n"
	out, err := InjectContract(src, "inj.ail", "f", "ensures { result > 0 }")
	if err != nil {
		t.Fatalf("InjectContract: %v", err)
	}
	if !reparseClean(t, out) {
		t.Fatal("injected equation-form source no longer parses")
	}
	if !strings.Contains(out, "ensures { result > 0 }") {
		t.Errorf("contract missing from output:\n%s", out)
	}
	// contract must sit BEFORE the body delimiter `=`
	if strings.Index(out, "ensures") > strings.Index(out, "= x + 1") {
		t.Errorf("contract injected after `=`:\n%s", out)
	}
}

func TestInjectContract_BlockForm(t *testing.T) {
	src := "module test/inj\nexport func g(x: int) -> int ! {} {\n  x + 1\n}\n"
	out, err := InjectContract(src, "inj.ail", "g", "requires { x >= 0 }")
	if err != nil {
		t.Fatalf("InjectContract: %v", err)
	}
	if !reparseClean(t, out) {
		t.Fatal("injected block-form source no longer parses")
	}
	if !strings.Contains(out, "requires { x >= 0 }") {
		t.Errorf("contract missing from output:\n%s", out)
	}
}

func TestInjectContract_NotFound(t *testing.T) {
	src := "module test/inj\nexport func f(x: int) -> int ! {} = x\n"
	if _, err := InjectContract(src, "inj.ail", "nope", "ensures { result > 0 }"); err == nil {
		t.Error("expected error for missing function, got nil")
	}
}

// TestInjectContract_SkipsExistingContract: a candidate that ALREADY carries a contract must be left
// unchanged (injecting a second clause would duplicate it and break parsing).
func TestInjectContract_SkipsExistingContract(t *testing.T) {
	src := "module test/inj\nexport func f(x: int) -> int ! {} ensures { result > 0 } = x + 1\n"
	out, err := InjectContract(src, "inj.ail", "f", "ensures { result > x }")
	if err != nil {
		t.Fatalf("InjectContract: %v", err)
	}
	if out != src {
		t.Errorf("expected source unchanged when a contract is already present, got:\n%s", out)
	}
	if !reparseClean(t, out) {
		t.Fatal("unchanged source must still parse")
	}
}
