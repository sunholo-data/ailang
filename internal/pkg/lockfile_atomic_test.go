package pkg

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TST-PKGPAR-14
// A reader must never see a half-written ailang.lock. Measured before the fix
// (M-PKG-PARALLEL-AGENTS V12): 16 of ~9,700 concurrent runs failed with
// "unexpected end of JSON input" because Save used a plain os.WriteFile.
func TestLockFile_SaveIsAtomicUnderConcurrentReads(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Opening a file that is being renamed over fails transiently on Windows
		// (sharing violation), so readers there can error for reasons unrelated to
		// torn writes. The atomic-replace guarantee under test is the POSIX one.
		t.Skip("rename-over-open-file semantics differ on Windows")
	}
	dir := t.TempDir()
	packages := make([]LockedPackage, 0, 40)
	for i := 0; i < 40; i++ { // a lock big enough that a torn write is observable
		packages = append(packages, LockedPackage{
			Name:        "vendor/pkg" + strings.Repeat("x", i),
			Version:     "1.0.0",
			ContentHash: "sha256:" + strings.Repeat("a", 64),
			Source:      "path",
			Path:        "../p",
			Exports:     []string{"m/a", "m/b", "m/c"},
		})
	}
	lf := NewLockFile(packages, "ailang lock test")
	if err := lf.Save(dir); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var failures atomic.Int64
	var firstErr atomic.Value
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for time.Now().Before(deadline) {
			if err := lf.Save(dir); err != nil {
				failures.Add(1)
				firstErr.CompareAndSwap(nil, "save: "+err.Error())
			}
		}
	}()
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				if _, err := LoadLockFile(dir); err != nil {
					failures.Add(1)
					firstErr.CompareAndSwap(nil, "load: "+err.Error())
				}
			}
		}()
	}
	wg.Wait()

	if n := failures.Load(); n > 0 {
		t.Fatalf("%d failures under concurrent save/load; first: %v", n, firstErr.Load())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != LockFileName {
			t.Errorf("leftover file after saves: %s", e.Name())
		}
	}
}
