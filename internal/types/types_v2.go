package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// TypeV2 represents types with proper kind tracking
type TypeV2 interface {
	Type
	GetKind() Kind
}

// TVar2 represents a type variable with kind
type TVar2 struct {
	Name string
	Kind Kind
}

func (t *TVar2) String() string { return t.Name }
func (t *TVar2) GetKind() Kind  { return t.Kind }

func (t *TVar2) Equals(other Type) bool {
	if o, ok := other.(*TVar2); ok {
		return t.Name == o.Name && t.Kind.Equals(o.Kind)
	}
	return false
}

func (t *TVar2) Substitute(subs map[string]Type) Type {
	if sub, ok := subs[t.Name]; ok {
		return sub
	}
	return t
}

// RowVar represents a row variable with kind
type RowVar struct {
	Name string
	Kind Kind // Should be KRow(KEffect) or KRow(KRecord)
}

func (r *RowVar) String() string { return r.Name }
func (r *RowVar) GetKind() Kind  { return r.Kind }

func (r *RowVar) Equals(other Type) bool {
	if o, ok := other.(*RowVar); ok {
		return r.Name == o.Name && r.Kind.Equals(o.Kind)
	}
	return false
}

func (r *RowVar) Substitute(subs map[string]Type) Type {
	if sub, ok := subs[r.Name]; ok {
		return sub
	}
	return r
}

// Row represents a row type (for both records and effects)
type Row struct {
	Kind       Kind            // KRow(KEffect) or KRow(KRecord)
	Labels     map[string]Type // For records: field->type, For effects: effect->unit
	Tail       *RowVar         // Optional row variable for extension
	Budgets    map[string]*int // For effects: optional budget limits (nil = unlimited)
	MinBudgets map[string]*int // For effects: optional minimum usage requirements (M-DX25 M4)
	// Params is per-effect parameter map: effect_name -> (param_key -> param_value).
	// NEW in M-EFFECT-REFINEMENT Phase 1 (v0.15.0). Nil/empty for non-parameterised
	// effects (back-compat). Only populated for effect rows (Kind == EffectRow);
	// records leave it nil. Used for invariant unification — !{Rand[mode=os]} does
	// NOT unify with !{Rand[mode=seeded]}, but DOES unify with bare !{Rand} after
	// default-mode desugar via DefaultModeFor.
	Params map[string]map[string]string
	// Provenance maps effect label names to the source span where that label was
	// introduced (e.g. the call site of an effectful builtin). Nil by default and
	// intentionally excluded from Equals — provenance never causes unification failures.
	Provenance map[string]ast.Span
}

func (r *Row) String() string {
	// Sort labels for canonical representation
	var keys []string
	for k := range r.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		if r.Kind.Equals(EffectRow) {
			// Effect head: name with optional [param=value, ...] block (alphabetical by key)
			head := k
			if r.Params != nil {
				if pmap, ok := r.Params[k]; ok && len(pmap) > 0 {
					pkeys := make([]string, 0, len(pmap))
					for pk := range pmap {
						pkeys = append(pkeys, pk)
					}
					sort.Strings(pkeys)
					paramParts := make([]string, len(pkeys))
					for i, pk := range pkeys {
						paramParts[i] = fmt.Sprintf("%s=%s", pk, pmap[pk])
					}
					head = fmt.Sprintf("%s[%s]", k, strings.Join(paramParts, ", "))
				}
			}
			// For effect rows, include budget annotations if present
			var annotations []string
			if r.MinBudgets != nil {
				if min, ok := r.MinBudgets[k]; ok && min != nil {
					annotations = append(annotations, fmt.Sprintf("@min=%d", *min))
				}
			}
			if r.Budgets != nil {
				if budget, ok := r.Budgets[k]; ok && budget != nil {
					annotations = append(annotations, fmt.Sprintf("@limit=%d", *budget))
				}
			}
			if len(annotations) > 0 {
				parts = append(parts, head+" "+strings.Join(annotations, " "))
			} else {
				parts = append(parts, head)
			}
		} else {
			parts = append(parts, fmt.Sprintf("%s: %s", k, r.Labels[k].String()))
		}
	}

	if r.Tail != nil {
		parts = append(parts, "..."+r.Tail.String())
	}

	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

