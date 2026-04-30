package types

import (
	"testing"
)

// Tests for M3: label propagation helpers and TLabelled wrapper.
// These test at the types-package level, independent of Core AST.

// --- TLabelled wrapper ---

func TestTLabelledString(t *testing.T) {
	base := &TCon{Name: "string"}
	email := LabelConst("email")
	lt := &TLabelled{Inner: base, L: email}
	want := "string<email>"
	if lt.String() != want {
		t.Errorf("TLabelled.String() = %q, want %q", lt.String(), want)
	}
}

func TestTLabelledStringBottom(t *testing.T) {
	base := &TCon{Name: "string"}
	lt := &TLabelled{Inner: base, L: LabelBottom()}
	// ⊥ label still shows for explicit TLabelled (use WithLabel to suppress it)
	got := lt.String()
	if got == "" {
		t.Error("TLabelled.String() should not be empty")
	}
}

// --- LabelOf ---

func TestLabelOfTLabelled(t *testing.T) {
	email := LabelConst("email")
	base := &TCon{Name: "string"}
	lt := &TLabelled{Inner: base, L: email}
	got := LabelOf(lt)
	if !LabelEqual(got, email) {
		t.Errorf("LabelOf(TLabelled) = %s, want %s", got, email)
	}
}

func TestLabelOfPlainType(t *testing.T) {
	plain := &TCon{Name: "int"}
	got := LabelOf(plain)
	if !LabelEqual(got, LabelBottom()) {
		t.Errorf("LabelOf(plain type) = %s, want ⊥", got)
	}
}

// --- StripLabel ---

func TestStripLabelTLabelled(t *testing.T) {
	email := LabelConst("email")
	base := &TCon{Name: "string"}
	lt := &TLabelled{Inner: base, L: email}
	got := StripLabel(lt)
	if got != base {
		t.Errorf("StripLabel(TLabelled) = %v, want base type %v", got, base)
	}
}

func TestStripLabelPlain(t *testing.T) {
	plain := &TCon{Name: "int"}
	got := StripLabel(plain)
	if got != plain {
		t.Errorf("StripLabel(plain) = %v, want same type %v", got, plain)
	}
}

// --- WithLabel ---

func TestWithLabelNonBottom(t *testing.T) {
	base := &TCon{Name: "string"}
	email := LabelConst("email")
	result := WithLabel(base, email)
	if lt, ok := result.(*TLabelled); !ok {
		t.Errorf("WithLabel(non-⊥) should return TLabelled, got %T", result)
	} else if !LabelEqual(lt.L, email) {
		t.Errorf("WithLabel label = %s, want %s", lt.L, email)
	}
}

func TestWithLabelBottom(t *testing.T) {
	base := &TCon{Name: "string"}
	result := WithLabel(base, LabelBottom())
	// ⊥ label should return the base type unwrapped (no unnecessary wrapping)
	if _, ok := result.(*TLabelled); ok {
		t.Error("WithLabel(⊥) should not wrap in TLabelled")
	}
	if result != base {
		t.Errorf("WithLabel(⊥) = %v, want base %v", result, base)
	}
}

func TestWithLabelReplaceExisting(t *testing.T) {
	base := &TCon{Name: "string"}
	user := LabelConst("user")
	email := LabelConst("email")
	// wrap with user first
	wrapped := WithLabel(base, user)
	// now re-wrap with email — should update the label, not double-wrap
	result := WithLabel(wrapped, email)
	lt, ok := result.(*TLabelled)
	if !ok {
		t.Fatalf("expected TLabelled, got %T", result)
	}
	if !LabelEqual(lt.L, email) {
		t.Errorf("re-wrapped label = %s, want %s", lt.L, email)
	}
	// Inner should be the original base (not another TLabelled)
	if _, nested := lt.Inner.(*TLabelled); nested {
		t.Error("WithLabel should not double-nest TLabelled wrappers")
	}
}

// --- PurePropagateLabel (APP-PURE rule) ---
// For a pure function call f(a, b): result label = returnLabel ⊔ label(a) ⊔ label(b)

func TestPurePropagateLabelIdentity(t *testing.T) {
	// id : string<α> -> string<α>  applied to  x : string<email>
	// arg label = email, func return label = α (or ⊥ if unresolved)
	// result label should be at least email (join of arg labels)
	email := LabelConst("email")
	argTypes := []Type{WithLabel(&TCon{Name: "string"}, email)}
	retLabel := LabelBottom() // unresolved return label treated as ⊥

	resultLabel := PurePropagateLabel(retLabel, argTypes)
	if !LabelEqual(resultLabel, email) {
		t.Errorf("pure id(string<email>): result label = %s, want %s", resultLabel, email)
	}
}

