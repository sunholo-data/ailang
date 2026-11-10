package types

import (
	"slices"
	"testing"
)

// isNewerScheme builds ∀a r. ({ver: a | r}, {ver: a | r}) -> bool ! {IO | e}
// with the given raw names, as the checker would export std/sem.is_newer.
func isNewerScheme(a, r, e string) *Scheme {
	rec := func() Type {
		return &TRecordOpen{
			Fields: map[string]Type{"ver": &TVar2{Name: a, Kind: Star}, "id": TString},
			Row:    &RowVar{Name: r, Kind: RecordRow},
		}
	}
	return &Scheme{
		TypeVars: []string{a},
		RowVars:  []string{e, r},
		Type: &TFunc2{
			Params:    []Type{rec(), rec()},
			Return:    TBool,
			EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{"IO": TUnit}, Tail: &RowVar{Name: e, Kind: EffectRow}},
		},
	}
}

// α-equivalent schemes canonicalise to the same value, whatever raw names (and
// therefore whatever map order or fresh counter) produced them (#2076).
func TestCanonicalizeScheme_AlphaEquivalentSchemesAgree(t *testing.T) {
	a, freeA := CanonicalizeScheme(isNewerScheme("α74", "ρ9", "ε3"))
	b, freeB := CanonicalizeScheme(isNewerScheme("α1", "r", "e"))
	if len(freeA) != 0 || len(freeB) != 0 {
		t.Fatalf("closed schemes reported free variables: %v %v", freeA, freeB)
	}
	if a.String() != b.String() {
		t.Fatalf("α-equivalent schemes canonicalise differently:\n  %s\n  %s", a, b)
	}
	if !slices.Equal(a.TypeVars, []string{"'t0"}) || !slices.Equal(a.RowVars, []string{"'r0", "'e0"}) {
		t.Fatalf("quantifiers not canonical in first-appearance order: type=%v row=%v", a.TypeVars, a.RowVars)
	}
	fn := a.Type.(*TFunc2)
	if fn.EffectRow.Tail.Name != "'e0" || !fn.EffectRow.Tail.Kind.Equals(EffectRow) {
		t.Fatalf("effect tail not renamed with its kind: %v", fn.EffectRow.Tail)
	}
	if fn.EffectRow.Labels["IO"] == nil {
		t.Fatal("effect labels lost by renaming")
	}
}

// T5b: a variable that is not quantified is reported, and spelled in its own
// namespace so it can never pass for a quantified one or collide with a fresh αN.
func TestCanonicalizeScheme_ReportsFreeVariables(t *testing.T) {
	s := isNewerScheme("α9", "r", "e")
	s.TypeVars = nil // α9 left free, as std/sem.is_newer was exported before #2157
	s.RowVars = []string{"e"}
	c, free := CanonicalizeScheme(s)
	if !slices.Equal(free, []string{"α9", "r"}) {
		t.Fatalf("free = %v, want [α9 r]", free)
	}
	if len(c.TypeVars) != 0 || !slices.Equal(c.RowVars, []string{"'e0"}) {
		t.Fatalf("free variables were quantified: type=%v row=%v", c.TypeVars, c.RowVars)
	}
	if got := c.Type.(*TFunc2).Params[0].(*TRecordOpen).Fields["ver"].String(); got != "'f0" {
		t.Fatalf("free type variable spelled %q, want 'f0", got)
	}
}

// Renaming preserves everything that is not a variable name.
func TestCanonicalizeScheme_PreservesStructure(t *testing.T) {
	budget := 3
	s := &Scheme{
		TypeVars: []string{"β"},
		Type: &TFunc2{
			ImplicitEffectContract: true,
			Params:                 []Type{&TRecord{Fields: map[string]Type{"x": &TVar2{Name: "β", Kind: Star}}, TypeName: "Point"}},
			Return:                 &TLabelled{Inner: &TVar2{Name: "β", Kind: Star}, L: LabelBottom()},
			EffectRow:              &Row{Kind: EffectRow, Labels: map[string]Type{"Net": TUnit}, Budgets: map[string]*int{"Net": &budget}},
		},
	}
	c, _ := CanonicalizeScheme(s)
	fn := c.Type.(*TFunc2)
	if !fn.ImplicitEffectContract {
		t.Error("effect-contract flag lost")
	}
	if rec := fn.Params[0].(*TRecord); rec.TypeName != "Point" || rec.Fields["x"].String() != "'t0" {
		t.Errorf("record not renamed faithfully: %#v", rec)
	}
	if _, ok := fn.Return.(*TLabelled); !ok {
		t.Errorf("label wrapper lost: %T", fn.Return)
	}
	if fn.EffectRow.Budgets["Net"] == nil || *fn.EffectRow.Budgets["Net"] != 3 {
		t.Error("effect budget lost")
	}
	if s.Type.(*TFunc2).Params[0].(*TRecord).Fields["x"].String() != "β" {
		t.Error("CanonicalizeScheme mutated its input")
	}
}

// #2157: a quantified record row is freshened as a RECORD row, not as an effect row.
func TestInstantiate_RecordRowKeepsItsKind(t *testing.T) {
	s, _ := CanonicalizeScheme(isNewerScheme("α1", "ρ2", "ε3"))
	kinds := map[string]Kind{}
	n := 0
	inst := s.Instantiate(func(k Kind) Type {
		n++
		name := []string{"x", "y", "z"}[n-1]
		kinds[name] = k
		if k.Equals(Star) {
			return &TVar2{Name: name, Kind: k}
		}
		return &RowVar{Name: name, Kind: k}
	})
	row := inst.(*TFunc2).Params[0].(*TRecordOpen).Row.(*RowVar)
	if !kinds[row.Name].Equals(RecordRow) {
		t.Fatalf("record row instantiated with kind %v", kinds[row.Name])
	}
	if tail := inst.(*TFunc2).EffectRow.Tail; !kinds[tail.Name].Equals(EffectRow) {
		t.Fatalf("effect row instantiated with kind %v", kinds[tail.Name])
	}
}

// #2157: generalisation sees variables under open records (the old collector
// skipped TRecordOpen, so `ver`'s type stayed free in exported schemes).
func TestGeneralizableVars_DescendsIntoOpenRecords(t *testing.T) {
	typ := isNewerScheme("α74", "ρ5", "ε1").Type
	tv, rows := generalizableVars(typ)
	if !slices.Equal(tv, []string{"α74"}) || !slices.Equal(rows, []string{"ρ5"}) {
		t.Fatalf("generalizableVars = %v, %v; want [α74], [ρ5]", tv, rows)
	}
}
