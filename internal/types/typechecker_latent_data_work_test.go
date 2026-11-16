package types

import (
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/ast"
)

// distinctExpansion expands G[x] to (G[(x, int)], G[(x, string)]) until x is
// levels tuples deep, where it is fn: 2^levels distinct applications, which
// no memo can share (the #2363 round-3 darr shape).
func distinctExpansion(fn Type, levels int) func(Type) Type {
	return func(t Type) Type {
		app, ok := t.(*TApp)
		if !ok {
			return t
		}
		depth := 0
		for x := app.Args[0]; ; depth++ {
			tup, ok := x.(*TTuple)
			if !ok {
				break
			}
			x = tup.Elements[0]
		}
		if depth >= levels {
			return fn
		}
		g := func(second Type) Type {
			return &TApp{Constructor: app.Constructor, Args: []Type{&TTuple{Elements: []Type{app.Args[0], second}}}}
		}
		return &TTuple{Elements: []Type{g(TInt), g(TString)}}
	}
}

// The scan charges every node it visits to the declaration's work budget and
// ends in TC_TYPE_WORK_001, saying it was expanding aliases.
func TestDataArrows_ExpansionChargesTheWorkBudget(t *testing.T) {
	fn := latentArrow("e")
	work := newUnifyWork(ast.Pos{}, func() int { return 3 })
	work.floor, work.next = 1000, 1000
	err := func() (err error) {
		defer recoverLatentDataDepth(&err)
		dataArrowsWith(&TApp{Constructor: &TCon{Name: "G"}, Args: []Type{TInt}}, distinctExpansion(fn, 12), work)
		return nil
	}()
	if err == nil || !strings.Contains(err.Error(), TC_TYPE_WORK_001) || !strings.Contains(err.Error(), "too much work expanding type aliases") ||
		!strings.Contains(err.Error(), "more than 1,288 steps for 3 nodes") {
		t.Fatalf("want %s from the scan, got %v", TC_TYPE_WORK_001, err)
	}
}

// Past its memo cap the scan forgets finished nodes instead of growing, and
// still finds every arrow: 2^16 distinct applications, within the floor.
func TestDataArrows_MemoCapStillFindsTheArrows(t *testing.T) {
	fn := latentArrow("e")
	var arrows []*TFunc2
	within(t, 10*time.Second, func() {
		arrows = dataArrowsWith(&TApp{Constructor: &TCon{Name: "G"}, Args: []Type{TInt}}, distinctExpansion(fn, 16), nil)
	})
	if len(arrows) == 0 {
		t.Fatal("no arrow found")
	}
	for _, a := range arrows {
		if a != fn {
			t.Fatalf("found an arrow that is not fn: %v", a)
		}
	}
}
