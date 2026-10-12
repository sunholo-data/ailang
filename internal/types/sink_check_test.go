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
