package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestPreludeAndImportedPrintln_SameCapabilityGate pins the fix for a capability
// soundness hole found 2026-08-17.
//
// `println` reaches the runtime by two different paths:
//
//	bare `println`              -> prelude injection -> eval.registerBuiltins
//	`import std/io (println)`   -> _io_println -> effects.Call -> RequireCap
//
// Both declare the SAME effect (`! {IO}`) and both pass effect checking, but only
// the imported path consulted the capability set. The prelude builtin wrote to
// stdout via fmt directly, so a program could perform IO with the WRONG caps, or
// with NO caps at all — defeating the point of a capability system.
//
// It surfaced through an eval: benchmarks/prompt_injection.yml granted only
// Declassify while requiring stdout, so it passed for solutions using the
// evading bare form and FAILED solutions that correctly imported std/io.
//
// The invariant: for a given capability set, both spellings must agree.
func TestPreludeAndImportedPrintln_SameCapabilityGate(t *testing.T) {
	binary := testutil.FindAilangBinary(t)
	dir := t.TempDir()

	sources := map[string]string{
		"bare": `module bare
export func main() -> () ! {IO} { println("1") }
`,
		"imported": `module imported
import std/io (println)
export func main() -> () ! {IO} { println("1") }
`,
	}
	for name, src := range sources {
		if err := os.WriteFile(filepath.Join(dir, name+".ail"), []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	capCases := []struct {
		name      string
		caps      []string
		wantAllow bool
	}{
		{"no caps", nil, false},
		{"wrong cap", []string{"Declassify"}, false},
		{"correct cap", []string{"IO"}, true},
	}

	for _, cc := range capCases {
		for _, form := range []string{"bare", "imported"} {
			t.Run(cc.name+"/"+form, func(t *testing.T) {
				args := []string{"run", "--entry", "main"}
				if len(cc.caps) > 0 {
					args = append(args, "--caps", strings.Join(cc.caps, ","))
				}
				args = append(args, form+".ail")

				cmd := exec.Command(binary, args...)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "OTEL_SDK_DISABLED=true")
				out, err := cmd.CombinedOutput()
				got := string(out)

				printed := strings.Contains(got, "\n1\n") || strings.HasPrefix(got, "1\n") ||
					strings.TrimSpace(got) == "1"
				blocked := strings.Contains(got, "requires capability")

				if cc.wantAllow {
					if err != nil || !printed {
						t.Fatalf("%s with caps %v: expected the program to run and print 1.\nerr=%v\noutput:\n%s",
							form, cc.caps, err, got)
					}
					return
				}
				// Denied: must NOT perform the effect, and must say why.
				if printed {
					t.Fatalf("CAPABILITY BYPASS: %s performed IO with caps %v (expected refusal).\noutput:\n%s",
						form, cc.caps, got)
				}
				if !blocked {
					t.Fatalf("%s with caps %v: expected a capability error, got none.\noutput:\n%s",
						form, cc.caps, got)
				}
			})
		}
	}
}
