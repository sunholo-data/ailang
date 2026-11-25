package smt

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// The type checker refuses a non-exhaustive list match, so no fixture reaches
// these; the encoder's own coverage check is what lets it take the last arm
// untested, and must not credit a pattern with lengths it does not match.
func TestListCoverage(t *testing.T) {
	activeADTContext = &parametricContext{adts: map[string][]ADTVariant{}}
	defer func() { activeADTContext = nil }()
	v := func(n string) core.CorePattern { return &core.VarPattern{Name: n} }
	w := &core.WildcardPattern{}
	lit := &core.LitPattern{Value: int64(1)}
	list := func(tail core.CorePattern, els ...core.CorePattern) core.CorePattern {
		lp := &core.ListPattern{Elements: els}
		if tail != nil {
			lp.Tail = &tail
		}
		return lp
	}
	empty := list(nil)
	cases := []struct {
		name string
		arms []core.CorePattern
		want bool
	}{
		{"empty and cons", []core.CorePattern{empty, list(v("r"), v("x"))}, true},
		{"cons alone", []core.CorePattern{list(v("r"), v("x"))}, false},
		{"empty alone", []core.CorePattern{empty}, false},
		{"fixed lengths then wildcard", []core.CorePattern{list(nil, v("a"), v("b")), w}, true},
		{"0, 1, 2 exactly", []core.CorePattern{empty, list(nil, v("a")), list(nil, v("a"), v("b"))}, false},
		{"0, 1 and 2+", []core.CorePattern{empty, list(nil, v("a")), list(v("r"), v("a"), v("b"))}, true},
		{"literal head", []core.CorePattern{empty, list(v("r"), lit)}, false},
		{"literal head then cons", []core.CorePattern{empty, list(v("r"), lit), list(v("r"), w)}, true},
		{"refutable tail", []core.CorePattern{empty, list(empty, v("x"))}, false},
		{"refutable tail and the rest", []core.CorePattern{empty, list(empty, v("x")), list(nil, v("x")), list(v("r"), v("x"), v("y"))}, true},
	}
	for _, c := range cases {
		rows := make([][]core.CorePattern, len(c.arms))
		for i, a := range c.arms {
			rows[i] = []core.CorePattern{a}
		}
		if got := patternCoverage(rows, []string{"(Seq Int)"}, 0); got != c.want {
			t.Errorf("%s: coverage %v, want %v", c.name, got, c.want)
		}
	}
}

// The encoder takes the last arm untested only when coverage holds: a match
// the checker would not let through is a skip, never an ite whose fall-through
// value is invented. The exhaustive one is the length-and-element chain.
func TestListMatchEncoding(t *testing.T) {
	activeADTContext = &parametricContext{adts: map[string][]ADTVariant{}, resultSort: "(Seq Int)"}
	defer func() { activeADTContext = nil }()
	var tail core.CorePattern = &core.VarPattern{Name: "rest"}
	cons := &core.ListPattern{Elements: []core.CorePattern{&core.VarPattern{Name: "x"}}, Tail: &tail}
	match := func(arms ...core.MatchArm) *core.Match {
		return &core.Match{Scrutinee: &core.Var{Name: "result"}, Arms: arms}
	}
	x := &core.Var{Name: "x"}
	got, err := EncodeExpr(match(
		core.MatchArm{Pattern: &core.ListPattern{}, Body: &core.Lit{Kind: core.IntLit, Value: int64(0)}},
		core.MatchArm{Pattern: cons, Body: x},
	))
	want := "(ite (= (seq.len result) 0) 0 (let (($l_x (seq.nth result 0)) ($l_rest (seq.extract result 1 (- (seq.len result) 1)))) $l_x))"
	if err != nil || got != want {
		t.Fatalf("exhaustive: %s %v\nwant %s", got, err, want)
	}
	_, err = EncodeExpr(match(core.MatchArm{Pattern: cons, Body: x}))
	if !errors.Is(err, ErrUnresolvableTypes) || !strings.Contains(err.Error(), "not provably exhaustive") {
		t.Fatalf("no empty arm: err = %v, want a not-exhaustive skip", err)
	}
}
