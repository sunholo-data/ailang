package exhaust

// DefaultMaxSteps is the per-match budget of recursive usefulness steps
// (design §7). Usefulness is exponential in the worst case; exceeding the
// budget is ErrBudget (TC_MATCH_004, "add a final `_ =>` arm"), never a
// silent pass.
const DefaultMaxSteps = 1_000_000

// DefaultMaxWitnesses caps how many missing patterns Check reports.
const DefaultMaxWitnesses = 5

// Options tunes Check. Zero values take the defaults.
type Options struct {
	MaxSteps     int
	MaxWitnesses int
}

// Result is the verdict on one match.
type Result struct {
	// Exhaustive is true when the unguarded arms cover every value.
	Exhaustive bool
	// Missing holds up to MaxWitnesses concrete patterns no unguarded arm
	// matches, in constructor declaration order; empty when Exhaustive.
	Missing []Pattern
	// Redundant lists, in order, the indices of arms that can never be
	// reached: every value they match is matched by an earlier unguarded arm.
	Redundant []int
	// Steps is the number of usefulness steps the check took.
	Steps int
}

// Check decides exhaustiveness and redundancy for a match with the given
// arms over a single scrutinee. Guarded arms do not count toward coverage
// (their guard may be false) but are checked for their own reachability.
//
// Any error means the verdict is unknown and the caller must fail closed.
func Check(arms []Arm, u Universe, opts Options) (*Result, error) {
	if opts.MaxSteps <= 0 {
		opts.MaxSteps = DefaultMaxSteps
	}
	if opts.MaxWitnesses <= 0 {
		opts.MaxWitnesses = DefaultMaxWitnesses
	}
	n := &normalizer{u: u, cache: map[string][]CtorSig{}}
	c := &checker{max: opts.MaxSteps}
	res := &Result{}

	all := make([]row, len(arms))
	for i, a := range arms {
		p, err := n.pattern(a.Pattern)
		if err != nil {
			return nil, err
		}
		all[i] = row{p}
	}
	// A scrutinee of a single-constructor type (tuple, record, unit) is
	// expanded once up front: every redundancy query would otherwise
	// re-specialise — re-copy — the whole matrix by that one constructor.
	top, err := analyse(all, nil)
	if err != nil {
		return nil, err
	}
	var single *head
	width := 1
	if top.fam == famTuple || top.fam == famRecord || top.fam == famUnit {
		single = top.allHeads()[0]
		width = single.arity
		for i := range all {
			all[i] = specializeRow(all[i], single)
		}
	}

	var unguarded []row
	for i, q := range all {
		ok, err := c.useful(unguarded, q)
		if err != nil {
			return nil, err
		}
		if !ok {
			res.Redundant = append(res.Redundant, i)
		}
		if !arms[i].Guarded {
			unguarded = append(unguarded, q)
		}
	}

	ws, err := c.missing(unguarded, width, opts.MaxWitnesses)
	if err != nil {
		return nil, err
	}
	res.Exhaustive = len(ws) == 0
	for _, w := range ws {
		p := w[0]
		if single != nil {
			p = &pat{h: single, args: w}
		}
		res.Missing = append(res.Missing, public(p))
	}
	res.Steps = c.steps
	return res, nil
}

// public converts an internal witness back to surface form, re-sugaring
// [] / :: chains into list patterns.
func public(p *pat) Pattern {
	if p.wild {
		if p.exclFam != famNone {
			return Wild{Domain: p.exclFam.String(), Excluding: renderLits(p.exclFam, p.excl)}
		}
		return Wild{}
	}
	h := p.h
	switch h.fam {
	case famADT:
		args := make([]Pattern, len(p.args))
		for i, a := range p.args {
			args[i] = public(a)
		}
		return Ctor{TypeName: h.typ, Name: h.name, Args: args}
	case famBool:
		return Lit{Value: h.name == "true"}
	case famUnit:
		return Lit{Value: nil}
	case famTuple:
		elems := make([]Pattern, len(p.args))
		for i, a := range p.args {
			elems[i] = public(a)
		}
		return Tuple{Elems: elems}
	case famRecord:
		fields := make(map[string]Pattern, len(h.fields))
		for i, f := range h.fields {
			fields[f] = public(p.args[i])
		}
		return Record{Fields: fields}
	case famList:
		var elems []Pattern
		for !p.wild && p.h.fam == famList && p.h.name == consName {
			elems = append(elems, public(p.args[0]))
			p = p.args[1]
		}
		if !p.wild && p.h.name == nilName {
			return List{Elems: elems}
		}
		return List{Elems: elems, Rest: public(p)}
	}
	// Literal families appear in witnesses only as excluding wildcards.
	return Wild{}
}
