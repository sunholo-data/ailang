package types

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// EffectfulVariantSuggester is injected by clients with a stdlib index. Library
// callers without an index still receive an honest generic remedy.
var EffectfulVariantSuggester func(module, name string) []string

func wrapTypeEqFailure(constraint TypeEq, err error) error {
	var mismatch *EffectRowMismatchError
	if constraint.App == nil || !errors.As(err, &mismatch) {
		return fmt.Errorf("type unification failed at %v: %w", constraint.Path, err)
	}
	path := applicationDiagnosticPathNoNote(constraint.App)
	if mismatch.ParamIdx < 0 || mismatch.nested {
		return fmt.Errorf("type unification failed at %v: %w", path, err)
	}
	return fmt.Errorf("%s (solver: type unification failed at %v: %w)", effectMismatchLead(constraint.App, mismatch), path, err)
}

func effectLabelsExcept(row, other *Row) string {
	var labels []string
	for label := range row.Labels {
		if _, ok := other.Labels[label]; !ok {
			labels = append(labels, label)
		}
	}
	sort.Strings(labels)
	return strings.Join(labels, ", ")
}

func effectMismatchLead(app *AppConstraintInfo, mismatch *EffectRowMismatchError) string {
	declared, actual := mismatch.DeclaredRow, mismatch.ActualRow
	extra, missing := effectLabelsExcept(actual, declared), effectLabelsExcept(declared, actual)
	callee := "this call"
	if app.Callee != "" {
		callee = "`" + app.Callee + "`"
	}
	i := mismatch.ParamIdx
	word := "argument"
	if i < len(app.ArgIsLambda) && app.ArgIsLambda[i] {
		word = "lambda"
	}
	location := ""
	if i < len(app.ArgSpans) && app.ArgSpans[i].Line > 0 {
		location = " (at " + app.ArgSpans[i].String() + ")"
	}
	purity := ""
	if len(declared.Labels) == 0 && declared.Tail == nil {
		purity = " (pure)"
	}
	lead := fmt.Sprintf("%s requires parameter %d to be a function with effect row `%s`%s, but this %s%s", callee, i, declared, purity, word, location)
	if extra != "" && missing == "" {
		lead += " performs effects {" + extra + "}. "
	} else {
		lead += fmt.Sprintf(" has effect row `%s` (performs {%s}, missing {%s}). ", actual, extra, missing)
	}
	remedy := "Use an effect-accepting combinator if one is exported"
	if EffectfulVariantSuggester != nil {
		variants := append([]string(nil), EffectfulVariantSuggester(app.FromModule, app.Callee)...)
		sort.Strings(variants)
		if len(variants) > 0 {
			remedy = "Use an effect-accepting combinator such as `" + strings.Join(variants, "`, `") + "`"
		}
	}
	return lead + remedy + ", or rewrite the traversal as a recursive function whose declared effects cover the callback's effects."
}
