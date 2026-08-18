package smt

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func TestSeqElemSort(t *testing.T) {
	tests := []struct {
		in       string
		wantElem string
		wantOK   bool
	}{
		{"(Seq String)", "String", true},
		{"(Seq Int)", "Int", true},
		{"(Seq Block)", "Block", true},
		// Nested: the element is itself a sequence sort, so the whole inner
		// term must come back, not just its head.
		{"(Seq (Seq String))", "(Seq String)", true},
		{"(Seq (Seq (Seq Int)))", "(Seq (Seq Int))", true},
		// Not sequence sorts — these must NOT yield an element sort, otherwise
		// a record sort would be mistaken for a sequence and the hint would
		// annotate an empty list with nonsense.
		{"Int", "", false},
		{"String", "", false},
		{"MdParseState", "", false},
		{"", "", false},
		{"(Seq )", "", false},
		{"(Array Int Int)", "", false},
	}
	for _, tc := range tests {
		gotElem, gotOK := seqElemSort(tc.in)
		if gotOK != tc.wantOK || gotElem != tc.wantElem {
			t.Errorf("seqElemSort(%q) = (%q, %v), want (%q, %v)",
				tc.in, gotElem, gotOK, tc.wantElem, tc.wantOK)
		}
	}
}

func TestEncodeListWithSortHint(t *testing.T) {
	empty := &core.List{}
	tests := []struct {
		name string
		list *core.List
		hint string
		want string
	}{
		{"empty at string", empty, "(Seq String)", "(as seq.empty (Seq String))"},
		{"empty at ADT", empty, "(Seq Block)", "(as seq.empty (Seq Block))"},
		{"empty at nested", empty, "(Seq (Seq String))", "(as seq.empty (Seq (Seq String)))"},
		// No usable hint ⇒ unchanged behaviour, so the fix can only ever
		// REPLACE an ill-sorted default, never introduce a new guess.
		{"empty, non-seq hint", empty, "MdParseState", "(as seq.empty (Seq Int))"},
		{"empty, no hint", empty, "", "(as seq.empty (Seq Int))"},
		// A non-empty literal whose single element is itself empty: the hint has
		// to reach one level down or the inner literal defaults to Int again.
		{
			"nested empty inside non-empty",
			&core.List{Elements: []core.CoreExpr{&core.List{}}},
			"(Seq (Seq String))",
			"(seq.unit (as seq.empty (Seq String)))",
		},
	}
	for _, tc := range tests {
		got, err := encodeListWithSortHint(tc.list, tc.hint)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestInlineEmptyListBindings(t *testing.T) {
	emptyList := func() core.CoreExpr { return &core.List{} }

	t.Run("inlines an empty-list binding into its use site", func(t *testing.T) {
		// ANF: let $t = [] in { items: $t }
		in := &core.Let{
			Name:  "$t",
			Value: emptyList(),
			Body:  &core.Record{Fields: map[string]core.CoreExpr{"items": &core.Var{Name: "$t"}}},
		}
		out := inlineEmptyListBindings(in)
		rec, ok := out.(*core.Record)
		if !ok {
			t.Fatalf("got %T, want the let to be gone and the record to surface", out)
		}
		if !isEmptyListLiteral(rec.Fields["items"]) {
			t.Fatalf("field 'items' = %T, want the empty list literal substituted in", rec.Fields["items"])
		}
	})

	t.Run("leaves a non-empty-list binding alone", func(t *testing.T) {
		// The rewrite is only sound because an empty list is a pure constant.
		// Anything else must keep its binding.
		in := &core.Let{
			Name:  "$t",
			Value: &core.List{Elements: []core.CoreExpr{&core.Lit{Kind: core.IntLit, Value: int64(1)}}},
			Body:  &core.Var{Name: "$t"},
		}
		if out := inlineEmptyListBindings(in); out != core.CoreExpr(in) {
			t.Fatalf("got %T, want the original let unchanged", out)
		}
	})

	t.Run("stops at a shadowing binder", func(t *testing.T) {
		// let $t = [] in (let $t = <int> in $t) — the inner $t is a different
		// variable, so substituting into it would change the program's meaning.
		inner := &core.Let{
			Name:  "$t",
			Value: &core.Lit{Kind: core.IntLit, Value: int64(7)},
			Body:  &core.Var{Name: "$t"},
		}
		out := inlineEmptyListBindings(&core.Let{Name: "$t", Value: emptyList(), Body: inner})
		got, ok := out.(*core.Let)
		if !ok {
			t.Fatalf("got %T, want the shadowing let to survive", out)
		}
		if v, ok := got.Body.(*core.Var); !ok || v.Name != "$t" {
			t.Fatalf("inner body = %#v, want the shadowed var untouched", got.Body)
		}
	})

	t.Run("non-let expressions pass through", func(t *testing.T) {
		e := &core.Var{Name: "x"}
		if out := inlineEmptyListBindings(e); out != core.CoreExpr(e) {
			t.Fatalf("got %T, want the expression returned as-is", out)
		}
	})
}
