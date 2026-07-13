package types

import (
	"strings"
	"testing"
)

// M-XMOD-ALIAS-POLY: unit tests over the parameterized-alias expansion added to
// expandAlias/Unify. These exercise the unifier directly with a hand-built
// alias env + params env, mirroring how the pipeline wires them
// (NewUnifierWithAliasesAndParams). Two layers:
//   1. Positive: an APPLIED alias (`Box[int]`) instantiates its body and unifies
//      structurally with the expanded concrete type.
//   2. Non-regression: a real parameterized ADT head (NOT in aliasEnv) stays
//      nominal — expansion must NOT fire. This is the critical guardrail.
//   3. Arity: wrong argument count → coded TC_ALIAS_ARITY_001, directional.

// polyUnifier builds a unifier with the given alias bodies + param lists.
func polyUnifier(aliases map[string]Type, params map[string][]string) *Unifier {
	return NewUnifierWithAliasesAndParams(aliases, params)
}

// boxAliasEnv: `type Box[a] = { items: [a] }`.
func boxAliasEnv() (map[string]Type, map[string][]string) {
	body := &TRecord{
		Fields: map[string]Type{
			"items": &TList{Element: &TVar2{Name: "a", Kind: Star}},
		},
	}
	return map[string]Type{"Box": body}, map[string][]string{"Box": {"a"}}
}

// TestAliasPoly_Record_Expands: Box[int] unifies with { items: [int] }.
func TestAliasPoly_Record_Expands(t *testing.T) {
	aliases, params := boxAliasEnv()
	u := polyUnifier(aliases, params)

	applied := &TApp{Constructor: &TCon{Name: "Box"}, Args: []Type{TInt}}
	concrete := &TRecord{
		Fields: map[string]Type{"items": &TList{Element: TInt}},
	}

	if _, err := u.Unify(applied, concrete, Substitution{}); err != nil {
		t.Fatalf("Box[int] must unify with { items: [int] }, got: %v", err)
	}
}

// TestAliasPoly_BareParam_Expands: `type Ident[a] = a`, Ident[int] unifies int.
func TestAliasPoly_BareParam_Expands(t *testing.T) {
	aliases := map[string]Type{"Ident": &TVar2{Name: "a", Kind: Star}}
	params := map[string][]string{"Ident": {"a"}}
	u := polyUnifier(aliases, params)

	applied := &TApp{Constructor: &TCon{Name: "Ident"}, Args: []Type{TInt}}
	if _, err := u.Unify(applied, TInt, Substitution{}); err != nil {
		t.Fatalf("Ident[int] must unify with int, got: %v", err)
	}
}

// TestAliasPoly_Tuple_Expands: `type Pair[a,b] = (a,b)`, Pair[int,string].
func TestAliasPoly_Tuple_Expands(t *testing.T) {
	aliases := map[string]Type{
		"Pair": &TTuple{Elements: []Type{
			&TVar2{Name: "a", Kind: Star},
			&TVar2{Name: "b", Kind: Star},
		}},
	}
	params := map[string][]string{"Pair": {"a", "b"}}
	u := polyUnifier(aliases, params)

	applied := &TApp{Constructor: &TCon{Name: "Pair"}, Args: []Type{TInt, TString}}
	concrete := &TTuple{Elements: []Type{TInt, TString}}
	if _, err := u.Unify(applied, concrete, Substitution{}); err != nil {
		t.Fatalf("Pair[int,string] must unify with (int, string), got: %v", err)
	}
}

// TestAliasPoly_Function_Expands: `type Fn[a,b] = (a) -> b`, Fn[int,int].
func TestAliasPoly_Function_Expands(t *testing.T) {
	aliases := map[string]Type{
		"Fn": &TFunc2{
			Params: []Type{&TVar2{Name: "a", Kind: Star}},
			Return: &TVar2{Name: "b", Kind: Star},
		},
	}
	params := map[string][]string{"Fn": {"a", "b"}}
	u := polyUnifier(aliases, params)

	applied := &TApp{Constructor: &TCon{Name: "Fn"}, Args: []Type{TInt, TInt}}
	concrete := &TFunc2{Params: []Type{TInt}, Return: TInt}
	if _, err := u.Unify(applied, concrete, Substitution{}); err != nil {
		t.Fatalf("Fn[int,int] must unify with (int) -> int, got: %v", err)
	}
}

// TestAliasPoly_Nested_Expands: Box[Box[int]] instantiates recursively.
func TestAliasPoly_Nested_Expands(t *testing.T) {
	aliases, params := boxAliasEnv()
	u := polyUnifier(aliases, params)

	inner := &TApp{Constructor: &TCon{Name: "Box"}, Args: []Type{TInt}}
	applied := &TApp{Constructor: &TCon{Name: "Box"}, Args: []Type{inner}}
	// Box[Box[int]] == { items: [ { items: [int] } ] }
	concrete := &TRecord{
		Fields: map[string]Type{
			"items": &TList{Element: &TRecord{
				Fields: map[string]Type{"items": &TList{Element: TInt}},
			}},
		},
	}
	if _, err := u.Unify(applied, concrete, Substitution{}); err != nil {
		t.Fatalf("Box[Box[int]] must unify with nested record, got: %v", err)
	}
}

// --- CRITICAL NON-REGRESSION: real ADTs stay nominal -----------------------

