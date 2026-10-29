package main

import (
	"github.com/sunholo-data/ailang/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExplicitEmptyCallbackRejectsBeforeExecution(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	bin := buildAilang(t)
	stdlib, err := filepath.Abs(filepath.Join("..", "..", "std"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", stdlib)
	for _, tt := range []struct{ name, decl, call, lib string }{
		{"direct", `func apply(f:(int)->int ! {}) -> int = f(1)`, `apply(noisy)`, ""},
		{"record", `type H = {run:(int)->int ! {}}
func apply(h:H)->int = h.run(1)`, `apply({run:noisy})`, ""},
		{"imported", `import lib (apply)`, `apply(noisy)`, `module lib
export func apply(f:(int)->int ! {}) -> int = f(1)`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "main.ail")
			imports, decl := tt.decl, ""
			if tt.lib == "" {
				imports = ""
				decl = tt.decl
			} else if err := os.WriteFile(filepath.Join(dir, "lib.ail"), []byte(tt.lib), 0600); err != nil {
				t.Fatal(err)
			}
			source := "module main\nimport std/io (println)\n" + imports + "\nfunc noisy(x:int)->int ! {IO} = { println(\"EMPTY_CALLBACK_SIDE_EFFECT\"); x }\n" + decl + "\nexport func main()->() ! {IO} {let _ = " + tt.call + "; ()}\n"
			if err := os.WriteFile(file, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			for _, bytecode := range []bool{false, true} {
				args := []string{"run", "--quiet", "--caps", "IO", "--entry", "main"}
				if bytecode {
					args = append(args, "--bytecode")
				}
				out, errout, code := testutil.RunBounded(t, dir, 60*time.Second, bin, append(args, file)...)
				if code == 0 || !strings.Contains(errout, "IO") || strings.Contains(out, "EMPTY_CALLBACK_SIDE_EFFECT") {
					t.Fatalf("bytecode=%v expected static rejection before output: code=%d out=%s err=%s", bytecode, code, out, errout)
				}
			}
		})
	}
}

func TestExplicitEmptyCallbackStorageRuns(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "main.ail")
	source := `module main
import std/io (println)
type Hooks = {probe:(int)->int ! {}, render:(int)->int ! {IO}}
pure func inc(x:int)->int = x+1
func noisy(x:int)->int ! {IO} = {println("EMPTY_CALLBACK_SIDE_EFFECT"); x}
pure func keep(h:Hooks)->Hooks = h
pure func build()->Hooks = keep({probe:inc,render:noisy})
export func main()->() ! {IO} {let h=build();println(show(h.probe(41)))}
`
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	for _, bytecode := range []bool{false, true} {
		args := []string{"run", "--quiet", "--caps", "IO", "--entry", "main"}
		if bytecode {
			args = append(args, "--bytecode")
		}
		out, errout, code := runAilangBin(t, bin, append(args, file)...)
		if code != 0 || strings.TrimSpace(out) != "42" {
			t.Fatalf("bytecode=%v expected pure storage, code=%d out=%s err=%s", bytecode, code, out, errout)
		}
	}
}
