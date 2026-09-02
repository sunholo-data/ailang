package testutil

import (
	"os"
	"testing"
)

func TestSetHomeDir(t *testing.T) {
	dir := t.TempDir()
	SetHomeDir(t, dir)

	for _, name := range []string{"HOME", "USERPROFILE", "home"} {
		if got := os.Getenv(name); got != dir {
			t.Errorf("%s = %q, want %q", name, got, dir)
		}
	}
	got, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir(): %v", err)
	}
	if got != dir {
		t.Errorf("os.UserHomeDir() = %q, want %q", got, dir)
	}
}

func TestSetHomeDirEmptyMakesUserHomeDirFail(t *testing.T) {
	SetHomeDir(t, "")
	if got, err := os.UserHomeDir(); err == nil {
		t.Fatalf("os.UserHomeDir() = %q, want an error", got)
	}
}
