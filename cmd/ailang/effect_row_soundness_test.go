package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEffectRowGuardRejectsBeforeExecution(t *testing.T) {
	runRowGuardCases(t, rowGuardCurrentCases())
}

// #1958 M2 (830934d23) was reverted for v0.56.1: it rejected valid callbacks
// because inference closes multi-effect lambda rows to {}. These cases come
// back with M2; see design_docs/implemented/v0_56_0/m-effect-row-open-closed-soundness.md.
func TestEffectRowGuardRejectsBeforeExecutionM2Deferred(t *testing.T) {
	t.Skip("#1958 M2 reverted for v0.56.1: needs inference to keep multi-effect lambda rows open first")
	runRowGuardCases(t, map[string]string{
		"map": `func loud(x: int) -> int ! {IO} { println("ROW1958_SIDE_EFFECT"); x }
pure func hidden(g: (int) -> int ! {IO}) -> [int] = map(g, [1])
export func main() -> () ! {IO} { let _ = hidden(loud); () }`,
		"filter": `func loud(x: int) -> bool ! {IO} { println("ROW1958_SIDE_EFFECT"); true }
pure func hidden(g: (int) -> bool ! {IO}) -> [int] = filter(g, [1])
export func main() -> () ! {IO} { let _ = hidden(loud); () }`,
		"foldl": `func loud(a: int, x: int) -> int ! {IO} { println("ROW1958_SIDE_EFFECT"); a + x }
pure func hidden(g: (int, int) -> int ! {IO}) -> int = foldl(g, 0, [1])
export func main() -> () ! {IO} { let _ = hidden(loud); () }`,
		"record": `type Hook = { run: (int) -> int ! {IO} }
func loud(x: int) -> int ! {IO} { println("ROW1958_SIDE_EFFECT"); x }
pure func hidden(h: Hook) -> [int] = map(h.run, [1])
export func main() -> () ! {IO} { let _ = hidden({ run: loud }); () }`,
		"mixed": `func loud(x: int) -> int ! {IO} { println("ROW1958_SIDE_EFFECT"); x }
pure func hidden[e](g: (int) -> int ! {IO, e}) -> [int] = map(g, [1])
export func main() -> () ! {IO} { let _ = hidden(loud); () }`,
	})
}

func rowGuardCurrentCases() map[string]string {
	cases := map[string]string{}
	for _, api := range []string{"stepWithStream", "stepWithStreamRecorded"} {
		cases[api] = fmt.Sprintf(`func loud(c: StreamChunk) -> () ! {IO} = println("ROW1958_SIDE_EFFECT")
func hidden() -> () ! {AI} { let _ = %s("", [], [], [], loud); () }
export func main() -> () ! {AI, IO} = hidden()`, api)
	}
	return cases
}

func runRowGuardCases(t *testing.T, cases map[string]string) {
	t.Helper()
	t.Setenv("AILANG_NO_CACHE", "1")
	bin := buildAilang(t)
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "row_soundness.ail")
			source := "module row_soundness\nimport std/io (println)\nimport std/list (map, filter, foldl)\nimport std/ai (stepWithStream, stepWithStreamRecorded, StreamChunk)\n" + body
			if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			for _, bytecode := range []bool{false, true} {
				args := []string{"run", "--quiet", "--caps", "AI,IO", "--ai-stub", "--entry", "main"}
				if bytecode {
					args = append(args, "--bytecode")
				}
				stdout, stderr, code := runAilangBin(t, bin, append(args, file)...)
				if code == 0 || !strings.Contains(stderr, "IO") {
					t.Fatalf("bytecode=%v must reject hidden IO, code=%d stdout=%s stderr=%s", bytecode, code, stdout, stderr)
				}
				if strings.Contains(stdout, "ROW1958_SIDE_EFFECT") {
					t.Fatalf("bytecode=%v executed a rejected callback: %s", bytecode, stdout)
				}
			}
		})
	}
}

func TestEffectRowGuardDeclaredEffectsRun(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "row_declared.ail")
	source := `module row_declared
import std/io (println)
import std/list (mapE)
import std/ai (stepWithStream, stepWithStreamRecorded, StreamChunk)
func loud(x: int) -> int ! {IO} { println("declared IO"); x }
func traverse[e](g: (int) -> int ! {e}) -> [int] ! {e} = mapE(g, [1])
pure func quiet(c: StreamChunk) -> () = ()
func aiOnly() -> () ! {AI} {
 let _ = stepWithStream("", [], [], [], quiet);
 let _ = stepWithStreamRecorded("", [], [], [], quiet);
 ()
}
export func main() -> () ! {AI, IO} {
 let _ = traverse(loud);
 aiOnly();
 println("pure AI callbacks")
}`
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, bytecode := range []bool{false, true} {
		args := []string{"run", "--quiet", "--caps", "AI,IO", "--ai-stub", "--entry", "main"}
		if bytecode {
			args = append(args, "--bytecode")
		}
		stdout, stderr, code := runAilangBin(t, bin, append(args, file)...)
		if code != 0 || strings.TrimSpace(stdout) != "declared IO\npure AI callbacks" {
			t.Fatalf("bytecode=%v declared effects must run: code=%d stdout=%s stderr=%s", bytecode, code, stdout, stderr)
		}
	}
}
