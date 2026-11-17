package runner

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// shadowDump writes every generated shadowing case as its own program, with
// its renamed twin beside it, for a CLI sweep against another binary:
//
//	go test ./internal/runner -run TestShadowGeneratedEngines -args -shadow-dump=DIR
var shadowDump = flag.String("shadow-dump", "", "directory to write the generated shadowing programs to")

// shadowGenSeed and shadowGenCases fix the generated corpus: the same seed
// always yields the same programs.
const (
	shadowGenSeed  = 2375
	shadowGenCases = 360
)

// shadowType is the type of the shadowed name. Each value level k reads as a
// distinct int (1, 20, 300, 4000), so an output names the binding it read.
type shadowType int

const (
	shInt shadowType = iota
	shStr
	shRec
	shFn
	shTypeCount
)

func (t shadowType) String() string { return [...]string{"int", "string", "record", "function"}[t] }

var shadowMagnitude = []int{1, 20, 300, 4000}

func (t shadowType) sig() string { return [...]string{"int", "string", "{a: int}", "int -> int"}[t] }

func (t shadowType) val(k int) string {
	m := shadowMagnitude[k]
	switch t {
	case shInt:
		return fmt.Sprint(m)
	case shStr:
		return fmt.Sprintf("%q", fmt.Sprintf("s%d", k))
	case shRec:
		return fmt.Sprintf("{a: %d}", m)
	}
	return fmt.Sprintf(`\x. x + %d`, m)
}

// use reads a binding of this type as an int.
func (t shadowType) use(name string) string {
	switch t {
	case shInt:
		return name
	case shStr:
		return fmt.Sprintf(`(if %s == "s0" then 1 else if %s == "s1" then 20 else if %s == "s2" then 300 else 4000)`, name, name, name)
	case shRec:
		return name + ".a"
	}
	return name + "(0)"
}

// shBinding is the binding of the shadowed name visible at some point: a
// local or parameter, or the top-level function v (global).
type shBinding struct {
	name   string
	global bool
}

// shadowGen renders one case. With twin set, every binder of the shadowed
// name gets its own name, so the program has no shadowing at all and its
// output is the oracle for the shadowing version: the two are generated from
// the same random stream, so they have the same shape.
type shadowGen struct {
	r     *rand.Rand
	t     shadowType
	depth int
	twin  bool
	n     int      // fresh-name counter for helper names
	vn    int      // fresh-name counter for twin binders
	trail []string // shapes and positions, outermost first
}

func (g *shadowGen) fresh(p string) string {
	g.n++
	return fmt.Sprintf("%s%d", p, g.n)
}

// bind names a new binder of the shadowed name.
func (g *shadowGen) bind() shBinding {
	if !g.twin {
		return shBinding{name: "v"}
	}
	g.vn++
	return shBinding{name: fmt.Sprintf("v_%d", g.vn)}
}

func (g *shadowGen) use(b shBinding) string {
	if b.global {
		return "v()"
	}
	return g.t.use(b.name)
}

var shadowPositions = []string{
	"seq", "closure", "argLater", "argEarlier", "opLater", "opEarlier", "listLater", "listEarlier",
	"tuple", "record", "cond", "scrutinee", "interp", "hofArg", "hofBody", "cmpeq",
}

// eq is a bool expression comparing binding b, at its own type, with the
// level-k value.
func (g *shadowGen) eq(b shBinding, k int) string {
	if b.global {
		return fmt.Sprintf("v() == %d", shadowMagnitude[k])
	}
	switch g.t {
	case shInt:
		return fmt.Sprintf("%s == %d", b.name, shadowMagnitude[k])
	case shStr:
		return fmt.Sprintf("%s == %q", b.name, fmt.Sprintf("s%d", k))
	case shRec:
		return fmt.Sprintf("%s.a == %d", b.name, shadowMagnitude[k])
	}
	return fmt.Sprintf("%s(0) == %d", b.name, shadowMagnitude[k])
}

