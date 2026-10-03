package golang

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/mathx"
)

// #1465: the helpers emitted into compiled programs must be bit-identical to
// internal/mathx (the interpreter/VM implementation). This compiles the
// emitted source with the Go toolchain and compares outputs.
func TestMathxHelperSourceMatchesMathx(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	src, err := mathxHelperSource()
	if err != nil {
		t.Fatal(err)
	}

	inputs := []float64{0.2064590551107192, 1.229317398921793, 0.5, -0.5, 2, 3.7, 10, -10, 0.7, 0.99,
		1e-300, 1e300, 1 << 29, 1e22, 123.456, 0.66, 2.5, 709.78, -745.13, 1.0001}
	unary := []string{"Exp", "Log", "Log10", "Sin", "Cos", "Tan", "Asin", "Acos", "Atan"}

	var body strings.Builder
	body.WriteString("package main\n\nimport (\n\t\"fmt\"\n\t\"math\"\n\t\"math/bits\"\n)\n\nvar _ = bits.Mul64\n\n")
	body.WriteString(src)
	body.WriteString("func main() {\n\txs := []float64{")
	for i, x := range inputs {
		if i > 0 {
			body.WriteString(", ")
		}
		fmt.Fprintf(&body, "math.Float64frombits(%#x)", math.Float64bits(x))
	}
	body.WriteString("}\n\tfor _, x := range xs {\n")
	for _, fn := range unary {
		fmt.Fprintf(&body, "\t\tfmt.Println(math.Float64bits(%s%s(x)))\n", mathxPrefix, fn)
	}
	fmt.Fprintf(&body, "\t\tfmt.Println(math.Float64bits(%sAtan2(x, 0.75)))\n", mathxPrefix)
	fmt.Fprintf(&body, "\t\tfmt.Println(math.Float64bits(%sPow(math.Abs(x), 1.37)))\n", mathxPrefix)
	body.WriteString("\t}\n}\n")

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(body.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module mathxcheck\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(goBin, "run", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOWORK=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("emitted helpers failed to build/run: %v\n%s", err, out)
	}

	fns := map[string]func(float64) float64{
		"Exp": mathx.Exp, "Log": mathx.Log, "Log10": mathx.Log10, "Sin": mathx.Sin, "Cos": mathx.Cos,
		"Tan": mathx.Tan, "Asin": mathx.Asin, "Acos": mathx.Acos, "Atan": mathx.Atan,
	}
	var want []uint64
	for _, x := range inputs {
		for _, fn := range unary {
			want = append(want, math.Float64bits(fns[fn](x)))
		}
		want = append(want, math.Float64bits(mathx.Atan2(x, 0.75)), math.Float64bits(mathx.Pow(math.Abs(x), 1.37)))
	}
	lines := strings.Fields(string(out))
	if len(lines) != len(want) {
		t.Fatalf("got %d outputs, want %d:\n%s", len(lines), len(want), out)
	}
	for i, l := range lines {
		got, err := strconv.ParseUint(l, 10, 64)
		if err != nil {
			t.Fatalf("bad output line %q", l)
		}
		gotF, wantF := math.Float64frombits(got), math.Float64frombits(want[i])
		if got != want[i] && !(math.IsNaN(gotF) && math.IsNaN(wantF)) {
			t.Errorf("output %d: emitted helper gave %#x, mathx gives %#x", i, got, want[i])
		}
	}
}

// Generated user identifiers pass through ToPascalCase/ToCamelCase, which drop
// underscores, and module prefixes use a double underscore — so no generated
// name can begin with the single-underscore "ailmathx_" helper prefix.
func TestMathxHelperNamesAreReserved(t *testing.T) {
	for _, n := range []string{"ailmathx_Exp", "ailmathx_exp", "ailmathxExp", "_ailmathx_Sin"} {
		for _, got := range []string{ToGoVarName(n), ToGoFuncName(n, true), ToGoFuncName(n, false)} {
			if strings.HasPrefix(got, mathxPrefix) {
				t.Errorf("user identifier %q maps to %q, which collides with the mathx helper namespace", n, got)
			}
		}
	}
	src, err := mathxHelperSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range []string{"Exp", "Log", "Log10", "Sin", "Cos", "Tan", "Asin", "Acos", "Atan", "Atan2", "Pow"} {
		if !strings.Contains(src, "func "+mathxPrefix+fn+"(") {
			t.Errorf("helper source lacks %s%s", mathxPrefix, fn)
		}
	}
	if strings.Contains(src, "package ") || strings.Contains(src, "import ") {
		t.Error("helper source must not carry package/import clauses")
	}
}
