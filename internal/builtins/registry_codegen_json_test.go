package builtins

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The Go-codegen GetInt/AsNumber helpers must follow std/json's number
// contract: JInt is exact, an integral JNumber within int64 converts, a
// fractional or out-of-range one is None (never truncated). The helper bodies
// are compiled against minimal stand-ins for the generated Json/Option types
// and run.
func TestGoCodegenJSONNumberHelpers(t *testing.T) {
	body := func(name string) string {
		meta, ok := Registry[name]
		if !ok || meta.GoCodegen == nil || meta.GoCodegen.Helper == nil {
			t.Fatalf("no Go codegen helper for %s", name)
		}
		h := meta.GoCodegen.Helper
		return h.Signature + " {\n" + h.Body + "\n}\n"
	}
	prog := `package main

import (
	"fmt"
	"math"
)

const (
	JsonKindJInt    = "JInt"
	JsonKindJNumber = "JNumber"
	JsonKindJString = "JString"
)

type Json struct {
	Kind    string
	JInt    struct{ Value0 int64 }
	JNumber struct{ Value0 float64 }
}

type Option struct {
	some bool
	v    interface{}
}

func NewOptionSome(v interface{}) interface{} { return &Option{true, v} }
func NewOptionNone() interface{}              { return &Option{} }
func IsNone(o interface{}) interface{}        { return !o.(*Option).some }
func OptionGetOrElse(o, d interface{}) interface{} {
	if o.(*Option).some {
		return o.(*Option).v
	}
	return d
}

// JsonGet stand-in: the "object" is the value itself.
func JsonGet(obj, key interface{}) interface{} { return NewOptionSome(obj) }

func show(o interface{}) string {
	if !o.(*Option).some {
		return "None"
	}
	return fmt.Sprintf("Some(%v)", o.(*Option).v)
}

` + body("_json_getInt") + body("_json_asNumber") + `
func main() {
	num := func(f float64) *Json { j := &Json{Kind: JsonKindJNumber}; j.JNumber.Value0 = f; return j }
	in := func(n int64) *Json { j := &Json{Kind: JsonKindJInt}; j.JInt.Value0 = n; return j }
	fmt.Println(show(GetInt(in(1234567890123456789), "k")))
	fmt.Println(show(GetInt(num(30), "k")))
	fmt.Println(show(GetInt(num(1187.5), "k")))
	fmt.Println(show(GetInt(num(1e30), "k")))
	fmt.Println(show(GetInt(num(math.NaN()), "k")))
	fmt.Println(show(GetInt(&Json{Kind: JsonKindJString}, "k")))
	fmt.Println(show(AsNumber(in(7))))
	fmt.Println(show(AsNumber(num(2.5))))
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(prog), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module helpers\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helpers did not build/run: %v\n%s\n%s", err, out, prog)
	}
	want := "Some(1234567890123456789)\nSome(30)\nNone\nNone\nNone\nNone\nSome(7)\nSome(2.5)\n"
	if got := strings.ReplaceAll(string(out), "\r\n", "\n"); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
