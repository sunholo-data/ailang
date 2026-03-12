package smt

import (
	"strings"
	"testing"
)

// --- GenerateListUnrolling tests (M3_RECURSIVE_LIST_OPS) ---

func TestGenerateListUnrolling_Reverse_Depth1(t *testing.T) {
	decls := GenerateListUnrolling("_list_reverse", 1, "Int")
	if len(decls) != 2 {
		t.Fatalf("expected 2 declarations (level 0 + level 1), got %d", len(decls))
	}

	// Level 0: uninterpreted
	if !strings.Contains(decls[0], "declare-fun") {
		t.Errorf("level 0 should be declare-fun, got %q", decls[0])
	}
	if !strings.Contains(decls[0], "_list_reverse_0") {
		t.Errorf("level 0 should contain _list_reverse_0, got %q", decls[0])
	}

	// Level 1: define-fun with seq.len, seq.extract, seq.nth, seq.unit, seq.++
	if !strings.Contains(decls[1], "define-fun") {
		t.Errorf("level 1 should be define-fun, got %q", decls[1])
	}
	if !strings.Contains(decls[1], "_list_reverse_1") {
		t.Errorf("level 1 should contain _list_reverse_1, got %q", decls[1])
	}
	if !strings.Contains(decls[1], "_list_reverse_0") {
		t.Errorf("level 1 should reference _list_reverse_0, got %q", decls[1])
	}
	if !strings.Contains(decls[1], "seq.len") {
		t.Errorf("level 1 should use seq.len, got %q", decls[1])
	}
}

func TestGenerateListUnrolling_Reverse_Depth3(t *testing.T) {
	decls := GenerateListUnrolling("_list_reverse", 3, "Int")
	if len(decls) != 4 {
		t.Fatalf("expected 4 declarations (level 0-3), got %d", len(decls))
	}

	// Level 0: uninterpreted
	if !strings.Contains(decls[0], "_list_reverse_0") {
		t.Errorf("level 0 should be _list_reverse_0")
	}

	// Each level k should reference level k-1
	for k := 1; k <= 3; k++ {
		if !strings.Contains(decls[k], "_list_reverse_"+string(rune('0'+k))) {
			t.Errorf("level %d should define _list_reverse_%d", k, k)
		}
		if !strings.Contains(decls[k], "_list_reverse_"+string(rune('0'+k-1))) {
			t.Errorf("level %d should reference _list_reverse_%d", k, k-1)
		}
	}
}

func TestGenerateListUnrolling_Take_Depth1(t *testing.T) {
	decls := GenerateListUnrolling("_list_take", 1, "Int")
	if len(decls) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(decls))
	}

	// Level 0: uninterpreted with 2 args (Int, (Seq Int))
	if !strings.Contains(decls[0], "_list_take_0") {
		t.Errorf("level 0 should contain _list_take_0")
	}
	if !strings.Contains(decls[0], "declare-fun") {
		t.Errorf("level 0 should be declare-fun")
	}

	// Level 1: define-fun
	if !strings.Contains(decls[1], "_list_take_1") {
		t.Errorf("level 1 should contain _list_take_1")
	}
	if !strings.Contains(decls[1], "_list_take_0") {
		t.Errorf("level 1 should reference _list_take_0")
	}
	if !strings.Contains(decls[1], "seq.unit") {
		t.Errorf("level 1 should use seq.unit")
	}
}

func TestGenerateListUnrolling_Drop_Depth1(t *testing.T) {
	decls := GenerateListUnrolling("_list_drop", 1, "Int")
	if len(decls) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(decls))
	}

	if !strings.Contains(decls[0], "_list_drop_0") {
		t.Errorf("level 0 should contain _list_drop_0")
	}
	if !strings.Contains(decls[1], "_list_drop_1") {
		t.Errorf("level 1 should contain _list_drop_1")
	}
	if !strings.Contains(decls[1], "_list_drop_0") {
		t.Errorf("level 1 should reference _list_drop_0")
	}
	if !strings.Contains(decls[1], "seq.extract") {
		t.Errorf("level 1 should use seq.extract")
	}
}

func TestGenerateListUnrolling_Reverse_Depth2(t *testing.T) {
	decls := GenerateListUnrolling("_list_reverse", 2, "Int")
	if len(decls) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(decls))
	}

	// Level 2 should reference level 1
	if !strings.Contains(decls[2], "_list_reverse_2") {
		t.Errorf("level 2 should define _list_reverse_2")
	}
	if !strings.Contains(decls[2], "_list_reverse_1") {
		t.Errorf("level 2 should reference _list_reverse_1")
	}
}

func TestGenerateListUnrolling_Take_Depth2(t *testing.T) {
	decls := GenerateListUnrolling("_list_take", 2, "Int")
	if len(decls) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(decls))
	}

	if !strings.Contains(decls[2], "_list_take_2") {
		t.Errorf("level 2 should define _list_take_2")
	}
	if !strings.Contains(decls[2], "_list_take_1") {
		t.Errorf("level 2 should reference _list_take_1")
	}
}

func TestGenerateListUnrolling_Drop_Depth2(t *testing.T) {
	decls := GenerateListUnrolling("_list_drop", 2, "Int")
	if len(decls) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(decls))
	}

	if !strings.Contains(decls[2], "_list_drop_2") {
		t.Errorf("level 2 should define _list_drop_2")
	}
	if !strings.Contains(decls[2], "_list_drop_1") {
		t.Errorf("level 2 should reference _list_drop_1")
	}
}

func TestGenerateListUnrolling_UnknownOp(t *testing.T) {
	decls := GenerateListUnrolling("_list_unknown", 1, "Int")
	if decls != nil {
		t.Errorf("expected nil for unknown operation, got %v", decls)
	}
}

func TestRecursiveListBuiltins_MapContents(t *testing.T) {
	for _, name := range []string{"_list_reverse", "_list_take", "_list_drop"} {
		if _, ok := RecursiveListBuiltins[name]; !ok {
			t.Errorf("expected %q in RecursiveListBuiltins map", name)
		}
	}
}

func TestTopLevelUnrolledName(t *testing.T) {
	name := TopLevelUnrolledName("_list_reverse", 3)
	if name != "_list_reverse_3" {
		t.Errorf("expected _list_reverse_3, got %s", name)
	}
}
