package types

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// The work budget (#2329, #2363 review): checking a declaration may take an
// amount of unification work that grows with the declaration's size, and no
// more.
//
// A large type is caught by size where it is held (type_size.go); the work
// budget catches a walk over an expansion that never is held, such as two
// separately declared alias families that the equivalence check cannot see
// through (a function or record type inside), unified with each other: a few
// lines of source, exponential work. A flat budget cannot tell that apart from
// a long, honest, straight-line function, whose unification work grows with
// the square of its length (each let is checked against an environment that
// grows with it). So the budget is
//
//	workBudget(n) = workFloor + workPerNode2 × n²
//
// where n is the size of the declaration — for unification, the Core nodes
// the checker has typed so far; for the IFC parameter-bound walk, the written
// size of the parameter type and the module's type declarations. An
// exponential shape is tiny in source, so it gets about workFloor steps and
// is stopped in a second or two; long code gets a budget that grows with it.
//
// Constants measured 2026-11-16, sampling (steps, nodes typed) throughout the
// check of every declaration in the repository's .ail files, the 14,780
// banked passing solutions, and generated straight-line code (800 to 5,000
// generic-call lets, a 2,000-line state machine, a 1,000-arm match): the
// largest (steps - workFloor/50) / nodes² anywhere is 0.14 (the state machine),
// so with workPerNode2 = 32 every declaration stays at least 50x below its
// budget at every point (over 230x once its size dominates). No declaration
// came near the floor with few nodes typed.
const (
	workFloor    = 1 << 21 // 2,097,152 steps: about a second of unification
	workPerNode2 = 32      // steps allowed per (node²)
)

// workBudget is the work allowed for a declaration of n nodes.
func workBudget(n int) int64 {
	b := int64(workFloor) + int64(workPerNode2)*int64(n)*int64(n)
	if b < 0 || n > 1<<24 {
		return 1 << 62
	}
	return b
}

// TC_TYPE_WORK_001 codes a declaration whose checking takes more unification
// work than its size allows.
const TC_TYPE_WORK_001 = "TC_TYPE_WORK_001"

// typeWorkExceeded unwinds inference when a declaration exceeds its work
// budget; the declaration boundary (recoverLatentDataDepth) turns it into the
// coded error.
type typeWorkExceeded struct {
	pos          string
	steps, limit int64
	nodes        int
}

func (e typeWorkExceeded) Error() string {
	where := ""
	if e.pos != "" {
		where = " at " + e.pos
	}
	return fmt.Sprintf("%s: checking this declaration%s takes too much unification work for its size: "+
		"more than %s steps for %s nodes (the limit at that size). Two type aliases that expand to the same large type "+
		"one level at a time are compared pair by pair; give the repeated part a data type (`type Pair = Pair(T, T)`), "+
		"which is never expanded, or split the declaration",
		TC_TYPE_WORK_001, where, groupDigits(e.limit), groupDigits(int64(e.nodes)))
}

// unifyWork counts the unification steps of one declaration.
type unifyWork struct {
	steps int64
	next  int64      // steps at which the budget is next consulted
	typed func() int // nodes typed so far in the declaration
	floor int64      // workFloor; smaller only in tests
	pos   ast.Pos    // the declaration, for the diagnostic
}

func newUnifyWork(pos ast.Pos, typed func() int) *unifyWork {
	return &unifyWork{typed: typed, floor: workFloor, next: workFloor, pos: pos}
}

// step counts one unification step. The budget is consulted only when the
// steps pass the last limit seen, so ordinary declarations never compute it.
func (w *unifyWork) step() {
	if w == nil {
		return
	}
	if w.steps++; w.steps <= w.next {
		return
	}
	n := 0
	if w.typed != nil {
		n = w.typed()
	}
	limit := workBudget(n) - workFloor + w.floor
	if w.steps > limit {
		panic(typeWorkExceeded{pos: positionString(w.pos), steps: w.steps, limit: limit, nodes: n})
	}
	w.next = limit
}
