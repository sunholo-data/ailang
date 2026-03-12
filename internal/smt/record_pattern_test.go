package smt

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// --- Record Pattern Encoding Tests ---

// TestEncodePattern_RecordPattern_Basic tests basic record pattern encoding:
// {x, y} with Point record type → (mk_Point x y)
func TestEncodePattern_RecordPattern_Basic(t *testing.T) {
	setupRecordTestContext()
	defer teardownRecordTestContext()

	pat := &core.RecordPattern{
		Fields: map[string]core.CorePattern{
			"x": &core.VarPattern{Name: "x"},
			"y": &core.VarPattern{Name: "y"},
		},
	}

	got, err := encodePattern(pat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(mk_Point x y)" {
		t.Errorf("got %q, want %q", got, "(mk_Point x y)")
	}
}

// TestEncodePattern_RecordPattern_ThreeFields tests alphabetical field ordering:
// {c, a, b} → (mk_Record_a_b_c a b c)
func TestEncodePattern_RecordPattern_ThreeFields(t *testing.T) {
	// Setup a 3-field record type
	activeRecordTypes = map[string]*RecordTypeInfo{
		"Record_a_b_c": {
			SortName:   "Record_a_b_c",
			CtorName:   "mk_Record_a_b_c",
			FieldNames: []string{"a", "b", "c"},
			FieldSorts: map[string]string{"a": "Int", "b": "Int", "c": "Int"},
		},
	}
	activeFieldSetToSort = map[string]string{
		"a,b,c": "Record_a_b_c",
	}
	defer teardownRecordTestContext()

	pat := &core.RecordPattern{
		Fields: map[string]core.CorePattern{
			"c": &core.VarPattern{Name: "c"},
			"a": &core.VarPattern{Name: "a"},
			"b": &core.VarPattern{Name: "b"},
		},
	}

	got, err := encodePattern(pat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(mk_Record_a_b_c a b c)" {
		t.Errorf("got %q, want %q", got, "(mk_Record_a_b_c a b c)")
	}
}

// TestEncodePattern_RecordPattern_WithWildcard tests record pattern with wildcard sub-patterns:
// {x: _, y: v} → (mk_Point _ v)
func TestEncodePattern_RecordPattern_WithWildcard(t *testing.T) {
	setupRecordTestContext()
	defer teardownRecordTestContext()

	pat := &core.RecordPattern{
		Fields: map[string]core.CorePattern{
			"x": &core.WildcardPattern{},
			"y": &core.VarPattern{Name: "v"},
		},
	}

	got, err := encodePattern(pat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(mk_Point _ v)" {
		t.Errorf("got %q, want %q", got, "(mk_Point _ v)")
	}
}

// TestEncodeMatch_RecordPattern tests a full match expression with record pattern:
// match p { {x, y} => x + y } → (match p (((mk_Point x y) (+ x y))))
func TestEncodeMatch_RecordPattern(t *testing.T) {
	setupRecordTestContext()
	defer teardownRecordTestContext()

	m := &core.Match{
		Scrutinee: &core.Var{Name: "p"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.RecordPattern{
					Fields: map[string]core.CorePattern{
						"x": &core.VarPattern{Name: "x"},
						"y": &core.VarPattern{Name: "y"},
					},
				},
				Body: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "add_Int"}},
					Args: []core.CoreExpr{
						&core.Var{Name: "x"},
						&core.Var{Name: "y"},
					},
				},
			},
		},
	}

	got, err := EncodeExpr(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(match p (((mk_Point x y) (+ x y))))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEncodePattern_NestedRecordPattern tests nested record patterns:
// {origin: {x, y}, radius: r} with a Circle record containing a Point
func TestEncodePattern_NestedRecordPattern(t *testing.T) {
	// Setup nested record types: Point{x, y} and Circle{origin: Point, radius: Int}
	activeRecordTypes = map[string]*RecordTypeInfo{
		"Point": {
			SortName:   "Point",
			CtorName:   "mk_Point",
			FieldNames: []string{"x", "y"},
			FieldSorts: map[string]string{"x": "Int", "y": "Int"},
		},
		"Circle": {
			SortName:   "Circle",
			CtorName:   "mk_Circle",
			FieldNames: []string{"origin", "radius"},
			FieldSorts: map[string]string{"origin": "Point", "radius": "Int"},
		},
	}
	activeFieldSetToSort = map[string]string{
		"x,y":           "Point",
		"origin,radius": "Circle",
	}
	defer teardownRecordTestContext()

	pat := &core.RecordPattern{
		Fields: map[string]core.CorePattern{
			"origin": &core.RecordPattern{
				Fields: map[string]core.CorePattern{
					"x": &core.VarPattern{Name: "ox"},
					"y": &core.VarPattern{Name: "oy"},
				},
			},
			"radius": &core.VarPattern{Name: "r"},
		},
	}

	got, err := encodePattern(pat)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "(mk_Circle (mk_Point ox oy) r)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestEncodePattern_RecordPattern_UnknownType tests that an error is returned
// when the record pattern's fields don't match any known record type.
func TestEncodePattern_RecordPattern_UnknownType(t *testing.T) {
	setupRecordTestContext()
	defer teardownRecordTestContext()

	pat := &core.RecordPattern{
		Fields: map[string]core.CorePattern{
			"foo": &core.VarPattern{Name: "foo"},
			"bar": &core.VarPattern{Name: "bar"},
		},
	}

	_, err := encodePattern(pat)
	if err == nil {
		t.Fatal("expected error for unknown record pattern, got nil")
	}
	if !strings.Contains(err.Error(), "unknown record type") {
		t.Errorf("expected 'unknown record type' error, got: %v", err)
	}
}

// TestEncodePattern_RecordPattern_NoActiveContext tests that an error is returned
// when no record context is active.
func TestEncodePattern_RecordPattern_NoActiveContext(t *testing.T) {
	// Ensure no active context
	activeRecordTypes = nil
	activeFieldSetToSort = nil

	pat := &core.RecordPattern{
		Fields: map[string]core.CorePattern{
			"x": &core.VarPattern{Name: "x"},
			"y": &core.VarPattern{Name: "y"},
		},
	}

	_, err := encodePattern(pat)
	if err == nil {
		t.Fatal("expected error when no record context is active, got nil")
	}
}

// TestPatternDepth_RecordPattern tests that patternDepth correctly handles
// RecordPattern (depth 1 for flat, depth 2 for nested constructor/record in fields).
func TestPatternDepth_RecordPattern(t *testing.T) {
	// Flat record pattern: depth 1
	flat := &core.RecordPattern{
		Fields: map[string]core.CorePattern{
			"x": &core.VarPattern{Name: "x"},
			"y": &core.VarPattern{Name: "y"},
		},
	}
	if d := patternDepth(flat); d != 1 {
		t.Errorf("flat RecordPattern depth = %d, want 1", d)
	}

	// Nested record pattern (record inside record): depth 2
	nested := &core.RecordPattern{
		Fields: map[string]core.CorePattern{
			"inner": &core.RecordPattern{
				Fields: map[string]core.CorePattern{
					"a": &core.VarPattern{Name: "a"},
				},
			},
			"val": &core.VarPattern{Name: "v"},
		},
	}
	if d := patternDepth(nested); d != 2 {
		t.Errorf("nested RecordPattern depth = %d, want 2", d)
	}
}
