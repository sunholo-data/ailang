package exhaust

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

var testUniverse = StaticUniverse{
	"Color":  {{"Red", 0}, {"Green", 0}, {"Blue", 0}},
	"Option": {{"Some", 1}, {"None", 0}},
	"Result": {{"Ok", 1}, {"Err", 1}},
	"Shape":  {{"Circle", 1}, {"Rect", 2}},
	"Day":    {{"Mon", 0}, {"Tue", 0}, {"Wed", 0}, {"Thu", 0}, {"Fri", 0}, {"Sat", 0}, {"Sun", 0}},
	// Same constructor names as Color, different type: must not mix.
	"Light": {{"Red", 0}, {"Green", 0}, {"Blue", 0}},
}

// Pattern builders.
var w = Wild{}

func c(typ, name string, args ...Pattern) Ctor { return Ctor{TypeName: typ, Name: name, Args: args} }
func col(name string) Ctor                     { return c("Color", name) }
func some(p Pattern) Ctor                      { return c("Option", "Some", p) }
func none() Ctor                               { return c("Option", "None") }
func ok(p Pattern) Ctor                        { return c("Result", "Ok", p) }
func er(p Pattern) Ctor                        { return c("Result", "Err", p) }
func l(v any) Lit                              { return Lit{Value: v} }
func lst(ps ...Pattern) List                   { return List{Elems: ps} }
func spread(rest Pattern, ps ...Pattern) List  { return List{Elems: ps, Rest: rest} }
func tup(ps ...Pattern) Tuple                  { return Tuple{Elems: ps} }
func rec(kv ...any) Record {
	r := Record{Fields: map[string]Pattern{}}
	for i := 0; i < len(kv); i += 2 {
		r.Fields[kv[i].(string)] = kv[i+1].(Pattern)
	}
	return r
}
func arms(ps ...Pattern) []Arm {
	out := make([]Arm, len(ps))
	for i, p := range ps {
		out[i] = Arm{Pattern: p}
	}
	return out
}
func guarded(p Pattern) Arm { return Arm{Pattern: p, Guarded: true} }

type checkCase struct {
	name      string
	arms      []Arm
	missing   []string // Describe() of each witness; nil = exhaustive
	redundant []int
}

