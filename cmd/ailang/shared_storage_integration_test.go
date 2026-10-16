package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// Each runCLI invocation starts a separate OS process using the freshly built
// binary; reopening a Go connection alone would miss CLI/lifecycle regressions.
func TestCrossRunLibrary(t *testing.T) {
	t.Setenv("AILANG_SHAREDMEM_DB", "")
	t.Setenv("AILANG_SHAREDINDEX_DB", "")
	t.Setenv("AILANG_EMBED_CACHE_DB", "")
	file := "examples/reference/cross_run_library.ail"
	run := func(path, entry string) string {
		t.Helper()
		out, stderr, code := runCLI(t, "run", "--caps", "IO,AI,SharedMem,SharedIndex", "--ai-stub", "--sharedmem-db", path, "--sharedindex-db", path, "--embed-cache-db", path, "--entry", entry, file)
		if code != 0 {
			t.Fatalf("%s exit %d: %s %s", entry, code, out, stderr)
		}
		return strings.TrimSpace(out)
	}
	path := filepath.Join(t.TempDir(), "library.db")
	if out := run(path, "seed"); out != "seeded" {
		t.Fatalf("seed: %q", out)
	}
	if out := run(path, "query"); out != "The crew remembers a blue star beyond the Archive" {
		t.Fatalf("query: %q", out)
	}
	if out := run(filepath.Join(t.TempDir(), "fresh.db"), "query"); out != "no match" {
		t.Fatalf("fresh: %q", out)
	}
}

func TestPersistentStorageFlagsRequireCapabilities(t *testing.T) {
	for _, flag := range []string{"--sharedmem-db", "--sharedindex-db", "--embed-cache-db"} {
		t.Run(flag, func(t *testing.T) {
			out, stderr, code := runCLI(t, "run", flag, filepath.Join(t.TempDir(), "store.db"), "--entry", "main", "examples/runnable/hello.ail")
			if code == 0 || !strings.Contains(out+stderr, flag) {
				t.Fatalf("flag must fail explicitly: code %d, %s %s", code, out, stderr)
			}
		})
	}
}
