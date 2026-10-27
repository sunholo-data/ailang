package eval_harness

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestMaintenanceSnapshotReadDoesNotEscapeDuringAncestorRename(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX rename/symlink concurrency control; Windows path guards covered separately")
	}
	root := t.TempDir()
	app := filepath.Join(root, "app")
	secret := filepath.Join(root, "controller")
	owned := filepath.Join(app, "owned")
	parked := filepath.Join(app, "parked")
	for _, dir := range []string{app, secret, owned} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(owned, "source"), []byte("PUBLIC"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secret, "source"), []byte("PRIVATE_ORACLE"), 0600); err != nil {
		t.Fatal(err)
	}
	data, _, err := maintenanceReadRegular(app, "owned/source")
	if err != nil || string(data) != "PUBLIC" {
		t.Fatal("positive source control failed", err)
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			if os.Rename(owned, parked) == nil {
				if os.Symlink(secret, owned) == nil {
					runtime.Gosched()
					_ = os.Remove(owned)
				}
				_ = os.Rename(parked, owned)
			}
		}
	}()
	defer func() { close(done); wg.Wait() }()
	for i := 0; i < 2000; i++ {
		data, _, err := maintenanceReadRegular(app, "owned/source")
		if err == nil && string(data) != "PUBLIC" {
			t.Fatalf("controller read outside app during candidate rename: %q", data)
		}
	}
}
