package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// writeFileAtomic replaces path with data so that a concurrent reader sees either
// the old file or the new one, never a prefix of the new one: the bytes go to a
// temp file in the same directory (same filesystem, so the rename is atomic on
// POSIX), are fsynced, and are renamed over path.
//
// M-PKG-PARALLEL-AGENTS V12: LockFile.Save used os.WriteFile, and agents sharing
// a checkout read truncated locks ("unexpected end of JSON input", 16 of ~9,700
// runs). Windows refuses to rename over a file another process has open, so the
// rename is retried briefly there; POSIX renames do not fail that way.
func writeFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return renameReplacing(tmpName, path)
}

// renameReplacing renames src over dst, retrying a bounded number of times on
// Windows where an open reader makes the replace fail transiently.
func renameReplacing(src, dst string) error {
	attempts := 1
	if runtime.GOOS == "windows" {
		attempts = 20
	}
	var err error
	for i := 0; i < attempts; i++ {
		if err = os.Rename(src, dst); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("replace %s: %w", dst, err)
}
