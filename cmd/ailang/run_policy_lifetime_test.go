//go:build unix

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The supervisor's lifeline is independent of its process group. Every failure
// cleanup below is scoped to positive PIDs acknowledged by this unique fixture.
func TestRunPolicy_SupervisorDeathStopsWorker(t *testing.T) {
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skip("POSIX sleep fixture unavailable")
	}
	bystander := exec.Command(sleep, "60")
	if err := bystander.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bystander.Process.Kill(); _ = bystander.Wait() }()
	bin := buildAilang(t)
	dir := t.TempDir()
	pol := writePolicy(t, dir, "security_mode = \"trusted_host\"\nallowed_caps = [\"IO\", \"Process\"]\nprocess_allow = [\"sh\"]\nentry = \"main\"\ntimeout_ms = 60000\nmax_output_bytes = 0\nfs_sandbox = \"${SANDBOX}\"\n")
	sandbox := filepath.Join(dir, "sandbox")
	writeAil(t, sandbox, "prog.ail", `module prog
import std/process (exec)
export func main() -> () ! {IO, Process} = {
  let result = exec("sh", ["-c", "echo $PPID > worker.pid.tmp; mv worker.pid.tmp worker.pid; echo $$ > subprocess.pid.tmp; mv subprocess.pid.tmp subprocess.pid; exec sleep 60"]);
  println(show(result))
}
`)
	cmd := exec.Command(bin, "run", "--policy", pol, "prog.ail")
	cmd.Dir = sandbox
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	pids := []int{}
	defer func() {
		for _, pid := range pids {
			if pid > 1 && !processGone(pid) {
				killProcess(pid)
			}
		}
	}()
	deadline := time.Now().Add(20 * time.Second)
	for _, name := range []string{"worker.pid", "subprocess.pid"} {
		for {
			raw, err := os.ReadFile(filepath.Join(sandbox, name))
			if err == nil {
				pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
				if err != nil || pid <= 1 {
					t.Fatalf("invalid fixture pid %q", raw)
				}
				pids = append(pids, pid)
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("fixture did not acknowledge %s", name)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	waited = true
	deadline = time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		allGone := true
		for _, pid := range pids {
			allGone = allGone && processGone(pid)
		}
		if allGone {
			if err := syscall.Kill(bystander.Process.Pid, 0); err != nil {
				t.Fatalf("unrelated bystander terminated: %v", err)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("policy worker/subprocess survived lost supervisor: %v", pids)
}
