package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestWarnSilentRatchet_WarnWhenVersionBumpsWithoutMessage(t *testing.T) {
	prevLF := &pkg.LockFile{
		Packages: []pkg.LockedPackage{
			{Name: "sunholo/myext", Version: "0.1.0", Source: "path", ContentHash: "sha256:aa", Effects: []string{}, Exports: []string{}},
		},
	}
	resolved := []pkg.ResolvedPackage{
		{Name: "sunholo/myext", Version: "0.1.1", Source: "path"},
	}

	// Redirect stderr to capture warning output.
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Point ~/.ailang to a temp dir with an empty DB (no upgrade-available messages).
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	warnSilentRatchet(resolved, prevLF)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	// Warning should be emitted: version changed with no upgrade-available message.
	if out == "" {
		t.Error("expected warning on stderr when version bumped without message, got nothing")
	}
	if !strings.Contains(out, "sunholo/myext") || !strings.Contains(out, "0.1.1") {
		t.Errorf("expected warning to mention package and new version, got: %q", out)
	}
	if !strings.Contains(out, "ailang publish") {
		t.Errorf("expected warning to mention 'ailang publish', got: %q", out)
	}
}

func TestWarnSilentRatchet_NoWarnWhenVersionUnchanged(t *testing.T) {
	prevLF := &pkg.LockFile{
		Packages: []pkg.LockedPackage{
			{Name: "sunholo/myext", Version: "0.1.0", Source: "path", ContentHash: "sha256:aa", Effects: []string{}, Exports: []string{}},
		},
	}
	resolved := []pkg.ResolvedPackage{
		{Name: "sunholo/myext", Version: "0.1.0", Source: "path"},
	}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	warnSilentRatchet(resolved, prevLF)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if out != "" {
		t.Errorf("expected no output when version unchanged, got: %q", out)
	}
}

func TestWarnSilentRatchet_NoWarnWhenMessageExists(t *testing.T) {
	prevLF := &pkg.LockFile{
		Packages: []pkg.LockedPackage{
			{Name: "sunholo/myext", Version: "0.1.0", Source: "path", ContentHash: "sha256:aa", Effects: []string{}, Exports: []string{}},
		},
	}
	resolved := []pkg.ResolvedPackage{
		{Name: "sunholo/myext", Version: "0.1.1", Source: "path"},
	}

	// Create a temp DB and insert an upgrade-available message for 0.1.1.
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	dbPath := messaging.GetDefaultDatabasePath()
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	env := &messaging.PackageMessageEnvelope{
		Schema:  messaging.PackageMessageSchema,
		Kind:    messaging.PkgMsgUpgradeAvailable,
		From:    messaging.FormatPackageInbox("sunholo/myext"),
		To:      []string{messaging.FormatPackageInbox("sunholo/myext")},
		Summary: "test bump",
		Status:  "open",
		Package: messaging.PackageRef{
			Name:        "sunholo/myext",
			FromVersion: "0.1.0",
			ToVersion:   "0.1.1",
			ChangeClass: "B",
		},
	}
	msg, err := env.ToInboxMessage()
	if err != nil {
		t.Fatalf("ToInboxMessage: %v", err)
	}
	if err := store.InsertInboxMessage(msg); err != nil {
		t.Fatalf("InsertInboxMessage: %v", err)
	}
	store.Close()

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	warnSilentRatchet(resolved, prevLF)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if out != "" {
		t.Errorf("expected no warning when upgrade-available message exists, got: %q", out)
	}
}

func TestWarnSilentRatchet_SkipsRegistryDeps(t *testing.T) {
	prevLF := &pkg.LockFile{
		Packages: []pkg.LockedPackage{
			{Name: "sunholo/myext", Version: "0.1.0", Source: "registry", ContentHash: "sha256:aa", Effects: []string{}, Exports: []string{}},
		},
	}
	resolved := []pkg.ResolvedPackage{
		// registry source — should NOT trigger ratchet warning
		{Name: "sunholo/myext", Version: "0.1.1", Source: "registry"},
	}

	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	warnSilentRatchet(resolved, prevLF)

	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	if out != "" {
		t.Errorf("expected no warning for registry dep, got: %q", out)
	}
}
