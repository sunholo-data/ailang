package smt

import (
	"fmt"
	"strings"
)

// SExpr is a parsed SMT-LIB S-expression: an atom (symbol, numeral, decimal)
// or a string literal, or a list.
type SExpr struct {
	Atom     string   // symbol / numeral / decimal (empty for lists and strings)
	Str      *string  // string literal contents, unescaped ("" quoting only)
	List     []*SExpr // children when IsList
	IsList   bool
	Original string // the exact source text, for display
}

// String renders the expression on one line (canonical spacing).
func (e *SExpr) String() string {
	switch {
	case e == nil:
		return ""
	case e.Str != nil:
		return `"` + strings.ReplaceAll(*e.Str, `"`, `""`) + `"`
	case !e.IsList:
		return e.Atom
	}
	parts := make([]string, len(e.List))
	for i, c := range e.List {
		parts[i] = c.String()
	}
	return "(" + strings.Join(parts, " ") + ")"
}

// Head returns the first atom of a list expression ("" otherwise).
func (e *SExpr) Head() string {
	if e == nil || !e.IsList || len(e.List) == 0 || e.List[0].IsList || e.List[0].Str != nil {
		return ""
	}
	return e.List[0].Atom
}

// ParseSExprs parses every top-level S-expression in src. Quoted |symbols| are
// kept as atoms including the bars; ; comments run to end of line.
func ParseSExprs(src string) ([]*SExpr, error) {
	p := &sexpParser{src: src}
	var out []*SExpr
	for {
		p.skipSpace()
		if p.pos >= len(p.src) {
			return out, nil
		}
		e, err := p.parse()
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
}

type sexpParser struct {
	src string
	pos int
}

func (p *sexpParser) skipSpace() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == ';':
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *sexpParser) parse() (*SExpr, error) {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("sexp: unexpected end of input")
	}
	start := p.pos
	switch c := p.src[p.pos]; {
	case c == '(':
		p.pos++
		e := &SExpr{IsList: true}
		for {
			p.skipSpace()
			if p.pos >= len(p.src) {
				return nil, fmt.Errorf("sexp: unclosed list at %d", start)
			}
			if p.src[p.pos] == ')' {
				p.pos++
				e.Original = p.src[start:p.pos]
				return e, nil
			}
			child, err := p.parse()
			if err != nil {
				return nil, err
			}
			e.List = append(e.List, child)
		}
	case c == ')':
		return nil, fmt.Errorf("sexp: unexpected ')' at %d", p.pos)
	case c == '"':
		p.pos++
		var b strings.Builder
		for {
			if p.pos >= len(p.src) {
				return nil, fmt.Errorf("sexp: unterminated string at %d", start)
			}
			if p.src[p.pos] == '"' {
				if p.pos+1 < len(p.src) && p.src[p.pos+1] == '"' { // "" is an escaped quote
					b.WriteByte('"')
					p.pos += 2
					continue
				}
				p.pos++
				break
			}
			b.WriteByte(p.src[p.pos])
			p.pos++
		}
		s := b.String()
		return &SExpr{Str: &s, Original: p.src[start:p.pos]}, nil
	case c == '|':
		end := strings.IndexByte(p.src[p.pos+1:], '|')
		if end < 0 {
			return nil, fmt.Errorf("sexp: unterminated |symbol| at %d", start)
		}
		p.pos += end + 2
		return &SExpr{Atom: p.src[start:p.pos], Original: p.src[start:p.pos]}, nil
	default:
		for p.pos < len(p.src) {
			c := p.src[p.pos]
			if c == '(' || c == ')' || c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '"' || c == ';' {
				break
			}
			p.pos++
		}
		return &SExpr{Atom: p.src[start:p.pos], Original: p.src[start:p.pos]}, nil
	}
}
