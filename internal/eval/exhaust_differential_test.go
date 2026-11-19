package eval

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types/exhaust"
)

// Differential property test of the exhaustiveness checker against the
// evaluator's own matcher (m-v1-match-exhaustiveness.md §9 M1).
//
// Random small types (non-recursive ADTs of ≤4 constructors, lists, tuples,
// records, bools, ints; depth ≤3) and random arm sets (some guarded) are
// generated as Core patterns, converted with exhaust.FromCore, and checked.
// Every value of the type is enumerated — exactly: patterns never look more
// than two list elements deep, so lists up to length 3 represent all lengths,
// and the int domain {0,1,2,99} has a value no literal pattern names. Then:
//
//   - Exhaustive ⇔ every value is matched by some unguarded arm (matchPattern);
//   - every witness, instantiated, is matched by no unguarded arm;
//   - arm i is redundant ⇔ no value reaches it (matched by arm i and by no
//     earlier unguarded arm).

type dkind int

const (
	dBool dkind = iota
	dInt
	dADT
	dList
	dTuple
	dRecord
)

type dctor struct {
	name string
	args []*dtype
}

type dtype struct {
	kind   dkind
	adt    string // dADT: type identity
	ctors  []dctor
	elem   *dtype   // dList
	elems  []*dtype // dTuple
	fields []string // dRecord, sorted
	ftypes []*dtype
}

type dgen struct {
	r     *rand.Rand
	nADT  int
	nCtor int
	univ  exhaust.StaticUniverse
}

func (g *dgen) typ(depth int) *dtype {
	k := dkind(g.r.Intn(6))
	if depth == 0 {
		k = dkind(g.r.Intn(2))
	}
	t := &dtype{kind: k}
	switch k {
	case dADT:
		t.adt = fmt.Sprintf("T%d", g.nADT)
		g.nADT++
		var sigs []exhaust.CtorSig
		for i, n := 0, 1+g.r.Intn(4); i < n; i++ {
			c := dctor{name: fmt.Sprintf("C%d", g.nCtor)}
			g.nCtor++
			for j, m := 0, g.r.Intn(3); j < m; j++ {
				c.args = append(c.args, g.typ(depth-1))
			}
			t.ctors = append(t.ctors, c)
			sigs = append(sigs, exhaust.CtorSig{Name: c.name, Arity: len(c.args)})
		}
		g.univ[t.adt] = sigs
	case dList:
		t.elem = g.typ(depth - 1)
	case dTuple:
		for i, n := 0, 2+g.r.Intn(2); i < n; i++ {
			t.elems = append(t.elems, g.typ(depth-1))
		}
	case dRecord:
		for i, n := 0, 1+g.r.Intn(3); i < n; i++ {
			t.fields = append(t.fields, string(rune('a'+i)))
			t.ftypes = append(t.ftypes, g.typ(depth-1))
		}
	}
	return t
}

// values enumerates every value of t (lists up to length 3). It gives up
// (nil, false) beyond limit values, so the enumeration is always complete.
func values(t *dtype, limit int) ([]Value, bool) {
	switch t.kind {
	case dBool:
		return []Value{&BoolValue{Value: false}, &BoolValue{Value: true}}, true
	case dInt:
		return []Value{&IntValue{Value: 0}, &IntValue{Value: 1}, &IntValue{Value: 2}, &IntValue{Value: 99}}, true
	case dADT:
		var out []Value
		for _, c := range t.ctors {
			combos, ok := product(c.args, limit)
			if !ok {
				return nil, false
			}
			for _, fs := range combos {
				out = append(out, &TaggedValue{TypeName: t.adt, CtorName: c.name, Fields: fs})
			}
			if len(out) > limit {
				return nil, false
			}
		}
		return out, true
	case dList:
		out := []Value{&ListValue{}}
		for n := 1; n <= 3; n++ {
			ts := make([]*dtype, n)
			for i := range ts {
				ts[i] = t.elem
			}
			combos, ok := product(ts, limit)
			if !ok {
				return nil, false
			}
			for _, es := range combos {
				out = append(out, &ListValue{Elements: es})
			}
			if len(out) > limit {
				return nil, false
			}
		}
		return out, true
	case dTuple:
		combos, ok := product(t.elems, limit)
		if !ok {
			return nil, false
		}
		out := make([]Value, len(combos))
		for i, es := range combos {
			out[i] = &TupleValue{Elements: es}
		}
		return out, true
	case dRecord:
		combos, ok := product(t.ftypes, limit)
		if !ok {
			return nil, false
		}
		out := make([]Value, len(combos))
		for i, fs := range combos {
			m := map[string]Value{}
			for j, f := range t.fields {
				m[f] = fs[j]
			}
			out[i] = &RecordValue{Fields: m}
		}
		return out, true
	}
	panic("unreachable dkind")
}

