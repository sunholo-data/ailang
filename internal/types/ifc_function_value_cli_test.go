package types

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestIFCFunctionValueFreshCLI(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	name := "ailang"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(t.TempDir(), name)
	out, stderr, code := testutil.RunBounded(t, root, 120*time.Second, "go", "build", "-o", bin, "./cmd/ailang")
	if code != 0 {
		t.Fatalf("build: %s%s", out, stderr)
	}
	t.Setenv("AILANG_NO_CACHE", "1")
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	for _, row := range fvRows {
		t.Run(row.id, func(t *testing.T) {
			// Fresh path + disabled cache: no earlier source can share this fixture key.
			file := filepath.Join(t.TempDir(), "fv.ail")
			if err := os.WriteFile(file, []byte(fvSource(row)), 0600); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := testutil.RunBounded(t, root, 30*time.Second, bin, "check", file)
			if row.want == "" {
				if code != 0 {
					t.Fatalf("clean control rejected: %s%s", stdout, stderr)
				}
			} else {
				diagnostic := "forbids it"
				if row.want == ParamLabelCoverError {
					diagnostic = "does not cover it"
				}
				if code != 1 || !strings.Contains(stdout+stderr, "information-flow violation") || !strings.Contains(stdout+stderr, diagnostic) {
					t.Fatalf("want IFC %s refusal, code=%d: %s%s", row.want, code, stdout, stderr)
				}
			}
		})
	}
	checkIFCCapitalizedImports(t, root, bin)
	for _, body := range []string{`let g = gate in g(fromInbox("hi"))`, `apply1(gate, fromInbox("hi"))`, `let w = \y. gate(y) in w(fromInbox("hi"))`} {
		file := filepath.Join(t.TempDir(), "stop.ail")
		src := "module stop\nimport std/io (println)\n" + fvPrelude + fvApply1 + `export func main() -> () ! {IO} { let _ = ` + body + `; println("IFC1719_UNSAFE") }`
		if err := os.WriteFile(file, []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
		for _, vm := range []bool{false, true} {
			args := []string{"run", "--quiet", "--caps", "IO", "--entry", "main"}
			if vm {
				args = append(args, "--bytecode")
			}
			args = append(args, file)
			stdout, stderr, code := testutil.RunBounded(t, root, 30*time.Second, bin, args...)
			if code == 0 || !strings.Contains(stderr+stdout, "information-flow violation") || strings.Contains(stdout, "IFC1719_UNSAFE") {
				t.Fatalf("vm=%v rejected program executed or missed IFC gate: code=%d %s%s", vm, code, stdout, stderr)
			}
		}
	}
}

func checkIFCCapitalizedImports(t *testing.T, root, bin string) {
	t.Helper()
	dir := t.TempDir()
	lib := `module lib
export func Apply(f: string -> string, x: string) -> string { f(x) }
export func apply(f: string -> string, x: string) -> string { f(x) }
`
	if err := os.WriteFile(filepath.Join(dir, "lib.ail"), []byte(lib), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ module, imported string }{{"upper", "Apply"}, {"lower", "apply"}} {
		src := "module " + tc.module + "\nimport lib (" + tc.imported + ")\n" + fvPrelude + `export func main() -> string { ` + tc.imported + `(gate, fromInbox("TAINT")) }`
		file := filepath.Join(dir, tc.module+".ail")
		if err := os.WriteFile(file, []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := testutil.RunBounded(t, dir, 30*time.Second, bin, "check", file)
		if code != 1 || !strings.Contains(stdout+stderr, "information-flow violation") {
			t.Fatalf("%s imported function must obey opaque fallback: code=%d %s%s", tc.imported, code, stdout, stderr)
		}
		for _, vm := range []bool{false, true} {
			args := []string{"run", "--quiet", "--entry", "main"}
			if vm {
				args = append(args, "--bytecode")
			}
			args = append(args, file)
			stdout, stderr, code := testutil.RunBounded(t, dir, 30*time.Second, bin, args...)
			if code == 0 || strings.Contains(stdout, "TAINT") || !strings.Contains(stderr+stdout, "information-flow violation") {
				t.Fatalf("%s vm=%v ran a leak: %d %s%s", tc.imported, vm, code, stdout, stderr)
			}
		}
	}
	src := "module stored\nimport std/option (Some)\n" + fvPrelude + `export func main() -> string { let stored = Some(gate) in "ok" }`
	file := filepath.Join(dir, "stored.ail")
	if err := os.WriteFile(file, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := testutil.RunBounded(t, root, 30*time.Second, bin, "check", file)
	if code != 0 {
		t.Fatalf("imported ADT sink storage rejected: %s%s", stdout, stderr)
	}
}
