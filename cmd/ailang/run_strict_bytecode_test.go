package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_RunStrictBytecodeAlone(t *testing.T) {
	for _, tc := range []struct {
		name, source, caps string
		fails              bool
	}{
		{"vm_entry", "export func main() -> int = 7", "", false},
		{"clock", "import std/clock (now)\nexport func main() -> int ! {Clock} = now()", "Clock", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := filepath.Join(t.TempDir(), tc.name+".ail")
			if err := os.WriteFile(src, []byte("module test/"+tc.name+"\n"+tc.source+"\n"), 0644); err != nil {
				t.Fatal(err)
			}
			args := []string{"run", "--verbose", "--relax-modules", "--caps", tc.caps}
			_, stderr, code := runCLI(t, append(append([]string{}, args...), src)...)
			if code != 0 {
				t.Fatalf("evaluator failed: %s", stderr)
			}
			stdout, stderr, code := runCLI(t, append(args, "--strict-bytecode", src)...)
			if tc.fails {
				if code == 0 || !strings.Contains(stderr, "bytecode execution failed") {
					t.Fatalf("expected strict failure: exit=%d stderr=%s", code, stderr)
				}
			} else if code != 0 || !strings.Contains(stderr, "via bytecode VM") || !strings.Contains(stdout, "7") {
				t.Fatalf("expected VM result: exit=%d stdout=%s stderr=%s", code, stdout, stderr)
			}
		})
	}
}
