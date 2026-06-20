package astedit

import (
	"strings"
	"testing"
)

func TestReplaceDecl(t *testing.T) {
	src := "module sd\n\nexport func a() -> int { 1 }\n\nexport func b() -> int { 2 }\n"

	out, err := ReplaceDecl(src, "sd.ail", "a", "export func a() -> int { 42 }")
	if err != nil {
		t.Fatalf("ReplaceDecl: %v", err)
	}
	if !strings.Contains(out, "42") {
		t.Errorf("new body (42) missing:\n%s", out)
	}
	if strings.Contains(out, "{ 1 }") {
		t.Errorf("old decl body not fully replaced — span too narrow:\n%s", out)
	}
	if !strings.Contains(out, "func b()") {
		t.Errorf("func b not preserved:\n%s", out)
	}
	// Result must re-parse with both functions intact.
	if FindFunc(out, "sd.ail", "a") == nil {
		t.Errorf("result does not re-parse with func a:\n%s", out)
	}
	if FindFunc(out, "sd.ail", "b") == nil {
		t.Errorf("result lost func b:\n%s", out)
	}
}

func TestReplaceDecl_NotFound(t *testing.T) {
	if _, err := ReplaceDecl("module m\nexport func a() -> int { 1 }\n", "m.ail", "missing", "x"); err == nil {
		t.Error("expected error for missing declaration")
	}
}
