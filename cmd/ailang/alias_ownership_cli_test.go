package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the actual compiler before both runtime engines. The unused callback
// still belongs to the caller: forwarding does not make its IO/FS row disappear.
func TestAliasOwnershipCLI(t *testing.T) {
	bin := buildAilang(t)
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	t.Setenv("AILANG_NO_CACHE", "1")
	t.Setenv("AILANG_RELAX_MODULES", "1")
	for _, imported := range []bool{false, true} {
		for _, effect := range []string{"IO", "FS"} {
			for _, field := range []string{"authority", "appraisal"} {
				for _, declared := range []bool{false, true} {
					t.Run(fmt.Sprintf("imported_%v/%s/%s/declared_%v", imported, effect, field, declared), func(t *testing.T) {
						dir := t.TempDir()
						input := filepath.ToSlash(filepath.Join(dir, "input.txt"))
						if err := os.WriteFile(filepath.Join(dir, "input.txt"), []byte("fixture"), 0600); err != nil {
							t.Fatal(err)
						}
						action := "println(\"callback ran\")"
						if effect == "FS" {
							action = fmt.Sprintf("readFile(%q)", input)
						}
						row := ""
						if declared {
							row = " ! {" + effect + "}"
						}
						source := `module main
import std/io (println)
import std/fs (readFile)
type Policies = {authority: int -> int, appraisal: int -> int}
pure func accept(p: Policies) -> int = acceptReady(p)
pure func acceptReady(p: Policies) -> int = p.authority(7)
pure func quiet(x: int) -> int = x
func loud(x: int) -> int ! {` + effect + `} { let _ = ` + action + `; x }
func caller() -> int` + row + ` = accept(` + map[string]string{"authority": "{authority: loud, appraisal: quiet}", "appraisal": "{authority: quiet, appraisal: loud}"}[field] + `)
export func main() -> () ! {IO, FS} = println(show(caller()))
`
						if imported {
							decl := "type Policies = {authority: int -> int, appraisal: int -> int}"
							source = strings.Replace(source, decl, "import policies (Policies)", 1)
							if err := os.WriteFile(filepath.Join(dir, "policies.ail"), []byte("module policies\nexport "+decl+"\n"), 0600); err != nil {
								t.Fatal(err)
							}
						}
						path := filepath.Join(dir, "main.ail")
						if err := os.WriteFile(path, []byte(source), 0600); err != nil {
							t.Fatal(err)
						}
						for _, bytecode := range []bool{false, true} {
							args := []string{"run", "--quiet", "--caps", "IO,FS", "--entry", "main"}
							if bytecode {
								args = append(args, "--bytecode", "--verbose")
								// std/fs.readFile is evaluator-only today. The normal
								// bytecode entry supports that bridge; strict mode
								// remains required for rejection and every other case.
								if !(declared && effect == "FS" && field == "authority") {
									args = append(args, "--strict-bytecode")
								}
							}
							res := runCLIIn(t, bin, t.TempDir(), dir, append(args, path)...)
							stdout, stderr, code := res.stdout, res.stderr, res.exitCode
							if declared {
								if bytecode && (!strings.Contains(stderr, "via bytecode VM") || strings.Contains(stderr, "falling back to evaluator")) {
									t.Fatalf("expected VM dispatch with explicit interop support: %s", stderr)
								}
								expected := "7"
								if effect == "IO" && field == "authority" {
									expected = "callback ran\n7"
								}
								if code != 0 || strings.ReplaceAll(strings.TrimSpace(stdout), "\r\n", "\n") != expected {
									t.Fatalf("bytecode=%v declared call: code=%d stdout=%q stderr=%s", bytecode, code, stdout, stderr)
								}
							} else {
								if code == 0 || stdout != "" || !strings.Contains(stderr, "Missing effects: "+effect) || !strings.Contains(stderr, "function 'caller'") {
									t.Fatalf("bytecode=%v rejection: code=%d stdout=%q stderr=%s", bytecode, code, stdout, stderr)
								}
							}
						}
					})
				}
			}
		}
	}
}
