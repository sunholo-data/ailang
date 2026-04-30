package types

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// TestSinkCheckViolation: string<email> reaching string{not email} → violation
func TestSinkCheckViolation(t *testing.T) {
	argType := WithLabel(&TCon{Name: "string"}, LabelConst("email"))
	refinement := &ast.RefinementExpr{NotLabel: "email"}

	err := CheckSinkRefinement(argType, refinement)
	if err == nil {
		t.Fatal("expected sink violation, got nil")
	}
	if err.ArgLabel.String() != LabelConst("email").String() {
		t.Errorf("error arg label = %s, want <email>", err.ArgLabel)
	}
	if err.SinkLabel != "email" {
		t.Errorf("error sink label = %q, want \"email\"", err.SinkLabel)
	}
}

// TestSinkCheckPass: string<sanitized> reaching string{not email} → no violation
func TestSinkCheckPass(t *testing.T) {
	argType := WithLabel(&TCon{Name: "string"}, LabelConst("sanitized"))
	refinement := &ast.RefinementExpr{NotLabel: "email"}

	err := CheckSinkRefinement(argType, refinement)
	if err != nil {
		t.Errorf("expected no violation, got: %v", err)
	}
}

// TestSinkCheckUnlabelledPass: unlabelled string reaching string{not email} → no violation (⊥ is safe)
func TestSinkCheckUnlabelledPass(t *testing.T) {
	argType := &TCon{Name: "string"} // no label = ⊥
	refinement := &ast.RefinementExpr{NotLabel: "email"}

	err := CheckSinkRefinement(argType, refinement)
	if err != nil {
		t.Errorf("unlabelled arg should pass sink: got %v", err)
	}
}

// TestSinkCheckJoinViolation: string<email ⊔ user> reaching string{not email} → violation
func TestSinkCheckJoinViolation(t *testing.T) {
	joined := LabelJoin(LabelConst("email"), LabelConst("user"))
	argType := WithLabel(&TCon{Name: "string"}, joined)
	refinement := &ast.RefinementExpr{NotLabel: "email"}

	err := CheckSinkRefinement(argType, refinement)
	if err == nil {
		t.Fatal("join label containing email should violate {not email} sink")
	}
}

// TestSinkCheckNoRefinement: nil refinement → no violation (not a sink)
func TestSinkCheckNoRefinement(t *testing.T) {
	argType := WithLabel(&TCon{Name: "string"}, LabelConst("email"))
	err := CheckSinkRefinement(argType, nil)
	if err != nil {
		t.Errorf("nil refinement should not trigger sink check, got: %v", err)
	}
}

// TestDeclassCheckViolation: input<email>, output<sanitized>, no Declassify → violation
func TestDeclassCheckViolation(t *testing.T) {
	inputLabel := LabelConst("email")
	outputLabel := LabelConst("sanitized")
	effectRow := []string{} // no Declassify

	err := CheckDeclassify(inputLabel, outputLabel, effectRow)
	if err == nil {
		t.Fatal("expected DECLASS violation, got nil")
	}
	if !err.NeedsDeclarrify {
		t.Error("violation should set NeedsDeclarrify = true")
	}
}

// TestDeclassCheckPass: input<email>, output<sanitized>, with Declassify → no violation
func TestDeclassCheckPass(t *testing.T) {
	inputLabel := LabelConst("email")
	outputLabel := LabelConst("sanitized")
	effectRow := []string{"Declassify"}

	err := CheckDeclassify(inputLabel, outputLabel, effectRow)
	if err != nil {
		t.Errorf("function with Declassify should pass DECLASS check, got: %v", err)
	}
}

// TestDeclassCheckIdentity: input<α>, output<α> (same label var) → no violation without Declassify
func TestDeclassCheckIdentity(t *testing.T) {
	// Both input and output carry the same label variable α
	α := LabelVar("α")
	effectRow := []string{} // no Declassify needed for identity

	err := CheckDeclassify(α, α, effectRow)
	if err != nil {
		t.Errorf("identity function (same label in/out) should not require Declassify, got: %v", err)
	}
}

// TestDeclassCheckBothBottom: ⊥ → ⊥, no Declassify → no violation (unlabelled functions are fine)
func TestDeclassCheckBothBottom(t *testing.T) {
	err := CheckDeclassify(LabelBottom(), LabelBottom(), []string{})
	if err != nil {
		t.Errorf("⊥-to-⊥ should not require Declassify, got: %v", err)
	}
}

// TestDeclassCheckSameConst: input<email>, output<email> → no violation (no label change)
func TestDeclassCheckSameConst(t *testing.T) {
	email := LabelConst("email")
	err := CheckDeclassify(email, email, []string{})
	if err != nil {
		t.Errorf("same label in/out should not require Declassify, got: %v", err)
	}
}

// TestSinkErrorMessage: violation error carries readable message
func TestSinkErrorMessage(t *testing.T) {
	argType := WithLabel(&TCon{Name: "string"}, LabelConst("email"))
	refinement := &ast.RefinementExpr{NotLabel: "email"}

	err := CheckSinkRefinement(argType, refinement)
	if err == nil {
		t.Fatal("expected violation")
	}
	msg := err.Error()
	if msg == "" {
		t.Error("SinkError.Error() should not be empty")
	}
	// Should mention "email" and the sink constraint
	if !containsAny(msg, "email", "not") {
		t.Errorf("error message %q should mention the label and sink constraint", msg)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
