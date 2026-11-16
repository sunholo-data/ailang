package types

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// Bounded rendering of types in diagnostics (#2329 review).
//
// A type error through a doubling alias (`type P0[a] = P1[(a, a)]` ...,
// `pure func f(x: P0[int]) -> int = x`) mismatches the expanded type, and
// `String()` writes that type out in full: 2^n leaves, ~460 KB of message at
// n=16 and gigabytes at n=30, allocated while the error is being built. The
// diagnostic sites render types through displayType instead. A type of up to
// displayTypeNodes nodes renders exactly as String() does; a larger one is
// written depth-first until displayTypeChars characters and then cut with an
// ellipsis and its size, so the cost is bounded in time and memory however
// large the type. String() itself is unchanged: interface digests and keys
// still see the whole type (types that large are rejected by TC_TYPE_SIZE_001
// before they reach an interface).

const (
	displayTypeNodes = 4096 // render exactly up to this tree size
	displayTypeChars = 2048 // budget for a larger type
)

// displayType renders t for a diagnostic, bounded (see above).
func displayType(t Type) string {
	if t == nil {
		return "<nil>"
	}
	n := newTypeSizer().size(t)
	if n <= displayTypeNodes {
		return t.String()
	}
	p := &boundedPrinter{budget: displayTypeChars}
	p.write(t)
	return fmt.Sprintf("%s… (a type of %s nodes, shown in part)", p.b.String(), groupDigits(n))
}

// boundedPrinter writes a type depth-first until its budget runs out.
type boundedPrinter struct {
	b      strings.Builder
	budget int
}

func (p *boundedPrinter) full() bool { return p.b.Len() >= p.budget }

func (p *boundedPrinter) str(s string) {
	if p.full() {
		return
	}
	if room := p.budget - p.b.Len(); len(s) > room {
		for room > 0 && !utf8.RuneStart(s[room]) {
			room--
		}
		s = s[:room]
	}
	p.b.WriteString(s)
}

func (p *boundedPrinter) list(open, close string, ts []Type) {
	p.str(open)
	for i, t := range ts {
		if p.full() {
			return
		}
		if i > 0 {
			p.str(", ")
		}
		p.write(t)
	}
	p.str(close)
}

func (p *boundedPrinter) write(t Type) {
	if p.full() {
		return
	}
	switch x := t.(type) {
	case nil:
		p.str("<nil>")
	case *TList:
		p.list("[", "]", []Type{x.Element})
	case *TArray:
		p.list("Array[", "]", []Type{x.Element})
	case *TMap:
		p.list("Map[", "]", []Type{x.Key, x.Value})
	case *TTuple:
		p.list("(", ")", x.Elements)
	case *TApp:
		p.write(x.Constructor)
		p.list("[", "]", x.Args)
	case *TFunc2:
		p.list("(", ") -> ", x.Params)
		p.write(x.Return)
	case *TRecord:
		p.fields(x.TypeName, x.Fields)
	case *TRecordOpen:
		p.fields("", x.Fields)
	case *TCon, *TVar, *TVar2, *RowVar:
		p.str(x.String())
	default:
		p.str("…")
	}
}

func (p *boundedPrinter) fields(name string, fields map[string]Type) {
	if name != "" {
		p.str(name)
		return
	}
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	p.str("{")
	for i, n := range names {
		if p.full() {
			return
		}
		if i > 0 {
			p.str(", ")
		}
		p.str(n + ": ")
		p.write(fields[n])
	}
	p.str("}")
}
