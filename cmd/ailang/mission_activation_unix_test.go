//go:build darwin || linux

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestActivationGateHelper(t *testing.T) {
	dir := os.Getenv("AILANG_TEST_ACTIVATION_GATE")
	if dir == "" {
		return
	}
	var gate [1]byte
	if _, err := io.ReadFull(os.Stdin, gate[:]); err != nil {
		os.Exit(23)
	}
	p, err := readActivationProcess(dir, "gate")
	if err != nil || p.Phase != "armed" || p.SessionID != os.Getpid() || gate[0] != 'R' {
		os.Exit(24)
	}
	sid, err := unix.Getsid(0)
	if err != nil || sid != p.SessionID {
		os.Exit(25)
	}
	if err = os.WriteFile(filepath.Join(dir, "released"), []byte("verified"), 0600); err != nil {
		os.Exit(26)
	}
	os.Exit(0)
}
func TestActivationSupervisorDurableReleaseGate(t *testing.T) {
	dir := t.TempDir()
	p := activationProcess{Version: 1, OperationID: "gate", WorkItemID: "work", WorkFile: filepath.Join(dir, "gate.work-item.json"), SpecDigest: strings.Repeat("a", 64), LauncherPID: os.Getpid(), Phase: "launching"}
	if err := createActivationProcess(dir, p); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestActivationGateHelper$")
	cmd.Env = append(os.Environ(), "AILANG_TEST_ACTIVATION_GATE="+dir)
	if err := superviseActivationCommand(context.Background(), dir, &p, &bytes.Buffer{}, cmd); err != nil {
		t.Fatal(err)
	}
	got, err := readActivationProcess(dir, "gate")
	if err != nil || got.Phase != "exited" || got.SessionID < 1 {
		t.Fatalf("missing process evidence: %+v %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "released")); err != nil {
		t.Fatal("child did not pass durable gate")
	}
}
func TestActivationParentDeathBeforeReleaseCannotDispatch(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestActivationGateHelper$")
	cmd.Env = append(os.Environ(), "AILANG_TEST_ACTIVATION_GATE="+dir)
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	cmd.Stdin = read
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = write.Close()
	err = cmd.Wait()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 23 {
		t.Fatalf("gate did not refuse EOF: %v", err)
	}
	if _, err = os.Stat(filepath.Join(dir, "released")); !os.IsNotExist(err) {
		t.Fatal("child ran without durable release")
	}
}
