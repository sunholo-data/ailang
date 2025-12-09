package golang

import (
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

func TestIsIfElseChain(t *testing.T) {
	// Test case 1: Simple if-else (no chain)
	simpleIf := &core.If{
		Cond: &core.Var{Name: "x"},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Else: &core.Lit{Kind: core.IntLit, Value: int64(1)},
	}
	if isIfElseChain(simpleIf) {
		t.Error("Simple if-else should not be a chain")
	}

	// Test case 2: If-else with else being another If (direct chain)
	chainIf := &core.If{
		Cond: &core.Var{Name: "x"},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Else: &core.If{
			Cond: &core.Var{Name: "y"},
			Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
			Else: &core.Lit{Kind: core.IntLit, Value: int64(2)},
		},
	}
	if !isIfElseChain(chainIf) {
		t.Error("Direct if-else chain should be detected")
	}

	// Test case 3: If-else with else being Let containing If (wrapped chain)
	wrappedChainIf := &core.If{
		Cond: &core.Var{Name: "x"},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Else: &core.Let{
			Name:  "tmp",
			Value: &core.Lit{Kind: core.BoolLit, Value: true},
			Body: &core.If{
				Cond: &core.Var{Name: "tmp"},
				Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
				Else: &core.Lit{Kind: core.IntLit, Value: int64(2)},
			},
		},
	}
	if !isIfElseChain(wrappedChainIf) {
		t.Error("Let-wrapped if-else chain should be detected")
	}
}

func TestIsLetIfChain(t *testing.T) {
	// Test case 1: Let with non-If body
	simpleLet := &core.Let{
		Name:  "x",
		Value: &core.Lit{Kind: core.IntLit, Value: int64(1)},
		Body:  &core.Var{Name: "x"},
	}
	if isLetIfChain(simpleLet) {
		t.Error("Simple let should not be a chain")
	}

	// Test case 2: Let with simple If body (no chain)
	letWithSimpleIf := &core.Let{
		Name:  "x",
		Value: &core.Lit{Kind: core.BoolLit, Value: true},
		Body: &core.If{
			Cond: &core.Var{Name: "x"},
			Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
			Else: &core.Lit{Kind: core.IntLit, Value: int64(1)},
		},
	}
	if isLetIfChain(letWithSimpleIf) {
		t.Error("Let with simple if should not be a chain")
	}

	// Test case 3: Let with If body that has Let-wrapped else (chain)
	letWithChainIf := &core.Let{
		Name:  "tmp1",
		Value: &core.Lit{Kind: core.BoolLit, Value: true},
		Body: &core.If{
			Cond: &core.Var{Name: "tmp1"},
			Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
			Else: &core.Let{
				Name:  "tmp2",
				Value: &core.Lit{Kind: core.BoolLit, Value: true},
				Body: &core.If{
					Cond: &core.Var{Name: "tmp2"},
					Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
					Else: &core.Lit{Kind: core.IntLit, Value: int64(2)},
				},
			},
		},
	}
	if !isLetIfChain(letWithChainIf) {
		t.Error("Let with if-else chain should be detected")
	}
}

func TestCollectIfChain(t *testing.T) {
	// Build a 3-branch chain: if tmp1 then 0 else let tmp2 = ... in if tmp2 then 1 else 2
	chain := &core.If{
		Cond: &core.Var{Name: "tmp1"},
		Then: &core.Lit{Kind: core.IntLit, Value: int64(0)},
		Else: &core.Let{
			Name:  "tmp2",
			Value: &core.Lit{Kind: core.BoolLit, Value: true},
			Body: &core.If{
				Cond: &core.Var{Name: "tmp2"},
				Then: &core.Lit{Kind: core.IntLit, Value: int64(1)},
				Else: &core.Lit{Kind: core.IntLit, Value: int64(2)},
			},
		},
	}

	branches, lets := collectIfChain(chain)

	if len(branches) != 3 {
		t.Errorf("Expected 3 branches, got %d", len(branches))
	}

	if len(lets) != 1 {
		t.Errorf("Expected 1 let binding, got %d", len(lets))
	}

	// Check branch conditions
	if branches[0].Cond == nil {
		t.Error("First branch should have condition")
	}
	if branches[1].Cond == nil {
		t.Error("Second branch should have condition")
	}
	if branches[2].Cond != nil {
		t.Error("Third (final) branch should not have condition")
	}
}