func TestPurePropagateLabelConcat(t *testing.T) {
	// concat(s1, s2) where s1: string<a>, s2: string<b>  → result: string<a ⊔ b>
	a := LabelConst("a")
	b := LabelConst("b")
	argTypes := []Type{
		WithLabel(&TCon{Name: "string"}, a),
		WithLabel(&TCon{Name: "string"}, b),
	}
	retLabel := LabelBottom()

	resultLabel := PurePropagateLabel(retLabel, argTypes)
	expected := LabelJoin(a, b)
	if !LabelEqual(resultLabel, expected) {
		t.Errorf("concat label = %s, want %s", resultLabel, expected)
	}
}

func TestPurePropagateLabelNoLabel(t *testing.T) {
	// f(x) where x has no label → result has ⊥
	argTypes := []Type{&TCon{Name: "int"}}
	retLabel := LabelBottom()
	resultLabel := PurePropagateLabel(retLabel, argTypes)
	if !LabelEqual(resultLabel, LabelBottom()) {
		t.Errorf("pure f(unlabelled): result label = %s, want ⊥", resultLabel)
	}
}

func TestPurePropagateLabelJoinReturnAndArg(t *testing.T) {
	// f returns string<user>, arg is string<email>
	// result: string<user ⊔ email>
	user := LabelConst("user")
	email := LabelConst("email")
	argTypes := []Type{WithLabel(&TCon{Name: "string"}, email)}
	retLabel := user

	resultLabel := PurePropagateLabel(retLabel, argTypes)
	expected := LabelJoin(user, email)
	if !LabelEqual(resultLabel, expected) {
		t.Errorf("join result = %s, want %s", resultLabel, expected)
	}
}

// --- JoinLabels (match arm join) ---

func TestJoinLabelsMatchArms(t *testing.T) {
	// match arm 1: string<email>, arm 2: string (no label / ⊥)
	// result: string<email>
	email := LabelConst("email")
	arm1 := WithLabel(&TCon{Name: "string"}, email)
	arm2 := &TCon{Name: "string"}

	resultLabel := JoinLabels(arm1, arm2)
	if !LabelEqual(resultLabel, email) {
		t.Errorf("match join = %s, want %s", resultLabel, email)
	}
}

func TestJoinLabelsMatchBothLabelled(t *testing.T) {
	pii := LabelConst("pii")
	user := LabelConst("user")
	arm1 := WithLabel(&TCon{Name: "string"}, pii)
	arm2 := WithLabel(&TCon{Name: "string"}, user)

	resultLabel := JoinLabels(arm1, arm2)
	expected := LabelJoin(pii, user)
	if !LabelEqual(resultLabel, expected) {
		t.Errorf("match join both = %s, want %s", resultLabel, expected)
	}
}

// --- TLabelled participates in existing type machinery ---

func TestTLabelledEquals(t *testing.T) {
	email := LabelConst("email")
	base := &TCon{Name: "string"}
	lt1 := &TLabelled{Inner: base, L: email}
	lt2 := &TLabelled{Inner: base, L: email}
	if !lt1.Equals(lt2) {
		t.Error("identical TLabelled should be equal")
	}
}

func TestTLabelledEqualsPlainFalse(t *testing.T) {
	email := LabelConst("email")
	lt := &TLabelled{Inner: &TCon{Name: "string"}, L: email}
	plain := &TCon{Name: "string"}
	if lt.Equals(plain) {
		t.Error("TLabelled should not equal plain type (different structure)")
	}
}

func TestTLabelledSubstitute(t *testing.T) {
	a := &TVar{Name: "a"}
	email := LabelConst("email")
	lt := &TLabelled{Inner: a, L: email}
	subs := map[string]Type{"a": &TCon{Name: "int"}}
	result := lt.Substitute(subs)
	lt2, ok := result.(*TLabelled)
	if !ok {
		t.Fatalf("Substitute should return TLabelled, got %T", result)
	}
	if !LabelEqual(lt2.L, email) {
		t.Error("Substitute should preserve label")
	}
	if lt2.Inner.String() != "int" {
		t.Errorf("Substitute inner = %s, want int", lt2.Inner.String())
	}
}
