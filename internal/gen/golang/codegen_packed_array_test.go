package golang

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestPackedArrayTypeMapping(t *testing.T) {
	mapper := NewTypeMapper()
	for _, tc := range []struct {
		element types.Type
		want    GoType
	}{
		{types.TFloat, "FloatArrayVal"}, {types.TInt, "ArrayVal"}, {&types.TVar{Name: "a"}, "ArrayVal"},
	} {
		got, err := mapper.MapType(&types.TArray{Element: tc.element})
		if err != nil || got != tc.want {
			t.Errorf("Array[%v] -> %q,%v; want %q", tc.element, got, err, tc.want)
		}
	}
	adt := NewADTGenerator("packedfixture")
	if got := adt.mapASTType(&ast.ArrayType{Element: &ast.SimpleType{Name: "float"}}); got != "FloatArrayVal" {
		t.Errorf("AST Array[float] -> %q", got)
	}
	if IsUserDefinedType("FloatArrayVal") || isRecordValueType("FloatArrayVal") {
		t.Fatal("packed array misclassified as a record/ADT")
	}
}

// Execute generated wrappers and runtime operations, rather than merely checking
// source strings. Exported literals and decoded packed arrays use different stores.
func TestPackedArrayGeneratedBoundaries(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	prog := &core.Program{
		Decls: []core.CoreExpr{
			&core.Let{Name: "literal", Value: &core.Lambda{Params: []string{"_"}, Body: &core.Array{Elements: []core.CoreExpr{&core.Lit{Kind: core.FloatLit, Value: 0.25}, &core.Lit{Kind: core.FloatLit, Value: 0.5}}}}, Body: &core.Var{Name: "literal"}},
			&core.Let{Name: "identity", Value: &core.Lambda{Params: []string{"xs"}, Body: &core.Var{Name: "xs"}}, Body: &core.Var{Name: "identity"}},
			&core.Let{Name: "shadowTrim", Value: &core.Lambda{Params: []string{"_"}, Body: &core.Let{
				Name: "trim", Value: &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}},
				Body: &core.App{Func: &core.Var{Name: "trim"}, Args: []core.CoreExpr{&core.Lit{Kind: core.StringLit, Value: " value "}}},
			}}, Body: &core.Var{Name: "shadowTrim"}},
			&core.Let{Name: "qualifiedTrim", Value: &core.Lambda{Params: []string{"_"}, Body: &core.Let{
				Name: "trim", Value: &core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}},
				Body: &core.App{Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/string", Name: "trim"}}, Args: []core.CoreExpr{&core.Lit{Kind: core.StringLit, Value: " value "}}},
			}}, Body: &core.Var{Name: "qualifiedTrim"}},
		},
		Meta: map[string]*core.DeclMeta{"literal": {Name: "literal", IsExport: true}, "identity": {Name: "identity", IsExport: true}, "shadowTrim": {Name: "shadowTrim", IsExport: true}, "qualifiedTrim": {Name: "qualifiedTrim", IsExport: true}},
	}
	g := New("packedfixture")
	g.RegisterFunctionType("literal", []GoType{"struct{}"}, "FloatArrayVal")
	g.RegisterFunctionType("identity", []GoType{"FloatArrayVal"}, "FloatArrayVal")
	source, err := g.Generate(prog)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":       "module packedfixture\n\ngo 1.21\n",
		"generated.go": string(source),
		"packed_test.go": `package packedfixture
import("testing";"reflect")
func TestPackedSetAndUnsafeGet(t *testing.T){
 original:=FloatArrayVal{0.0,0.25}
 got:=Set(original,int64(0),0.5)
 packed,ok:=got.(FloatArrayVal);if !ok||!reflect.DeepEqual(packed,FloatArrayVal{0.5,0.25}){t.Fatalf("Set packed: %T %v",got,got)}
 if original[0]!=0 {t.Fatal("Set mutated source")}
 if UnsafeGet(packed,int64(0))!=0.5 {t.Fatal("UnsafeGet packed")}
 boxed:=Set(original,int64(1),"mixed").(ArrayVal);if boxed[0]!=0.0||boxed[1]!="mixed"||original[1]!=0.25{t.Fatal("generic Set promotion",boxed)}
}
func TestTypedExports(t *testing.T){
 got:=Literal();if !reflect.DeepEqual(got,FloatArrayVal{0.25,0.5}){t.Fatal(got)}
 original:=FloatArrayVal{0.0,0.25};returned:=Identity(original)
 if &original[0]!=&returned[0] {t.Fatal("packed export performed a copy")}
 if !reflect.DeepEqual(ConvertToFloatArray(ArrayVal{0.0,0.5}),FloatArrayVal{0.0,0.5}){t.Fatal("boxed literal conversion")}
 if len(ConvertToFloatArray(ArrayVal{}))!=0 {t.Fatal("empty conversion")}
}
func TestInvalidPackedConversion(t *testing.T){
 for _,x:=range []interface{}{ArrayVal{0.0,"bad"},int64(1)} {func(){defer func(){if recover()==nil {t.Errorf("accepted %T %v",x,x)}}();ConvertToFloatArray(x)}()}
}
func TestLexicalFunctionValues(t *testing.T){
 if got:=ShadowTrim();got!=" value " {t.Fatalf("local trim was replaced by stdlib helper: %q",got)}
 if got:=QualifiedTrim();got!="value" {t.Fatalf("explicit std/string.trim lost qualification: %q",got)}
}
`,
	}
	for name, s := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(goBin, "test", "-count=1", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("packed runtime/wrappers: %v\n%s", err, out)
	}
}
