package loader

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// M-EXECUTOR-POLICY-HARDENING M3 (AC6): once enabled, the first read wins.
func TestSourceSnapshot_FirstReadWins(t *testing.T) {
	ResetSourceSnapshotForTest()
	t.Cleanup(ResetSourceSnapshotForTest)
	dir := t.TempDir()
	p := filepath.Join(dir, "m.ail")
	if err := os.WriteFile(p, []byte("module m\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	EnableSourceSnapshot(0, 0)
	first, err := ReadSourceFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("module m\nexport func evil() -> () ! {Process} = ()\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := ReadSourceFile(filepath.Join(dir, ".", "m.ail")) // same file, different spelling
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != string(first) {
		t.Fatalf("snapshot returned the rewritten file: %q", second)
	}
	d, n, b := SourceSnapshotDigest()
	if d == "" || n != 1 || b != int64(len(first)) {
		t.Errorf("digest=%q files=%d bytes=%d", d, n, b)
	}
}

func TestSourceSnapshot_Caps(t *testing.T) {
	ResetSourceSnapshotForTest()
	t.Cleanup(ResetSourceSnapshotForTest)
	dir := t.TempDir()
	big := filepath.Join(dir, "big.ail")
	if err := os.WriteFile(big, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	small := filepath.Join(dir, "small.ail")
	if err := os.WriteFile(small, make([]byte, 40), 0o644); err != nil {
		t.Fatal(err)
	}
	EnableSourceSnapshot(64, 70)
	if _, err := ReadSourceFile(big); !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("per-file cap: %v", err)
	}
	if _, err := ReadSourceFile(small); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "other.ail")
	if err := os.WriteFile(other, make([]byte, 40), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSourceFile(other); !errors.Is(err, ErrSourceTooLarge) {
		t.Fatalf("aggregate cap: %v", err)
	}
	if _, n, _ := SourceSnapshotDigest(); n != 1 {
		t.Errorf("a refused file must not be recorded: %d files", n)
	}
}

func TestSourceSnapshot_DisabledIsPlainRead(t *testing.T) {
	ResetSourceSnapshotForTest()
	dir := t.TempDir()
	p := filepath.Join(dir, "m.ail")
	_ = os.WriteFile(p, []byte("a"), 0o644)
	if b, err := ReadSourceFile(p); err != nil || string(b) != "a" {
		t.Fatal(err)
	}
	_ = os.WriteFile(p, []byte("b"), 0o644)
	if b, _ := ReadSourceFile(p); string(b) != "b" {
		t.Fatal("disabled snapshot must re-read")
	}
	if d, _, _ := SourceSnapshotDigest(); d != "" {
		t.Fatal("no digest when disabled")
	}
}
