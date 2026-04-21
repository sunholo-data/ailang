package builtins

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
)

// canonicalKey produces a deterministic canonical string representation of a value
// for use as a Go map key. This is NOT a hash — it is a collision-free canonical
// encoding where same value always produces the same key.
//
// Records are serialized with sorted keys to ensure determinism regardless of
// Go map iteration order. Type-tagged prefixes prevent cross-type collisions.
func canonicalKey(v eval.Value) string {
	switch val := v.(type) {
	case *eval.IntValue:
		return fmt.Sprintf("i:%d", val.Value)
	case *eval.FloatValue:
		// Phase 1: %g format is sufficient for string-heavy workloads.
		// Full NaN/-0.0 normalization deferred to Phase 2 (M-HASH-COLLECTIONS).
		return fmt.Sprintf("f:%g", val.Value)
	case *eval.StringValue:
		// Length-prefix the string to avoid collisions like "s:a,b" vs "s:a" + ",b"
		return fmt.Sprintf("s:%d:%s", len(val.Value), val.Value)
	case *eval.BoolValue:
		if val.Value {
			return "b:1"
		}
		return "b:0"
	case *eval.UnitValue:
		return "u"
	case *eval.ListValue:
		var b strings.Builder
		b.WriteString("l:[")
		for i, elem := range val.Elements {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(canonicalKey(elem))
		}
		b.WriteByte(']')
		return b.String()
	case *eval.ArrayValue:
		var b strings.Builder
		b.WriteString("a:[")
		for i, elem := range val.Elements {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(canonicalKey(elem))
		}
		b.WriteByte(']')
		return b.String()
	case *eval.TupleValue:
		var b strings.Builder
		b.WriteString("tp:(")
		for i, elem := range val.Elements {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(canonicalKey(elem))
		}
		b.WriteByte(')')
		return b.String()
	case *eval.RecordValue:
		// Sort keys for determinism — canonical order, not Go map order
		keys := make([]string, 0, len(val.Fields))
		for k := range val.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString("r:{")
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(k)
			b.WriteByte(':')
			b.WriteString(canonicalKey(val.Fields[k]))
		}
		b.WriteByte('}')
		return b.String()
	case *eval.TaggedValue:
		var b strings.Builder
		b.WriteString("t:")
		b.WriteString(val.CtorName)
		b.WriteByte('(')
		for i, field := range val.Fields {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(canonicalKey(field))
		}
		b.WriteByte(')')
		return b.String()
	case *eval.BytesValue:
		return fmt.Sprintf("by:%x", val.Value)
	default:
		// Safe fallback for unknown types (functions, closures, etc.)
		return "x:" + v.String()
	}
}
