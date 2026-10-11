package runlife

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFrameValidation(t *testing.T) {
	for _, input := range []string{`{"version":1,"kind":"ready"} {}` + "\n", `{"version":2,"kind":"ready"}` + "\n", strings.Repeat("x", 1025) + "\n", `{"version":1,"kind":"receipt","pid":123}` + "\n", `{"version":1,"kind":"unknown"}` + "\n"} {
		if _, err := readFrame(strings.NewReader(input)); err == nil {
			t.Errorf("accepted malformed lifecycle frame %q", input)
		}
	}
	for _, kind := range []string{"ready", "stop", "receipt"} {
		if _, err := readFrame(strings.NewReader(`{"version":1,"kind":"` + kind + `"}` + "\n")); err != nil {
			t.Errorf("valid %s: %v", kind, err)
		}
	}
}

func TestPrepareOnlySameExecutableRun(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("private run lifetime requires supported process-group platform")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"version"}, {"runaway"}, nil} {
		child, err := Prepare(exec.Command(exe, args...), nil)
		if err != nil || child != nil {
			t.Fatalf("non-run registered: %v %v", child, err)
		}
	}
	cmd := exec.Command(exe, "run")
	child, err := Prepare(cmd, nil)
	if err != nil || child == nil {
		t.Fatalf("same exe not registered: %v", err)
	}
	child.Abort()
	link := filepath.Join(t.TempDir(), "runtime-alias")
	if err := os.Symlink(exe, link); err == nil {
		child, err = Prepare(exec.Command(link, "run"), nil)
		if err != nil || child == nil {
			t.Fatalf("symlink identity rejected: %v", err)
		}
		child.Abort()
	}
	other := filepath.Join(t.TempDir(), "different-runtime")
	if err := os.WriteFile(other, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	child, err = Prepare(exec.Command(other, "run"), nil)
	if err != nil || child != nil {
		t.Fatalf("other exe registered: %v %v", child, err)
	}
}
