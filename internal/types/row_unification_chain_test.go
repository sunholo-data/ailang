package types

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// A recursive SCC can bind a row through several principal-unifier remainders.
// Unifying its final call with the same row must resolve all of those aliases;
// rewriting an intermediate alias would disconnect the remaining calls.
func TestRowUnificationResolvesAliasChain(t *testing.T) {
	for _, kind := range []Kind{EffectRow, RecordRow} {
		t.Run(kind.String(), func(t *testing.T) {
			row := func(name string) *Row {
				return &Row{Kind: kind, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: kind}}
			}
			first, last := row("first"), row("last")
			sub := Substitution{}
			names := []string{"first", "a", "b", "c", "d", "last"}
			for i := 0; i < len(names)-1; i++ {
				next := row(names[i+1])
				sub[names[i]] = next
			}
			solved, err := NewRowUnifier().UnifyRows(first, last, sub)
			if err != nil {
				t.Fatal(err)
			}
			if _, changed := solved["last"]; changed {
				t.Fatalf("same resolved tail was rebound: %v", solved)
			}
			solved["last"] = &Row{Kind: kind, Labels: map[string]Type{}}
			for _, name := range names {
				got := zonkApplicationRow(solved, row(name))
				if got.Tail != nil || !got.Kind.Equals(kind) {
					t.Fatalf("%s disconnected from closed recursive row: %v", name, got)
				}
			}
		})
	}
}

func TestRowUnificationClosesMixedAliasChain(t *testing.T) {
	for _, kind := range []Kind{EffectRow, RecordRow} {
		t.Run(kind.String(), func(t *testing.T) {
			row := func(name string) *Row {
				return &Row{Kind: kind, Labels: map[string]Type{}, Tail: &RowVar{Name: name, Kind: kind}}
			}
			sub := Substitution{"a": row("b"), "b": row("c"), "c": &Row{Kind: kind, Labels: map[string]Type{"payload": TInt}, Tail: &RowVar{Name: "d", Kind: kind}}}
			_, err := NewRowUnifier().UnifyRows(row("a"), &Row{Kind: kind, Labels: map[string]Type{"payload": TInt}}, sub)
			if err != nil {
				t.Fatal(err)
			}
			got := zonkApplicationRow(sub, row("a"))
			if got.Tail != nil || got.Labels["payload"] != TInt || !got.Kind.Equals(kind) {
				t.Fatalf("closed row lost labels or remained open: %v", got)
			}
		})
	}
}

func TestRowUnificationAliasMetadataAndCycle(t *testing.T) {
	budget, minimum := 7, 2
	first := &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "a", Kind: EffectRow}}
	extension := &Row{Kind: EffectRow, Labels: map[string]Type{"IO": TUnit}, Tail: &RowVar{Name: "b", Kind: EffectRow}, Budgets: map[string]*int{"IO": &budget}, MinBudgets: map[string]*int{"IO": &minimum}, Params: map[string]map[string]string{"IO": {"mode": "replay"}}, Provenance: map[string]ast.Span{"IO": {Start: ast.Pos{File: "recursive.ail"}}}}
	sub := Substitution{"a": extension, "b": &RowVar{Name: "c", Kind: EffectRow}, "c": EmptyEffectRow()}
	resolved := NewRowUnifier().applySubToRow(sub, first)
	if resolved.Tail != nil || !resolved.Kind.Equals(EffectRow) || resolved.Labels["IO"] != TUnit || *resolved.Budgets["IO"] != budget || *resolved.MinBudgets["IO"] != minimum || resolved.Params["IO"]["mode"] != "replay" || resolved.Provenance["IO"].Start.File != "recursive.ail" {
		t.Fatalf("resolved row lost effect metadata: %#v", resolved)
	}
	*resolved.Budgets["IO"] = 100
	*resolved.MinBudgets["IO"] = 100
	resolved.Params["IO"]["mode"] = "changed"
	if extension.Params["IO"]["mode"] != "replay" || *extension.Budgets["IO"] != 7 || *extension.MinBudgets["IO"] != 2 || first.Tail.Name != "a" {
		t.Fatal("resolution mutated its source")
	}
	cycle := Substitution{"a": &RowVar{Name: "b", Kind: EffectRow}, "b": &RowVar{Name: "a", Kind: EffectRow}}
	if got := NewRowUnifier().applySubToRow(cycle, first); got.Tail == nil {
		t.Fatal("cyclic alias was silently treated as pure")
	}
}

func TestRowUnificationUnresolvedTailDetached(t *testing.T) {
	tail := &RowVar{Name: "b", Kind: EffectRow}
	sub := Substitution{"a": &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: tail}}
	resolved := NewRowUnifier().applySubToRow(sub, &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "a", Kind: EffectRow}})
	resolved.Tail.Name = "changed"
	if tail.Name != "b" {
		t.Fatal("resolution mutated the source alias")
	}
}
