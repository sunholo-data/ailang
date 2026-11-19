package types

import (
	"fmt"
	"io"
	"strconv"
)

// Formatting a type with fmt (%s, %v, %q) is bounded by construction (#2363
// review): every diagnostic built with fmt.Errorf or fmt.Sprintf formats its
// types through Format, never through an unbounded String(). A type within
// maxTypeTreeNodes — every type a checked program can hold — formats exactly
// as String() does, so ordinary messages are byte-identical; a larger one,
// which can only be met on its way to TC_TYPE_SIZE_001 or inside a type error
// through an alias, is written in part (displayType). String() itself stays
// exact: digests and keys call it directly.
func formatType(t Type, f fmt.State, verb rune) {
	var s string
	if exceedsNodes(t, maxTypeTreeNodes) {
		s = displayType(t)
	} else {
		s = t.String()
	}
	switch verb {
	case 'q':
		s = strconv.Quote(s)
	case 's', 'v':
	default:
		s = "%!" + string(verb) + "(" + s + ")"
	}
	_, _ = io.WriteString(f, s)
}

// exceedsNodes reports whether t's tree has more than limit nodes, visiting at
// most limit+1 of them.
func exceedsNodes(t Type, limit int64) bool {
	c := nodeCounter{limit: limit}
	return c.over(t)
}

type nodeCounter struct{ n, limit int64 }

func (c *nodeCounter) any(ts []Type) bool {
	for _, t := range ts {
		if c.over(t) {
			return true
		}
	}
	return false
}

func (c *nodeCounter) fields(m map[string]Type) bool {
	for _, t := range m {
		if c.over(t) {
			return true
		}
	}
	return false
}

func (c *nodeCounter) over(t Type) bool {
	if t == nil {
		return false
	}
	if c.n++; c.n > c.limit {
		return true
	}
	switch x := t.(type) {
	case *TList:
		return c.over(x.Element)
	case *TArray:
		return c.over(x.Element)
	case *TMap:
		return c.over(x.Key) || c.over(x.Value)
	case *TTuple:
		return c.any(x.Elements)
	case *TApp:
		return c.over(x.Constructor) || c.any(x.Args)
	case *TFunc2:
		return c.any(x.Params) || c.over(x.Return) || (x.EffectRow != nil && c.over(x.EffectRow))
	case *TRecord:
		return c.fields(x.Fields) || c.over(x.Row)
	case *TRecordOpen:
		return c.fields(x.Fields) || c.over(x.Row)
	case *TRecord2:
		return x.Row != nil && c.over(x.Row)
	case *Row:
		return c.fields(x.Labels)
	case *TLabelled:
		return c.over(x.Inner)
	}
	return false
}

// Format makes fmt render the type bounded (formatType).
func (t *TList) Format(f fmt.State, verb rune)       { formatType(t, f, verb) }
func (t *TArray) Format(f fmt.State, verb rune)      { formatType(t, f, verb) }
func (t *TMap) Format(f fmt.State, verb rune)        { formatType(t, f, verb) }
func (t *TTuple) Format(f fmt.State, verb rune)      { formatType(t, f, verb) }
func (t *TRecord) Format(f fmt.State, verb rune)     { formatType(t, f, verb) }
func (t *TRecordOpen) Format(f fmt.State, verb rune) { formatType(t, f, verb) }
func (t *TApp) Format(f fmt.State, verb rune)        { formatType(t, f, verb) }
func (t *TFunc2) Format(f fmt.State, verb rune)      { formatType(t, f, verb) }
func (t *TRecord2) Format(f fmt.State, verb rune)    { formatType(t, f, verb) }
func (r *Row) Format(f fmt.State, verb rune)         { formatType(r, f, verb) }
func (t *TLabelled) Format(f fmt.State, verb rune)   { formatType(t, f, verb) }
