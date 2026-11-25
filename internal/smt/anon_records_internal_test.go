package smt

import "testing"

// Across modules the ADT pass of each sees only its own records, so a
// collision shows as two declarations of one name with different bodies.
func TestConflictingRecordSorts(t *testing.T) {
	a := DeclareRecordDatatype("Record_desc_key", map[string]string{"key": "String", "desc": "String"})
	b := DeclareRecordDatatype("Record_desc_key", map[string]string{"key": "String", "desc": "Json"})
	c := DeclareRecordDatatype("Record_k_v", map[string]string{"k": "Int", "v": "Int"})
	got := conflictingRecordSorts([]string{a, c, a, c}, nil)
	if len(got) != 0 {
		t.Fatalf("identical redeclarations are not a conflict: %v", got)
	}
	got = conflictingRecordSorts([]string{a, c, b}, []string{"Record_x_y"})
	if !got["Record_desc_key"] || !got["Record_x_y"] || got["Record_k_v"] || len(got) != 2 {
		t.Fatalf("conflicts: %v", got)
	}
	needed := map[string]bool{"Record_x_y": true, "Record_desc_key": true, "Int": true}
	if n := firstNeededConflict(needed, got); n != "Record_desc_key" {
		t.Fatalf("first needed conflict %q", n)
	}
	if n := firstNeededConflict(map[string]bool{"Record_k_v": true}, got); n != "" {
		t.Fatalf("unneeded conflict reported: %q", n)
	}
}