// TestAliasPoly_ADT_StaysNominal_Option: Option[a] head is NOT in aliasEnv, so
// expandAlias must return it unchanged (no instantiation). Two different
// instantiations must NOT unify to a record — Option[int] vs { items: [int] }
// must FAIL, proving expansion did not fire on the ADT.
func TestAliasPoly_ADT_StaysNominal_Option(t *testing.T) {
	// Only Box is an alias. Option is a genuine ADT (absent from aliasEnv).
	aliases, params := boxAliasEnv()
	u := polyUnifier(aliases, params)

	optionInt := &TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{TInt}}

	// Option[int] must NOT expand into anything: unifying it with a record fails.
	rec := &TRecord{Fields: map[string]Type{"items": &TList{Element: TInt}}}
	if _, err := u.Unify(optionInt, rec, Substitution{}); err == nil {
		t.Fatal("Option[int] must NOT expand/unify with a record — ADT must stay nominal")
	}

	// Option[int] unifies with itself (nominal identity preserved).
	if _, err := u.Unify(optionInt,
		&TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{TInt}},
		Substitution{}); err != nil {
		t.Fatalf("Option[int] must unify with Option[int] (nominal), got: %v", err)
	}
}

// TestAliasPoly_ADT_StaysNominal_Result: Result[a,b] must stay nominal.
func TestAliasPoly_ADT_StaysNominal_Result(t *testing.T) {
	aliases, params := boxAliasEnv()
	u := polyUnifier(aliases, params)

	resultIS := &TApp{Constructor: &TCon{Name: "Result"}, Args: []Type{TInt, TString}}
	// Result[int,string] vs Result[string,int] must NOT unify (args differ),
	// which is only meaningful because Result is treated nominally (not expanded).
	other := &TApp{Constructor: &TCon{Name: "Result"}, Args: []Type{TString, TInt}}
	if _, err := u.Unify(resultIS, other, Substitution{}); err == nil {
		t.Fatal("Result[int,string] must NOT unify with Result[string,int]")
	}
}

// TestAliasPoly_UserTree_StaysNominal: a user parameterized ADT `type Tree[a]`
// whose head TCon is not in aliasEnv is untouched.
func TestAliasPoly_UserTree_StaysNominal(t *testing.T) {
	aliases, params := boxAliasEnv()
	u := polyUnifier(aliases, params)

	treeInt := &TApp{Constructor: &TCon{Name: "Tree"}, Args: []Type{TInt}}
	if _, err := u.Unify(treeInt, TInt, Substitution{}); err == nil {
		t.Fatal("Tree[int] must NOT expand/unify with int — user ADT must stay nominal")
	}
}

// --- Arity diagnostics ------------------------------------------------------

// TestAliasArity_TooMany: Box[int, string] on a 1-param alias → coded, directional.
func TestAliasArity_TooMany(t *testing.T) {
	aliases, params := boxAliasEnv()
	u := polyUnifier(aliases, params)

	applied := &TApp{Constructor: &TCon{Name: "Box"}, Args: []Type{TInt, TString}}
	_, err := u.Unify(applied,
		&TRecord{Fields: map[string]Type{"items": &TList{Element: TInt}}},
		Substitution{})
	if err == nil {
		t.Fatal("Box[int, string] (arity 2 on 1-param alias) must fail")
	}
	for _, want := range []string{
		"TC_ALIAS_ARITY_001",
		"expects 1 type argument(s)",
		"but 2 provided",
		"Suggestion:",
		"remove the extra 1",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("arity-too-many error missing %q\ngot: %v", want, err)
		}
	}
}

// TestAliasArity_NullaryApplied: a nullary alias applied with args → coded.
func TestAliasArity_NullaryApplied(t *testing.T) {
	aliases := map[string]Type{"Row": &TRecord{Fields: map[string]Type{"x": TInt}}}
	// No params entry for Row => nullary.
	u := polyUnifier(aliases, map[string][]string{})

	applied := &TApp{Constructor: &TCon{Name: "Row"}, Args: []Type{TInt}}
	_, err := u.Unify(applied, TInt, Substitution{})
	if err == nil {
		t.Fatal("Row[int] on a nullary alias must fail")
	}
	for _, want := range []string{
		"TC_ALIAS_ARITY_001",
		"expects 0 type argument(s)",
		"but 1 provided",
		"takes no type arguments",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("nullary-applied error missing %q\ngot: %v", want, err)
		}
	}
}

// TestAliasArityMismatchMsg_Directional: unit test on the rendered text.
func TestAliasArityMismatchMsg_Directional(t *testing.T) {
	// too many
	msg := aliasArityMismatchMsg("Box", 1, 2)
	if !strings.Contains(msg, "TC_ALIAS_ARITY_001") || !strings.Contains(msg, "remove the extra 1") {
		t.Errorf("too-many msg wrong:\n%s", msg)
	}
	// too few
	msg = aliasArityMismatchMsg("Pair", 2, 1)
	if !strings.Contains(msg, "supply the missing 1") {
		t.Errorf("too-few msg wrong:\n%s", msg)
	}
	// nullary applied
	msg = aliasArityMismatchMsg("Row", 0, 1)
	if !strings.Contains(msg, "takes no type arguments") {
		t.Errorf("nullary msg wrong:\n%s", msg)
	}
}

// TestAliasPoly_NullaryAliasUnchanged: a plain nullary alias (TCon head, no
// args) still expands via the existing TCon path — M-XMOD-ALIAS pack regression.
func TestAliasPoly_NullaryAliasUnchanged(t *testing.T) {
	aliases := map[string]Type{"MyInt": TInt}
	u := polyUnifier(aliases, map[string][]string{})

	if _, err := u.Unify(&TCon{Name: "MyInt"}, TInt, Substitution{}); err != nil {
		t.Fatalf("nullary alias MyInt must still expand to int, got: %v", err)
	}
}
