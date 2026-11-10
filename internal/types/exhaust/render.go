package exhaust

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// maxListedLiterals caps how many covered literals a "(any int other than
// …)" note names (design §5).
const maxListedLiterals = 5

// Render prints a pattern in surface syntax that `ailang check` accepts:
// `Blue`, `Some(Err(_))`, `[]`, `[_, ..._]`, `(false, false)`,
// `{a: false, ...}`. A record whose fields are all wildcards renders as `_`;
// any other record renders open.
func Render(p Pattern) string {
	var b strings.Builder
	render(&b, p)
	return b.String()
}

// Describe is Render plus, for a witness standing for the rest of an
// infinite literal domain, which literals are already covered:
// `_ (any int other than 0, 1)`.
func Describe(p Pattern) string {
	s := Render(p)
	if w, ok := p.(Wild); ok && w.Domain != "" && len(w.Excluding) > 0 {
		lits := w.Excluding
		more := ""
		if len(lits) > maxListedLiterals {
			lits, more = lits[:maxListedLiterals], ", …"
		}
		s += fmt.Sprintf(" (any %s other than %s%s)", w.Domain, strings.Join(lits, ", "), more)
	}
	return s
}

func render(b *strings.Builder, p Pattern) {
	switch p := p.(type) {
	case Wild, *Wild, nil:
		b.WriteString("_")
	case Ctor:
		b.WriteString(p.Name)
		if len(p.Args) > 0 {
			b.WriteByte('(')
			renderList(b, p.Args)
			b.WriteByte(')')
		}
	case Lit:
		b.WriteString(renderLit(p.Value))
	case Tuple:
		if len(p.Elems) == 0 {
			b.WriteString("()")
			return
		}
		b.WriteByte('(')
		renderList(b, p.Elems)
		b.WriteByte(')')
	case List:
		b.WriteByte('[')
		renderList(b, p.Elems)
		if p.Rest != nil {
			if len(p.Elems) > 0 {
				b.WriteString(", ")
			}
			// A spread must bind a name (PAT_SPREAD_NEEDS_IDENT); `..._` parses.
			b.WriteString("...")
			render(b, p.Rest)
		}
		b.WriteByte(']')
	case Record:
		// Always open (`, ...`): the matrix knows only the fields the arms
		// name, not the record type's, and a closed pattern would have to
		// list every field to type-check.
		names := make([]string, 0, len(p.Fields))
		for f, fp := range p.Fields {
			if isWild(fp) {
				continue
			}
			names = append(names, f)
		}
		if len(names) == 0 {
			b.WriteString("_")
			return
		}
		sort.Strings(names)
		b.WriteByte('{')
		for i, f := range names {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(f)
			b.WriteString(": ")
			render(b, p.Fields[f])
		}
		b.WriteString(", ...}")
	default:
		fmt.Fprintf(b, "<%T>", p)
	}
}

func renderList(b *strings.Builder, ps []Pattern) {
	for i, p := range ps {
		if i > 0 {
			b.WriteString(", ")
		}
		render(b, p)
	}
}

func isWild(p Pattern) bool {
	switch p.(type) {
	case Wild, *Wild, nil:
		return true
	}
	return false
}

func renderLit(v any) string {
	switch v := v.(type) {
	case nil:
		return "()"
	case string:
		return strconv.Quote(v)
	case float64:
		s := strconv.FormatFloat(v, 'g', -1, 64)
		if !strings.ContainsAny(s, ".eEN") {
			s += ".0"
		}
		return s
	}
	return fmt.Sprint(v)
}

// renderLits renders the keys of an infinite family's literal constructors.
func renderLits(f family, keys []string) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		switch f {
		case famString:
			out[i] = strconv.Quote(k)
		case famFloat:
			if !strings.ContainsAny(k, ".eEN") {
				k += ".0"
			}
			out[i] = k
		default:
			out[i] = k
		}
	}
	return out
}