func product(ts []*dtype, limit int) ([][]Value, bool) {
	out := [][]Value{{}}
	for _, t := range ts {
		vs, ok := values(t, limit)
		if !ok || len(out)*len(vs) > limit {
			return nil, false
		}
		var next [][]Value
		for _, prefix := range out {
			for _, v := range vs {
				next = append(next, append(append([]Value{}, prefix...), v))
			}
		}
		out = next
	}
	return out, true
}

// pat generates a Core pattern of type t; spreads have at most two prefix
// elements and a cons tail is a variable or a list of at most one element.
func (g *dgen) pat(t *dtype, depth int) core.CorePattern {
	if depth == 0 || g.r.Intn(4) == 0 {
		if g.r.Intn(2) == 0 {
			return &core.WildcardPattern{}
		}
		return &core.VarPattern{Name: "v"}
	}
	switch t.kind {
	case dBool:
		return &core.LitPattern{Value: g.r.Intn(2) == 0}
	case dInt:
		return &core.LitPattern{Value: int64(g.r.Intn(3))}
	case dADT:
		c := t.ctors[g.r.Intn(len(t.ctors))]
		p := &core.ConstructorPattern{Name: c.name, TypeName: t.adt}
		for _, a := range c.args {
			p.Args = append(p.Args, g.pat(a, depth-1))
		}
		return p
	case dList:
		n := g.r.Intn(3)
		lp := &core.ListPattern{}
		for i := 0; i < n; i++ {
			lp.Elements = append(lp.Elements, g.pat(t.elem, depth-1))
		}
		switch g.r.Intn(3) {
		case 0: // fixed length
		case 1:
			var tail core.CorePattern = &core.VarPattern{Name: "rest"}
			lp.Tail = &tail
		default:
			if n <= 1 {
				inner := &core.ListPattern{}
				if g.r.Intn(2) == 0 {
					inner.Elements = []core.CorePattern{g.pat(t.elem, depth-1)}
				}
				var tail core.CorePattern = inner
				lp.Tail = &tail
			}
		}
		return lp
	case dTuple:
		tp := &core.TuplePattern{}
		for _, e := range t.elems {
			tp.Elements = append(tp.Elements, g.pat(e, depth-1))
		}
		return tp
	case dRecord:
		rp := &core.RecordPattern{Fields: map[string]core.CorePattern{}, Rest: true}
		for i, f := range t.fields {
			if len(rp.Fields) == 0 || g.r.Intn(2) == 0 {
				rp.Fields[f] = g.pat(t.ftypes[i], depth-1)
			}
		}
		return rp
	}
	panic("unreachable dkind")
}

// instantiate builds a value matching witness w (a wildcard takes the first
// value of its type; an int wildcard excluding literals takes 99).
func instantiate(t *dtype, w exhaust.Pattern) Value {
	switch w := w.(type) {
	case exhaust.Wild:
		vs, _ := values(t, 1<<20)
		if t.kind == dInt {
			return &IntValue{Value: 99}
		}
		return vs[0]
	case exhaust.Ctor:
		for _, c := range t.ctors {
			if c.name == w.Name {
				fs := make([]Value, len(c.args))
				for i, a := range c.args {
					fs[i] = instantiate(a, w.Args[i])
				}
				return &TaggedValue{TypeName: t.adt, CtorName: c.name, Fields: fs}
			}
		}
	case exhaust.Lit:
		if b, ok := w.Value.(bool); ok {
			return &BoolValue{Value: b}
		}
	case exhaust.List:
		lv := &ListValue{}
		for _, e := range w.Elems {
			lv.Elements = append(lv.Elements, instantiate(t.elem, e))
		}
		return lv
	case exhaust.Tuple:
		tv := &TupleValue{}
		for i, e := range w.Elems {
			tv.Elements = append(tv.Elements, instantiate(t.elems[i], e))
		}
		return tv
	case exhaust.Record:
		rv := &RecordValue{Fields: map[string]Value{}}
		for i, f := range t.fields {
			fp, ok := w.Fields[f]
			if !ok {
				fp = exhaust.Wild{}
			}
			rv.Fields[f] = instantiate(t.ftypes[i], fp)
		}
		return rv
	}
	panic(fmt.Sprintf("cannot instantiate %#v at %#v", w, t))
}

