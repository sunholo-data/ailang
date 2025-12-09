package types

import (
	"testing"
)

// TestTRecord2Unification tests TRecord2 unification cases
func TestTRecord2Unification(t *testing.T) {
	u := NewUnifier()

	tests := []struct {
		name    string
		t1      Type
		t2      Type
		wantErr bool
	}{
		{
			name:    "empty records unify",
			t1:      &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{}}},
			t2:      &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{}}},
			wantErr: false,
		},
		{
			name: "closed records with same fields unify",
			t1: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   nil,
			}},
			t2: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   nil,
			}},
			wantErr: false,
		},
		{
			name: "closed records with different fields fail",
			t1: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   nil,
			}},
			t2: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"y": &TCon{Name: "Int"}},
				Tail:   nil,
			}},
			wantErr: true,
		},
		{
			name: "open record unifies with closed record (subsumption)",
			t1: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   &RowVar{Name: "r", Kind: RecordRow},
			}},
			t2: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}, "y": &TCon{Name: "String"}},
				Tail:   nil,
			}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := u.Unify(tt.t1, tt.t2, make(Substitution))
			if (err != nil) != tt.wantErr {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTRecordConversion tests conversion between TRecord and TRecord2
func TestTRecordConversion(t *testing.T) {
	tests := []struct {
		name string
		old  *TRecord
	}{
		{
			name: "simple closed record",
			old: &TRecord{
				Fields: map[string]Type{"x": &TCon{Name: "Int"}},
				Row:    nil,
			},
		},
		{
			name: "open record with row var",
			old: &TRecord{
				Fields: map[string]Type{"x": &TCon{Name: "Int"}},
				Row:    &RowVar{Name: "r", Kind: RecordRow},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to TRecord2 and back
			new := TRecordToTRecord2(tt.old)
			if new == nil {
				t.Fatal("TRecordToTRecord2 returned nil")
			}

			back := TRecord2ToTRecord(new)
			if back == nil {
				t.Fatal("TRecord2ToTRecord returned nil")
			}

			// Check fields match
			if len(back.Fields) != len(tt.old.Fields) {
				t.Errorf("field count mismatch: got %d, want %d", len(back.Fields), len(tt.old.Fields))
			}

			for name, typ := range tt.old.Fields {
				if backTyp, ok := back.Fields[name]; !ok {
					t.Errorf("field %s missing in converted record", name)
				} else if !typ.Equals(backTyp) {
					t.Errorf("field %s type mismatch: got %v, want %v", name, backTyp, typ)
				}
			}
		})
	}
}

// TestRowOccursCheck tests the occurs check in row unification
func TestRowOccursCheck(t *testing.T) {
	u := NewUnifier()

	// Create a row variable
	rowVar := &RowVar{Name: "r", Kind: RecordRow}

	// Try to unify r with {x: int | r} - should fail occurs check
	row1 := &Row{
		Kind:   RecordRow,
		Labels: map[string]Type{},
		Tail:   rowVar,
	}

	row2 := &Row{
		Kind:   RecordRow,
		Labels: map[string]Type{"x": &TCon{Name: "Int"}},
		Tail:   rowVar, // Same row var - occurs!
	}

	_, err := u.unifyRows(row1, row2, make(Substitution))
	if err == nil {
		t.Error("expected occurs check to fail, but it succeeded")
	}
}

// TestOpenClosedInteractions tests Day 3 open-closed cases
func TestOpenClosedInteractions(t *testing.T) {
	u := NewUnifier()

	tests := []struct {
		name    string
		t1      Type
		t2      Type
		wantErr bool
	}{
		{
			name: "{x:int | ρ} ~ {x:int,y:bool} succeeds (open-closed)",
			t1: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   &RowVar{Name: "r", Kind: RecordRow},
			}},
			t2: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}, "y": &TCon{Name: "Bool"}},
				Tail:   nil,
			}},
			wantErr: false,
		},
		{
			name: "{x:int} ~ {x:int | ρ} succeeds (closed-open, common fields)",
			t1: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   nil,
			}},
			t2: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   &RowVar{Name: "r", Kind: RecordRow},
			}},
			wantErr: false,
		},
		{
			name: "order independence: {x:int,y:bool} ~ {y:bool,x:int}",
			t1: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}, "y": &TCon{Name: "Bool"}},
				Tail:   nil,
			}},
			t2: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"y": &TCon{Name: "Bool"}, "x": &TCon{Name: "Int"}},
				Tail:   nil,
			}},
			wantErr: false,
		},
		{
			name: "nested openness: {u:{id:int | ρ}} ~ {u:{id:int,email:string}}",
			t1: &TRecord2{Row: &Row{
				Kind: RecordRow,
				Labels: map[string]Type{
					"u": &TRecord2{Row: &Row{
						Kind:   RecordRow,
						Labels: map[string]Type{"id": &TCon{Name: "Int"}},
						Tail:   &RowVar{Name: "r", Kind: RecordRow},
					}},
				},
				Tail: nil,
			}},
			t2: &TRecord2{Row: &Row{
				Kind: RecordRow,
				Labels: map[string]Type{
					"u": &TRecord2{Row: &Row{
						Kind: RecordRow,
						Labels: map[string]Type{
							"id":    &TCon{Name: "Int"},
							"email": &TCon{Name: "String"},
						},
						Tail: nil,
					}},
				},
				Tail: nil,
			}},
			wantErr: false,
		},
		{
			name: "field type mismatch fails: {x:int} ~ {x:bool}",
			t1: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   nil,
			}},
			t2: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Bool"}},
				Tail:   nil,
			}},
			wantErr: true,
		},
		{
			name: "missing field in closed fails: {x:int,y:bool} ~ {x:int}",
			t1: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}, "y": &TCon{Name: "Bool"}},
				Tail:   nil,
			}},
			t2: &TRecord2{Row: &Row{
				Kind:   RecordRow,
				Labels: map[string]Type{"x": &TCon{Name: "Int"}},
				Tail:   nil,
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := u.Unify(tt.t1, tt.t2, make(Substitution))
			if (err != nil) != tt.wantErr {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTypeAliasExpansion tests M-BUGFIX: type alias expansion during unification
// This fixes: "cannot unify type constructor Coord with *types.TRecord"
func TestTypeAliasExpansion(t *testing.T) {
	// Create unifier with alias environment
	// Simulates: type Coord = { x: int, y: int }
	aliases := map[string]Type{
		"Coord": &TRecord{
			Fields: map[string]Type{
				"x": &TCon{Name: "int"},
				"y": &TCon{Name: "int"},
			},
		},
	}
	u := NewUnifierWithAliases(aliases)

	tests := []struct {
		name    string
		t1      Type
		t2      Type
		wantErr bool
	}{
		{
			name: "TCon(Coord) ~ TRecord{x,y} succeeds with alias expansion",
			t1:   &TCon{Name: "Coord"},
			t2: &TRecord{
				Fields: map[string]Type{
					"x": &TCon{Name: "int"},
					"y": &TCon{Name: "int"},
				},
			},
			wantErr: false,
		},
		{
			name: "TRecord{x,y} ~ TCon(Coord) succeeds with alias expansion (reversed)",
			t1: &TRecord{
				Fields: map[string]Type{
					"x": &TCon{Name: "int"},
					"y": &TCon{Name: "int"},
				},
			},
			t2:      &TCon{Name: "Coord"},
			wantErr: false,
		},
		{
			name: "TCon(Coord) ~ TRecord{x,y,z} fails (extra field)",
			t1:   &TCon{Name: "Coord"},
			t2: &TRecord{
				Fields: map[string]Type{
					"x": &TCon{Name: "int"},
					"y": &TCon{Name: "int"},
					"z": &TCon{Name: "int"},
				},
			},
			wantErr: true,
		},
		{
			name: "TCon(Coord) ~ TRecord{x:int} fails (missing field)",
			t1:   &TCon{Name: "Coord"},
			t2: &TRecord{
				Fields: map[string]Type{
					"x": &TCon{Name: "int"},
				},
			},
			wantErr: true,
		},
		{
			name:    "Unregistered alias TCon(Unknown) ~ TRecord fails",
			t1:      &TCon{Name: "Unknown"},
			t2:      &TRecord{Fields: map[string]Type{"x": &TCon{Name: "int"}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := u.Unify(tt.t1, tt.t2, make(Substitution))
			if (err != nil) != tt.wantErr {
				t.Errorf("Unify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTypeAliasExpansion_NoAliases tests that unification without aliases still works
func TestTypeAliasExpansion_NoAliases(t *testing.T) {
	// Standard unifier without alias expansion
	u := NewUnifier()

	// TCon(Coord) ~ TRecord should fail without aliases
	t1 := &TCon{Name: "Coord"}
	t2 := &TRecord{
		Fields: map[string]Type{
			"x": &TCon{Name: "int"},
			"y": &TCon{Name: "int"},
		},
	}

	_, err := u.Unify(t1, t2, make(Substitution))
	if err == nil {
		t.Error("Expected unification to fail without alias expansion")
	}
}
