package vm

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// M-V1-SHOW-TOTAL: `show` is total, deterministic and injective per type, and
// the evaluator and the bytecode VM render every value byte for byte the same.
// design_docs/planned/v1_0_0/m-v1-show-total.md

// toVMForShow builds the VM value that a VM program would hold for v. Unlike
// the runtime bridge (EvalToBytecode) it covers every shape show can meet —
// user ADTs, closures, maps — and it encodes map keys with the VM's OWN key
// encoder (bytecode.MapKey), so the order a VM map renders in is measured,
// not copied from the evaluator.
func toVMForShow(t testing.TB, v eval.Value) bytecode.Value {
	t.Helper()
	conv := func(xs []eval.Value) []bytecode.Value {
		out := make([]bytecode.Value, len(xs))
		for i, x := range xs {
			out[i] = toVMForShow(t, x)
		}
		return out
	}
	switch x := v.(type) {
	case *eval.IntValue:
		return bytecode.NewInt(int64(x.Value))
	case *eval.FloatValue:
		return bytecode.NewFloat(x.Value)
	case *eval.BoolValue:
		return bytecode.NewBool(x.Value)
	case *eval.StringValue:
		return bytecode.NewString(x.Value)
	case *eval.UnitValue:
		return bytecode.Unit()
	case *eval.ListValue:
		return bytecode.NewList(conv(x.Elements))
	case *eval.TupleValue:
		return bytecode.NewTuple(conv(x.Elements))
	case *eval.ArrayValue:
		if fs, packed := x.Floats(); packed {
			return bytecode.NewFloatArray(fs)
		}
		return bytecode.NewArray(conv(x.Elements()))
	case *eval.RecordValue:
		fields := make([]bytecode.RecordField, 0, len(x.Fields))
		for name, fv := range x.Fields {
			fields = append(fields, bytecode.RecordField{Name: name, Value: toVMForShow(t, fv)})
		}
		return bytecode.NewRecord(fields)
	case *eval.MapValue:
		entries := make(map[string]bytecode.MapEntry, len(x.Entries))
		for _, e := range x.Entries {
			k := toVMForShow(t, e.Key)
			enc, err := bytecode.MapKey(k)
			if err != nil {
				t.Fatalf("VM map key %s: %v", eval.Show(e.Key), err)
			}
			entries[enc] = bytecode.MapEntry{Key: k, Value: toVMForShow(t, e.Value)}
		}
		return bytecode.NewMap(entries)
	case *eval.TaggedValue:
		return bytecode.NewADT(0, x.CtorName, conv(x.Fields))
	case *eval.BytesValue:
		return bytecode.NewBytes(&bytecode.BytesObj{B: x.Value, Filename: x.Filename, MimeType: x.MimeType})
	case *eval.FunctionValue, *eval.BuiltinFunction, *eval.ConstructorClosure:
		return bytecode.NewClosure(nil, nil)
	}
	t.Fatalf("toVMForShow: unsupported %T", v)
	return bytecode.Value{}
}

// --- value builders ---------------------------------------------------------

// Variables, not constants: Go folds 0.1 + 0.2 exactly at compile time.
var pointOne, pointTwo = 0.1, 0.2

func sInt(n int) eval.Value             { return &eval.IntValue{Value: n} }
func sFlt(f float64) eval.Value         { return &eval.FloatValue{Value: f} }
func sStr(s string) eval.Value          { return &eval.StringValue{Value: s} }
func sBool(b bool) eval.Value           { return &eval.BoolValue{Value: b} }
func sList(xs ...eval.Value) eval.Value { return &eval.ListValue{Elements: xs} }
func sTup(xs ...eval.Value) eval.Value  { return &eval.TupleValue{Elements: xs} }
func sCtor(name string, xs ...eval.Value) eval.Value {
	return &eval.TaggedValue{TypeName: "T", CtorName: name, Fields: xs}
}
func sRec(kv ...any) eval.Value {
	r := &eval.RecordValue{Fields: map[string]eval.Value{}}
	for i := 0; i < len(kv); i += 2 {
		r.Fields[kv[i].(string)] = kv[i+1].(eval.Value)
	}
	return r
}
func sMap(t testing.TB, kv ...eval.Value) eval.Value {
	m := &eval.MapValue{Entries: map[string]*eval.MapEntry{}}
	for i := 0; i < len(kv); i += 2 {
		k, err := eval.MapKey(kv[i])
		if err != nil {
			t.Fatal(err)
		}
		m.Entries[k] = &eval.MapEntry{Key: kv[i], Value: kv[i+1]}
	}
	return m
}
func showInts(lo, hi int) []eval.Value {
	out := make([]eval.Value, 0, hi-lo)
	for i := lo; i < hi; i++ {
		out = append(out, sInt(i))
	}
	return out
}
func joinInts(lo, hi int) string {
	parts := make([]string, 0, hi-lo)
	for i := lo; i < hi; i++ {
		parts = append(parts, strconv.Itoa(i))
	}
	return strings.Join(parts, ", ")
}