func matchesAny(ps []core.CorePattern, v Value) bool {
	for _, p := range ps {
		if _, ok := matchPattern(p, v); ok {
			return true
		}
	}
	return false
}

func TestExhaustivenessDifferential(t *testing.T) {
	const trials = 4000
	checked, nonExh, redundant := 0, 0, 0
	for seed := int64(0); checked < trials && seed < trials*20; seed++ {
		g := &dgen{r: rand.New(rand.NewSource(seed)), univ: exhaust.StaticUniverse{}}
		ty := g.typ(1 + g.r.Intn(3))
		vals, ok := values(ty, 3000)
		if !ok {
			continue
		}
		var arms []exhaust.Arm
		var corePats []core.CorePattern
		guarded := map[int]bool{}
		for i, n := 0, 1+g.r.Intn(5); i < n; i++ {
			cp := g.pat(ty, 3)
			p, err := exhaust.FromCore(cp)
			if err != nil {
				t.Fatalf("seed %d: FromCore: %v", seed, err)
			}
			gd := g.r.Intn(5) == 0
			guarded[i] = gd
			arms = append(arms, exhaust.Arm{Pattern: p, Guarded: gd})
			corePats = append(corePats, cp)
		}
		res, err := exhaust.Check(arms, g.univ, exhaust.Options{})
		if err != nil {
			t.Fatalf("seed %d: Check: %v", seed, err)
		}
		checked++
		describe := func() string {
			var b strings.Builder
			for i := range corePats {
				fmt.Fprintf(&b, "\n  arm %d (guarded=%v): %s", i, guarded[i], exhaust.Render(arms[i].Pattern))
			}
			return b.String()
		}

		var unguarded []core.CorePattern
		reached := make([]bool, len(corePats))
		allCovered := true
		var uncovered Value
		for _, v := range vals {
			var before []core.CorePattern
			covered := false
			for i, p := range corePats {
				if _, ok := matchPattern(p, v); ok && !matchesAny(before, v) {
					reached[i] = true
				}
				if !guarded[i] {
					before = append(before, p)
					if _, ok := matchPattern(p, v); ok {
						covered = true
					}
				}
			}
			if !covered && allCovered {
				allCovered, uncovered = false, v
			}
		}
		for i, p := range corePats {
			if !guarded[i] {
				unguarded = append(unguarded, p)
			}
		}

		if res.Exhaustive != allCovered {
			t.Fatalf("seed %d: checker exhaustive=%v, evaluator all-covered=%v (uncovered: %v; witnesses %v)%s",
				seed, res.Exhaustive, allCovered, uncovered, res.Missing, describe())
		}
		if !res.Exhaustive {
			nonExh++
		}
		for _, w := range res.Missing {
			v := instantiate(ty, w)
			if matchesAny(unguarded, v) {
				t.Fatalf("seed %d: witness %s (value %v) is matched by an unguarded arm%s", seed, exhaust.Describe(w), v, describe())
			}
		}
		isRed := map[int]bool{}
		for _, i := range res.Redundant {
			isRed[i] = true
			redundant++
		}
		for i := range corePats {
			if isRed[i] == reached[i] {
				t.Fatalf("seed %d: arm %d redundant=%v but reachable-by-evaluator=%v%s", seed, i, isRed[i], reached[i], describe())
			}
		}
	}
	if checked < trials {
		t.Fatalf("only %d trials had enumerable types", checked)
	}
	t.Logf("%d trials: %d non-exhaustive, %d redundant arms", checked, nonExh, redundant)
}
