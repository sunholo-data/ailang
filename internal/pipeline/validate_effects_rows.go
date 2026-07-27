package pipeline

import (
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// cloneEffectRow protects source-declared rows from collection. Validation
// treats the declaration map as immutable, including nested maps and pointers.
func cloneEffectRow(row *types.Row) *types.Row {
	if row == nil {
		return nil
	}
	cloned := &types.Row{
		Kind: row.Kind,
		Tail: row.Tail,
	}
	if row.Labels != nil {
		cloned.Labels = make(map[string]types.Type, len(row.Labels))
		for name, labelType := range row.Labels {
			cloned.Labels[name] = labelType
		}
	}
	cloned.Budgets = cloneBudgetMap(row.Budgets)
	cloned.MinBudgets = cloneBudgetMap(row.MinBudgets)
	if row.Params != nil {
		cloned.Params = make(map[string]map[string]string, len(row.Params))
		for effect, params := range row.Params {
			cloned.Params[effect] = cloneStringMap(params)
		}
	}
	if row.Provenance != nil {
		cloned.Provenance = make(map[string]ast.Span, len(row.Provenance))
		for effect, span := range row.Provenance {
			cloned.Provenance[effect] = span
		}
	}
	return cloned
}

func cloneBudgetMap(src map[string]*int) map[string]*int {
	if src == nil {
		return nil
	}
	dst := make(map[string]*int, len(src))
	for effect, budget := range src {
		if budget != nil {
			value := *budget
			dst[effect] = &value
		}
	}
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

// unionRequiredEffectRows is the validation collector's conflict-preserving
// union. UnionEffectRows intentionally chooses its left parameter map when two
// modes conflict; validation cannot discard either requirement. Conflicting
// values are represented as a stable, sorted set so strict M1 subsumption
// rejects every single-mode declaration independent of traversal order.
func unionRequiredEffectRows(a, b *types.Row) *types.Row {
	if a == nil {
		return cloneEffectRow(b)
	}
	if b == nil {
		return cloneEffectRow(a)
	}
	merged := types.UnionEffectRows(a, b)
	if merged == nil {
		return nil
	}
	for effect := range merged.Labels {
		left := effectiveValidationParams(a, effect)
		right := effectiveValidationParams(b, effect)
		if len(left) == 0 && len(right) == 0 {
			continue
		}
		if merged.Params == nil {
			merged.Params = make(map[string]map[string]string)
		}
		merged.Params[effect] = mergeRequirementParams(left, right)
	}
	return merged
}

func effectiveValidationParams(row *types.Row, effect string) map[string]string {
	if row != nil && len(row.Params[effect]) > 0 {
		return row.Params[effect]
	}
	if key, value, ok := types.DefaultModeFor(effect); ok {
		return map[string]string{key: value}
	}
	return nil
}

func mergeRequirementParams(a, b map[string]string) map[string]string {
	keys := make(map[string]struct{}, len(a)+len(b))
	for key := range a {
		keys[key] = struct{}{}
	}
	for key := range b {
		keys[key] = struct{}{}
	}
	merged := make(map[string]string, len(keys))
	for key := range keys {
		values := make(map[string]struct{})
		for _, value := range []string{a[key], b[key]} {
			for _, item := range strings.Split(value, "|") {
				if item != "" {
					values[item] = struct{}{}
				}
			}
		}
		sorted := make([]string, 0, len(values))
		for value := range values {
			sorted = append(sorted, value)
		}
		sort.Strings(sorted)
		merged[key] = strings.Join(sorted, "|")
	}
	return merged
}
