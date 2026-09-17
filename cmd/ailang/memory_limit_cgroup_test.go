package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-V1-MEMORY-FOOTPRINT M4 (D-D): `--max-memory cgroup` derives the Go soft
// memory limit from the container's cgroup limit x0.9 — opt-in, never
// inferred. "max", a missing file and the v1 "unlimited" sentinel all mean
// no limit, and say so.

func fakeCgroup(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestCgroupLimitV2(t *testing.T) {
	root := fakeCgroup(t, map[string]string{"memory.max": "1073741824\n"})
	got, src, err := cgroupMemoryLimitFrom(root)
	if err != nil || got != 1<<30 || !strings.Contains(src, "memory.max") {
		t.Fatalf("got %d %q %v", got, src, err)
	}
}

func TestCgroupLimitV2MaxMeansNone(t *testing.T) {
	root := fakeCgroup(t, map[string]string{"memory.max": "max\n"})
	got, src, err := cgroupMemoryLimitFrom(root)
	if err != nil || got != 0 || !strings.Contains(src, "no limit") {
		t.Fatalf("got %d %q %v", got, src, err)
	}
}

func TestCgroupLimitV1FallbackAndSentinel(t *testing.T) {
	root := fakeCgroup(t, map[string]string{"memory/memory.limit_in_bytes": "536870912\n"})
	got, _, err := cgroupMemoryLimitFrom(root)
	if err != nil || got != 512<<20 {
		t.Fatalf("v1: got %d %v", got, err)
	}
	root = fakeCgroup(t, map[string]string{"memory/memory.limit_in_bytes": "9223372036854771712\n"})
	got, src, err := cgroupMemoryLimitFrom(root)
	if err != nil || got != 0 || !strings.Contains(src, "no limit") {
		t.Fatalf("v1 sentinel: got %d %q %v", got, src, err)
	}
}

func TestCgroupLimitMissingIsNoLimitNotError(t *testing.T) {
	got, src, err := cgroupMemoryLimitFrom(t.TempDir())
	if err != nil || got != 0 || !strings.Contains(src, "no limit") {
		t.Fatalf("got %d %q %v", got, src, err)
	}
}

func TestResolveMemoryLimitPrecedence(t *testing.T) {
	root := fakeCgroup(t, map[string]string{"memory.max": "1000000000\n"})
	cgroupRootForTest = root
	t.Cleanup(func() { cgroupRootForTest = "" })

	// Nothing asked: nothing applied.
	t.Setenv("AILANG_MEMLIMIT", "")
	if n, src, err := resolveMemoryLimit(""); err != nil || n != 0 || src != "none" {
		t.Fatalf("default: %d %q %v", n, src, err)
	}
	// Explicit size.
	if n, _, err := resolveMemoryLimit("256MB"); err != nil || n != 256<<20 {
		t.Fatalf("size: %d %v", n, err)
	}
	// Opt-in cgroup: x0.9.
	n, src, err := resolveMemoryLimit("cgroup")
	if err != nil || n != 900000000 || !strings.Contains(src, "cgroup") {
		t.Fatalf("cgroup: %d %q %v", n, src, err)
	}
	// Env spelling, same values.
	t.Setenv("AILANG_MEMLIMIT", "cgroup")
	if n, _, err := resolveMemoryLimit(""); err != nil || n != 900000000 {
		t.Fatalf("env cgroup: %d %v", n, err)
	}
	t.Setenv("AILANG_MEMLIMIT", "64MB")
	if n, _, err := resolveMemoryLimit(""); err != nil || n != 64<<20 {
		t.Fatalf("env size: %d %v", n, err)
	}
	// Flag beats env.
	if n, _, err := resolveMemoryLimit("128MB"); err != nil || n != 128<<20 {
		t.Fatalf("flag over env: %d %v", n, err)
	}
	// Malformed is an error, not a fallback.
	if _, _, err := resolveMemoryLimit("lots"); err == nil {
		t.Fatal("malformed accepted")
	}
	t.Setenv("AILANG_MEMLIMIT", "lots")
	if _, _, err := resolveMemoryLimit(""); err == nil {
		t.Fatal("malformed env accepted")
	}
}