// One negative and one positive case per pattern form of design §4.1, plus
// the nesting, guard, list-length and record cases M1 names.
var checkCases = []checkCase{
	// wildcard / variable
	{name: "wildcard covers", arms: arms(w)},
	{name: "no arms", arms: nil, missing: []string{"_"}},

	// constructors
	{name: "ctor missing Blue", arms: arms(col("Red"), col("Green")), missing: []string{"Blue"}},
	{name: "ctor all", arms: arms(col("Blue"), col("Red"), col("Green"))},
	{name: "ctor all then wildcard redundant", arms: arms(col("Red"), col("Green"), col("Blue"), w), redundant: []int{3}},
	{name: "ctor repeated arm redundant", arms: arms(col("Red"), col("Red"), w), redundant: []int{1}},
	{name: "ctor missing several in declaration order", arms: arms(col("Green")), missing: []string{"Red", "Blue"}},
	{name: "witnesses capped at 5", arms: arms(c("Day", "Wed")), missing: []string{"Mon", "Tue", "Thu", "Fri", "Sat"}},
	{name: "ctor args wildcard", arms: arms(some(w), none())},
	{name: "ctor missing None", arms: arms(some(w)), missing: []string{"None"}},

	// nested constructors
	{name: "nested Some(Err(_))", arms: arms(some(ok(w)), none()), missing: []string{"Some(Err(_))"}},
	{name: "nested covered", arms: arms(some(ok(w)), some(er(w)), none())},
	{name: "nested two-arg ctor", arms: arms(c("Shape", "Circle", w), c("Shape", "Rect", l(true), w)), missing: []string{"Rect(false, _)"}},

	// bool: finite
	{name: "bool missing false", arms: arms(l(true)), missing: []string{"false"}},
	{name: "bool both", arms: arms(l(false), l(true))},

	// int / float / string: infinite
	{name: "int literals never complete", arms: arms(l(0), l(1)), missing: []string{"_ (any int other than 0, 1)"}},
	{name: "int64 literal same as int", arms: arms(l(0), l(int64(0)), w), redundant: []int{1}},
	{name: "int with catch-all", arms: arms(l(0), w)},
	{name: "string literals", arms: arms(l("a")), missing: []string{`_ (any string other than "a")`}},
	{name: "float literals", arms: arms(l(1.0), l(2.5)), missing: []string{"_ (any float other than 1.0, 2.5)"}},
	{name: "many literals listed at most five", arms: arms(l(1), l(2), l(3), l(4), l(5), l(6)), missing: []string{"_ (any int other than 1, 2, 3, 4, 5, …)"}},

	// lists
	{name: "list missing []", arms: arms(lst(w), lst(w, w)), missing: []string{"[]"}},
	{name: "list missing long", arms: arms(lst(), lst(w)), missing: []string{"[_, _, ..._]"}},
	{name: "list nil and cons", arms: arms(lst(), spread(w, w))},
	{name: "cons only", arms: arms(spread(w, w)), missing: []string{"[]"}},
	{name: "nil only", arms: arms(lst()), missing: []string{"[_, ..._]"}},
	{name: "fixed lengths then spread", arms: arms(lst(), lst(w), spread(w, w, w))},
	{name: "spread after cover redundant", arms: arms(lst(), spread(w, w), lst(w)), redundant: []int{2}},
	{name: "list element nested", arms: arms(lst(), spread(w, l(true))), missing: []string{"[false, ..._]"}},

	// tuples
	{name: "tuple missing (false, false)", arms: arms(tup(l(true), w), tup(w, l(true))), missing: []string{"(false, false)"}},
	{name: "tuple covered", arms: arms(tup(l(true), w), tup(l(false), w))},

	// unit
	{name: "unit literal", arms: arms(l(nil))},
	{name: "unit empty tuple", arms: arms(tup())},

	// records: one constructor, fields = union named in the column
	{name: "record disjoint fields", arms: arms(rec("a", l(true)), rec("b", l(true))), missing: []string{"{a: false, b: false, ...}"}},
	{name: "record one field", arms: arms(rec("a", l(true))), missing: []string{"{a: false, ...}"}},
	{name: "record covered", arms: arms(rec("a", l(true)), rec("a", l(false), "b", w))},
	{name: "record disjoint covered", arms: arms(rec("a", l(true)), rec("b", l(false)), rec("a", l(false), "b", l(true)))},
	{name: "record all-wild redundant", arms: arms(rec("a", w), rec("b", l(true))), redundant: []int{1}},

	// guards never cover; guarded arms are still checked for reachability
	{name: "guarded arm does not cover", arms: []Arm{guarded(col("Red")), {Pattern: col("Green")}, {Pattern: col("Blue")}}, missing: []string{"Red"}},
	{name: "guarded-only arms", arms: []Arm{guarded(w), guarded(w)}, missing: []string{"_"}},
	{name: "guard then fallback", arms: []Arm{guarded(col("Red")), {Pattern: w}}},
	{name: "guarded arm after cover redundant", arms: []Arm{{Pattern: w}, guarded(col("Red"))}, redundant: []int{1}},
	{name: "guarded arm does not shadow later arm", arms: []Arm{guarded(w), {Pattern: col("Red")}, {Pattern: col("Green")}, {Pattern: col("Blue")}}},
}

func TestCheck(t *testing.T) {
	for _, tc := range checkCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Check(tc.arms, testUniverse, Options{})
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			var got []string
			for _, m := range res.Missing {
				got = append(got, Describe(m))
			}
			if res.Exhaustive != (tc.missing == nil) || !reflect.DeepEqual(got, tc.missing) {
				t.Errorf("exhaustive=%v missing=%q, want missing=%q", res.Exhaustive, got, tc.missing)
			}
			if !reflect.DeepEqual(res.Redundant, tc.redundant) {
				t.Errorf("redundant=%v, want %v", res.Redundant, tc.redundant)
			}
		})
	}
}

