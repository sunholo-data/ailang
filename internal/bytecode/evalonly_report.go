package bytecode

import "sort"

// EvalOnlyEntry is one evaluator-only prototype the strict VM would trap on:
// what `ailang check --strict-bytecode` reports as BC001, what a strict run
// rejects before dispatch, and what a non-strict run warns it will bridge.
type EvalOnlyEntry struct {
	Name   string // canonical prototype name ("module.func", or bare)
	File   string // source file of the declaration ("" = unknown)
	Line   int    // declaration line (0 = unknown)
	Reason string // the compiler's EvalReason, verbatim
}

// EvalOnlyReachable returns every EvalOnly prototype reachable from roots,
// sorted by name. A root that is itself EvalOnly is included.
//
// Reachability follows NestedProtos, which is the only way bytecode can name
// another prototype: every call, first-class function reference and lambda
// goes through OpClosure on a NestedProtos index. That makes the walk exact
// for what the VM can dispatch — it is the set a strict run would trap on —
// and keeps imported evaluator-only helpers the program never touches (std
// carries several, e.g. closure-arg builtins) out of the report.
func (img *BytecodeImage) EvalOnlyReachable(roots []int) []EvalOnlyEntry {
	seen := make(map[int]bool, len(img.Prototypes))
	stack := append([]int(nil), roots...)
	var out []EvalOnlyEntry
	for len(stack) > 0 {
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if idx < 0 || idx >= len(img.Prototypes) || seen[idx] {
			continue
		}
		seen[idx] = true
		p := img.Prototypes[idx]
		if p.EvalOnly {
			out = append(out, EvalOnlyEntry{Name: p.Name, File: p.File, Line: p.Line, Reason: p.EvalReason})
			continue // no body, so nothing further runs on the VM from here
		}
		stack = append(stack, p.NestedProtos...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ProtoIndex returns the image index of p, or -1 when p is not in the image.
func (img *BytecodeImage) ProtoIndex(p *FuncPrototype) int {
	for i, q := range img.Prototypes {
		if q == p {
			return i
		}
	}
	return -1
}
