package types

import (
	"testing"
)

// TestLabelBottom tests the bottom element identity law: ⊥ ⊔ L = L
func TestLabelBottom(t *testing.T) {
	email := LabelConst("email")
	bottom := LabelBottom()

	joined := LabelJoin(bottom, email)
	if !LabelEqual(joined, email) {
		t.Errorf("⊥ ⊔ <email> should equal <email>, got %s", joined)
	}

	joined2 := LabelJoin(email, bottom)
	if !LabelEqual(joined2, email) {
		t.Errorf("<email> ⊔ ⊥ should equal <email>, got %s", joined2)
	}

	joined3 := LabelJoin(bottom, bottom)
	if !LabelEqual(joined3, bottom) {
		t.Errorf("⊥ ⊔ ⊥ should equal ⊥, got %s", joined3)
	}
}

// TestLabelIdempotence: L ⊔ L = L
func TestLabelIdempotence(t *testing.T) {
	email := LabelConst("email")
	joined := LabelJoin(email, email)
	if !LabelEqual(joined, email) {
		t.Errorf("<email> ⊔ <email> should equal <email>, got %s", joined)
	}
}

// TestLabelCommutativity: <email> ⊔ <user> equals <user> ⊔ <email> after canonicalisation
func TestLabelCommutativity(t *testing.T) {
	email := LabelConst("email")
	user := LabelConst("user")

	ab := LabelJoin(email, user)
	ba := LabelJoin(user, email)
	if !LabelEqual(ab, ba) {
		t.Errorf("<email> ⊔ <user> should Equal <user> ⊔ <email>, got %s vs %s", ab, ba)
	}
}

// TestLabelAssociativity: ((<a> ⊔ <b>) ⊔ <c>) Equal (<a> ⊔ (<b> ⊔ <c>))
func TestLabelAssociativity(t *testing.T) {
	a := LabelConst("a")
	b := LabelConst("b")
	c := LabelConst("c")

	lhs := LabelJoin(LabelJoin(a, b), c)
	rhs := LabelJoin(a, LabelJoin(b, c))
	if !LabelEqual(lhs, rhs) {
		t.Errorf("(<a> ⊔ <b>) ⊔ <c> should Equal <a> ⊔ (<b> ⊔ <c>), got %s vs %s", lhs, rhs)
	}
}

// TestLabelEvalNot tests the refinement predicate EvalNot(L, ℓ): true iff ℓ ∉ L
func TestLabelEvalNot(t *testing.T) {
	email := LabelConst("email")
	user := LabelConst("user")
	bottom := LabelBottom()

	// EvalNot(⊥, <email>) == true (email is not in ⊥)
	if !EvalNot(bottom, email) {
		t.Error("EvalNot(⊥, <email>) should be true")
	}

	// EvalNot(<email>, <email>) == false
	if EvalNot(email, email) {
		t.Error("EvalNot(<email>, <email>) should be false")
	}

	// EvalNot(<email> ⊔ <user>, <email>) == false (email is present in the join)
	emailUser := LabelJoin(email, user)
	if EvalNot(emailUser, email) {
		t.Errorf("EvalNot(<email> ⊔ <user>, <email>) should be false")
	}

	// EvalNot(<email> ⊔ <user>, <pii>) == true (pii is not in join)
	pii := LabelConst("pii")
	if !EvalNot(emailUser, pii) {
		t.Error("EvalNot(<email> ⊔ <user>, <pii>) should be true")
	}
}

// TestLabelSubsumes tests Subsumes(L, ℓ): ℓ ⊆ L
func TestLabelSubsumes(t *testing.T) {
	email := LabelConst("email")
	user := LabelConst("user")
	bottom := LabelBottom()

	if LabelSubsumes(bottom, email) {
		t.Error("⊥ should not subsume <email>")
	}
	if !LabelSubsumes(email, email) {
		t.Error("<email> should subsume <email>")
	}
	joined := LabelJoin(email, user)
	if !LabelSubsumes(joined, email) {
		t.Error("<email> ⊔ <user> should subsume <email>")
	}
	if !LabelSubsumes(joined, user) {
		t.Error("<email> ⊔ <user> should subsume <user>")
	}
	pii := LabelConst("pii")
	if LabelSubsumes(joined, pii) {
		t.Error("<email> ⊔ <user> should not subsume <pii>")
	}
}

// TestLabelString tests human-readable rendering
func TestLabelString(t *testing.T) {
	bottom := LabelBottom()
	if bottom.String() != "⊥" {
		t.Errorf("bottom.String() = %q, want \"⊥\"", bottom.String())
	}

	email := LabelConst("email")
	if email.String() != "<email>" {
		t.Errorf("email.String() = %q, want \"<email>\"", email.String())
	}

	// Join renders with canonical (sorted) order
	user := LabelConst("user")
	joined := LabelJoin(email, user) // email < user alphabetically
	s := joined.String()
	if s != "<email> ⊔ <user>" {
		t.Errorf("joined.String() = %q, want \"<email> ⊔ <user>\"", s)
	}

	// Reverse order should still canonicalise
	joined2 := LabelJoin(user, email)
	s2 := joined2.String()
	if s2 != "<email> ⊔ <user>" {
		t.Errorf("reversed joined.String() = %q, want \"<email> ⊔ <user>\"", s2)
	}
}

// TestLabelVar tests label variables (used for polymorphic labels in generics)
func TestLabelVar(t *testing.T) {
	α := LabelVar("α")
	if α.String() != "α" {
		t.Errorf("LabelVar(α).String() = %q, want \"α\"", α.String())
	}
	// A var subsumes itself
	if !LabelSubsumes(α, α) {
		t.Error("LabelVar α should subsume itself")
	}
	// ⊥ does not subsume a var
	bottom := LabelBottom()
	if LabelSubsumes(bottom, α) {
		t.Error("⊥ should not subsume a label var")
	}
}

// TestLabelJoinThreeWay: triple join normalises correctly (run -count=20 to catch map nondeterminism)
func TestLabelJoinThreeWay(t *testing.T) {
	a := LabelConst("alpha")
	b := LabelConst("beta")
	c := LabelConst("gamma")

	// All orderings should produce the same Equal result
	variants := []Label{
		LabelJoin(LabelJoin(a, b), c),
		LabelJoin(LabelJoin(a, c), b),
		LabelJoin(LabelJoin(b, a), c),
		LabelJoin(LabelJoin(b, c), a),
		LabelJoin(LabelJoin(c, a), b),
		LabelJoin(LabelJoin(c, b), a),
	}
	ref := variants[0]
	for i, v := range variants[1:] {
		if !LabelEqual(ref, v) {
			t.Errorf("ordering variant %d produced different label: %s vs %s", i+1, ref, v)
		}
	}

	// String representation should be canonical and stable
	s := ref.String()
	for i, v := range variants[1:] {
		if v.String() != s {
			t.Errorf("ordering variant %d string %q != canonical %q", i+1, v.String(), s)
		}
	}
}

// TestLabelEqualSelf tests that every label equals itself (reflexivity)
func TestLabelEqualSelf(t *testing.T) {
	labels := []Label{
		LabelBottom(),
		LabelConst("email"),
		LabelVar("L"),
		LabelJoin(LabelConst("a"), LabelConst("b")),
	}
	for _, l := range labels {
		if !LabelEqual(l, l) {
			t.Errorf("LabelEqual(%s, %s) should be true (reflexivity)", l, l)
		}
	}
}
