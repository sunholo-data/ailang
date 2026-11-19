package vm_test

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runner"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/types"
)

// End-to-end property test of the exhaustiveness CHECK against both ENGINES
// (M-V1-MATCH-EXHAUSTIVENESS M2). internal/eval's differential test pins the
// algorithm on Core patterns; this one goes through source text, the parser,
// the elaborator, the type checker's constructor identities and universes,
// and the run time, so a wiring fault (a wrong universe, a lost stamp, a
// pattern form converted differently) shows up as a disagreement.
//
// Each trial writes a module with two random non-recursive ADTs, a random
// scrutinee type (an ADT, Option, a tuple, a record, a list of bool, int or
// string) and a random arm set whose guards are all `if false`, plus a main
// that applies the match to EVERY value of the type (exactly: lists to length
// 3, as no pattern looks deeper than two elements, and the int/string domains
// include a value no literal names). With such guards the check's verdict
// must equal the engines':
//
//   - `check` accepts  ⇔ no value reaches RT002, on the evaluator and on the
//     strict VM (no false error; no accepted program that fails at run time);
//   - `check` rejects  ⇒ the error is TC_MATCH_002 at the match's position,
//     and RT002 fires there on both engines when the check is switched off;
//   - an arm reported unreachable (TC_MATCH_003) is never selected.
//
// A second family uses `if true` guards: the check must still never accept a
// program that fails at run time (guarded arms do not count, so a rejection
// is allowed even when the guard always holds).
func TestMatchExhaustivenessCheckAgreesWithEngines(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("AILANG_NO_CACHE", "1")
	trials := 160
	rng := rand.New(rand.NewSource(20261111))
	stats := map[string]int{}
	for i := 0; i < trials; i++ {
		g := newMatchGen(rng)
		guardTrue := i%4 == 3
		name := fmt.Sprintf("mxp%d", i)
		src := g.module(name, guardTrue)
		t.Run(name, func(t *testing.T) {
			checkMatchTrial(t, name, src, guardTrue, stats)
		})
	}
	t.Logf("trials %d: %v", trials, stats)
	if stats["accepted"] == 0 || stats["rejected"] == 0 || stats["redundant"] == 0 {
		t.Errorf("the generator must exercise accepted, rejected and redundant cases: %v", stats)
	}
}

func checkMatchTrial(t *testing.T, name, src string, guardTrue bool, stats map[string]int) {
	file := filepath.Join(t.TempDir(), name+".ail")
	if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	res, cerr := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: true}, pipeline.Source{Filename: file, Code: src})
	if cerr == nil && len(res.Errors) > 0 {
		cerr = res.Errors[0]
	}
	var redundant []int
	for _, w := range res.Warnings {
		if mw, ok := w.(*types.MatchWarning); ok {
			redundant = append(redundant, mw.Arm)
		}
	}
	if cerr != nil {
		var me *types.MatchError
		if !errors.As(cerr, &me) || me.Code != types.TC_MATCH_002 {
			t.Fatalf("check failed with something other than TC_MATCH_002: %v\n%s", cerr, src)
		}
		stats["rejected"]++
		restore := types.SetMatchExhaustivenessErrorsForTest(false)
		defer restore()
	} else {
		stats["accepted"]++
	}
	if len(redundant) > 0 {
		stats["redundant"]++
	}
	for _, strict := range []bool{false, true} {
		out, rerr := runMatchModule(t, file, src, strict)
		var mf *ailerrors.MatchFailureError
		failed := errors.As(rerr, &mf)
		if rerr != nil && !failed {
			t.Fatalf("strict=%v: run failed with %v\n%s", strict, rerr, src)
		}
		switch {
		case cerr == nil && failed:
			t.Fatalf("UNSOUND: check accepted, strict=%v run hit %v\n%s", strict, mf, src)
		case cerr != nil && !failed && !guardTrue:
			t.Fatalf("FALSE ERROR: check said %v, but every value matched on strict=%v (output %s)\n%s", cerr, strict, out, src)
		case cerr != nil && failed:
			var me *types.MatchError
			errors.As(cerr, &me)
			if !strings.HasSuffix(mf.Pos, fmt.Sprintf(":%d:%d", me.Pos.Line, me.Pos.Column)) {
				t.Errorf("strict=%v: RT002 at %s, TC_MATCH_002 at %s", strict, mf.Pos, me.Pos)
			}
		}
		if !failed && !guardTrue {
			selected := map[string]bool{}
			flat := strings.ReplaceAll(strings.TrimSpace(out), "][", ", ")
			for _, a := range strings.Split(strings.Trim(flat, "[]"), ", ") {
				selected[a] = true
			}
			for _, r := range redundant {
				if selected[fmt.Sprint(r)] {
					t.Fatalf("arm %d was reported unreachable but was selected (strict=%v): %s\n%s", r, strict, out, src)
				}
			}
		}
	}
}

