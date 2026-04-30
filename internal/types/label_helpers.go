package types

import "fmt"

// TLabelled wraps any Type with an information-flow label.
// Use WithLabel / LabelOf / StripLabel rather than constructing directly.
type TLabelled struct {
	Inner Type
	L     Label
}

func (t *TLabelled) String() string {
	if _, isBot := t.L.(labelBottom); isBot {
		return fmt.Sprintf("%s<⊥>", t.Inner.String())
	}
	// LabelConst.String() returns "<name>" — extract the inner name to avoid "string<<email>>"
	ls := t.L.String()
	if len(ls) >= 2 && ls[0] == '<' && ls[len(ls)-1] == '>' {
		ls = ls[1 : len(ls)-1]
	}
	return fmt.Sprintf("%s<%s>", t.Inner.String(), ls)
}

func (t *TLabelled) Equals(other Type) bool {
	if o, ok := other.(*TLabelled); ok {
		return t.Inner.Equals(o.Inner) && LabelEqual(t.L, o.L)
	}
	return false
}

func (t *TLabelled) Substitute(subs map[string]Type) Type {
	return &TLabelled{Inner: t.Inner.Substitute(subs), L: t.L}
}

// --- Label accessors ---

// LabelOf returns the IFC label carried by t, or ⊥ if t is unlabelled.
func LabelOf(t Type) Label {
	if lt, ok := t.(*TLabelled); ok {
		return lt.L
	}
	return LabelBottom()
}

// StripLabel removes the TLabelled wrapper, returning the underlying type.
func StripLabel(t Type) Type {
	if lt, ok := t.(*TLabelled); ok {
		return lt.Inner
	}
	return t
}

// WithLabel wraps t with label l. Returns t unchanged when l is ⊥ (no-op).
// Replaces an existing label if t is already a TLabelled, to avoid double-nesting.
func WithLabel(t Type, l Label) Type {
	if _, isBot := l.(labelBottom); isBot {
		return StripLabel(t) // strip any existing wrapper too
	}
	base := StripLabel(t) // unwrap before re-wrapping
	return &TLabelled{Inner: base, L: l}
}

// --- Propagation helpers ---

// PurePropagateLabel computes the output label for a pure function application.
// Rule (APP-PURE): output label = returnLabel ⊔ join(LabelOf(arg) for each arg)
func PurePropagateLabel(returnLabel Label, argTypes []Type) Label {
	result := returnLabel
	for _, arg := range argTypes {
		result = LabelJoin(result, LabelOf(arg))
	}
	return result
}

// JoinLabels returns the join of the labels carried by two types.
// Used for match-arm joins, record construction, and let-binding propagation.
func JoinLabels(a, b Type) Label {
	return LabelJoin(LabelOf(a), LabelOf(b))
}

// JoinLabelList returns the join of labels across a slice of types.
func JoinLabelList(types []Type) Label {
	result := LabelBottom()
	for _, t := range types {
		result = LabelJoin(result, LabelOf(t))
	}
	return result
}
