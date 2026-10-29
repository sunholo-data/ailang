package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// Compile and execute the actual emitted helper. A metadata-only test would
// miss an Inline that formats correctly but omits invalid-precision guards.
func TestFormatFixedGeneratedGo(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	prog := &core.Program{
		Decls: []core.CoreExpr{&core.Let{
			Name: "renderFixed",
			Value: &core.Lambda{Params: []string{"x", "d"}, Body: &core.App{
				Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "formatFixed"}},
				Args: []core.CoreExpr{&core.Var{Name: "x"}, &core.Var{Name: "d"}},
			}},
			Body: &core.Var{Name: "renderFixed"},
		}},
		Meta: map[string]*core.DeclMeta{"renderFixed": {Name: "renderFixed", IsExport: true}},
	}
	source, err := New("fixedcheck").Generate(prog)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":       "module fixedcheck\n\ngo 1.21\n",
		"generated.go": string(source),
		"generated_test.go": `package fixedcheck
import("fmt"; "math"; "strings"; "testing")
func TestValues(t *testing.T) {
  for _, c := range []struct{x float64; d int64; want string}{
    {-27.060000000000002,2,"-27.06"}, {0.125,2,"0.12"},
    {2.675,2,"2.67"}, {0.1,20,"0.10000000000000000555"},
    {math.Copysign(0,-1),2,"-0.00"}, {math.NaN(),2,"NaN"},
    {math.Inf(1),2,"+Inf"}, {math.Inf(-1),2,"-Inf"},
    {3.14159,0,"3"}, {1,1000,"1."+strings.Repeat("0",1000)},
  } { if got:=RenderFixed(c.x,c.d); got!=c.want { t.Errorf("got %v want %q",got,c.want) } }
}
func TestErrors(t *testing.T) {
  for _, d := range []int64{-1,1001,math.MinInt64,math.MaxInt64} {
    func(){ defer func(){ r:=recover(); if r==nil || !strings.Contains(fmt.Sprint(r),"decimals") || !strings.Contains(fmt.Sprint(r),"0..1000") { t.Errorf("precision %d: expected range panic, got %v",d,r) } }(); RenderFixed(1.25,d) }()
  }
}
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(goBin, "test", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated Go failed: %v\n%s", err, out)
	}
}
