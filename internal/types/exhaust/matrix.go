package exhaust

import (
	"fmt"
	"sort"
)

// row is one pattern vector of the matrix.
type row []*pat

// column summarises the first column of a matrix (plus, for usefulness, the
// first pattern of the query vector): its family, the distinct constructors
// it names, and — for records — the union of the fields named in it.
type column struct {
	fam    family
	heads  []*head // distinct, in first-seen order
	fields []string
}

func analyse(rows []row, extra *pat) (column, error) {
	var c column
	seen := map[string]bool{}
	add := func(p *pat) error {
		if p.wild {
			return nil
		}
		h := p.h
		if c.fam == famNone {
			c.fam = h.fam
		} else if c.fam != h.fam {
			return fmt.Errorf("%w: %s and %s patterns", ErrIllTyped, c.fam, h.fam)
		}
		if h.fam == famADT && len(c.heads) > 0 && (c.heads[0].typ != h.typ || !sameSigs(c.heads[0].sigs, h.sigs)) {
			return fmt.Errorf("%w: constructors %s and %s belong to different types", ErrIllTyped, c.heads[0].name, h.name)
		}
		if h.fam == famTuple && len(c.heads) > 0 && c.heads[0].arity != h.arity {
			return fmt.Errorf("%w: tuples of %d and %d elements", ErrIllTyped, c.heads[0].arity, h.arity)
		}
		if h.fam == famRecord {
			c.fields = unionFields(c.fields, h.fields)
		}
		if !seen[h.name] {
			seen[h.name] = true
			c.heads = append(c.heads, h)
		}
		return nil
	}
	for _, r := range rows {
		if err := add(r[0]); err != nil {
			return c, err
		}
	}
	if extra != nil {
		if err := add(extra); err != nil {
			return c, err
		}
	}
	return c, nil
}

