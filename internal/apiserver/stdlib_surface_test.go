package apiserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/loader"
)

// M-SERVEAPI-OPERATOR-SURFACE D4: a stdlib module is never local surface.
// Daneel measured 23 std/* endpoints in the banner of a module importing
// std/stream: the embedded stdlib's display path "<embedded>/std/stream.ail"
// is relative, so filepath.Abs put it under a base path equal to the cwd.

func stdLoaded(filePath, mod string) *loader.LoadedModule {
	return &loader.LoadedModule{
		Path: mod,
		File: &ast.File{Path: filePath, Module: &ast.ModuleDecl{Path: mod}},
		Iface: &iface.Iface{Module: mod, Exports: map[string]*iface.IfaceItem{
			"defaultConfig": {Name: "defaultConfig"},
		}},
	}
}

func TestRegisterModule_EmbeddedStdlibIsNotLocal(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	srv := New(cwd, Config{Port: "0"})
	defer srv.Close()

	// Instrument check: the synthetic path really does land under the base
	// path, so the test exercises the bug rather than the prefix filter.
	abs, _ := filepath.Abs("<embedded>/std/stream.ail")
	if rel, err := filepath.Rel(cwd, abs); err != nil || rel == abs {
		t.Fatalf("instrument check: %q should resolve under %q", abs, cwd)
	}

	key, created, err := srv.registerModule(stdLoaded("<embedded>/std/stream.ail", "std/stream"))
	if err != nil {
		t.Fatal(err)
	}
	if key != "" || created {
		t.Fatalf("embedded std/stream registered as local surface (key %q)", key)
	}
	if n := len(srv.GetModules()); n != 0 {
		t.Fatalf("modules = %d, want 0", n)
	}
	if d := srv.DroppedModules(); len(d) != 0 {
		t.Fatalf("a stdlib module is not a drop, got %+v", d)
	}
}

func TestRegisterModule_OnDiskStdlibIsNeverLocalOrDropped(t *testing.T) {
	base := t.TempDir()
	srv := New(base, Config{Port: "0"})
	defer srv.Close()

	// A stdlib root inside the project (the ./std tier) and one outside it.
	inside := filepath.Join(base, "std", "stream.ail")
	outside := filepath.Join(t.TempDir(), "std", "list.ail")
	for _, p := range []string{inside, outside} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("module x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct{ path, mod string }{{inside, "std/stream"}, {outside, "std/list"}} {
		if key, _, err := srv.registerModule(stdLoaded(c.path, c.mod)); err != nil || key != "" {
			t.Fatalf("%s registered (key %q, err %v)", c.mod, key, err)
		}
	}
	if n := len(srv.GetModules()); n != 0 {
		t.Fatalf("modules = %d, want 0", n)
	}
	if d := srv.DroppedModules(); len(d) != 0 {
		t.Fatalf("stdlib modules must not be recorded as drops (they fill /api/_health): %+v", d)
	}

	// Positive control: a local, non-std module with the same shape registers.
	local := filepath.Join(base, "talk.ail")
	if err := os.WriteFile(local, []byte("module talk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if key, _, err := srv.registerModule(stdLoaded(local, "talk")); err != nil || key == "" {
		t.Fatalf("positive control: local module not registered (key %q, err %v)", key, err)
	}
}
