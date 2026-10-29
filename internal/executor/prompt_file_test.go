package executor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWritePromptFile(t *testing.T) {
	path, cleanup, err := WritePromptFile("ailang-test-prompt-*.md", "TEACHING PROMPT: ailang run")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "TEACHING PROMPT: ailang run" {
		t.Fatalf("file content = %q, %v", got, err)
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		// Windows reports every writable file as 0666; its ACLs are not mode bits.
		t.Errorf("mode = %v, want owner-only: the file holds the whole prompt", info.Mode().Perm())
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("prompt file survived cleanup: %v", err)
	}
}

func TestWritePromptFile_NoTempDirIsAnError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	t.Setenv("TMPDIR", missing) // unix
	t.Setenv("TMP", missing)    // windows
	t.Setenv("TEMP", missing)
	if _, _, err := WritePromptFile("x-*", "p"); err == nil {
		t.Fatal("WritePromptFile succeeded with no usable temp dir")
	}
}