// runMatchModule compiles and runs main on one engine.
func runMatchModule(t *testing.T, file, src string, strict bool) (string, error) {
	t.Helper()
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: true}, pipeline.Source{Filename: file, Code: src})
	if err != nil {
		t.Fatalf("compile for run: %v", err)
	}
	rt := runtime.NewModuleRuntime(".")
	if res.DictReg != nil {
		rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	err = runner.ExecuteModuleEntrypoint(rt, runner.ModuleExecParams{Filename: file, Iface: res.Interface, Modules: res.Modules,
		Entry: "main", ArgsJSON: "null", Quiet: true, Print: true, BytecodeMode: strict, StrictBytecode: strict, PipelineResult: &res})
	w.Close()
	os.Stdout = old
	buf := make([]byte, 1<<20)
	n, _ := r.Read(buf)
	r.Close()
	return string(buf[:n]), err
}

// --- generator ---

type mty struct {
	kind  string // "bool", "adt", "opt", "tuple", "rec", "list", "int", "str"
	adt   *madt
	elems []*mty // opt: [a]; tuple: elements; rec: x, y; list: [bool]
}

type mctor struct {
	name string
	args []*mty
}

type madt struct {
	name  string
	ctors []mctor
}

type matchGen struct {
	rng  *rand.Rand
	adts []*madt
	vars int
}

func newMatchGen(rng *rand.Rand) *matchGen {
	g := &matchGen{rng: rng}
	boolT := &mty{kind: "bool"}
	a := &madt{name: "A"}
	for i := 0; i < 2+rng.Intn(2); i++ {
		c := mctor{name: fmt.Sprintf("A%d", i)}
		for j := 0; j < rng.Intn(3); j++ {
			c.args = append(c.args, boolT)
		}
		a.ctors = append(a.ctors, c)
	}
	b := &madt{name: "B"}
	aT := &mty{kind: "adt", adt: a}
	for i := 0; i < 1+rng.Intn(3); i++ {
		c := mctor{name: fmt.Sprintf("B%d", i)}
		for j := 0; j < rng.Intn(3); j++ {
			c.args = append(c.args, []*mty{boolT, aT, {kind: "opt", elems: []*mty{boolT}}}[rng.Intn(3)])
		}
		b.ctors = append(b.ctors, c)
	}
	if len(b.ctors) == 1 && len(b.ctors[0].args) == 0 {
		// `type B = B0` declares an alias of a type named B0, not an ADT.
		b.ctors[0].args = []*mty{aT}
	}
	g.adts = []*madt{a, b}
	return g
}

func (g *matchGen) scrutineeType() *mty {
	boolT := &mty{kind: "bool"}
	aT := &mty{kind: "adt", adt: g.adts[0]}
	bT := &mty{kind: "adt", adt: g.adts[1]}
	switch g.rng.Intn(9) {
	case 0:
		return aT
	case 1:
		return bT
	case 2:
		return &mty{kind: "opt", elems: []*mty{bT}}
	case 3:
		return &mty{kind: "tuple", elems: []*mty{aT, boolT}}
	case 4:
		return &mty{kind: "rec", elems: []*mty{boolT, aT}}
	case 5:
		return &mty{kind: "list", elems: []*mty{boolT}}
	case 6:
		return &mty{kind: "int"}
	case 7:
		return &mty{kind: "str"}
	}
	return &mty{kind: "tuple", elems: []*mty{boolT, &mty{kind: "opt", elems: []*mty{aT}}}}
}

func (t *mty) src() string {
	switch t.kind {
	case "bool":
		return "bool"
	case "adt":
		return t.adt.name
	case "opt":
		return "Option[" + t.elems[0].src() + "]"
	case "tuple":
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = e.src()
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case "rec":
		return "{x: " + t.elems[0].src() + ", y: " + t.elems[1].src() + "}"
	case "list":
		return "[" + t.elems[0].src() + "]"
	case "int":
		return "int"
	}
	return "string"
}

func (g *matchGen) wild() string {
	if g.rng.Intn(2) == 0 {
		return "_"
	}
	g.vars++
	return fmt.Sprintf("v%d", g.vars)
}