func sameSigs(a, b []CtorSig) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) > 0 && &a[0] == &b[0] {
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func unionFields(a, b []string) []string {
	if len(a) == 0 {
		return b
	}
	set := make(map[string]bool, len(a)+len(b))
	for _, f := range a {
		set[f] = true
	}
	grew := false
	for _, f := range b {
		if !set[f] {
			set[f] = true
			grew = true
		}
	}
	if !grew {
		return a
	}
	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// complete reports whether the column's constructors cover its family.
func (c column) complete() bool {
	switch c.fam {
	case famNone:
		return false
	case famADT:
		return len(c.heads) == len(c.heads[0].sigs)
	case famBool, famList:
		return len(c.heads) == 2
	case famTuple, famUnit, famRecord:
		return true
	}
	return false // infinite literal families
}

// allHeads lists every constructor of a complete column's family, in
// declaration order, with a record's arity widened to the column's fields.
func (c column) allHeads() []*head {
	switch c.fam {
	case famADT:
		h0 := c.heads[0]
		out := make([]*head, len(h0.sigs))
		for i, s := range h0.sigs {
			out[i] = &head{fam: famADT, name: s.Name, arity: s.Arity, sigs: h0.sigs, typ: h0.typ}
		}
		return out
	case famBool:
		return []*head{{fam: famBool, name: "false"}, {fam: famBool, name: "true"}}
	case famList:
		return []*head{{fam: famList, name: nilName}, {fam: famList, name: consName, arity: 2}}
	case famRecord:
		return []*head{c.recordHead()}
	}
	return []*head{c.heads[0]} // tuple, unit
}

func (c column) recordHead() *head {
	return &head{fam: famRecord, arity: len(c.fields), fields: c.fields}
}

// missingHeads lists the constructors of an incomplete column's family that
// it does not name, as witness patterns (wildcard arguments). For an
// infinite literal family the one witness is a wildcard excluding the
// literals named.
func (c column) missingHeads() []*pat {
	present := make(map[string]bool, len(c.heads))
	for _, h := range c.heads {
		present[h.name] = true
	}
	var out []*pat
	switch {
	case c.fam.infinite():
		excl := make([]string, len(c.heads))
		for i, h := range c.heads {
			excl[i] = h.name
		}
		return []*pat{{wild: true, exclFam: c.fam, excl: excl}}
	case c.fam == famADT, c.fam == famBool, c.fam == famList:
		for _, h := range c.allHeads() {
			if !present[h.name] {
				out = append(out, &pat{h: h, args: wilds(h.arity)})
			}
		}
	}
	return out
}

func wilds(n int) []*pat {
	out := make([]*pat, n)
	for i := range out {
		out[i] = wildPat
	}
	return out
}

// args returns p's sub-patterns under constructor h (p's head equals h, or
// p is a wildcard). A record pattern is widened to h's fields; an unnamed
// field is a wildcard.
func args(p *pat, h *head) []*pat {
	if p.wild {
		return wilds(h.arity)
	}
	if h.fam != famRecord || len(p.h.fields) == len(h.fields) {
		return p.args
	}
	out := make([]*pat, len(h.fields))
	j := 0
	for i, f := range h.fields {
		if j < len(p.h.fields) && p.h.fields[j] == f {
			out[i] = p.args[j]
			j++
		} else {
			out[i] = wildPat
		}
	}
	return out
}

// checkHead is analyse's type-consistency check for a constructor query,
// against the column's first constructor only: the column itself was checked
// as a whole when its rows were analysed. It keeps a literal query against a
// long literal column linear rather than building the column summary.
func checkHead(rows []row, h *head) error {
	for _, r := range rows {
		if r[0].wild {
			continue
		}
		o := r[0].h
		switch {
		case o.fam != h.fam:
			return fmt.Errorf("%w: %s and %s patterns", ErrIllTyped, o.fam, h.fam)
		case h.fam == famADT && (o.typ != h.typ || !sameSigs(o.sigs, h.sigs)):
			return fmt.Errorf("%w: constructors %s and %s belong to different types", ErrIllTyped, o.name, h.name)
		case h.fam == famTuple && o.arity != h.arity:
			return fmt.Errorf("%w: tuples of %d and %d elements", ErrIllTyped, o.arity, h.arity)
		}
		return nil
	}
	return nil
}

// specialize is S(c, P): the rows whose first pattern is c or a wildcard,
// with that pattern replaced by its arguments.
func specialize(rows []row, h *head) []row {
	var out []row // lazily: most literal specialisations keep few rows
	for _, r := range rows {
		if !r[0].wild && !r[0].h.same(h) {
			continue
		}
		out = append(out, specializeRow(r, h))
	}
	return out
}

func specializeRow(r row, h *head) row {
	if h.arity == 0 {
		return r[1:] // rows are never mutated, so sharing the tail is safe
	}
	a := args(r[0], h)
	nr := make(row, 0, len(a)+len(r)-1)
	nr = append(nr, a...)
	return append(nr, r[1:]...)
}

// defaultMatrix is D(P): the rows whose first pattern is a wildcard, without
// it.
func defaultMatrix(rows []row) []row {
	var out []row
	for _, r := range rows {
		if r[0].wild {
			out = append(out, r[1:])
		}
	}
	return out
}

// checker carries the step budget across one match's recursion.
type checker struct {
	steps, max int
}

func (c *checker) tick() error {
	c.steps++
	if c.steps > c.max {
		return fmt.Errorf("%w (%d steps)", ErrBudget, c.max)
	}
	return nil
}

// useful is U(P, q): is there a value matched by q and by no row of P?
func (c *checker) useful(rows []row, q row) (bool, error) {
	if err := c.tick(); err != nil {
		return false, err
	}
	if len(q) == 0 {
		return len(rows) == 0, nil
	}
	if !q[0].wild {
		h := q[0].h
		if h.fam == famRecord {
			col, err := analyse(rows, q[0])
			if err != nil {
				return false, err
			}
			h = col.recordHead()
		} else if err := checkHead(rows, h); err != nil {
			return false, err
		}
		return c.useful(specialize(rows, h), specializeRow(q, h))
	}
	col, err := analyse(rows, nil)
	if err != nil {
		return false, err
	}
	if col.complete() {
		for _, h := range col.allHeads() {
			ok, err := c.useful(specialize(rows, h), specializeRow(q, h))
			if ok || err != nil {
				return ok, err
			}
		}
		return false, nil
	}
	return c.useful(defaultMatrix(rows), q[1:])
}

// missing returns up to limit pattern vectors of width n that no row of P
// matches (Maranget §3.1, "computing missing patterns"); none means P is
// exhaustive.
func (c *checker) missing(rows []row, n, limit int) ([]row, error) {
	if err := c.tick(); err != nil {
		return nil, err
	}
	if n == 0 {
		if len(rows) == 0 {
			return []row{{}}, nil
		}
		return nil, nil
	}
	// A row of wildcards matches every vector: nothing is missing. Without
	// this the search below specialises through every constructor of every
	// column first (an 18-column bool tuple ending in `_` hit the budget).
	for _, r := range rows {
		if allWild(r) {
			return nil, nil
		}
	}
	col, err := analyse(rows, nil)
	if err != nil {
		return nil, err
	}
	var out []row
	if col.complete() {
		for _, h := range col.allHeads() {
			ws, err := c.missing(specialize(rows, h), h.arity+n-1, limit-len(out))
			if err != nil {
				return nil, err
			}
			for _, w := range ws {
				out = append(out, append(row{{h: h, args: w[:h.arity]}}, w[h.arity:]...))
			}
			if len(out) >= limit {
				break
			}
		}
		return out, nil
	}
	ws, err := c.missing(defaultMatrix(rows), n-1, limit)
	if err != nil || len(ws) == 0 {
		return nil, err
	}
	heads := []*pat{wildPat}
	if col.fam != famNone {
		heads = col.missingHeads()
	}
	for _, h := range heads {
		for _, w := range ws {
			out = append(out, append(row{h}, w...))
			if len(out) >= limit {
				return out, nil
			}
		}
	}
	return out, nil
}

func allWild(r row) bool {
	for _, p := range r {
		if !p.wild {
			return false
		}
	}
	return true
}
