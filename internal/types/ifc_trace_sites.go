package types

import (
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Trace-redaction sites (M-TRACE-LABEL-AWARE,
// design_docs/planned/v0_36_0/m-trace-label-aware.md).
//
// The tracer runs below the type system: labels are erased before runtime and
// the collector receives only rendered strings, so it cannot tell a <secret>
// value from any other. This pass is the one place that knows, so it reports
// every call site at which a labelled value is passed (an argument), captured
// (the callee value — a closure over a secret) or produced (the result). The
// pipeline stamps each site's label onto the elaborated core.App, and the
// evaluator redacts the trace for the dynamic extent of that call.
//
// Labels are taken from EVERY walk the checker makes of a call — the reporting
// walk, the silent ⊥-seeded body walks, and the summary walks — with label
// variables (ρ, "whatever the caller passes") erased. Each concrete label any
// walk sees is a real flow in some context, so the union over-approximates;
// for a redaction policy that is the safe direction.

// CheckModuleIFCWithTraceSites is CheckModuleIFC plus the trace-redaction
// sites: each labelled call's FuncCall.Pos mapped to its sorted,
// comma-joined label names ("pii,secret").
func CheckModuleIFCWithTraceSites(file *ast.File) ([]*TypeCheckError, map[ast.Pos]string) {
	return CheckModuleIFCWithImportsAndTraceSites(file, IFCImports{})
}

// CheckModuleIFCWithImportsAndTraceSites is the production gate (pipeline,
// REPL): CheckModuleIFCWithImports plus the trace-redaction sites, from ONE
// walk so the redaction policy sees exactly the flows the gate checked —
// including labels produced by imported callees (#1134).
func CheckModuleIFCWithImportsAndTraceSites(file *ast.File, imports IFCImports) ([]*TypeCheckError, map[ast.Pos]string) {
	if file == nil {
		return nil, nil
	}
	if err := imports.validate(); err != nil {
		return []*TypeCheckError{err}, nil
	}
	sites := map[ast.Pos]map[string]bool{}
	errs := guardIFCAnalysis(func() []*TypeCheckError {
		c := newIFCChecker(file)
		c.useImports(imports)
		c.traceSites = sites
		return c.run(file)
	})
	out := make(map[ast.Pos]string, len(sites))
	for pos, names := range sites {
		list := make([]string, 0, len(names))
		for n := range names {
			list = append(list, n)
		}
		sort.Strings(list)
		out[pos] = strings.Join(list, ",")
	}
	return errs, out
}

// noteTraceSite records the concrete label names of l at a call site. Only
// constant parts are kept: a label variable is not a fact about any value.
func (c *ifcChecker) noteTraceSite(pos ast.Pos, l Label) {
	for _, part := range normalisedParts(l) {
		lc, ok := part.(labelConst)
		if !ok {
			continue
		}
		names := c.traceSites[pos]
		if names == nil {
			names = map[string]bool{}
			c.traceSites[pos] = names
		}
		names[lc.name] = true
	}
}
