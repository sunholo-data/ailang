package runner

import (
	"os"
	"path/filepath"
	"testing"
)

// ailang#2052: `--entry main` on the VM must run the ENTRY module's main.
// FindEntryProto used to take the first prototype ending in ".main" or
// "_main", so an imported module whose id sorted ahead of the entry's, or a
// local `run_main` declared before `main`, ran instead — silently, exit 0,
// whenever the wrong function compiled natively.

func TestVMEntryIgnoresImportedMain(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	files := map[string]string{
		"ailang.toml": `[package]
name = "aa/p"
version = "0.1.0"
edition = "1"
[exports]
modules = ["aa/p/tools/gen", "aa/p/tools/verify"]
[effects]
max = ["IO"]
`,
		// Imported by package path, so its id ("aa/p/tools/gen") sorts ahead
		// of the entry's file-path id ("tools/verify").
		"tools/gen.ail": `module aa/p/tools/gen
import std/io (println)
export pure func helper() -> int = 1
export func main() -> () ! {IO} { println("WRONG gen.main") }
`,
		"tools/verify.ail": `module aa/p/tools/verify
import aa/p/tools/gen (helper)
import std/io (println)
export func main() -> () ! {IO} { println("verify.main ${show(helper())}") }
`,
	}
	for name, src := range files {
		if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, "tools/verify.ail", "", engine); got != "verify.main 1\n" {
				t.Fatalf("stdout = %q, want the entry module's main", got)
			}
		})
	}
}

func TestVMEntryIgnoresUnderscoreSuffix(t *testing.T) {
	t.Chdir(t.TempDir())
	source := `module entry_suffix
import std/io (println)
pure func run_main() -> int = 7
export func main() -> () ! {IO} { println("main ${show(run_main())}") }
`
	for _, engine := range allEngines {
		t.Run(engine, func(t *testing.T) {
			if got := runOnEngine(t, "entry_suffix.ail", source, engine); got != "main 7\n" {
				t.Fatalf("stdout = %q, want main (not run_main)", got)
			}
		})
	}
}
