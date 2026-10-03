//go:build unix

package proctree

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func alive(pid int) bool { return syscall.Kill(pid, 0) == nil }

// The measured failure: a command started DETACHED (its own process group, like pi's bash
// tool) inside the workspace survives a kill of the agent's group. ReapWorkspace must find it
// by cwd and kill it, and must leave a process outside the workspace alone.
func TestReapWorkspace_KillsDetachedStragglerInWorkspace(t *testing.T) {
	ws := t.TempDir()
	other := t.TempDir()

	straggler := exec.Command("/bin/sh", "-c", "sleep 300")
	straggler.Dir = ws
	straggler.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := straggler.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = syscall.Kill(-straggler.Process.Pid, syscall.SIGKILL) }()

	bystander := exec.Command("/bin/sh", "-c", "sleep 300")
	bystander.Dir = other
	bystander.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := bystander.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bystander.Process.Kill(); _, _ = bystander.Process.Wait() }()

	time.Sleep(200 * time.Millisecond) // let both reach their cwd before the scan

	found, err := ReapWorkspace(ws)
	if err != nil {
		t.Fatalf("ReapWorkspace: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("found no process in the workspace; the detached straggler survived")
	}
	// The straggler sleeps 300s, so it exits within the deadline ONLY if the reap killed it.
	// (A bare Wait here passed with the kill removed: it simply waited the sleep out.)
	exited := make(chan struct{})
	go func() { _, _ = straggler.Process.Wait(); close(exited) }()
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatalf("straggler %d still running 5s after reap", straggler.Process.Pid)
	}
	if !alive(bystander.Process.Pid) {
		t.Errorf("bystander %d outside the workspace was killed", bystander.Process.Pid)
	}
}

func TestReapWorkspace_RefusesRootAndHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	for _, dir := range []string{"/", "/tmp", home} {
		if _, err := ReapWorkspace(dir); err == nil {
			t.Errorf("ReapWorkspace(%q) did not refuse", dir)
		}
	}
	if found, err := ReapWorkspace(""); err != nil || found != nil {
		t.Errorf("empty dir: found=%v err=%v, want a no-op", found, err)
	}
}
