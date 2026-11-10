package exhaust

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
)

// Errors the check fails closed with. ErrBudget maps to TC_MATCH_004; the
// others are internal errors (the program was type-checked, so they mean a
// checker or universe bug, never "assume exhaustive").
var (
	ErrUnknownType = errors.New("exhaust: no constructor universe for type")
	ErrIllTyped    = errors.New("exhaust: patterns in one column do not share a type")
	ErrBudget      = errors.New("exhaust: step budget exceeded")
)

// family is the kind of type a column ranges over; it fixes the
// constructor signature the algorithm uses.
type family uint8

const (
	famNone   family = iota // a wildcard-only column: no constructor seen
	famADT                  // a user ADT: constructors from the Universe
	famBool                 // finite {false, true}
	famInt                  // infinite literal domains: never complete
	famFloat                //
	famString               //
	famList                 // [] | h :: t
	famTuple                // one constructor of arity n
	famUnit                 // one nullary constructor
	famRecord               // one constructor; arity = fields named in the column
)

func (f family) String() string {
	return [...]string{"", "ADT", "bool", "int", "float", "string", "list", "tuple", "unit", "record"}[f]
}

// infinite reports whether the family has infinitely many constructors, so
// that no set of its literals is ever complete.
func (f family) infinite() bool { return f == famInt || f == famFloat || f == famString }

const (
	nilName  = "[]"
	consName = "::"
)

// head is a constructor of some family. For a record, fields lists the field
// names it carries (sorted); args follow that order.
type head struct {
	fam    family
	name   string    // ADT ctor; literal key; nilName/consName; "" otherwise
	arity  int       //
	sigs   []CtorSig // famADT: the type's full constructor list
	typ    string    // famADT: Ctor.TypeName, for rendering
	fields []string  // famRecord
}

func (h *head) same(o *head) bool {
	if h.fam != o.fam {
		return false
	}
	switch h.fam {
	case famTuple, famUnit, famRecord:
		return true // single-constructor families
	}
	return h.name == o.name
}

// pat is the internal pattern: a wildcard or a constructor applied to
// sub-patterns. Lists are desugared to [] / ::, literals to nullary
// constructors of their family.
type pat struct {
	wild bool
	h    *head
	args []*pat
	// witness only: the infinite-domain literals this wildcard excludes.
	exclFam family
	excl    []string
}

var wildPat = &pat{wild: true}

// normalizer converts public Patterns, resolving ADT universes once per type.
type normalizer struct {
	u     Universe
	cache map[string][]CtorSig
}

func (n *normalizer) pattern(p Pattern) (*pat, error) {
	switch p := p.(type) {
	case nil:
		return nil, fmt.Errorf("exhaust: nil pattern")
	case Wild:
		return wildPat, nil
	case *Wild:
		return wildPat, nil
	case Ctor:
		return n.ctor(p)
	case *Ctor:
		return n.ctor(*p)
	case Lit:
		return lit(p.Value)
	case *Lit:
		return lit(p.Value)
	case List:
		return n.list(p)
	case *List:
		return n.list(*p)
	case Tuple:
		return n.tuple(p.Elems)
	case *Tuple:
		return n.tuple(p.Elems)
	case Record:
		return n.record(p.Fields)
	case *Record:
		return n.record(p.Fields)
	}
	return nil, fmt.Errorf("exhaust: unknown pattern form %T", p)
}

func (n *normalizer) ctor(c Ctor) (*pat, error) {
	sigs, ok := n.cache[c.TypeName]
	if !ok {
		sigs, ok = n.u.CtorsOf(c)
		if !ok || len(sigs) == 0 {
			return nil, fmt.Errorf("%w %q (constructor %s)", ErrUnknownType, c.TypeName, c.Name)
		}
		n.cache[c.TypeName] = sigs
	}
	arity := -1
	for _, s := range sigs {
		if s.Name == c.Name {
			arity = s.Arity
			break
		}
	}
	if arity < 0 {
		return nil, fmt.Errorf("%w: constructor %s is not one of type %q's constructors", ErrUnknownType, c.Name, c.TypeName)
	}
	if arity != len(c.Args) {
		return nil, fmt.Errorf("%w: constructor %s takes %d arguments, pattern has %d", ErrIllTyped, c.Name, arity, len(c.Args))
	}
	args, err := n.all(c.Args)
	if err != nil {
		return nil, err
	}
	return &pat{h: &head{fam: famADT, name: c.Name, arity: arity, sigs: sigs, typ: c.TypeName}, args: args}, nil
}

func (n *normalizer) list(l List) (*pat, error) {
	var tail *pat
	if l.Rest == nil {
		tail = &pat{h: &head{fam: famList, name: nilName}}
	} else {
		var err error
		if tail, err = n.pattern(l.Rest); err != nil {
			return nil, err
		}
	}
	for i := len(l.Elems) - 1; i >= 0; i-- {
		e, err := n.pattern(l.Elems[i])
		if err != nil {
			return nil, err
		}
		tail = &pat{h: &head{fam: famList, name: consName, arity: 2}, args: []*pat{e, tail}}
	}
	return tail, nil
}

func (n *normalizer) tuple(elems []Pattern) (*pat, error) {
	if len(elems) == 0 {
		return &pat{h: &head{fam: famUnit}}, nil
	}
	args, err := n.all(elems)
	if err != nil {
		return nil, err
	}
	return &pat{h: &head{fam: famTuple, arity: len(args)}, args: args}, nil
}

func (n *normalizer) record(fields map[string]Pattern) (*pat, error) {
	names := make([]string, 0, len(fields))
	for f := range fields {
		names = append(names, f)
	}
	sort.Strings(names)
	args := make([]*pat, len(names))
	for i, f := range names {
		a, err := n.pattern(fields[f])
		if err != nil {
			return nil, err
		}
		args[i] = a
	}
	return &pat{h: &head{fam: famRecord, arity: len(names), fields: names}, args: args}, nil
}

func (n *normalizer) all(ps []Pattern) ([]*pat, error) {
	out := make([]*pat, len(ps))
	for i, p := range ps {
		q, err := n.pattern(p)
		if err != nil {
			return nil, err
		}
		out[i] = q
	}
	return out, nil
}

// lit converts a literal value to a nullary constructor of its family. The
// key is canonical, so 1 (int) and 1 (int64) are the same constructor.
func lit(v any) (*pat, error) {
	var h head
	switch v := v.(type) {
	case nil:
		h = head{fam: famUnit}
	case bool:
		h = head{fam: famBool, name: strconv.FormatBool(v)}
	case int:
		h = head{fam: famInt, name: strconv.FormatInt(int64(v), 10)}
	case int64:
		h = head{fam: famInt, name: strconv.FormatInt(v, 10)}
	case float64:
		if math.IsNaN(v) {
			// NaN never equals itself: the pattern matches nothing. Keep it a
			// distinct, never-complete literal so it covers nothing.
			h = head{fam: famFloat, name: "NaN"}
		} else {
			h = head{fam: famFloat, name: strconv.FormatFloat(v, 'g', -1, 64)}
		}
	case string:
		h = head{fam: famString, name: v}
	default:
		return nil, fmt.Errorf("exhaust: unsupported literal %T", v)
	}
	return &pat{h: &h}, nil
}