func (r *Row) GetKind() Kind { return r.Kind }

func (r *Row) Equals(other Type) bool {
	if o, ok := other.(*Row); ok {
		if !r.Kind.Equals(o.Kind) {
			return false
		}
		if len(r.Labels) != len(o.Labels) {
			return false
		}
		for k, v := range r.Labels {
			if ov, ok := o.Labels[k]; !ok || !v.Equals(ov) {
				return false
			}
		}
		// Compare budgets for effect rows
		if r.Kind.Equals(EffectRow) {
			if !equalBudgets(r.Budgets, o.Budgets) {
				return false
			}
			// Compare per-effect params (invariant; nil and empty are equivalent)
			if !equalParams(r.Params, o.Params) {
				return false
			}
		}
		if r.Tail == nil && o.Tail == nil {
			return true
		}
		if r.Tail != nil && o.Tail != nil {
			return r.Tail.Equals(o.Tail)
		}
		return false
	}
	return false
}

// equalParams compares two per-effect parameter maps for invariant equality.
// nil and empty maps are treated as equivalent. Used by Row.Equals.
func equalParams(a, b map[string]map[string]string) bool {
	// Strip empty inner maps before counting (nil == empty inner)
	aLen := 0
	for _, m := range a {
		if len(m) > 0 {
			aLen++
		}
	}
	bLen := 0
	for _, m := range b {
		if len(m) > 0 {
			bLen++
		}
	}
	if aLen != bLen {
		return false
	}
	for effect, am := range a {
		if len(am) == 0 {
			continue
		}
		bm := b[effect]
		if len(am) != len(bm) {
			return false
		}
		for k, v := range am {
			if bv, ok := bm[k]; !ok || bv != v {
				return false
			}
		}
	}
	return true
}

// equalBudgets compares two budget maps for equality
func equalBudgets(a, b map[string]*int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		// One is nil, one is not - check if non-nil is empty
		if a == nil && len(b) == 0 {
			return true
		}
		if b == nil && len(a) == 0 {
			return true
		}
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if av == nil && bv == nil {
			continue
		}
		if av == nil || bv == nil {
			return false
		}
		if *av != *bv {
			return false
		}
	}
	return true
}

func (r *Row) Substitute(subs map[string]Type) Type {
	labels := make(map[string]Type)
	for k, v := range r.Labels {
		labels[k] = v.Substitute(subs)
	}

	// Preserve budgets
	var budgets map[string]*int
	if r.Budgets != nil {
		budgets = make(map[string]*int)
		for k, v := range r.Budgets {
			budgets[k] = v
		}
	}

	// Preserve effect params (M-EFFECT-REFINEMENT Phase 1)
	var params map[string]map[string]string
	if r.Params != nil {
		params = make(map[string]map[string]string)
		for effect, pmap := range r.Params {
			if len(pmap) == 0 {
				continue
			}
			inner := make(map[string]string, len(pmap))
			for k, v := range pmap {
				inner[k] = v
			}
			params[effect] = inner
		}
		if len(params) == 0 {
			params = nil
		}
	}

	var tail *RowVar
	if r.Tail != nil {
		if sub, ok := subs[r.Tail.Name]; ok {
			// If substituting the tail, need to merge
			if subRow, ok := sub.(*Row); ok {
				// Merge labels
				for k, v := range subRow.Labels {
					labels[k] = v
				}
				// Merge budgets from substituted row
				if subRow.Budgets != nil {
					if budgets == nil {
						budgets = make(map[string]*int)
					}
					for k, v := range subRow.Budgets {
						budgets[k] = v
					}
				}
				// Merge params from substituted row
				if subRow.Params != nil {
					if params == nil {
						params = make(map[string]map[string]string)
					}
					for effect, pmap := range subRow.Params {
						if len(pmap) == 0 {
							continue
						}
						inner := make(map[string]string, len(pmap))
						for pk, pv := range pmap {
							inner[pk] = pv
						}
						params[effect] = inner
					}
				}
				tail = subRow.Tail
			} else if subVar, ok := sub.(*RowVar); ok {
				tail = subVar
			}
		} else {
			tail = r.Tail
		}
	}

	return &Row{
		Kind:    r.Kind,
		Labels:  labels,
		Tail:    tail,
		Budgets: budgets,
		Params:  params,
	}
}