// inner is an int expression evaluated where cur is the visible binding at
// level k. Below the deepest level it places a deeper shadow in one of the
// positions an expression can take, next to a read of cur by an earlier or
// later sibling: a block there used to float its let over the sibling (#2388).
func (g *shadowGen) inner(k int, cur shBinding) string {
	if k == g.depth {
		if g.r.Intn(2) == 0 {
			return g.use(cur)
		}
		c, z := g.fresh("c"), g.fresh("z")
		return fmt.Sprintf(`{ let %s = \%s. %s + %s; %s(0) }`, c, z, g.use(cur), z, c)
	}
	pos := shadowPositions[g.r.Intn(len(shadowPositions))]
	g.trail = append(g.trail, "@"+pos)
	u := g.use(cur)
	sh := func() string { return g.shadow(k+1, cur) }
	switch pos {
	case "seq":
		w := g.fresh("w")
		return fmt.Sprintf("{ let %s = %s; %s * 7 + %s }", w, sh(), u, w)
	case "closure":
		c, z, w := g.fresh("c"), g.fresh("z"), g.fresh("w")
		return fmt.Sprintf(`{ let %s = \%s. %s + %s; let %s = %s; %s(%s) }`, c, z, u, z, w, sh(), c, w)
	case "argLater":
		return fmt.Sprintf("add2(%s, %s)", sh(), u)
	case "argEarlier":
		return fmt.Sprintf("add2(%s, %s)", u, sh())
	case "opLater":
		return fmt.Sprintf("(%s) * 7 + %s", sh(), u)
	case "opEarlier":
		return fmt.Sprintf("%s * 7 + (%s)", u, sh())
	case "listLater":
		return fmt.Sprintf("lsum([%s, %s])", sh(), u)
	case "listEarlier":
		return fmt.Sprintf("lsum([%s, %s])", u, sh())
	case "tuple":
		a, b := g.fresh("a"), g.fresh("b")
		return fmt.Sprintf("match (%s, %s) { (%s, %s) => %s * 7 + %s }", sh(), u, a, b, a, b)
	case "record":
		rr := g.fresh("rr")
		return fmt.Sprintf("{ let %s = {p: %s, q: %s}; %s.p * 7 + %s.q }", rr, sh(), u, rr, rr)
	case "cond":
		return fmt.Sprintf("(if (%s) > 0 then %s * 7 + 1 else 0)", sh(), u)
	case "scrutinee":
		n := g.fresh("n")
		return fmt.Sprintf("match %s { %s => %s * 7 + %s }", sh(), n, n, u)
	case "interp":
		n := g.fresh("n")
		return fmt.Sprintf(`match stringToInt("${show(%s)}0${show(%s)}") { Some(%s) => %s %% 1000003, None => 0 - 1 }`, sh(), u, n, n)
	case "hofArg":
		z := g.fresh("z")
		return fmt.Sprintf(`ap(\%s. %s + %s, %s)`, z, z, u, sh())
	case "cmpeq":
		// An equality on the shadowed name at its own type, beside a shadow.
		return fmt.Sprintf("(if %s then 1 else 2) * 7 + (%s)", g.eq(cur, k), sh())
	}
	// hofBody: the shadow sits in a lambda body passed to a higher-order call.
	z := g.fresh("z")
	return fmt.Sprintf(`ap(\%s. (%s) * 7 + %s, %s)`, z, sh(), z, u)
}

var shadowShapes = []string{"block", "ifarm", "matchpat", "matcharm", "lamparam", "lambody", "tuplepat", "recpat", "letin", "seqrebind", "letrec", "guardpat", "guardouter", "guardnested"}

