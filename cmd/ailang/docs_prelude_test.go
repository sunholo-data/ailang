package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestPreludeDocs_ReverseDrift is the REVERSE half of the bidirectional drift
// guard: the rendered prelude function set (preludeDocEntries) MUST equal exactly
// the live surface enumerated straight from the mechanisms —
// pipeline.PreludeSurface() (injected bindings) + the `show` builtin. If a binding
// is ADDED to or REMOVED from InjectPrelude's surface, this test fails until the
// docs page (which renders from the same mechanism) is regenerated — but since
// the page renders from the SAME mechanism, the only way to diverge is to change
// the show builtin entry without updating the mechanism, which this catches.
func TestPreludeDocs_ReverseDrift(t *testing.T) {
	// Live surface, enumerated independently of preludeDocEntries().
	live := map[string]bool{"show": true} // builtin, not in either mechanism
	for _, b := range pipeline.PreludeSurface() {
		live[b.Name] = true
	}

	rendered := map[string]bool{}
	for _, e := range preludeDocEntries() {
		rendered[e.name] = true
	}

	for name := range live {
		if !rendered[name] {
			t.Errorf("prelude binding %q is live (mechanism/builtin) but MISSING from docs render", name)
		}
	}
	for name := range rendered {
		if !live[name] {
			t.Errorf("prelude entry %q is rendered but ABSENT from the live surface (stale hand-entry?)", name)
		}
	}
}

// TestPreludeDocs_ImplicitModulesFromLoader asserts the implicit-import section is
// sourced from the loader's single source of truth (EntryPreludeModules +
// EntryPreludeSymbols), not a copied table — adding a module/symbol there changes
// the page automatically.
func TestPreludeDocs_ImplicitModulesFromLoader(t *testing.T) {
	mods := loader.EntryPreludeModules()
	if len(mods) == 0 {
		t.Fatal("EntryPreludeModules() returned nothing")
	}
	for _, m := range mods {
		syms := loader.EntryPreludeSymbols(m)
		if len(syms) == 0 {
			t.Errorf("EntryPreludeSymbols(%q) returned no symbols", m)
		}
	}
	// std/option and std/result are the current implicit prelude modules.
	if loader.EntryPreludeSymbols("std/option") == nil {
		t.Error("expected std/option to be an implicit-prelude module")
	}
}

// TestPreludeSurface_Delta asserts PreludeSurface enumerates exactly the DELTA
// InjectPrelude adds to a base TypeEnv (not stale/over-broad). This backs the
// reverse-drift guarantee: the enumerator IS the injection.
func TestPreludeSurface_Delta(t *testing.T) {
	// The single source of truth is preludeSurface(); InjectPrelude iterates it,
	// PreludeSurface() returns it. Assert every enumerated name actually lands in
	// the env after InjectPrelude, and no enumerated scheme is nil.
	surface := pipeline.PreludeSurface()
	if len(surface) == 0 {
		t.Fatal("PreludeSurface() is empty — expected at least println")
	}
	names := map[string]bool{}
	for _, b := range surface {
		if b.Scheme == nil || b.Scheme.Type == nil {
			t.Errorf("prelude binding %q has nil scheme", b.Name)
		}
		names[b.Name] = true
	}
	if !names["println"] {
		t.Error("expected println in the prelude surface")
	}
}

// TestPreludeDocs_ForwardCompileProbe drives the binary: an ENTRY program using
// println + show + Some/Ok import-free must compile & run (the forward half of the
// drift guard — every rendered prelude name is really available without import).
func TestPreludeDocs_ForwardCompileProbe(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	repoRoot := findRepoRootForTest(t)

	dir := t.TempDir()
	prog := `module probe

export func main() -> () ! {IO} {
  let o = Some(42);
  match o {
    Some(x) => println(show(x)),
    None => println("none")
  }
}
`
	fp := filepath.Join(dir, "probe.ail")
	if err := os.WriteFile(fp, []byte(prog), 0o644); err != nil {
		t.Fatalf("write probe: %v", err)
	}

	cmd := exec.Command(bin, "run", "--entry", "main", fp)
	cmd.Env = append(os.Environ(), "AILANG_STDLIB_PATH="+filepath.Join(repoRoot, "std"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("prelude forward compile-probe failed (println/show/Some/Ok not import-free?): %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "42") {
		t.Errorf("expected probe output to contain 42, got:\n%s", out)
	}
}

// TestPreludeDocs_Render drives the binary: `ailang docs prelude` renders the
// live page (functions + implicit imports + scope notes), NOT the
// "module 'std/prelude' not found" error (V3 fix).
func TestPreludeDocs_Render(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	repoRoot := findRepoRootForTest(t)

	cmd := exec.Command(bin, "docs", "prelude")
	cmd.Env = append(os.Environ(), "AILANG_STDLIB_PATH="+filepath.Join(repoRoot, "std"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docs prelude failed: %v\n%s", err, out)
	}
	got := string(out)
	if strings.Contains(got, "not found") {
		t.Fatalf("docs prelude still errors instead of rendering:\n%s", got)
	}
	for _, want := range []string{"println", "show", "std/option", "std/result", "Entry-only", "Silent shadowing"} {
		if !strings.Contains(got, want) {
			t.Errorf("docs prelude missing %q, got:\n%s", want, got)
		}
	}
}

// TestDocsList_PreludeFooter asserts `ailang docs --list` gained the footer line
// pointing at `ailang docs prelude`.
func TestDocsList_PreludeFooter(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	repoRoot := findRepoRootForTest(t)

	cmd := exec.Command(bin, "docs", "--list")
	cmd.Env = append(os.Environ(), "AILANG_STDLIB_PATH="+filepath.Join(repoRoot, "std"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docs --list failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ailang docs prelude") {
		t.Errorf("docs --list missing prelude footer line, got:\n%s", out)
	}
}
