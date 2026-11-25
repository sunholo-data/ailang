//go:build !windows

package agy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// A run on a binary that is not the pinned version is refused BEFORE agy is
// asked to do anything: `ailang exec agy` (the mission lane probe) never called
// HealthCheck, so a self-updated 1.5.16 ran unpinned on 2026-11-24.
func TestExecute_RefusesUnpinnedBinary(t *testing.T) {
	setupHomes(t)
	m := newMockAgy(t, "1.5.16", "edit", 0, "")
	e := newExec(t, m)
	task := &executor.Task{ID: "t1", Directive: "x", Workspace: t.TempDir(), Model: "gemini-4-neon-high", Timeout: 30 * time.Second}
	res, err := e.Execute(context.Background(), task)
	if err == nil || !strings.Contains(err.Error(), "1.5.16") || !strings.Contains(err.Error(), ExpectedVersion) {
		t.Fatalf("Execute on an unpinned binary: res=%+v err=%v (want a version-mismatch error)", res, err)
	}
	if _, statErr := os.Stat(m.argsFile); statErr == nil {
		t.Fatal("agy was run with a prompt despite the version mismatch")
	}
}

// The pinned install (config.AgyBinFor) wins over PATH and ~/.local/bin, which
// belong to interactive agy and update themselves.
func TestDefaultAgyPath_PrefersPinnedInstall(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv("AILANG_AGY_HOME_ROOT", "")
	onPath := t.TempDir()
	t.Setenv("PATH", onPath)
	write := func(p string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	local := filepath.Join(home, ".local", "bin", "agy")
	write(local)
	write(filepath.Join(onPath, "agy"))
	if got := defaultAgyPath(); got != filepath.Join(onPath, "agy") {
		t.Fatalf("without a pinned install: %s, want the PATH agy", got)
	}
	pinned := config.AgyBinFor(home)
	write(pinned)
	if got := defaultAgyPath(); got != pinned {
		t.Fatalf("with a pinned install: %s, want %s", got, pinned)
	}
	if !strings.HasPrefix(pinned, filepath.Join(home, ".ailang", "state", "agy")) {
		t.Fatalf("pinned path %s is not under the agy HOME root", pinned)
	}
}
