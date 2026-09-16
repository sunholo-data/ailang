package executor

import (
	"context"
	"github.com/sunholo-data/ailang/internal/testutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMaterializeAgentPolicy_ReadOnlyOutsideWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix permission bits")
	}
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	// The 0555 dir is the property under test; give TempDir's cleanup its bits back.
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(home, ".ailang", "agent-policy"), 0o755) })
	ws := filepath.Join(home, "work")
	p, err := MaterializeAgentPolicy("allowed_caps = [\"IO\"]\n", ws)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(filepath.Dir(p)) == ws || p == "" {
		t.Fatalf("policy path %q must be outside the workspace", p)
	}
	if err := os.WriteFile(p, []byte("allowed_caps = [\"Net\"]\n"), 0o644); err == nil {
		t.Fatal("the materialised policy must not be writable")
	}
	// Re-materialising (a second task in the same container) must succeed.
	if _, err := MaterializeAgentPolicy("allowed_caps = []\n", ws); err != nil {
		t.Fatalf("re-materialise: %v", err)
	}
	if p2, err := MaterializeAgentPolicy("fs_sandbox = \"${WORKSPACE}\"\n", ws); err != nil {
		t.Fatal(err)
	} else if b, _ := os.ReadFile(p2); !strings.Contains(string(b), "fs_sandbox = \""+ws+"\"") {
		t.Fatalf("${WORKSPACE} not expanded: %s", b)
	}
	if got, _ := MaterializeAgentPolicy("", ws); got != "" {
		t.Fatalf("no content must mean no path (default-deny), got %q", got)
	}
}

func TestVersionProbe_BoundedOnAHangingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix shell fake")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "hang")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 60 &\nwait\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	p := &VersionProbe{CLI: "x", Path: bin}
	start := time.Now()
	if got := p.Identity(context.Background()); got != "" {
		t.Fatalf("hanging binary must be unmeasured, got %q", got)
	}
	if el := time.Since(start); el > VersionProbeTimeout+2*time.Second {
		t.Fatalf("probe took %v; must be bounded by %v", el, VersionProbeTimeout)
	}
	start = time.Now()
	_ = p.Identity(context.Background())
	if el := time.Since(start); el > time.Second {
		t.Fatalf("second Identity() re-probed (%v); a failed probe must be remembered", el)
	}
}