// pattern returns a random pattern of type t; depth bounds nesting.
func (g *matchGen) pattern(t *mty, depth int) string {
	if depth <= 0 || g.rng.Intn(5) == 0 {
		return g.wild()
	}
	switch t.kind {
	case "bool":
		return []string{"true", "false"}[g.rng.Intn(2)]
	case "adt":
		c := t.adt.ctors[g.rng.Intn(len(t.adt.ctors))]
		if len(c.args) == 0 {
			return c.name
		}
		args := make([]string, len(c.args))
		for i, a := range c.args {
			args[i] = g.pattern(a, depth-1)
		}
		return c.name + "(" + strings.Join(args, ", ") + ")"
	case "opt":
		if g.rng.Intn(3) == 0 {
			return "None"
		}
		return "Some(" + g.pattern(t.elems[0], depth-1) + ")"
	case "tuple":
		parts := make([]string, len(t.elems))
		for i, e := range t.elems {
			parts[i] = g.pattern(e, depth-1)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case "rec":
		switch g.rng.Intn(3) {
		case 0:
			return "{x: " + g.pattern(t.elems[0], depth-1) + ", ...}"
		case 1:
			return "{y: " + g.pattern(t.elems[1], depth-1) + ", ...}"
		}
		return "{x: " + g.pattern(t.elems[0], depth-1) + ", y: " + g.pattern(t.elems[1], depth-1) + "}"
	case "list":
		e := func() string { return g.pattern(t.elems[0], depth-1) }
		switch g.rng.Intn(6) {
		case 0:
			return "[]"
		case 1:
			return "[" + e() + "]"
		case 2:
			return "[" + e() + ", " + e() + "]"
		case 3:
			g.vars++
			return fmt.Sprintf("[%s, ...r%d]", e(), g.vars)
		case 4:
			g.vars++
			return fmt.Sprintf("[%s, %s, ...r%d]", e(), e(), g.vars)
		}
		g.vars++
		return fmt.Sprintf("%s :: t%d", e(), g.vars)
	case "int":
		return fmt.Sprint(g.rng.Intn(3))
	}
	return []string{`"a"`, `"b"`}[g.rng.Intn(2)]
}

// values enumerates every value of t (lists to length 3; the int and string
// domains carry one value no pattern names).
func (g *matchGen) values(t *mty) []string {
	switch t.kind {
	case "bool":
		return []string{"true", "false"}
	case "adt":
		var out []string
		for _, c := range t.adt.ctors {
			if len(c.args) == 0 {
				out = append(out, c.name)
				continue
			}
			for _, args := range g.product(c.args) {
				out = append(out, c.name+"("+strings.Join(args, ", ")+")")
			}
		}
		return out
	case "opt":
		out := []string{"None"}
		for _, v := range g.values(t.elems[0]) {
			out = append(out, "Some("+v+")")
		}
		return out
	case "tuple":
		var out []string
		for _, vs := range g.product(t.elems) {
			out = append(out, "("+strings.Join(vs, ", ")+")")
		}
		return out
	case "rec":
		var out []string
		for _, vs := range g.product(t.elems) {
			out = append(out, "{x: "+vs[0]+", y: "+vs[1]+"}")
		}
		return out
	case "list":
		out := []string{"[]"}
		prev := [][]string{{}}
		for n := 1; n <= 3; n++ {
			var next [][]string
			for _, p := range prev {
				for _, v := range g.values(t.elems[0]) {
					next = append(next, append(append([]string(nil), p...), v))
				}
			}
			for _, l := range next {
				out = append(out, "["+strings.Join(l, ", ")+"]")
			}
			prev = next
		}
		return out
	case "int":
		return []string{"0", "1", "2", "77"}
	}
	return []string{`"a"`, `"b"`, `"zz"`}
}

func (g *matchGen) product(ts []*mty) [][]string {
	out := [][]string{{}}
	for _, t := range ts {
		var next [][]string
		for _, p := range out {
			for _, v := range g.values(t) {
				next = append(next, append(append([]string(nil), p...), v))
			}
		}
		out = next
	}
	return out
}

func (g *matchGen) module(name string, guardTrue bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n\n", name)
	for _, a := range g.adts {
		parts := make([]string, len(a.ctors))
		for i, c := range a.ctors {
			parts[i] = c.name
			if len(c.args) > 0 {
				as := make([]string, len(c.args))
				for j, at := range c.args {
					as[j] = at.src()
				}
				parts[i] += "(" + strings.Join(as, ", ") + ")"
			}
		}
		fmt.Fprintf(&b, "type %s = %s\n", a.name, strings.Join(parts, " | "))
	}
	st := g.scrutineeType()
	arms := 1 + g.rng.Intn(5)
	fmt.Fprintf(&b, "\npure func f(v: %s) -> int = match v {\n", st.src())
	for i := 1; i <= arms; i++ {
		guard := ""
		if g.rng.Intn(5) == 0 {
			guard = " if false"
			if guardTrue {
				guard = " if true"
			}
		}
		sep := ","
		if i == arms {
			sep = ""
		}
		fmt.Fprintf(&b, "  %s%s => %d%s\n", g.pattern(st, 3), guard, i, sep)
	}
	b.WriteString("}\n\n")
	// The calls are split into chunks of 12: one function with a hundred
	// calls exceeds the VM's register window (BC001).
	vals := g.values(st)
	var chunks []string
	for start := 0; start < len(vals); start += 12 {
		end := min(start+12, len(vals))
		calls := make([]string, 0, end-start)
		for _, v := range vals[start:end] {
			calls = append(calls, "f("+v+")")
		}
		fn := fmt.Sprintf("c%d", len(chunks))
		fmt.Fprintf(&b, "pure func %s() -> string = show([%s])\n", fn, strings.Join(calls, ", "))
		chunks = append(chunks, "${"+fn+"()}")
	}
	fmt.Fprintf(&b, "\nexport func main() -> string = %q\n", strings.Join(chunks, ""))
	return b.String()
}
