package iface

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-IFACE-COMPACT-ADT-FIELDS: the compact interface must carry constructor field
// signatures and render record types, instead of bare names + "<*types.TRecord>".

func identCanon() func(string) string {
	return func(s string) string { return s }
}

func TestRenderConstructor_RecordFields(t *testing.T) {
	rec := &types.TRecord{Fields: map[string]types.Type{
		"level": &types.TCon{Name: "int"},
		"text":  &types.TCon{Name: "string"},
	}}
	got := renderConstructor("HeadingBlock", []types.Type{rec})
	want := "HeadingBlock({level: int, text: string})"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderConstructor_Nullary(t *testing.T) {
	if got := renderConstructor("None", nil); got != "None" {
		t.Errorf("got %q, want None", got)
	}
}

func TestRenderConstructor_Positional(t *testing.T) {
	got := renderConstructor("Pair", []types.Type{
		&types.TCon{Name: "int"}, &types.TCon{Name: "string"},
	})
	if got != "Pair(int, string)" {
		t.Errorf("got %q, want Pair(int, string)", got)
	}
}

// M-IFACE-RECORD-FIELDS: a record type alias must render its fields, so an agent can build/
// destructure it from the compact iface instead of cat-ing the source.
func TestRenderTypeAlias_Record(t *testing.T) {
	rec := &types.TRecord{Fields: map[string]types.Type{
		"colSpan": &types.TCon{Name: "int"},
		"merged":  &types.TCon{Name: "bool"},
		"text":    &types.TCon{Name: "string"},
	}}
	got := renderTypeAlias(rec)
	want := "{colSpan: int, merged: bool, text: string}"
	if got != want {
		t.Errorf("renderTypeAlias got %q, want %q", got, want)
	}
}

func TestFormatTypeCanonical_RecordNoLeak(t *testing.T) {
	rec := &types.TRecord{Fields: map[string]types.Type{"x": &types.TCon{Name: "float"}}}
	got := formatTypeCanonical(rec, identCanon())
	if strings.Contains(got, "<*") {
		t.Fatalf("record leaked a Go internal type: %q", got)
	}
	if got != "{x: float}" {
		t.Errorf("got %q, want {x: float}", got)
	}
}

func TestFormatTypeCanonical_NominalRecordRendersName(t *testing.T) {
	rec := &types.TRecord{
		TypeName: "TableCell",
		Fields:   map[string]types.Type{"v": &types.TCon{Name: "string"}},
	}
	if got := formatTypeCanonical(rec, identCanon()); got != "TableCell" {
		t.Errorf("got %q, want TableCell", got)
	}
}

func TestFormatTypeCanonical_OpenRow(t *testing.T) {
	rec := &types.TRecord{
		Fields: map[string]types.Type{"x": &types.TCon{Name: "int"}},
		Row:    &types.Row{Labels: map[string]types.Type{}, Tail: &types.RowVar{Name: "r"}},
	}
	got := formatTypeCanonical(rec, identCanon())
	if !strings.Contains(got, "...r") {
		t.Errorf("open record should render an open tail; got %q", got)
	}
}

// A self-referential record must never hang the renderer (depth guard). If the guard
// were missing this test would time out rather than fail — that is the signal.
func TestFormatTypeCanonical_CycleSafe(t *testing.T) {
	rec := &types.TRecord{Fields: map[string]types.Type{}}
	rec.Fields["self"] = rec // cycle
	got := formatTypeCanonical(rec, identCanon())
	if !strings.Contains(got, "_") {
		t.Errorf("expected depth-guard marker '_'; got %q", got)
	}
}
