package types

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// #2244: an alias of an ADT is that ADT. These unit tests pin the one place
// that answers "which ADT is this?" (nominal_head.go) and each consumer of it;
// source-level coverage is internal/pipeline/match_alias_test.go and
// internal/vm/match_alias_test.go.

// aliasTC is a checker that knows m's Color, Hue = Color, Tint = Hue,
// IntOpt = Option[int], Opt[a] = Option[a], P = { x: int } and the record
// newtype W = W({ x: int }).
func aliasTC() *CoreTypeChecker {
	tc := newMatchTC()
	color := &TCon{Name: "Color", Module: "m"}
	option := &TCon{Name: "Option", Module: "std/option"}
	tc.RegisterTypeAlias(NominalKey("m", "Hue"), color)
	tc.RegisterTypeAlias(NominalKey("m", "Tint"), &TCon{Name: "Hue", Module: "m"})
	tc.RegisterTypeAlias(NominalKey("m", "IntOpt"), &TApp{Constructor: option, Args: []Type{TInt}})
	tc.RegisterTypeAlias(NominalKey("m", "Opt"), &TApp{Constructor: option, Args: []Type{&TVar2{Name: "a", Kind: Star}}})
	tc.RegisterTypeAliasParams(NominalKey("m", "Opt"), []string{"a"})
	rec := &TRecord{Fields: map[string]Type{"x": TInt}}
	tc.RegisterTypeAlias(NominalKey("m", "P"), rec)
	tc.RegisterTypeAlias(NominalKey("m", "W"), rec)
	tc.RegisterNewtypeADT(NominalKey("m", "W"))
	return tc
}

func TestNominalHead(t *testing.T) {
	tc := aliasTC()
	u := tc.aliasUnifier()
	hue := &TCon{Name: "Hue", Module: "m"}
	cases := []struct {
		name string
		t    Type
		want string // KeyOf the head, "" for none
	}{
		{"ADT", &TCon{Name: "Color", Module: "m"}, "m::Color"},
		{"alias", hue, "m::Color"},
		{"alias of an alias", &TCon{Name: "Tint", Module: "m"}, "m::Color"},
		{"alias of an applied ADT", &TCon{Name: "IntOpt", Module: "m"}, "std/option::Option"},
		{"applied parameterised alias", &TApp{Constructor: &TCon{Name: "Opt", Module: "m"}, Args: []Type{hue}}, "std/option::Option"},
		{"record newtype stays nominal", &TCon{Name: "W", Module: "m"}, "m::W"},
		{"alias of a record has no head", &TCon{Name: "P", Module: "m"}, ""},
		{"type variable", &TVar2{Name: "a", Kind: Star}, ""},
	}
	for _, c := range cases {
		got := ""
		if h := u.nominalHead(c.t); h != nil {
			got = h.KeyOf()
		}
		if got != c.want {
			t.Errorf("%s: head %q, want %q", c.name, got, c.want)
		}
	}
	// No unifier expands nothing.
	var none *Unifier
	if h := none.nominalHead(hue); h == nil || h.Name != "Hue" {
		t.Errorf("nil unifier: head %v, want Hue unexpanded", h)
	}
	// The checker-level name: an ADT's name, never a primitive's.
	if got := tc.scrutineeADTName(hue); got != "Color" {
		t.Errorf("scrutineeADTName(Hue) = %q, want Color", got)
	}
	tc.RegisterTypeAlias(NominalKey("m", "Age"), TInt)
	for _, nonADT := range []Type{&TCon{Name: "Age", Module: "m"}, &TCon{Name: "P", Module: "m"}, &TVar2{Name: "a", Kind: Star}} {
		if got := tc.scrutineeADTName(nonADT); got != "" {
			t.Errorf("scrutineeADTName(%s) = %q, want \"\"", nonADT, got)
		}
	}
	if got := writtenAlias(hue, "Color"); got != "Hue" {
		t.Errorf("writtenAlias(Hue, Color) = %q", got)
	}
	if got := writtenAlias(&TCon{Name: "Color", Module: "m"}, "Color"); got != "" {
		t.Errorf("writtenAlias(Color, Color) = %q, want \"\"", got)
	}
}