// TFunc2 represents a function type with effect row
type TFunc2 struct {
	Params    []Type
	EffectRow *Row // Row of kind KRow(KEffect)
	Return    Type
}

func (t *TFunc2) String() string {
	params := make([]string, len(t.Params))
	for i, p := range t.Params {
		params[i] = p.String()
	}

	effectStr := ""
	if t.EffectRow != nil && (len(t.EffectRow.Labels) > 0 || t.EffectRow.Tail != nil) {
		effectStr = fmt.Sprintf(" ! %s", t.EffectRow.String())
	}

	if len(params) == 1 {
		return fmt.Sprintf("%s -> %s%s", params[0], t.Return.String(), effectStr)
	}
	return fmt.Sprintf("(%s) -> %s%s", strings.Join(params, ", "), t.Return.String(), effectStr)
}

func (t *TFunc2) GetKind() Kind { return Star }

func (t *TFunc2) Equals(other Type) bool {
	if o, ok := other.(*TFunc2); ok {
		if len(t.Params) != len(o.Params) {
			return false
		}
		for i := range t.Params {
			if !t.Params[i].Equals(o.Params[i]) {
				return false
			}
		}
		if !t.Return.Equals(o.Return) {
			return false
		}
		if t.EffectRow == nil && o.EffectRow == nil {
			return true
		}
		if t.EffectRow != nil && o.EffectRow != nil {
			return t.EffectRow.Equals(o.EffectRow)
		}
		return false
	}
	return false
}

func (t *TFunc2) Substitute(subs map[string]Type) Type {
	params := make([]Type, len(t.Params))
	for i, p := range t.Params {
		params[i] = p.Substitute(subs)
	}

	var effectRow *Row
	if t.EffectRow != nil {
		if sub := t.EffectRow.Substitute(subs); sub != nil {
			effectRow = sub.(*Row)
		}
	}

	return &TFunc2{
		Params:    params,
		EffectRow: effectRow,
		Return:    t.Return.Substitute(subs),
	}
}

// TRecord2 represents a record type with row
type TRecord2 struct {
	Row *Row // Row of kind KRow(KRecord)
}

func (t *TRecord2) String() string {
	if t.Row == nil {
		return "{}"
	}
	return t.Row.String()
}

func (t *TRecord2) GetKind() Kind { return Star }

func (t *TRecord2) Equals(other Type) bool {
	if o, ok := other.(*TRecord2); ok {
		if t.Row == nil && o.Row == nil {
			return true
		}
		if t.Row != nil && o.Row != nil {
			return t.Row.Equals(o.Row)
		}
		return false
	}
	return false
}

func (t *TRecord2) Substitute(subs map[string]Type) Type {
	if t.Row == nil {
		return t
	}
	return &TRecord2{
		Row: t.Row.Substitute(subs).(*Row),
	}
}

// Scheme represents a type scheme with quantified variables
type Scheme struct {
	TypeVars    []string     // Quantified type variables
	RowVars     []string     // Quantified row variables
	Constraints []Constraint // Type class constraints
	Type        Type
}