func TestCheckFailsClosed(t *testing.T) {
	cases := []struct {
		name string
		arms []Arm
		want error
	}{
		{"unknown type", arms(c("Nope", "X")), ErrUnknownType},
		{"ctor not in universe", arms(c("Color", "Purple")), ErrUnknownType},
		{"ctor arity", arms(c("Option", "Some")), ErrIllTyped},
		{"two families in one column", arms(l(1), l("a")), ErrIllTyped},
		{"two ADTs in one column", arms(col("Red"), c("Light", "Green")), ErrIllTyped},
		{"tuple arities differ", arms(tup(w, w), tup(w, w, w)), ErrIllTyped},
		{"unsupported literal", arms(l(int32(1))), nil},
		{"nil pattern", []Arm{{}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Check(tc.arms, testUniverse, Options{})
			if err == nil {
				t.Fatalf("want an error, got %+v", res)
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCheckBudget(t *testing.T) {
	_, err := Check(arms(col("Red"), col("Green")), testUniverse, Options{MaxSteps: 2})
	if !errors.Is(err, ErrBudget) {
		t.Fatalf("err = %v, want ErrBudget", err)
	}
}

// Every rendered witness must be a pattern the parser accepts (design §12,
// "witness rendering drifts from parseable syntax").
func TestWitnessesParse(t *testing.T) {
	for _, tc := range checkCases {
		res, err := Check(tc.arms, testUniverse, Options{})
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range res.Missing {
			pat := Render(m)
			src := "module t\n\npure func f(x: int) -> int = match x {\n  " + pat + " => 0\n}\n"
			p := parser.New(lexer.New(src, "t.ail"))
			p.ParseFile()
			if errs := p.Errors(); len(errs) > 0 {
				t.Errorf("%s: witness %q does not parse: %v", tc.name, pat, errs[0])
			}
		}
	}
}

func TestRender(t *testing.T) {
	cases := map[string]Pattern{
		"Blue":            col("Blue"),
		"Some(Err(_))":    some(er(w)),
		"[]":              lst(),
		"[_, ..._]":       spread(w, w),
		"(false, false)":  tup(l(false), l(false)),
		"{a: false, ...}": rec("a", l(false), "b", w),
		"_":               rec("a", w),
		"()":              l(nil),
		`"a\n"`:           l("a\n"),
		"2.0":             l(2.0),
	}
	for want, p := range cases {
		if got := Render(p); got != want {
			t.Errorf("Render(%#v) = %q, want %q", p, got, want)
		}
	}
	if !strings.HasPrefix(Describe(Wild{Domain: "int", Excluding: []string{"0"}}), "_ (any int other than 0)") {
		t.Error("Describe dropped the excluded literals")
	}
}

// `x :: []` and `x :: [y, ...r]` are one list: Render does not print the
// tail as a nested list (`[_, ...[]]`).
func TestRenderConsOntoList(t *testing.T) {
	cases := map[string]Pattern{
		"[_]":          List{Elems: []Pattern{w}, Rest: lst()},
		"[_, _, ..._]": List{Elems: []Pattern{w}, Rest: List{Elems: []Pattern{w}, Rest: Wild{}}},
		"[_, _]":       List{Elems: []Pattern{w}, Rest: List{Elems: []Pattern{w}, Rest: lst()}},
		"[_, ..._]":    List{Elems: []Pattern{w}, Rest: Wild{}},
	}
	for want, p := range cases {
		if got := Render(p); got != want {
			t.Errorf("Render(%#v) = %q, want %q", p, got, want)
		}
	}
}

// A row of wildcards closes the missing-pattern search at once: an
// 18-column bool tuple, each column covered by its own two arms, then `_`.
func TestMissingStopsAtAWildcardRow(t *testing.T) {
	const n = 18
	var rows []Arm
	for c := 0; c < n; c++ {
		for _, b := range []bool{true, false} {
			elems := make([]Pattern, n)
			for i := range elems {
				elems[i] = w
			}
			elems[c] = l(b)
			rows = append(rows, Arm{Pattern: Tuple{Elems: elems}})
		}
	}
	res, err := missingOnly(append(rows, Arm{Pattern: w}))
	if err != nil || len(res) != 0 {
		t.Fatalf("got %v, %v", err, res)
	}
}

// missingOnly runs the witness search alone, with a small budget the old
// exhaustive descent (2^18 vectors) blows through.
func missingOnly(arms []Arm) ([]row, error) {
	n := &normalizer{u: testUniverse, cache: map[string][]CtorSig{}}
	rows := make([]row, len(arms))
	for i, a := range arms {
		p, err := n.pattern(a.Pattern)
		if err != nil {
			return nil, err
		}
		rows[i] = row{p}
	}
	c := &checker{max: 10_000}
	return c.missing(rows, 1, DefaultMaxWitnesses)
}
