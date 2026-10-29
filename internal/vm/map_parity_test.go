package vm_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runner"
	"github.com/sunholo-data/ailang/internal/runtime"
)

func runMapModule(t *testing.T, filename, source string, strict bool) (string, error) {
	t.Helper()
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: source != ""}, pipeline.Source{Filename: filename, Code: source})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("check: %v %v", err, res.Errors)
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
	defer func() { os.Stdout = old }()
	err = runner.ExecuteModuleEntrypoint(rt, runner.ModuleExecParams{Filename: filename, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null", Quiet: true, Print: true, BytecodeMode: strict, StrictBytecode: strict, PipelineResult: &res})
	w.Close()
	output, readErr := io.ReadAll(r)
	r.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(output), err
}

func TestMapStrictEngineParity(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	source := `module fl
import std/map (fromList, lookup)
export func main() -> int {
 let m = fromList([("a", 1), ("b", 2)]);
 match lookup(m, "b") { Some(v) => v, None => 0 }
}`
	for _, strict := range []bool{false, true} {
		t.Run(fmt.Sprintf("reporter/strict=%v", strict), func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "fl.ail")
			if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := runMapModule(t, filename, source, strict)
			if err != nil || strings.TrimSpace(got) != "2" {
				t.Fatalf("got %q err %v", got, err)
			}
		})
	}
	mapResult := `module map_result
import std/map (fromList)
export func main() -> Map[string, int] { fromList([("b", 2), ("a", 1)]) }
`
	for _, strict := range []bool{false, true} {
		filename := filepath.Join(t.TempDir(), "map_result.ail")
		if err := os.WriteFile(filename, []byte(mapResult), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := runMapModule(t, filename, mapResult, strict)
		if err != nil || strings.TrimSpace(got) != `Map{a: 1, b: 2}` {
			t.Fatalf("map entrypoint strict=%v: %q %v", strict, got, err)
		}
	}
	fixture := "examples/runnable/map_strict_parity.ail"
	interpreted, err := runMapModule(t, fixture, "", false)
	if err != nil {
		t.Fatal(err)
	}
	strict, err := runMapModule(t, fixture, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if strict != interpreted || strings.TrimSpace(strict) != "true" {
		t.Fatalf("strict=%q interpreter=%q", strict, interpreted)
	}
}

func TestMapPortKeepsUnportedXMLStrictFailure(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	source := `module xml_unported
import std/xml (nodeKind, Text)
export func main() -> bool { let kind = nodeKind(Text("x")); true }
`
	filename := filepath.Join(t.TempDir(), "xml_unported.ail")
	if err := os.WriteFile(filename, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = runMapModule(t, filename, source, true)
	if err == nil || !strings.Contains(err.Error(), "__xml_nodeKind") || !strings.Contains(err.Error(), "opaque-type") {
		t.Fatalf("expected explicit unported XML error: %v", err)
	}
}
