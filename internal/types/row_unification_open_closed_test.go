package types

import (
	"fmt"
	"strings"
	"testing"
)

// An open row means at least its listed labels. Its tail can accept extra
// closed-side labels, but cannot erase labels that the closed side forbids.
func TestRowUnification_OpenClosedOwnLabels(t *testing.T) {
	for _, kind := range []Kind{EffectRow, RecordRow} {
		for _, reverse := range []bool{false, true} {
			for _, tt := range []struct {
				name         string
				open, closed []string
				reject       bool
			}{
				{"cannot erase only label", []string{"IO"}, nil, true},
				{"cannot erase one of several", []string{"IO", "Net"}, []string{"Net"}, true},
				{"cannot exchange distinct labels", []string{"IO"}, []string{"FS"}, true},
				{"matching labels", []string{"IO"}, []string{"IO"}, false},
				{"tail accepts closed extras", []string{"IO"}, []string{"IO", "Net"}, false},
				{"empty open accepts labels", nil, []string{"IO"}, false},
				{"both empty", nil, nil, false},
			} {
				t.Run(fmt.Sprintf("%s/reverse=%v/%s", kind, reverse, tt.name), func(t *testing.T) {
					row := func(labels []string) *Row {
						m := make(map[string]Type)
						for _, l := range labels {
							m[l] = TUnit
						}
						return &Row{Kind: kind, Labels: m}
					}
					open, closed := row(tt.open), row(tt.closed)
					open.Tail = &RowVar{Name: "e", Kind: kind}
					a, b := open, closed
					if reverse {
						a, b = b, a
					}
					sub := Substitution{}
					got, err := NewRowUnifier().UnifyRows(a, b, sub)
					if tt.reject {
						if err == nil || !strings.Contains(err.Error(), "IO") || !strings.Contains(err.Error(), "closed row") {
							t.Fatalf("must reject dropped IO: sub=%v error=%v", got, err)
						}
						word := "effect"
						if kind.Equals(RecordRow) {
							word = "fields"
						}
						if !strings.Contains(err.Error(), word) {
							t.Fatalf("diagnostic must name %s: %v", word, err)
						}
						if _, bound := sub["e"]; bound {
							t.Fatal("rejected tail must not be bound")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					bound, ok := got["e"].(*Row)
					if !ok || bound.Tail != nil {
						t.Fatalf("tail must close: %v", got)
					}
					for _, label := range tt.closed {
						_, own := open.Labels[label]
						_, extra := bound.Labels[label]
						if extra == own {
							t.Fatalf("tail must contain exactly closed extras: open=%v bound=%v", open, bound)
						}
					}
				})
			}
		}
	}
}

func TestRowUnification_OpenClosedAfterTailSubstitution(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		open := &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "outer", Kind: EffectRow}}
		closed := &Row{Kind: EffectRow, Labels: map[string]Type{}}
		sub := Substitution{"outer": &Row{Kind: EffectRow, Labels: map[string]Type{"IO": TUnit}, Tail: &RowVar{Name: "inner", Kind: EffectRow}}}
		a, b := open, closed
		if reverse {
			a, b = b, a
		}
		if _, err := NewRowUnifier().UnifyRows(a, b, sub); err == nil {
			t.Fatal("substituted open-row labels must not disappear")
		}
	}
}