func TestForeignConstructorErrorNamesTheAlias(t *testing.T) {
	plain := NewMatchForeignConstructorError("Some", "Option", "Color", "", []string{"Some", "None"}, []string{"Red", "Blue"}, nil)
	if !strings.HasPrefix(plain.Message, "match arm constructor 'Some' belongs to ADT 'Option', not 'Color' (the scrutinee's type).\n  Color's constructors are: Blue, Red") {
		t.Errorf("without an alias: %q", plain.Message)
	}
	aliased := NewMatchForeignConstructorError("Some", "Option", "Color", "Hue", []string{"Some", "None"}, []string{"Red", "Blue"}, nil)
	if !strings.HasPrefix(aliased.Message, "match arm constructor 'Some' belongs to ADT 'Option', not 'Color' (the scrutinee's type, written as its alias Hue).") {
		t.Errorf("with an alias: %q", aliased.Message)
	}
}

// A constructor outside direct scope (matched by name, #1478 ladder) is
// checked against the scrutinee's ADT after alias expansion, and an unknown
// one is answered with that ADT's constructors.
func TestCtorScopeThroughAlias(t *testing.T) {
	tc := aliasTC()
	tc.SetDiagnosticConstructorTypes(map[string]string{"Red": "Color", "Green": "Color", "Blue": "Color", "Calm": "Mood"})
	hue := &TCon{Name: "Hue", Module: "m"}
	if err := tc.checkCtorPatternScope(&core.ConstructorPattern{Name: "Red"}, hue); err != nil {
		t.Errorf("Red on a Hue: %v", err)
	}
	err := tc.checkCtorPatternScope(&core.ConstructorPattern{Name: "Calm"}, hue)
	var tce *TypeCheckError
	if !errors.As(err, &tce) || tce.Kind != MatchForeignConstructorError ||
		!strings.Contains(tce.Message, "belongs to ADT 'Mood', not 'Color' (the scrutinee's type, written as its alias Hue)") {
		t.Errorf("Calm on a Hue: %v", err)
	}
	err = tc.checkCtorPatternScope(&core.ConstructorPattern{Name: "Gren"}, hue)
	if !errors.As(err, &tce) || tce.Kind != MatchUnknownConstructorError ||
		!strings.HasPrefix(tce.Suggestion, "did you mean 'Green'? Color's constructors are: Blue, Green, Red") {
		t.Errorf("Gren on a Hue: %v (suggestion %q)", err, tce.Suggestion)
	}
}

// Exhaustiveness on an aliased scrutinee: by-name constructors resolve to the
// alias's ADT, the message names both, and each resolved by-name pattern is
// stamped with its ADT so the back ends need not resolve the alias.
func TestVerifyMatchThroughAliasStampsByNamePatterns(t *testing.T) {
	tc := aliasTC()
	hue := &TCon{Name: "Hue", Module: "m"}
	byName := func(name string) (core.MatchArm, *core.ConstructorPattern) {
		p := &core.ConstructorPattern{Name: name}
		tc.recordCtorPatternType(p, hue)
		return core.MatchArm{Pattern: p, Body: &core.Lit{}}, p
	}
	red, pRed := byName("Red")
	green, pGreen := byName("Green")
	blue, _ := byName("Blue")
	full := matchOn(1, 10, hue, tc, red, green, blue)
	if _, err := tc.VerifyMatchExhaustiveness(full); err != nil {
		t.Fatalf("exhaustive match on Hue: %v", err)
	}
	if pRed.TypeName != "Color" || pRed.TypeModule != "m" || pGreen.TypeName != "Color" {
		t.Errorf("stamps = %s/%s, %s/%s; want m/Color", pRed.TypeModule, pRed.TypeName, pGreen.TypeModule, pGreen.TypeName)
	}

	red2, _ := byName("Red")
	partial := matchOn(2, 20, hue, tc, red2)
	_, err := tc.VerifyMatchExhaustiveness(partial)
	var errs MatchErrors
	if !errors.As(err, &errs) || len(errs) != 1 || errs[0].Message != "non-exhaustive match on Hue (= Color) — no arm matches: Blue, Green" {
		t.Errorf("partial match on Hue: %v", err)
	}

	// A pattern the checker never saw keeps no stamp: its ADT is a guess (here
	// the only loaded ADT declaring Blue).
	unseen := &core.ConstructorPattern{Name: "Blue"}
	m := matchOn(3, 30, &TCon{Name: "Color", Module: "m"}, tc, core.MatchArm{Pattern: unseen, Body: &core.Lit{}}, wildArm())
	if _, err := tc.VerifyMatchExhaustiveness(m); err != nil {
		t.Fatalf("unseen pattern: %v", err)
	}
	if unseen.TypeName != "" {
		t.Errorf("an unseen pattern was stamped %q", unseen.TypeName)
	}
}