// shadow is an int expression that rebinds the name to its level-k value.
func (g *shadowGen) shadow(k int, outer shBinding) string {
	shape := shadowShapes[g.r.Intn(len(shadowShapes))]
	if shape == "tuplepat" && g.t == shFn {
		// A lambda inside a tuple literal fails on the evaluator ("intrinsic
		// operations require OpLowering pass"), a separate bug (#2323).
		shape = "block"
	}
	g.trail = append(g.trail, shape)
	val := g.t.val(k)
	switch shape {
	case "block":
		nb := g.bind()
		return fmt.Sprintf("{ let %s = %s; %s }", nb.name, val, g.inner(k, nb))
	case "ifarm":
		nb := g.bind()
		return fmt.Sprintf("(if b then { let %s = %s; %s } else %s)", nb.name, val, g.inner(k, nb), g.use(outer))
	case "matchpat":
		nb := g.bind()
		return fmt.Sprintf("match Some(%s) { Some(%s) => %s, None => 0 }", val, nb.name, g.inner(k, nb))
	case "matcharm":
		z, nb := g.fresh("z"), g.bind()
		return fmt.Sprintf("match Some(%d) { Some(%s) => { let %s = %s; %s + %s }, None => %s }", k, z, nb.name, val, g.inner(k, nb), z, g.use(outer))
	case "lamparam":
		f, nb := g.fresh("f"), g.bind()
		return fmt.Sprintf(`{ let %s = \%s. %s; %s(%s) }`, f, nb.name, g.inner(k, nb), f, val)
	case "lambody":
		f, z, nb := g.fresh("f"), g.fresh("z"), g.bind()
		return fmt.Sprintf(`{ let %s = \%s. { let %s = %s; %s + %s }; %s(0) }`, f, z, nb.name, val, g.inner(k, nb), z, f)
	case "tuplepat":
		nb := g.bind()
		return fmt.Sprintf("match (%s, 0) { (%s, _) => %s }", val, nb.name, g.inner(k, nb))
	case "recpat":
		o, nb := g.fresh("o"), g.bind()
		field := "v" // field punning: the field is named v, whatever the variable is called
		if nb.name != "v" {
			field = "v: " + nb.name
		}
		return fmt.Sprintf("match {v: %s, %s: 5} { {%s, %s} => %s + %s }", val, o, field, o, g.inner(k, nb), o)
	case "letin":
		nb := g.bind()
		return fmt.Sprintf("(let %s = %s in %s)", nb.name, val, g.inner(k, nb))
	case "seqrebind":
		t, first := g.fresh("t"), g.bind()
		second := g.bind()
		return fmt.Sprintf("{ let %s = %s; let %s = %s; let %s = %s; %s + %s }",
			first.name, g.t.val((k+1)%4), t, g.t.use(first.name), second.name, val, g.inner(k, second), t)
	case "guardpat":
		// The guard reads the arm's own pattern variable.
		nb := g.bind()
		g2 := g.bind()
		return fmt.Sprintf("match Some(%s) { Some(%s) if %s > 0 => %s, Some(%s) => %s * 3, None => 0 }",
			val, nb.name, g.use(nb), g.inner(k, nb), g2.name, g.use(g2))
	case "guardouter":
		// The guard reads the OUTER binding while another arm binds the name.
		z, nb, g2 := g.fresh("z"), g.bind(), g.bind()
		return fmt.Sprintf("match Some(%s) { Some(%s) if %s > 0 => { let %s = %s; %s }, Some(%s) => %s * 3, None => 0 }",
			val, z, g.use(outer), nb.name, z, g.inner(k, nb), g2.name, g.use(g2))
	case "guardnested":
		// A match inside the guard binds the name again.
		nb, g2 := g.bind(), g.bind()
		return fmt.Sprintf("match Some(%s) { Some(%s) if (match Some(%s) { Some(%s) => %s > 0, None => false }) && %s > 0 => %s, _ => %s }",
			val, nb.name, g.t.val((k+1)%4), g2.name, g.use(g2), g.use(nb), g.inner(k, nb), g.use(outer))
	}
	// letrec: the name becomes a recursive function here; nothing nests inside.
	nb := g.bind()
	return fmt.Sprintf("{ letrec %s = \\n. if n <= 0 then 0 else n + %s(n - 1); %s(3) }", nb.name, nb.name, nb.name)
}

var shadowContexts = []string{"local", "param", "tailloop", "letin", "global", "escape", "tailarg", "generic"}

// shadowCase is one generated function (shadowing and twin) plus the main
// lines that print it.
type shadowCase struct {
	name, desc, decls, twinDecls, calls string
}

