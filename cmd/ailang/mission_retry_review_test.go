package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReviewBundleExclusiveCompletePublication(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "review.json")
	bodies := [][]byte{[]byte("{\"work\":true}\n"), []byte("{\"ready\":true}\n"), []byte("models: {}\n")}
	if err := writeReviewFiles(path, bodies); err != nil {
		t.Fatal(err)
	}
	for i, suffix := range []string{"", ".manifest.json", ".models.yml"} {
		got, err := os.ReadFile(path + suffix)
		if err != nil || !bytes.Equal(got, bodies[i]) {
			t.Fatalf("%s: %q %v", suffix, got, err)
		}
		info, err := os.Stat(path + suffix)
		if err != nil {
			t.Fatalf("stat %s: %v", suffix, err)
		}
		// Windows has NO assertable permission property here, not merely a weaker one: a file
		// created 0600 reports -rw-rw-rw- (0666), so even "no group or world bits" is false
		// there. Measured on the runner 2026-09-08 — an earlier attempt asserted exactly that
		// portable half and failed. The owner-only guarantee is a unix guarantee; assert it
		// where it exists and say plainly that it does not exist elsewhere.
		if runtime.GOOS == "windows" {
			continue
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("unsafe mode %v on %s", info.Mode().Perm(), suffix)
		}
	}
	if err := writeReviewFiles(path, [][]byte{[]byte("changed"), nil, nil}); err == nil {
		t.Fatal("overwrote existing bundle")
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, bodies[0]) {
		t.Fatal("original input changed")
	}
}
func TestReviewBundleRefusesSidecarCollisionAndSymlink(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "review.json")
	if err := os.Symlink(filepath.Join(dir, "victim"), path+".models.yml"); err != nil {
		t.Fatal(err)
	}
	if err := writeReviewFiles(path, [][]byte{[]byte("{}"), nil, nil}); err == nil {
		t.Fatal("accepted sidecar redirect")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatal("published runnable input before sidecars")
	}
	if _, err := os.Lstat(filepath.Join(dir, "victim")); !os.IsNotExist(err) {
		t.Fatal("followed symlink")
	}
}
func TestRetryReviewCLIRejectsBadInputsWithoutState(t *testing.T) {
	deps, db := iterationTestDeps(t)
	for _, args := range [][]string{nil, {""}, {"docs", "--work-item", "x", "--work-item", "y"}, {"docs", "--work-item", "x", "--output", "out", "--new-id", "y", "--max-tokens", "1000", "--timeout-seconds", "10", "--max-cost-usd", "1"}} {
		if err := runMissionRetryReview(context.Background(), args, &bytes.Buffer{}, deps); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatal("prepare created state")
	}
}
