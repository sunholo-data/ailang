package types

import "testing"

func latentArrow(tail string, labels ...string) *TFunc2 {
	row := &Row{Kind: EffectRow, Labels: map[string]Type{}}
	for _, l := range labels {
		row.Labels[l] = TUnit
	}
	if tail != "" {
		row.Tail = &RowVar{Name: tail, Kind: EffectRow}
	}
	return &TFunc2{Params: []Type{TInt}, EffectRow: row, Return: TInt, ImplicitEffectContract: true}
}

func arrowTails(arrows []*TFunc2) []string {
	var tails []string
	for _, fn := range arrows {
		if fn.EffectRow != nil && fn.EffectRow.Tail != nil {
			tails = append(tails, fn.EffectRow.Tail.Name)
		}
	}
	return tails
}

// AC6: every data position is reached; arrow params and returns are not.
func TestDataArrows_Positions(t *testing.T) {
	f := func(n string) *TFunc2 { return latentArrow(n) }
	tests := []struct {
		name string
		typ  Type
		want []string
	}{
		{"top", f("a"), []string{"a"}},
		{"list", &TList{Element: f("a")}, []string{"a"}},
		{"array", &TArray{Element: f("a")}, []string{"a"}},
		{"map", &TMap{Key: TString, Value: f("a")}, []string{"a"}},
		{"tuple", &TTuple{Elements: []Type{f("a"), TInt, f("b")}}, []string{"a", "b"}},
		{"app", &TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{f("a")}}, []string{"a"}},
		{"record_sorted", &TRecord{Fields: map[string]Type{"z": f("z"), "a": f("a")}}, []string{"a", "z"}},
		{"record_open", &TRecordOpen{Fields: map[string]Type{"f": f("a")}}, []string{"a"}},
		{"record2", &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"f": f("a")}}}, []string{"a"}},
		{"nested", &TList{Element: &TTuple{Elements: []Type{&TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{f("a")}}}}}, []string{"a"}},
		{"no_param_descent", &TFunc2{Params: []Type{f("p")}, EffectRow: EmptyEffectRow(), Return: f("r")}, nil},
		{"no_descent_inside_list", &TList{Element: &TFunc2{Params: []Type{&TList{Element: f("p")}}, Return: f("r")}}, nil},
		{"scalar", TInt, nil},
		{"nil", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := arrowTails(dataArrowsWith(tt.typ, nil))
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// AC6: alias heads are expanded at each level, and a recursive alias terminates.
func TestDataArrows_AliasExpansionAndCycles(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.RegisterTypeAlias("Cb", latentArrow("a"))
	tc.RegisterTypeAlias("R", &TRecord{Fields: map[string]Type{"f": &TCon{Name: "Cb"}}})
	tc.RegisterTypeAlias("Loop", &TList{Element: &TCon{Name: "Loop"}})

	if got := arrowTails(tc.dataArrows(&TList{Element: &TCon{Name: "R"}})); len(got) != 1 || got[0] != "a" {
		t.Fatalf("alias not expanded per level: %v", got)
	}
	if got := dataArrowsWith(&TCon{Name: "R"}, nil); len(got) != 0 {
		t.Fatalf("nil expander must not expand aliases: %v", got)
	}
	if got := tc.dataArrows(&TCon{Name: "Loop"}); len(got) != 0 {
		t.Fatalf("recursive alias: %v", got)
	}
	deep := Type(latentArrow("deep"))
	for i := 0; i <= maxDataArrowDepth+1; i++ {
		deep = &TList{Element: deep}
	}
	if got := dataArrowsWith(deep, nil); len(got) != 0 {
		t.Fatalf("depth cap not applied: %v", got)
	}
}

func TestLatentRow_ClosesOnlyConcreteTails(t *testing.T) {
	concrete := latentArrow("ε_annot1", "IO")
	concrete.ImplicitEffectContract, concrete.ConcreteEffectContract = false, true
	if row := latentRow(concrete); row.Tail != nil || row.Labels["IO"] == nil {
		t.Fatalf("concrete compatibility tail not closed: %v", row)
	}
	if concrete.EffectRow.Tail == nil {
		t.Fatal("latentRow mutated the arrow's own row")
	}
	implicit := latentArrow("ε_annot2")
	if row := latentRow(implicit); row.Tail == nil || row.Tail.Name != "ε_annot2" {
		t.Fatalf("implicit tail must stay open: %v", row)
	}
	rowPoly := latentArrow("e", "IO")
	rowPoly.ImplicitEffectContract = false
	if row := latentRow(rowPoly); row.Tail == nil {
		t.Fatalf("row-polymorphic tail must stay open: %v", row)
	}
	if row := latentRow(&TFunc2{Params: []Type{TInt}, Return: TInt}); row == nil || len(row.Labels) != 0 || row.Tail != nil {
		t.Fatalf("row-less arrow must cost nothing: %v", row)
	}
}

func TestNestedLatentRow_UnionOwnershipAndConflict(t *testing.T) {
	io, fs := latentArrow("", "IO"), latentArrow("", "FS")
	if row := nestedLatentRow([]*TFunc2{io, fs}, nil, nil); row == nil || row.Labels["IO"] == nil || row.Labels["FS"] == nil {
		t.Fatalf("union must keep every label (not first-wins): %v", row)
	}
	pure := nestedLatentRow([]*TFunc2{latentArrow(""), latentArrow("")}, nil, nil)
	if pure == nil || len(pure.Labels) != 0 || pure.Tail != nil {
		t.Fatalf("pure callbacks must publish an empty (non-nil) row: %v", pure)
	}
	if row := nestedLatentRow(nil, nil, nil); row != nil {
		t.Fatalf("no arrows must publish no row: %v", row)
	}
	two := []*TFunc2{latentArrow("e1"), latentArrow("e2")}
	if row := nestedLatentRow(two, nil, nil); row != nil {
		t.Fatalf("distinct unowned tails must be an explicit conflict, got %v", row)
	}
	if row := nestedLatentRow(two, nil, map[string]bool{"e1": true, "e2": true}); row == nil || row.Tail != nil {
		t.Fatalf("implicitly owned tails are the caller's caller's: %v", row)
	}
	if row := nestedLatentRow(two, nil, map[string]bool{"e1": true}); row == nil || row.Tail == nil || row.Tail.Name != "e2" {
		t.Fatalf("only the owned tail is dropped: %v", row)
	}
	zonk := func(r *Row) *Row {
		r = cloneApplicationRow(r)
		r.Tail = nil
		r.Labels["Net"] = TUnit
		return r
	}
	if row := nestedLatentRow(two, zonk, nil); row == nil || row.Labels["Net"] == nil {
		t.Fatalf("rows must pass through zonk before union: %v", row)
	}
}

// Site (a): a parameter that only CONTAINS an open arrow is a hand-off.
func TestResolvedCallbackMask_NestedPositions(t *testing.T) {
	tc := NewCoreTypeChecker()
	tc.RegisterTypeAlias("R", &TRecord{Fields: map[string]Type{"f": latentArrow("a")}})
	concrete := latentArrow("ε_annot9", "IO")
	concrete.ImplicitEffectContract, concrete.ConcreteEffectContract = false, true
	callee := &TFunc2{
		Params: []Type{
			&TTuple{Elements: []Type{latentArrow("a"), TInt}},
			&TList{Element: latentArrow("b")},
			&TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{latentArrow("c")}},
			&TCon{Name: "R"},
			&TList{Element: concrete},
			&TList{Element: TInt},
			latentArrow("top"),
		},
		EffectRow: EmptyEffectRow(),
		Return:    TInt,
	}
	want := []bool{true, true, true, true, false, false, true}
	got := tc.resolvedCallbackMask(callee)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mask = %v, want %v", got, want)
		}
	}
}