func renderShadowCase(seed int64, i int, twin bool) (decls, desc string) {
	r := rand.New(rand.NewSource(seed))
	g := &shadowGen{r: r, t: shadowType(r.Intn(int(shTypeCount))), depth: 1 + r.Intn(3), twin: twin}
	ctx := shadowContexts[r.Intn(len(shadowContexts))]
	name := fmt.Sprintf("case%d", i)
	v0 := g.t.val(0)
	outer := shBinding{name: "v"}
	switch ctx {
	case "local":
		decls = fmt.Sprintf(`func %s(b: bool) -> [int] {
  let v = %s;
  let g = \z. %s + z;
  let s = %s;
  let h = \z. %s + z;
  [s, %s, g(0), h(0)]
}`, name, v0, g.use(outer), g.shadow(1, outer), g.use(outer), g.use(outer))
	case "param":
		decls = fmt.Sprintf(`func %s_p(v: %s, b: bool) -> [int] {
  let s = %s;
  let h = \z. %s + z;
  [s, %s, h(0)]
}
func %s(b: bool) -> [int] = %s_p(%s, b)`, name, g.t.sig(), g.shadow(1, outer), g.use(outer), g.use(outer), name, name, v0)
	case "tailloop":
		decls = fmt.Sprintf(`func %s_l(i: int, acc: [int], v: %s, b: bool) -> [int] {
  if i <= 0 then acc else {
    let s = %s;
    %s_l(i - 1, [s + i, %s] ++ acc, v, b)
  }
}
func %s(b: bool) -> [int] = %s_l(3, [], %s, b)`, name, g.t.sig(), g.shadow(1, outer), name, g.use(outer), name, name, v0)
	case "letin":
		decls = fmt.Sprintf("func %s(b: bool) -> [int] =\n  let v = %s in\n  let s = %s in\n  [s, %s]",
			name, v0, g.shadow(1, outer), g.use(outer))
	case "tailarg":
		// The shadow sits in the arguments of the recursive tail call, next
		// to the parameter it shadows.
		decls = fmt.Sprintf(`func %s_l(i: int, acc: [int], v: %s, b: bool) -> [int] {
  if i <= 0 then acc else %s_l(i - 1, [(%s) + i, %s] ++ acc, v, b)
}
func %s(b: bool) -> [int] = %s_l(3, [], %s, b)`, name, g.t.sig(), name, g.shadow(1, outer), g.use(outer), name, name, v0)
	case "generic":
		// A pattern variable and a lambda parameter of a type parameter,
		// named like a concrete local: their comparisons must use their own
		// type (the operator lowering used the local's).
		kv, nb, nb2 := `"k"`, g.bind(), g.bind()
		if g.t != shInt {
			kv = "7"
		}
		decls = fmt.Sprintf(`func %s_g[a](o: Option[a], k2: a, b: bool) -> [int] {
  let v = %s;
  let s = %s;
  let e = match o { Some(%s) => if %s == k2 then 1 else 2, None => 0 };
  let f = \%s. if %s != k2 then 3 else 4;
  [s, e, f(k2), %s]
}
func %s(b: bool) -> [int] = %s_g(Some(%s), %s, b)`, name, v0, g.shadow(1, outer), nb.name, nb.name, nb2.name, nb2.name, g.use(outer), name, name, kv, kv)
	case "global":
		// The name is the top-level function v at level 0: a local that
		// shadows it must not capture the later call.
		outer = shBinding{global: true}
		decls = fmt.Sprintf(`func %s(b: bool) -> [int] {
  let s = %s;
  [s, v()]
}`, name, g.shadow(1, outer))
	default: // escape: a closure built inside the shadowing block outlives it
		w, nb := g.fresh("w"), g.bind()
		decls = fmt.Sprintf(`func %s(b: bool) -> [int] {
  let v = %s;
  let k = { let %s = %s; let %s = %s; \z. %s + z + %s };
  [k(0), %s]
}`, name, v0, nb.name, g.t.val(1), w, g.inner(1, nb), g.use(nb), w, g.use(outer))
	}
	desc = fmt.Sprintf("%s: type=%s context=%s depth=%d shapes=%s", name, g.t, ctx, g.depth, strings.Join(g.trail, ">"))
	return decls, desc
}

func genShadowCase(r *rand.Rand, i int) shadowCase {
	seed := r.Int63()
	decls, desc := renderShadowCase(seed, i, false)
	twin, _ := renderShadowCase(seed, i, true)
	return shadowCase{
		name:      fmt.Sprintf("case%d", i),
		desc:      desc,
		decls:     decls,
		twinDecls: twin,
		calls:     fmt.Sprintf("  println(show(case%d(true)));\n  println(show(case%d(false)))", i, i),
	}
}

const shadowHelpers = `import std/io (println)
import std/string (stringToInt)

func v() -> int = 77

func add2(a: int, b: int) -> int = a * 7 + b

func ap(f: int -> int, x: int) -> int = f(x)

func lsum(xs: [int]) -> int = match xs {
  [] => 0,
  x :: rest => x * 7 + lsum(rest)
}

`

func shadowModule(module string, cases []shadowCase, twin bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n\n%s", module, shadowHelpers)
	for _, c := range cases {
		d := c.decls
		if twin {
			d = c.twinDecls
		}
		fmt.Fprintf(&b, "-- %s\n%s\n\n", c.desc, d)
	}
	// Print in parts: one main with hundreds of calls would exceed the VM's
	// 256-register frame on its own.
	const per = 16
	var parts []string
	for lo := 0; lo < len(cases); lo += per {
		hi := min(lo+per, len(cases))
		calls := make([]string, 0, hi-lo)
		for _, c := range cases[lo:hi] {
			calls = append(calls, c.calls)
		}
		part := fmt.Sprintf("part%d", len(parts))
		fmt.Fprintf(&b, "func %s() -> () ! {IO} {\n%s\n}\n\n", part, strings.Join(calls, ";\n"))
		parts = append(parts, "  "+part+"()")
	}
	fmt.Fprintf(&b, "export func main() -> () ! {IO} {\n%s\n}\n", strings.Join(parts, ";\n"))
	return b.String()
}