// showSpecRows is every row of the spec tables in the design doc
// ("Current output per value kind" and "Specification: the new show"), with
// the NEW rendering.
func showSpecRows(t testing.TB) []struct {
	name string
	v    eval.Value
	want string
} {
	tree := sCtor("Leaf")
	for i := 1; i <= 5; i++ {
		tree = sCtor("Node", tree, sInt(i), sCtor("Leaf"))
	}
	nest := sInt(1)
	for i := 0; i < 5; i++ {
		nest = sList(nest)
	}
	ee := make([]eval.Value, 40)
	for i := range ee {
		ee[i] = sStr("éé")
	}
	return []struct {
		name string
		v    eval.Value
		want string
	}{
		{"int", sInt(42), "42"},
		{"negative int", sInt(-7), "-7"},
		{"max int", sInt(math.MaxInt64), "9223372036854775807"},
		{"min int", sInt(math.MinInt64), "-9223372036854775808"},
		{"float", sFlt(3.14), "3.14"},
		{"whole float", sFlt(2.0), "2.0"},
		{"negative zero", sFlt(math.Copysign(0, -1)), "-0.0"},
		{"zero", sFlt(0), "0.0"},
		{"0.1+0.2", sFlt(pointOne + pointTwo), "0.30000000000000004"},
		{"1e300", sFlt(1e300), "1" + strings.Repeat("0", 300) + ".0"},
		{"1e-300", sFlt(1e-300), "0." + strings.Repeat("0", 299) + "1"},
		{"smallest denormal", sFlt(5e-324), "0." + strings.Repeat("0", 323) + "5"},
		{"NaN", sFlt(math.NaN()), "NaN"},
		{"Inf", sFlt(math.Inf(1)), "Inf"},
		{"-Inf", sFlt(math.Inf(-1)), "-Inf"},
		{"true", sBool(true), "true"},
		{"false", sBool(false), "false"},
		{"top-level string is bare", sStr("hi \"q\"\n"), "hi \"q\"\n"},
		{"top-level empty string", sStr(""), ""},
		{"top-level invalid UTF-8 is passed through", sStr("a\xff"), "a\xff"},
		{"unit", &eval.UnitValue{}, "()"},
		{"list", sList(sInt(1), sInt(2), sInt(3)), "[1, 2, 3]"},
		{"empty list", sList(), "[]"},
		{"nested strings quoted", sList(sStr("a"), sStr("b, c")), `["a", "b, c"]`},
		{"empty nested string", sList(sStr(""), sStr("x")), `["", "x"]`},
		{"array", eval.NewArray([]eval.Value{sInt(1), sInt(2), sInt(3)}), "#[1, 2, 3]"},
		{"float array (packed)", eval.NewArray([]eval.Value{sFlt(1.5), sFlt(math.Copysign(0, -1))}), "#[1.5, -0.0]"},
		{"empty array", eval.NewArray(nil), "#[]"},
		{"tuple", sTup(sInt(1), sStr("x"), sBool(true)), `(1, "x", true)`},
		{"record sorted by name", sRec("b", sStr("two"), "a", sInt(1)), `{a: 1, b: "two"}`},
		{"empty record", sRec(), "{}"},
		{"record name byte order", sRec("b", sInt(1), "B", sInt(2), "_a", sInt(3), "a1", sInt(4)), "{B: 2, _a: 3, a1: 4, b: 1}"},
		{"ctor with args", sCtor("Circle", sInt(3)), "Circle(3)"},
		{"nullary ctor", sCtor("Dot"), "Dot"},
		{"ctor two args", sCtor("Rect", sInt(1), sInt(2)), "Rect(1, 2)"},
		{"Some string", sCtor("Some", sStr("x")), `Some("x")`},
		{"None", sCtor("None"), "None"},
		{"Ok", sCtor("Ok", sStr("y")), `Ok("y")`},
		{"Err", sCtor("Err", sStr("e")), `Err("e")`},
		{"5-deep tree, no depth limit", tree, "Node(Node(Node(Node(Node(Leaf, 1, Leaf), 2, Leaf), 3, Leaf), 4, Leaf), 5, Leaf)"},
		{"[[[[[1]]]]], no depth limit", nest, "[[[[[1]]]]]"},
		{"function", &eval.FunctionValue{}, "<function>"},
		{"builtin", &eval.BuiltinFunction{Name: "print"}, "<function>"},
		{"bytes", &eval.BytesValue{Value: []byte("hi")}, "<bytes:6869>"},
		{"bytes over 32, total", &eval.BytesValue{Value: []byte(strings.Repeat("\xab", 40))}, "<bytes:" + strings.Repeat("ab", 40) + ">"},
		{"upload bytes: content, not name", &eval.BytesValue{Value: []byte{0}, Filename: "f.pdf", MimeType: "application/pdf"}, "<bytes:00>"},
		{"empty bytes", &eval.BytesValue{}, "<bytes:>"},
		{"map in encoded-key order", sMap(t, sInt(9), sStr("a"), sInt(10), sStr("b"), sInt(2), sStr("c")), `Map{10: "b", 2: "c", 9: "a"}`},
		{"empty map", sMap(t), "Map{}"},
		{"map with tuple keys", sMap(t, sTup(sInt(1), sStr("t")), sCtor("Some", sFlt(1.5))), `Map{(1, "t"): Some(1.5)}`},
		{"set", sCtor("Set", sMap(t, sStr("b"), &eval.UnitValue{}, sStr("a"), &eval.UnitValue{})), `Set(Map{"a": (), "b": ()})`},
		{"range(0, 50), no width limit", sList(showInts(0, 50)...), "[" + joinInts(0, 50) + "]"},
		{"tuple of two wide lists", sTup(sList(showInts(0, 30)...), sList(showInts(0, 30)...)), "([" + joinInts(0, 30) + "], [" + joinInts(0, 30) + "])"},
		{"40 x éé stays valid UTF-8", sList(ee...), "[" + strings.TrimSuffix(strings.Repeat(`"éé", `, 40), ", ") + "]"},
		{"escapes", sList(sStr(`"`), sStr(`\`), sStr("\n\t\r"), sStr("${x}"), sStr("$y"), sStr("\x00\x7f\u0085\u200b"), sStr("é 😀"), sStr("\xff")),
			`["\"", "\\", "\n\t\r", "\${x}", "$y", "\u{0}\u{7F}\u{85}\u{200B}", "é 😀", "\xFF"]`},
		{"string nested in ctor in record in list", sList(sRec("k", sCtor("Named", sStr("n\"m"), sCtor("Dot")))), `[{k: Named("n\"m", Dot)}]`},
		{"Json JInt", sCtor("JObject", sList(sRec("key", sStr("n"), "value", sCtor("JInt", sInt(3))))), `JObject([{key: "n", value: JInt(3)}])`},
		{"indirect at top level keeps a string bare", &eval.IndirectValue{Cell: &eval.RefCell{Val: sStr("x"), Init: true}}, "x"},
	}
}

// TestShowSpecGolden: every spec row renders as the doc says, through the
// evaluator's renderer and through the VM's, byte for byte.
func TestShowSpecGolden(t *testing.T) {
	for _, row := range showSpecRows(t) {
		got := eval.Show(row.v)
		if got != row.want {
			t.Errorf("%s: evaluator\n got %q\nwant %q", row.name, got, row.want)
		}
		if _, ok := row.v.(*eval.IndirectValue); ok {
			continue // evaluator-only indirection
		}
		if vmGot := showValue(toVMForShow(t, row.v)); vmGot != got {
			t.Errorf("%s: engines differ\n eval %q\n   vm %q", row.name, got, vmGot)
		}
	}
}

// TestShowValidUTF8 is acceptance (3): the old byte-level cut split "é".
func TestShowValidUTF8(t *testing.T) {
	ee := make([]eval.Value, 40)
	for i := range ee {
		ee[i] = sStr("éé")
	}
	for _, v := range []eval.Value{sList(ee...), sList(sStr("\xff\xfe")), sRec("k", sStr("\xc3"))} {
		if s := eval.Show(v); !utf8.ValidString(s) {
			t.Errorf("show is not valid UTF-8: %q", s)
		}
	}
}

// --- linear cost ------------------------------------------------------------

// showAllocBytes reports the bytes allocated by one render of v.
func showAllocBytes(render func() string) (perOp int64, out string) {
	res := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			out = render()
		}
	})
	return res.AllocedBytesPerOp(), out
}

// TestShowDeepAndWideAreLinear is acceptance (2): a 1 000 000-deep user ADT and
// a 10 000 000-element list render with allocation linear in the output — no
// Go recursion (it would overflow the goroutine stack long before a million
// levels) and no per-level string concatenation (quadratic).
func TestShowDeepAndWideAreLinear(t *testing.T) {
	const depth = 1_000_000
	var chain eval.Value = sCtor("Z")
	var vmChain = bytecode.NewADT(0, "Z", nil)
	var left eval.Value = sCtor("Leaf")
	for i := 0; i < depth; i++ {
		chain = sCtor("S", chain)
		vmChain = bytecode.NewADT(0, "S", []bytecode.Value{vmChain})
		if i < depth/10 {
			left = sCtor("Node", left, sInt(1), sCtor("Leaf"))
		}
	}
	wantChain := strings.Repeat("S(", depth) + "Z" + strings.Repeat(")", depth)

	elems := make([]eval.Value, 10_000_000)
	one := sInt(7)
	for i := range elems {
		elems[i] = one
	}
	wide := sList(elems...)

	cases := []struct {
		name   string
		render func() string
		want   string
		// maxPerOutByte bounds allocated bytes per output byte: the output
		// buffer's growth plus the returned string, and for a left-nested
		// tree one small frame per pending level.
		maxPerOutByte int64
	}{
		{"1M-deep chain, evaluator", func() string { return eval.Show(chain) }, wantChain, 5},
		{"1M-deep chain, VM", func() string { return showValue(vmChain) }, wantChain, 5},
		{"100k-deep left-nested tree", func() string { return eval.Show(left) }, "", 16},
		{"10M-element list", func() string { return eval.Show(wide) }, "", 5},
	}
	for _, c := range cases {
		perOp, out := showAllocBytes(c.render)
		if c.want != "" && out != c.want {
			t.Fatalf("%s: wrong rendering (len %d, want %d)", c.name, len(out), len(c.want))
		}
		if limit := c.maxPerOutByte * int64(len(out)); perOp > limit {
			t.Errorf("%s: allocated %d bytes for %d bytes of output (%.1f/byte); want <= %d/byte",
				c.name, perOp, len(out), float64(perOp)/float64(len(out)), c.maxPerOutByte)
		} else {
			t.Logf("%s: %d bytes out, %d allocated (%.2f/byte)", c.name, len(out), perOp, float64(perOp)/float64(len(out)))
		}
	}
	if got := eval.Show(wide); len(got) != 3*len(elems) { // "7, " per element, minus ", " plus "[]"
		t.Fatalf("10M list rendered %d bytes, want %d", len(got), 3*len(elems))
	}
}

// --- showPreview ------------------------------------------------------------

func TestShowPreviewSemantics(t *testing.T) {
	big := sList(showInts(0, 50)...)
	full := eval.Show(big)
	n := utf8.RuneCountInString(full)
	ee := sList(sStr("éé"), sStr("éé"))
	cases := []struct {
		v    eval.Value
		max  int
		want string
	}{
		{big, n, full},
		{big, n + 1, full},
		{big, n - 1, full[:len(full)-1] + "…(+1 chars)"},
		{big, 10, full[:10] + "…(+" + strconv.Itoa(n-10) + " chars)"},
		{sList(sInt(1), sInt(2), sInt(3)), 5, "[1, 2…(+4 chars)"},
		{ee, 3, `["é…(+9 chars)`},
		{ee, 2, `["…(+10 chars)`},
		{big, 0, "…(+" + strconv.Itoa(n) + " chars)"},
		{big, -3, "…(+" + strconv.Itoa(n) + " chars)"},
		{sStr(""), 0, ""},
		{sStr(""), -1, ""},
		{sStr("abc"), 2, "ab…(+1 chars)"},
		{sStr("\xff\xfeok"), 1, "\xff…(+3 chars)"},
	}
	for _, c := range cases {
		got := eval.ShowPreview(c.v, c.max)
		if got != c.want {
			t.Errorf("ShowPreview(%s, %d) = %q, want %q", eval.Show(c.v), c.max, got, c.want)
		}
		vm, err := builtinShowPreview([]bytecode.Value{toVMForShow(t, c.v), bytecode.NewInt(int64(c.max))})
		if err != nil {
			t.Fatal(err)
		}
		if vm.AsString() != got {
			t.Errorf("ShowPreview(%s, %d): engines differ: eval %q, vm %q", eval.Show(c.v), c.max, got, vm.AsString())
		}
	}
}

// Property: for any budget, a preview is either the full rendering, or a
// prefix of it on a code-point boundary plus a marker whose count is exact.
func TestShowPreviewIsAnExactPrefix(t *testing.T) {
	for _, row := range showSpecRows(t) {
		full := eval.Show(row.v)
		n := utf8.RuneCountInString(full)
		for _, max := range []int{-1, 0, 1, 2, 3, 7, 40, n - 1, n, n + 1} {
			got := eval.ShowPreview(row.v, max)
			if max >= n || (max < 0 && n == 0) { // a negative budget is a budget of 0
				if got != full {
					t.Fatalf("%s @%d: fits, want full rendering, got %q", row.name, max, got)
				}
				continue
			}
			keep := max
			if keep < 0 {
				keep = 0
			}
			i := 0
			for r := 0; r < keep; r++ {
				_, size := utf8.DecodeRuneInString(full[i:])
				i += size
			}
			if want := full[:i] + eval.PreviewMarker(n-keep); got != want {
				t.Fatalf("%s @%d:\n got %q\nwant %q", row.name, max, got, want)
			}
		}
	}
}

// Acceptance: a 200-char preview of range(0, 1000000) allocates under 4 KB
// beyond the value, on both engines — the walk counts past the budget, it does
// not build the rendering.
func TestShowPreviewDoesNotMaterialise(t *testing.T) {
	big := sList(showInts(0, 1_000_000)...)
	vmBig := toVMForShow(t, big)
	vmArgs := []bytecode.Value{vmBig, bytecode.NewInt(200)}
	for name, render := range map[string]func() string{
		"evaluator": func() string { return eval.ShowPreview(big, 200) },
		"vm": func() string {
			v, _ := builtinShowPreview(vmArgs)
			return v.AsString()
		},
	} {
		perOp, out := showAllocBytes(render)
		if !strings.HasPrefix(out, "[0, 1, 2, ") || !strings.HasSuffix(out, " chars)") {
			t.Fatalf("%s: unexpected preview %q", name, out)
		}
		if perOp >= 4096 {
			t.Errorf("%s: preview allocated %d bytes; want < 4096", name, perOp)
		}
	}
}

// --- generated values: engines agree, show is injective ----------------------

// genType is a closed type grammar for the generator.
type genType struct {
	kind   string // int float bool string unit list array tuple record map option result tree
	elems  []*genType
	fields []string
}

// showFragments are biased toward the characters the escaper exists for.
var showFragments = []string{
	"", "a", "b", " ", `"`, `\`, ", ", "${", "$", "{", "}", "\n", "\t", "\r", "\x00", "\x7f", "\u0085", "\u200b",
	"é", "😀", "\xff", "\xc3", "[", "]", "(", ")", ": ", "Some(", `\"`, `", "`,
}

var genFloats = []float64{0, math.Copysign(0, -1), 1, -1, 0.1, 2.5, 1e21, 1e-7, 1e300, 5e-324, math.MaxFloat64, math.Inf(1), math.Inf(-1)}

type showGen struct {
	seed uint64
}

func (g *showGen) intn(n int) int {
	// splitmix64: deterministic and dependency-free.
	g.seed += 0x9e3779b97f4a7c15
	z := g.seed
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return int(z % uint64(n))
}

func (g *showGen) typ(depth int) *genType {
	leaves := []string{"int", "float", "bool", "string", "string", "unit"}
	if depth >= 3 || g.intn(3) == 0 {
		return &genType{kind: leaves[g.intn(len(leaves))]}
	}
	switch g.intn(8) {
	case 0:
		return &genType{kind: "list", elems: []*genType{g.typ(depth + 1)}}
	case 1:
		return &genType{kind: "array", elems: []*genType{g.typ(depth + 1)}}
	case 2:
		n := 1 + g.intn(3)
		t := &genType{kind: "tuple"}
		for i := 0; i < n; i++ {
			t.elems = append(t.elems, g.typ(depth+1))
		}
		return t
	case 3:
		all := []string{"a", "b", "name", "B", "x_1"}
		t := &genType{kind: "record"}
		for _, f := range all {
			if g.intn(2) == 0 {
				t.fields = append(t.fields, f)
				t.elems = append(t.elems, g.typ(depth+1))
			}
		}
		return t
	case 4:
		keys := []string{"int", "string"}
		return &genType{kind: "map", elems: []*genType{{kind: keys[g.intn(2)]}, g.typ(depth + 1)}}
	case 5:
		return &genType{kind: "option", elems: []*genType{g.typ(depth + 1)}}
	case 6:
		return &genType{kind: "result", elems: []*genType{g.typ(depth + 1), {kind: "string"}}}
	default:
		return &genType{kind: "tree"}
	}
}

func (g *showGen) str() string {
	var b strings.Builder
	for n := g.intn(4); n > 0; n-- {
		b.WriteString(showFragments[g.intn(len(showFragments))])
	}
	return b.String()
}

func (g *showGen) value(t testing.TB, ty *genType, depth int) eval.Value {
	switch ty.kind {
	case "int":
		return sInt(g.intn(21) - 10)
	case "float":
		return sFlt(genFloats[g.intn(len(genFloats))])
	case "bool":
		return sBool(g.intn(2) == 0)
	case "string":
		return sStr(g.str())
	case "unit":
		return &eval.UnitValue{}
	case "list", "array":
		n := g.intn(4)
		xs := make([]eval.Value, n)
		for i := range xs {
			xs[i] = g.value(t, ty.elems[0], depth+1)
		}
		if ty.kind == "array" {
			return eval.NewArray(xs)
		}
		return sList(xs...)
	case "tuple":
		xs := make([]eval.Value, len(ty.elems))
		for i, e := range ty.elems {
			xs[i] = g.value(t, e, depth+1)
		}
		return sTup(xs...)
	case "record":
		r := &eval.RecordValue{Fields: map[string]eval.Value{}}
		for i, f := range ty.fields {
			r.Fields[f] = g.value(t, ty.elems[i], depth+1)
		}
		return r
	case "map":
		var kv []eval.Value
		seen := map[string]bool{}
		for n := g.intn(4); n > 0; n-- {
			k := g.value(t, ty.elems[0], depth+1)
			enc, _ := eval.MapKey(k)
			if seen[enc] {
				continue
			}
			seen[enc] = true
			kv = append(kv, k, g.value(t, ty.elems[1], depth+1))
		}
		return sMap(t, kv...)
	case "option":
		if g.intn(3) == 0 {
			return sCtor("None")
		}
		return sCtor("Some", g.value(t, ty.elems[0], depth+1))
	case "result":
		if g.intn(2) == 0 {
			return sCtor("Ok", g.value(t, ty.elems[0], depth+1))
		}
		return sCtor("Err", g.value(t, ty.elems[1], depth+1))
	case "tree":
		if depth >= 8 || g.intn(3) == 0 {
			return sCtor("Leaf")
		}
		return sCtor("Node", g.value(t, ty, depth+1), sStr(g.str()), g.value(t, ty, depth+1))
	}
	t.Fatalf("unknown kind %s", ty.kind)
	return nil
}

// structKey is an obviously injective encoding of a value's structure: every
// node is tagged and every sequence and string is length-prefixed; floats are
// compared by bit pattern (so -0.0 and 0.0 differ, as show distinguishes them).
func structKey(v eval.Value) string {
	var b strings.Builder
	var walk func(v eval.Value)
	seq := func(tag string, xs []eval.Value) {
		fmt.Fprintf(&b, "%s%d(", tag, len(xs))
		for _, x := range xs {
			walk(x)
		}
		b.WriteByte(')')
	}
	walk = func(v eval.Value) {
		switch x := v.(type) {
		case *eval.IntValue:
			fmt.Fprintf(&b, "i%d;", x.Value)
		case *eval.FloatValue:
			fmt.Fprintf(&b, "f%x;", math.Float64bits(x.Value))
		case *eval.BoolValue:
			fmt.Fprintf(&b, "b%t;", x.Value)
		case *eval.StringValue:
			fmt.Fprintf(&b, "s%d:%s", len(x.Value), x.Value)
		case *eval.UnitValue:
			b.WriteString("u;")
		case *eval.ListValue:
			seq("L", x.Elements)
		case *eval.ArrayValue:
			seq("A", x.Elements())
		case *eval.TupleValue:
			seq("T", x.Elements)
		case *eval.TaggedValue:
			fmt.Fprintf(&b, "C%d:%s", len(x.CtorName), x.CtorName)
			seq("", x.Fields)
		case *eval.RecordValue:
			names := make([]string, 0, len(x.Fields))
			for k := range x.Fields {
				names = append(names, k)
			}
			sort.Strings(names)
			fmt.Fprintf(&b, "R%d(", len(names))
			for _, k := range names {
				fmt.Fprintf(&b, "%d:%s", len(k), k)
				walk(x.Fields[k])
			}
			b.WriteByte(')')
		case *eval.MapValue:
			keys := make([]string, 0, len(x.Entries))
			for k := range x.Entries {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			fmt.Fprintf(&b, "M%d(", len(keys))
			for _, k := range keys {
				walk(x.Entries[k].Key)
				walk(x.Entries[k].Value)
			}
			b.WriteByte(')')
		default:
			panic(fmt.Sprintf("structKey: %T", v))
		}
	}
	walk(v)
	return b.String()
}

// TestShowEnginesAgreeAndInjective is M2: values of 200 generated types, 60
// each (12 000 values; every pair within a type is compared, ~350 000 pairs).
// For each value the evaluator and the VM render identically, and the output
// is valid UTF-8 (nested invalid bytes are escaped). Within one type, equal
// renderings imply structurally equal values.
func TestShowEnginesAgreeAndInjective(t *testing.T) {
	g := &showGen{seed: 20261111}
	const types, perType = 200, 60
	values, collisions := 0, 0
	for ti := 0; ti < types; ti++ {
		ty := g.typ(0)
		seen := map[string]string{} // show -> structKey
		for i := 0; i < perType; i++ {
			v := g.value(t, ty, 0)
			values++
			s := eval.Show(v)
			if vs := showValue(toVMForShow(t, v)); vs != s {
				t.Fatalf("engines differ on a %s:\n eval %q\n   vm %q", ty.kind, s, vs)
			}
			if _, top := v.(*eval.StringValue); !top && !utf8.ValidString(s) {
				t.Fatalf("show of a %s is not valid UTF-8: %q", ty.kind, s)
			}
			k := structKey(v)
			if prev, ok := seen[s]; ok {
				if prev != k {
					t.Fatalf("show is not injective on a %s: two different values both render as\n%q\n%q\n%q", ty.kind, s, prev, k)
				}
				collisions++
			}
			seen[s] = k
		}
	}
	t.Logf("%d values, %d equal-rendering pairs (all structurally equal)", values, collisions)
}

// NaN is outside the injectivity domain but not outside engine agreement.
func TestShowNaNEnginesAgree(t *testing.T) {
	v := sList(sFlt(math.NaN()), sTup(sFlt(math.NaN()), sStr("n")))
	if e, m := eval.Show(v), showValue(toVMForShow(t, v)); e != m || e != `[NaN, (NaN, "n")]` {
		t.Fatalf("eval %q vm %q", e, m)
	}
}