func (s *Scheme) String() string {
	var vars []string
	vars = append(vars, s.TypeVars...)
	vars = append(vars, s.RowVars...)

	prefix := ""
	if len(vars) > 0 {
		prefix = fmt.Sprintf("∀%s. ", strings.Join(vars, " "))
	}

	constraintStr := ""
	if len(s.Constraints) > 0 {
		constraints := make([]string, len(s.Constraints))
		for i, c := range s.Constraints {
			constraints[i] = c.String()
		}
		constraintStr = fmt.Sprintf("(%s) => ", strings.Join(constraints, ", "))
	}

	return prefix + constraintStr + s.Type.String()
}

// QualifiedScheme represents a type scheme with explicit class constraints
// This is used during generalization to preserve non-ground constraints
// e.g., ∀α. Num α ⇒ α → α → α
type QualifiedScheme struct {
	Constraints []ClassConstraint // Non-ground class constraints
	Scheme      *Scheme           // The polymorphic type scheme
}

// NewQualifiedScheme creates a qualified scheme from constraints and type
func NewQualifiedScheme(constraints []ClassConstraint, typeVars []string, typ Type) *QualifiedScheme {
	// Convert ClassConstraint to Constraint for the Scheme
	var schemeConstraints []Constraint
	for _, cc := range constraints {
		schemeConstraints = append(schemeConstraints, Constraint{
			Class: cc.Class,
			Type:  cc.Type,
		})
	}

	return &QualifiedScheme{
		Constraints: constraints,
		Scheme: &Scheme{
			TypeVars:    typeVars,
			Constraints: schemeConstraints,
			Type:        typ,
		},
	}
}

// String returns a string representation of the qualified scheme
func (qs *QualifiedScheme) String() string {
	if qs.Scheme != nil {
		return qs.Scheme.String()
	}

	// Format constraints manually if no scheme
	if len(qs.Constraints) > 0 {
		var constraintStrs []string
		for _, c := range qs.Constraints {
			constraintStrs = append(constraintStrs, fmt.Sprintf("%s[%s]", c.Class, c.Type))
		}
		return fmt.Sprintf("(%s) => ?", strings.Join(constraintStrs, ", "))
	}
	return "?"
}

// Instantiate creates a fresh instance of the type scheme
func (s *Scheme) Instantiate(fresh func(Kind) Type) Type {
	subs := make(map[string]Type)

	// Fresh type variables
	for _, v := range s.TypeVars {
		subs[v] = fresh(Star)
	}

	// Fresh row variables for effects
	for _, v := range s.RowVars {
		// Determine kind from usage in type
		// For now, assume effect rows (can be refined)
		subs[v] = fresh(EffectRow)
	}

	return s.Type.Substitute(subs)
}

// Helper functions

// EmptyEffectRow creates an empty effect row
func EmptyEffectRow() *Row {
	return &Row{
		Kind:   EffectRow,
		Labels: make(map[string]Type),
		Tail:   nil,
	}
}

// EmptyRecordRow creates an empty record row
func EmptyRecordRow() *Row {
	return &Row{
		Kind:   RecordRow,
		Labels: make(map[string]Type),
		Tail:   nil,
	}
}

// CanonicalizeRow returns the canonical representation of a row
func CanonicalizeRow(r *Row) *Row {
	// Already sorted in String(), but let's ensure Labels are in canonical form
	return r
}

// GetKind returns the kind of any type
func GetKind(t Type) Kind {
	switch t := t.(type) {
	case *TVar2:
		return t.Kind
	case *RowVar:
		return t.Kind
	case *Row:
		return t.Kind
	case *TFunc2:
		return Star
	case *TRecord2:
		return Star
	case *TCon:
		return Star
	case *TList:
		return Star
	case *TArray:
		return Star
	case *TMap:
		return Star
	case *TTuple:
		return Star
	default:
		return Star
	}
}
