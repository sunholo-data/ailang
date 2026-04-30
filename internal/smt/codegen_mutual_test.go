package smt

import (
	"strings"
	"testing"
)

// TestSplitDeclareDatatype_Simple verifies extracting sort name + variants body
// from a simple non-recursive declaration.
func TestSplitDeclareDatatype_Simple(t *testing.T) {
	decl := `(declare-datatype Point ((mk_Point (x Int) (y Int))))`
	name, body := splitDeclareDatatype(decl)
	if name != "Point" {
		t.Errorf("name = %q, want Point", name)
	}
	wantBody := `((mk_Point (x Int) (y Int)))`
	if body != wantBody {
		t.Errorf("body = %q, want %q", body, wantBody)
	}
}

// TestSplitDeclareDatatype_NestedParens handles ADTs with multiple variants
// containing nested parens in field types like (Seq Block).
func TestSplitDeclareDatatype_NestedParens(t *testing.T) {
	decl := `(declare-datatype Block ((TextBlock (TextBlock_0 Int)) (SectionBlock (SectionBlock_0 (Seq Block)))))`
	name, body := splitDeclareDatatype(decl)
	if name != "Block" {
		t.Errorf("name = %q, want Block", name)
	}
	wantBody := `((TextBlock (TextBlock_0 Int)) (SectionBlock (SectionBlock_0 (Seq Block))))`
	if body != wantBody {
		t.Errorf("body = %q, want %q", body, wantBody)
	}
}

// TestDeclareDatatypesMutual_TwoSorts emits the plural form for a Block ↔
// Record_blocks_kind mutual recursion (the canonical docparse case).
func TestDeclareDatatypesMutual_TwoSorts(t *testing.T) {
	decls := []string{
		`(declare-datatype Block ((TextBlock (TextBlock_0 Int)) (SectionBlock (SectionBlock_0 Record_blocks_kind))))`,
		`(declare-datatype Record_blocks_kind ((mk_Record_blocks_kind (blocks (Seq Block)) (kind String))))`,
	}
	got := DeclareDatatypesMutual(decls)
	// Must contain plural form
	if !strings.HasPrefix(got, "(declare-datatypes") {
		t.Errorf("expected (declare-datatypes prefix, got: %s", got)
	}
	// Must include both sorts in the type list
	if !strings.Contains(got, "(Block 0)") {
		t.Errorf("expected (Block 0) in type list, got: %s", got)
	}
	if !strings.Contains(got, "(Record_blocks_kind 0)") {
		t.Errorf("expected (Record_blocks_kind 0) in type list, got: %s", got)
	}
	// Both variant lists must be present (their bodies live in the second tuple)
	if !strings.Contains(got, "TextBlock") || !strings.Contains(got, "SectionBlock") {
		t.Errorf("expected Block variants in output, got: %s", got)
	}
	if !strings.Contains(got, "mk_Record_blocks_kind") {
		t.Errorf("expected Record_blocks_kind constructor in output, got: %s", got)
	}
}

// TestFindSCCs_SelfRecursive verifies a self-referencing sort forms a 1-cycle.
func TestFindSCCs_SelfRecursive(t *testing.T) {
	decls := []string{
		`(declare-datatype List ((Nil) (Cons (head Int) (tail List))))`,
	}
	sccs := findSCCs(decls)
	if len(sccs) != 1 {
		t.Fatalf("expected 1 SCC, got %d: %v", len(sccs), sccs)
	}
	if len(sccs[0]) != 1 || sccs[0][0] != "List" {
		t.Errorf("expected [[List]], got %v", sccs)
	}
}

// TestFindSCCs_MutualRecursion verifies that two sorts referencing each other
// form a single SCC of size 2.
func TestFindSCCs_MutualRecursion(t *testing.T) {
	decls := []string{
		`(declare-datatype Block ((SectionBlock (SectionBlock_0 Record_blocks_kind))))`,
		`(declare-datatype Record_blocks_kind ((mk_Record_blocks_kind (blocks (Seq Block)))))`,
	}
	sccs := findSCCs(decls)
	if len(sccs) != 1 {
		t.Fatalf("expected 1 SCC for mutual recursion, got %d: %v", len(sccs), sccs)
	}
	if len(sccs[0]) != 2 {
		t.Fatalf("expected SCC of size 2, got %d: %v", len(sccs[0]), sccs[0])
	}
	got := map[string]bool{sccs[0][0]: true, sccs[0][1]: true}
	if !got["Block"] || !got["Record_blocks_kind"] {
		t.Errorf("expected SCC = {Block, Record_blocks_kind}, got %v", sccs[0])
	}
}

// TestFindSCCs_AcyclicGroup verifies that two non-recursive sorts produce
// two separate SCCs (no cycle).
func TestFindSCCs_AcyclicGroup(t *testing.T) {
	decls := []string{
		`(declare-datatype Inner ((mk_Inner (x Int))))`,
		`(declare-datatype Outer ((mk_Outer (inner Inner))))`,
	}
	sccs := findSCCs(decls)
	if len(sccs) != 2 {
		t.Fatalf("expected 2 SCCs (acyclic), got %d: %v", len(sccs), sccs)
	}
	for _, scc := range sccs {
		if len(scc) != 1 {
			t.Errorf("expected each SCC to be a singleton, got: %v", scc)
		}
	}
}
