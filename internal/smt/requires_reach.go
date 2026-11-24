package smt

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
)

// requires_reach.go — the D1 blast radius (M-VERIFY-SOUNDNESS-COVERAGE M0).
//
// For each contracted function, the reachable callees that carry a requires.
// Since M3 the calls to them in the function's own body are checked
// (RequiresChecked); the gap map reports the rest as not yet checked.
// Nothing here changes a verdict.

// annotateRequiresCallees fills RequiresCallees on every result whose function
// reaches (through its body or its contracts, as the encoder inlines them) a
// callee with at least one requires clause.
func annotateRequiresCallees(report *VerifyReport, prog *core.Program, imported map[string]*core.Program) {
	for i := range report.Results {
		r := &report.Results[i]
		meta := prog.Meta[r.Function]
		if meta == nil || len(meta.Contracts) == 0 {
			continue
		}
		_, body := unwrapLambda(findFuncBody(prog, r.Function))
		roots := append([]core.CoreExpr{body}, contractExprs(meta.Contracts)...)
		r.RequiresCallees = requiresCallees(roots, r.Function, prog, imported)
	}
}

// requiresCallees returns, sorted, the callees reachable from roots whose
// declaration has a requires clause.
func requiresCallees(roots []core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program) []string {
	names := collectCalleeCallsFromRoots(roots, selfName, prog, imported)
	sort.Strings(names)
	var out []string
	for _, name := range names {
		_, owner := findFuncBodyInAnyProg(name, prog, imported)
		if owner != nil && hasRequires(owner.Meta[name]) {
			out = append(out, name)
		}
	}
	return out
}

func hasRequires(meta *core.DeclMeta) bool {
	if meta == nil {
		return false
	}
	for _, c := range meta.Contracts {
		if c != nil && c.Kind == core.RequiresKind {
			return true
		}
	}
	return false
}
