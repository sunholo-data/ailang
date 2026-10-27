package builtins

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var bytesCodegenExports = []string{"fromInts", "toInts", "length", "slice", "byteAt", "concat", "fromString", "toString"}

func TestBytesCodegenSpecCoverage(t *testing.T) {
	for _, name := range bytesCodegenExports {
		spec := GetCodegenSpecByStdlibName("std/bytes", name)
		if spec == nil || spec.Helper == nil {
			t.Errorf("std/bytes.%s must resolve to a concrete Go helper", name)
			continue
		}
		if builtin := StdlibIndex[StdlibKey{Module: "std/bytes", Name: name}]; !strings.HasPrefix(builtin, "_bytes_") {
			t.Errorf("std/bytes.%s resolves outside bytes family: %s", name, builtin)
		}
	}
	for _, oldName := range []string{"_str_fromString", "_str_toString"} {
		spec := GetCodegenSpec(oldName)
		if spec == nil || spec.StdlibName != "" || spec.StdlibModule != "" {
			t.Errorf("legacy alias %s must remain directly available without a qualified claim", oldName)
		}
	}
	for _, name := range bytesCodegenExports {
		claims := 0
		for _, meta := range Registry {
			if s := meta.GoCodegen; s != nil && s.StdlibModule == "std/bytes" && s.StdlibName == name {
				claims++
			}
		}
		if claims != 1 {
			t.Errorf("std/bytes.%s has %d qualified claims, want 1", name, claims)
		}
	}
}

func TestBytesCodegenHelperBehavior(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	var source strings.Builder
	source.WriteString("package bytesfixture\nimport(\"fmt\";\"reflect\")\nvar _ = fmt.Sprint\nvar _ = reflect.ValueOf\n")
	source.WriteString(`type Option struct { Some bool; Value interface{} }
func NewOptionSome(v interface{}) interface{} { return Option{true,v} }
func NewOptionNone() interface{} { return Option{} }
func toInt64(v interface{}) int64 { switch x:=v.(type) {case int64: return x; case int: return int64(x); default: panic("not int") } }
`)
	for _, name := range bytesCodegenExports {
		spec := GetCodegenSpecByStdlibName("std/bytes", name)
		if spec == nil || spec.Helper == nil {
			t.Fatalf("std/bytes.%s lacks a helper", name)
		}
		fmt.Fprintf(&source, "%s {\n%s\n}\n", spec.Helper.Signature, spec.Helper.Body)
	}
	dir := t.TempDir()
	files := map[string]string{
		"go.mod":     "module bytesfixture\n\ngo 1.21\n",
		"helpers.go": source.String(),
		"helpers_test.go": `package bytesfixture
import("fmt";"math";"reflect";"strings";"testing")
func TestConversions(t *testing.T) {
 b:=BytesFromInts([]interface{}{int64(0),int64(128),int64(255)}).([]byte)
 if !reflect.DeepEqual(b,[]byte{0,128,255}) {t.Fatal(b)}
 if !reflect.DeepEqual(BytesToInts(b),[]interface{}{int64(0),int64(128),int64(255)}) {t.Fatal(BytesToInts(b))}
 if !reflect.DeepEqual(BytesFromInts([]int64{0,255}),[]byte{0,255}) {t.Fatal("typed list")}
 if len(BytesFromInts(nil).([]byte))!=0 {t.Fatal("nil list")}
 if BytesLength(b)!=int64(3) {t.Fatal(BytesLength(b))}
 utf8:=BytesFromString("é").([]byte)
 if !reflect.DeepEqual(utf8,[]byte{195,169}) || BytesToString(utf8)!="é" {t.Fatal(utf8)}
 raw:=[]byte{0xff,0xfe}; if BytesToString(raw)!=string(raw) {t.Fatal("raw string conversion drift")}
 if !reflect.DeepEqual(BytesFromString(""),[]byte{}) {t.Fatal("empty string")}
}
func TestBoundsAndOwnership(t *testing.T) {
 b:=[]byte{10,20,30}
 for _,c:=range []struct{s,n int64;want Option}{
   {0,0,Option{true,[]byte{}}}, {1,2,Option{true,[]byte{20,30}}},
   {3,0,Option{true,[]byte{}}}, {-1,1,Option{}}, {0,-1,Option{}},
   {4,0,Option{}}, {2,2,Option{}}, {1,math.MaxInt64,Option{}},
   {math.MaxInt64,1,Option{}},
 } {if got:=BytesSlice(b,c.s,c.n); !reflect.DeepEqual(got,c.want) {t.Errorf("slice(%d,%d)=%#v want %#v",c.s,c.n,got,c.want)}}
 for _,c:=range []struct{i int64;want Option}{{-1,Option{}},{0,Option{true,int64(10)}},{2,Option{true,int64(30)}},{3,Option{}},{math.MaxInt64,Option{}}} {if got:=BytesByteAt(b,c.i);!reflect.DeepEqual(got,c.want){t.Errorf("byteAt(%d)=%#v",c.i,got)}}
 sub:=BytesSlice(b,int64(0),int64(2)).(Option).Value.([]byte); sub[0]=99
 joined:=BytesConcat(b,[]byte{40}).([]byte); joined[1]=98
 if !reflect.DeepEqual(b,[]byte{10,20,30}) {t.Fatal("helper aliases input",b)}
}
func TestFromIntsRejectsOutOfRange(t *testing.T) {
 for _,n:=range []int64{-1,256,math.MinInt64,math.MaxInt64} {func(){defer func(){r:=recover();if r==nil||!strings.Contains(fmt.Sprint(r),"0-255"){t.Errorf("%d: %v",n,r)}}();BytesFromInts([]interface{}{n})}()}
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
		t.Fatalf("emitted byte helpers failed: %v\n%s", err, out)
	}
}
