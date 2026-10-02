package eval

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

// One renderer for `show`, shared by the `show` builtin (internal/builtins),
// the REPL/simple evaluator environment (eval_typed_helpers.go, eval_simple.go)
// and the bytecode VM (internal/vm). Each engine describes its own values as ShowNodes and
// RenderShow makes every formatting decision: the depth limit, the 80-column
// elision, float spelling, constructor syntax. Three hand-written renderers had
// drifted apart: the VM printed ADTs as "<adt#0 6>" and never elided (#1453),
// and the REPL quoted strings and spelled floats with %g.

// ShowKind says how RenderShow lays out a node.
type ShowKind uint8

const (
	// ShowText is a leaf already rendered by the engine (int, bool, string,
	// unit, bytes, functions).
	ShowText ShowKind = iota
	// ShowFloat is a float leaf; RenderShow spells it.
	ShowFloat
	// ShowList, ShowArray and ShowTuple render Items in order.
	ShowList
	ShowArray
	ShowTuple
	// ShowRecord renders Names[i]: Items[i], sorted by name.
	ShowRecord
	// ShowCtor renders Text (the constructor name) applied to Items.
	ShowCtor
	// ShowMap renders Items as alternating key, value, in the order given.
	ShowMap
	// ShowSame renders Items[0] at the same depth (an indirection).
	ShowSame
)

// ShowNode is one engine value as RenderShow sees it.
type ShowNode struct {
	Kind  ShowKind
	Text  string
	Float float64
	Items []any
	Names []string
}

const (
	maxDepth = 3
	// ShowMaxWidth is the column past which a rendered container is elided.
	ShowMaxWidth  = 80
	elisionPrefix = 20
	elisionSuffix = 20
)

// RenderShow renders v, using inspect to describe each value.
func RenderShow(v any, inspect func(any) ShowNode) string {
	return renderShow(v, 0, inspect)
}

func renderShow(v any, depth int, inspect func(any) ShowNode) string {
	if depth > maxDepth {
		return "..."
	}
	n := inspect(v)
	child := func(x any) string { return renderShow(x, depth+1, inspect) }
	switch n.Kind {
	case ShowFloat:
		return showFloat(n.Float)
	case ShowList:
		return showSeq(n.Items, child, "[", "]")
	case ShowArray:
		return showSeq(n.Items, child, "#[", "]")
	case ShowTuple:
		return showSeq(n.Items, child, "(", ")")
	case ShowRecord:
		if len(n.Names) == 0 {
			return "{}"
		}
		idx := make([]int, len(n.Names))
		for i := range idx {
			idx[i] = i
		}
		sort.Slice(idx, func(a, b int) bool { return n.Names[idx[a]] < n.Names[idx[b]] })
		parts := make([]string, len(idx))
		for i, j := range idx {
			parts[i] = n.Names[j] + ": " + child(n.Items[j])
		}
		return truncateIfNeeded("{" + strings.Join(parts, ", ") + "}")
	case ShowCtor:
		if len(n.Items) == 0 {
			return n.Text
		}
		parts := make([]string, len(n.Items))
		for i, x := range n.Items {
			parts[i] = child(x)
		}
		return n.Text + "(" + strings.Join(parts, ", ") + ")"
	case ShowMap:
		// Map{...} is deliberate debug notation: AILANG has no map literal.
		parts := make([]string, 0, len(n.Items)/2)
		for i := 0; i+1 < len(n.Items); i += 2 {
			parts = append(parts, child(n.Items[i])+": "+child(n.Items[i+1]))
		}
		return truncateIfNeeded("Map{" + strings.Join(parts, ", ") + "}")
	case ShowSame:
		return renderShow(n.Items[0], depth, inspect)
	}
	return n.Text
}

func showFloat(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Inf"
	case math.IsInf(f, -1):
		return "-Inf"
	}
	// 'f' keeps the decimal point visible; whole numbers get ".0".
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

func showSeq(items []any, child func(any) string, open, close string) string {
	parts := make([]string, len(items))
	for i, x := range items {
		parts[i] = child(x)
	}
	return truncateIfNeeded(open + strings.Join(parts, ", ") + close)
}

// truncateIfNeeded elides the middle of long strings to keep under ShowMaxWidth.
func truncateIfNeeded(s string) string {
	if len(s) <= ShowMaxWidth || elisionPrefix+elisionSuffix+3 >= len(s) {
		return s
	}
	return s[:elisionPrefix] + "..." + s[len(s)-elisionSuffix:]
}

// Show renders v the way the `show` builtin does.
func Show(v Value) string { return renderShow(v, 0, inspectShow) }

// ShowAt renders v as if it were nested depth levels deep.
func ShowAt(v Value, depth int) string { return renderShow(v, depth, inspectShow) }

func showItems(vs []Value) []any {
	out := make([]any, len(vs))
	for i, x := range vs {
		out[i] = x
	}
	return out
}

// inspectShow describes an evaluator value for RenderShow.
func inspectShow(x any) ShowNode {
	switch val := x.(type) {
	case *IntValue:
		return ShowNode{Text: strconv.Itoa(val.Value)}
	case *FloatValue:
		return ShowNode{Kind: ShowFloat, Float: val.Value}
	case *BoolValue:
		return ShowNode{Text: strconv.FormatBool(val.Value)}
	case *StringValue:
		return ShowNode{Text: val.Value} // identity for strings, no quotes
	case *ListValue:
		return ShowNode{Kind: ShowList, Items: showItems(val.Elements)}
	case *ArrayValue:
		return ShowNode{Kind: ShowArray, Items: showItems(val.Elements())}
	case *TupleValue:
		return ShowNode{Kind: ShowTuple, Items: showItems(val.Elements)}
	case *MapValue:
		keys := make([]string, 0, len(val.Entries))
		for key := range val.Entries {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		items := make([]any, 0, 2*len(keys))
		for _, key := range keys {
			items = append(items, val.Entries[key].Key, val.Entries[key].Value)
		}
		return ShowNode{Kind: ShowMap, Items: items}
	case *RecordValue:
		names := make([]string, 0, len(val.Fields))
		items := make([]any, 0, len(val.Fields))
		for k, fv := range val.Fields {
			names = append(names, k)
			items = append(items, fv)
		}
		return ShowNode{Kind: ShowRecord, Names: names, Items: items}
	case *TaggedValue:
		return ShowNode{Kind: ShowCtor, Text: val.CtorName, Items: showItems(val.Fields)}
	case *UnitValue:
		return ShowNode{Text: "()"}
	case *FunctionValue, *BuiltinFunction, *ConstructorClosure:
		return ShowNode{Text: "<function>"}
	case *BytesValue:
		return ShowNode{Text: val.String()}
	case *IndirectValue:
		if val.Cell == nil || !val.Cell.Init || val.Cell.Val == nil {
			return ShowNode{Text: "<uninitialized>"}
		}
		return ShowNode{Kind: ShowSame, Items: []any{val.Cell.Val}}
	case *ErrorValue:
		return ShowNode{Text: "Error: " + val.Message}
	case *tailCall:
		// Never escapes applyFunctionValue's loop; rendered only so a
		// violation of that invariant is visible rather than "<unknown>".
		return ShowNode{Text: val.String()}
	}
	return ShowNode{Text: "<unknown>"}
}
