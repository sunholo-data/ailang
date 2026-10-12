package pipeline

import (
	"github.com/sunholo-data/ailang/internal/testutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEffectRowVariableDischarge(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, tt := range []struct{ name, source, blame string }{
		{"pure", `func runIt(f: () -> int ! {e}) -> int ! {e} = f()
func quiet() -> int = 42
pure func caller() -> int = runIt(quiet)`, ""},
		{"twice_pure", `func runTwice(f: () -> int ! {e}) -> int = f() + f()
func noisy() -> int ! {IO} { let _ = println("LEAK"); 21 }
func caller() -> int ! {IO} = runTwice(noisy)`, "runTwice"},
		{"twice_generic", `func runTwice(f: () -> int ! {e}) -> int ! {e} = f() + f()
func noisy() -> int ! {IO} { let _ = println("LEAK"); 21 }
func caller() -> int ! {IO} = runTwice(noisy)`, ""},
		{"tail_not_absorbing_io", `func illegal[e]() -> () ! {e} = println("LEAK")`, "illegal"},
		{"wrong_fs", `func runIt(f: () -> int ! {e}) -> int ! {e} = f()
func noisy() -> int ! {IO} { let _ = println("LEAK"); 42 }
func caller() -> int ! {FS} = runIt(noisy)`, "caller"},
		{"independent_calls", `func runIt(f: () -> int ! {e}) -> int ! {e} = f()
func quiet() -> int = 21
func noisy() -> int ! {IO} { let _ = println("LEAK"); 21 }
func first() -> int ! {IO} = runIt(quiet) + runIt(noisy)
func second() -> int ! {IO} = runIt(noisy) + runIt(quiet)
pure func third() -> int = runIt(quiet)`, ""},
		{"return_only_argument", `func retOnly(x: int) -> int ! {e} = x
pure func caller() -> int = retOnly(42)`, ""},
		{"return_only", `func retOnly() -> int ! {e} = 42
pure func caller() -> int = retOnly()`, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := checkModules(t, map[string]string{"main.ail": "module main\nimport std/io (println)\n" + tt.source})
			if tt.blame == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "function '"+tt.blame+"'") {
				t.Fatalf("wrong effect rejection: %v", err)
			}
			if tt.name == "wrong_fs" && !strings.Contains(err.Error(), "Missing effects: IO") {
				t.Fatal(err)
			}
			if strings.Contains(err.Error(), "Suggested fix:") && !strings.Contains(err.Error(), "IO") && !strings.Contains(err.Error(), "Unresolved effect tail") {
				t.Fatalf("blank suggestion: %v", err)
			}
		})
	}
}

func TestEffectRowVariableRuntime(t *testing.T) {
	binary := testutil.FindAilangBinary(t)
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, generic := range []bool{false, true} {
		row := ""
		if generic {
			row = " ! {e}"
		}
		source := "module twice\nimport std/io (println)\nfunc runTwice(f: () -> int ! {e}) -> int" + row + " = f() + f()\nfunc noisy() -> int ! {IO} { let _ = println(\"CALLBACK_MARKER\"); 21 }\nexport func main() -> () ! {IO} { let _ = runTwice(noisy); () }\n"
		path := filepath.Join(t.TempDir(), "twice.ail")
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		output, err := exec.Command(binary, "run", "--caps", "IO", "--entry", "main", path).CombinedOutput()
		if generic {
			if err != nil || strings.Count(string(output), "CALLBACK_MARKER") != 2 {
				t.Fatalf("corrected helper: %v\n%s", err, output)
			}
			noCaps, err := exec.Command(binary, "run", "--entry", "main", path).CombinedOutput()
			if err == nil || !strings.Contains(string(noCaps), "capability") {
				t.Fatalf("capability backstop: %v\n%s", err, noCaps)
			}
		} else if err == nil || !strings.Contains(string(output), "function 'runTwice'") || strings.Contains(string(output), "CALLBACK_MARKER") {
			t.Fatalf("must reject before printing: %v\n%s", err, output)
		}
	}
}

func TestEffectRowVariableStreamingImports(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, module := range []string{"std/stream", "std/ai/streaming"} {
		for _, row := range []string{" ! {Stream, IO}", " ! {Stream}"} {
			source := "module main\nimport std/io (println)\nimport std/stream (StreamConn, StreamEvent)\nimport " + module + " (onEvent)\n" + `func handle(event: StreamEvent) -> bool ! {IO} { let _ = println("event"); false }
func sameTail[e](conn: StreamConn, cb: StreamEvent -> bool ! {e}) -> () ! {Stream, e} { onEvent(conn, cb); onEvent(conn, cb) }
func subscribe(conn: StreamConn) -> ()` + row + " = onEvent(conn, handle)\n"
			err := checkModules(t, map[string]string{"main.ail": source})
			if strings.Contains(row, "IO") {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "function 'subscribe'") || !strings.Contains(err.Error(), "Missing effects: IO") {
				t.Fatalf("wrong streaming diagnostic: %v", err)
			}
		}
	}
}

func TestEffectRowVariableDeclaredAndInferredImports(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, annotation := range []string{" ! {e}", ""} {
		helper := "module helper\nexport func apply[e](f: () -> int" + annotation + ") -> int" + annotation + " = f()\n"
		for _, row := range []string{" ! {IO}", ""} {
			main := "module main\nimport helper (apply)\nimport std/io (println)\nfunc noisy() -> int ! {IO} { let _ = println(\"CALL\"); 42 }\nfunc caller() -> int" + row + " = apply(noisy)\n"
			err := checkModules(t, map[string]string{"helper.ail": helper, "main.ail": main})
			if row != "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "function 'caller'") || !strings.Contains(err.Error(), "Missing effects: IO") {
				t.Fatalf("wrong importer diagnostic: %v", err)
			}
		}
	}
}

func TestEffectRowVariableEmptyDiffInvariant(t *testing.T) {
	err := formatEffectError("same", nil, nil)
	if !strings.Contains(err.Error(), "internal invariant") || strings.Contains(err.Error(), "Suggested fix") {
		t.Fatal(err)
	}
}