// TestShadowGeneratedEngines runs a fixed-seed corpus of shadowing programs:
// a name rebound by a block let, an if or match arm, a pattern variable (with
// guards that read it, read the outer binding, or hold a match of their own),
// a lambda parameter or body, a let-in, a sequential rebinding or a letrec;
// that shadow placed in a statement or in an expression position (call
// argument, operator operand, list, tuple or record element, if condition,
// match scrutinee, interpolation, higher-order argument, lambda body, next to
// an equality) beside a read of the shadowed name; nested up to three deep;
// int, string, record and function values; in a local, a parameter, a tail
// loop (in its body or in the recursive call's arguments), a let-in body, over
// a top-level function of the same name, in an escaping closure, and in a
// generic function whose pattern variable and lambda parameter of a type
// parameter share the name of a concrete local and are compared.
//
// The oracle is each program's twin, generated from the same random stream
// with every binder renamed apart, run on the evaluator: it has no shadowing,
// so neither the VM's flat name space (#2375), the elaborator floating a let
// over its siblings (#2388), nor a compile pass resolving a name to the wrong
// binding can change it. Every engine must print the twin's output for both
// versions.
func TestShadowGeneratedEngines(t *testing.T) {
	r := rand.New(rand.NewSource(shadowGenSeed))
	cases := make([]shadowCase, shadowGenCases)
	for i := range cases {
		cases[i] = genShadowCase(r, i)
	}
	if *shadowDump != "" {
		if err := os.MkdirAll(*shadowDump, 0o755); err != nil {
			t.Fatal(err)
		}
		for i, c := range cases {
			for _, twin := range []bool{false, true} {
				mod := fmt.Sprintf("shgen%03d", i)
				if twin {
					mod += "_twin"
				}
				src := shadowModule(mod, []shadowCase{c}, twin)
				if err := os.WriteFile(filepath.Join(*shadowDump, mod+".ail"), []byte(src), 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	dir := t.TempDir()
	shadowFile, twinFile := filepath.Join(dir, "shadow_gen.ail"), filepath.Join(dir, "shadow_twin.ail")
	shadowSrc, twinSrc := shadowModule("shadow_gen", cases, false), shadowModule("shadow_twin", cases, true)
	want := strings.Split(runOnEngine(t, twinFile, twinSrc, "eval"), "\n")
	if len(want) != 2*len(cases)+1 {
		t.Fatalf("twin on the evaluator printed %d lines, want %d", len(want)-1, 2*len(cases))
	}
	runs := []struct{ label, file, src, engine string }{
		{"shadowing/eval", shadowFile, shadowSrc, "eval"},
		{"shadowing/bytecode", shadowFile, shadowSrc, "bytecode"},
		{"shadowing/strict", shadowFile, shadowSrc, "strict"},
		{"twin/bytecode", twinFile, twinSrc, "bytecode"},
		{"twin/strict", twinFile, twinSrc, "strict"},
	}
	for _, run := range runs {
		got := strings.Split(runOnEngine(t, run.file, run.src, run.engine), "\n")
		if len(got) != len(want) {
			t.Fatalf("%s printed %d lines, oracle %d", run.label, len(got)-1, len(want)-1)
		}
		bad := 0
		for i := 0; i+1 < len(want); i++ {
			if got[i] != want[i] {
				bad++
				c := cases[i/2]
				t.Errorf("%s: %s(b=%v) = %s, oracle %s\n  %s", run.label, c.name, i%2 == 0, got[i], want[i], c.desc)
			}
		}
		if bad > 0 {
			t.Logf("%s: %d of %d lines differ from the oracle", run.label, bad, len(want)-1)
		}
	}
	// The shadowing module, lowered, must also satisfy the invariant itself.
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck}, pipeline.Source{Filename: shadowFile})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("compile %s: %v %v", shadowFile, err, res.Errors)
	}
	if n := checkLoweredUnique(t, "shadow_gen", res, map[string]bool{}); n < len(cases) {
		t.Fatalf("checked %d lowered functions, want at least %d", n, len(cases))
	}
}
