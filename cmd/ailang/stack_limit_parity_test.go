package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

const stackLimitPureSource = `module depth
pure func sum(n: int) -> int = if n <= 0 then 0 else 1 + sum(n - 1)
export pure func main(n: int) -> int = sum(n)
`

// IO-free entries ensure strict bytecode cannot hide an overflow via replay.
func TestStackLimitPureParity(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join(t.TempDir(), "depth.ail")
	writeFile(t, src, stackLimitPureSource)
	assertStackLimitFixtureChecks(t, bin, src)
	for _, tc := range []struct {
		name, depth, limit, wantError string
	}{
		{"default", "9000", "", ""},
		{"small", "200", "50", "overflow"},
		{"raised", "11001", "12000", ""},
		{"zero", "9000", "0", ""},
		{"negative", "9000", "-1", ""},
		{"zeroAbove", "11001", "0", "overflow"},
		{"negativeAbove", "11001", "-1", "overflow"},
		{"aboveDefault", "11001", "", "overflow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var outputs []string
			for _, bytecode := range []bool{false, true} {
				args := []string{"run", "--relax-modules", "--entry", "main", "--args-json", tc.depth}
				if tc.limit != "" {
					args = append(args, "--max-recursion-depth", tc.limit)
				}
				if bytecode {
					args = append(args, "--bytecode", "--strict-bytecode")
				}
				out, stderr, code := runWithStdin(t, bin, "", append(args, src)...)
				if tc.wantError != "" {
					want := "RT_REC_003"
					if bytecode {
						want = "stack overflow"
					}
					if code == 0 || !strings.Contains(stderr, want) {
						t.Errorf("bytecode=%v: exit=%d stdout=%q stderr=%s; want %s", bytecode, code, out, stderr, want)
					}
				} else {
					if code != 0 || strings.TrimSpace(out) != tc.depth {
						t.Errorf("bytecode=%v: exit=%d stdout=%q stderr=%s; want %s", bytecode, code, out, stderr, tc.depth)
					}
					outputs = append(outputs, out)
				}
			}
			if len(outputs) == 2 && outputs[0] != outputs[1] {
				t.Errorf("backend output differs: %q", outputs)
			}
		})
	}
}

const stackLimitListSource = `import std/io (readLine, println)
import std/list (length)
pure func rep(n: int, x: string) -> [string] = if n <= 0 then [] else x :: rep(n - 1, x)
`

func TestStackLimitServiceParity(t *testing.T) {
	assertStackLimitIOParity(t, "service", `module service
`+stackLimitListSource+`
func handle(line: string) -> () ! {IO} {
 let xs = rep(1170, line);
 println("HANDLED len=${show(length(xs))}")
}
func loop() -> () ! {IO} {
 let line = readLine(());
 if line == "" then () else {
  if line == "hello" then println("HELLO") else {
   println("ASK");
   handle(line);
   println("HANDLED-AFTER")
  };
  loop()
 }
}
export func main() -> () ! {IO} = loop()
`, "hello\nvoice\n", "HELLO\nASK\nHANDLED len=1170\nHANDLED-AFTER\n")
}

func TestStackLimitPrintExactlyOnce(t *testing.T) {
	assertStackLimitIOParity(t, "standalone", `module standalone
`+stackLimitListSource+`
export func main() -> () ! {IO} {
 println("START");
 let xs = rep(5000, "x");
 println("DONE len=${show(length(xs))}")
}
`, "", "START\nDONE len=5000\n")
}

func assertStackLimitIOParity(t *testing.T, module, source, stdin, want string) {
	t.Helper()
	bin := buildAilang(t)
	src := filepath.Join(t.TempDir(), module+".ail")
	writeFile(t, src, source)
	assertStackLimitFixtureChecks(t, bin, src)
	for _, bytecode := range []bool{false, true} {
		t.Run(fmt.Sprintf("bytecode=%v", bytecode), func(t *testing.T) {
			args := []string{"run", "--relax-modules", "--verbose", "--caps", "IO", "--entry", "main"}
			if bytecode {
				args = append(args, "--bytecode")
			}
			out, stderr, code := runWithStdin(t, bin, stdin, append(args, src)...)
			// Normalize native line endings for Windows CLI runs.
			out = strings.ReplaceAll(out, "\r\n", "\n")
			if code != 0 || out != want || strings.Contains(stderr, "falling back to evaluator") {
				t.Errorf("exit=%d stdout=%q stderr=%s; want %q without replay", code, out, stderr, want)
			}
		})
	}
}

func assertStackLimitFixtureChecks(t *testing.T, bin, src string) {
	t.Helper()
	out, stderr, code := runWithStdin(t, bin, "", "check", "--relax-modules", src)
	if code != 0 {
		t.Fatalf("fixture check: exit=%d stdout=%s stderr=%s", code, out, stderr)
	}
}
