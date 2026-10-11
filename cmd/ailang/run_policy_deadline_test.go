//go:build unix

package main

import (
	"bytes"
	"github.com/sunholo-data/ailang/internal/proctree"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Arm the worker about one second before the supervisor's absolute deadline. Continuing
// evaluation during a new cooperative grace would write after_limit about 200 ms past
// that boundary. The policy force path must not grant that execution time.
func TestRunPolicy_DeadlineStopsExecutionWithoutProtocolGrace(t *testing.T) {
	bin := buildAilang(t)
	root := t.TempDir()
	sandbox := filepath.Join(root, "sandbox")
	pol := writePolicy(t, root, "security_mode=\"restricted\"\nallowed_caps=[\"IO\",\"FS\",\"Clock\"]\nentry=\"main\"\nfs_sandbox=\"${SANDBOX}\"\ntimeout_ms=10000\n")
	writeAil(t, sandbox, "worker.ail", `module worker
import std/fs (writeFile, fileExists)
import std/clock (sleep)
func awaitArm() -> () ! {FS, Clock} {
 if fileExists("arm") then () else { sleep(5); awaitArm() }
}
export func main() -> () ! {FS, Clock} {
 writeFile("started", "worker entered main");
 awaitArm();
 sleep(1200);
 writeFile("after_limit", "execution continued past supervisor deadline")
}
`)
	cmd := exec.Command(bin, "run", "--policy", pol, "--chdir", sandbox, "worker.ail")
	proctree.SetGroup(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	started := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	finished := false
	defer func() {
		if !finished {
			_ = cmd.Process.Kill()
			<-waited
		}
	}()
	// The timer starts after policy resolution, immediately before the worker
	// is spawned. Observe that spawn rather than assuming CLI startup was free.
	spawnDeadline := started.Add(8 * time.Second)
	for len(policyFixturePIDs(t, root)) != 2 {
		if time.Now().After(spawnDeadline) {
			t.Fatal("policy worker process not admitted")
		}
		time.Sleep(5 * time.Millisecond)
	}
	timerObserved := time.Now()
	startup := timerObserved.Add(8 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(sandbox, "started")); err == nil {
			break
		}
		select {
		case err := <-waited:
			finished = true
			t.Fatalf("worker did not start: %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
		default:
		}
		if time.Now().After(startup) {
			t.Fatal("worker startup not acknowledged before limit")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if remaining := time.Until(timerObserved.Add(9000 * time.Millisecond)); remaining > 0 {
		time.Sleep(remaining)
	}
	if err := os.WriteFile(filepath.Join(sandbox, "arm"), []byte("armed"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-waited:
		finished = true
	case <-time.After(2 * time.Second):
		t.Fatal("policy execution continued in protocol grace")
	}
	if cmd.ProcessState.ExitCode() != 3 || !strings.Contains(stderr.String(), `"reason":"timeout"`) {
		t.Fatalf("policy timeout changed: %d %s", cmd.ProcessState.ExitCode(), stderr.String())
	}
	time.Sleep(500 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(sandbox, "after_limit")); err == nil {
		t.Fatal("worker executed after the hard policy boundary")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRunPolicy_OutputStopsExecutionWithoutProtocolGrace(t *testing.T) {
	bin := buildAilang(t)
	root := t.TempDir()
	sandbox := filepath.Join(root, "sandbox")
	pol := writePolicy(t, root, "security_mode=\"restricted\"\nallowed_caps=[\"IO\",\"FS\",\"Clock\"]\nentry=\"main\"\nfs_sandbox=\"${SANDBOX}\"\ntimeout_ms=30000\nmax_output_bytes=32\n")
	writeAil(t, sandbox, "worker.ail", `module worker
import std/fs (writeFile)
import std/io (println)
import std/clock (sleep)
export func main() -> () ! {FS, IO, Clock} {
 writeFile("started", "worker entered main");
 println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx");
 sleep(200);
 writeFile("after_limit", "execution continued after output cap")
}
`)
	_, stderr, code := runAilangBin(t, bin, "run", "--policy", pol, "--chdir", sandbox, "worker.ail")
	if code != 3 || !strings.Contains(stderr, `"reason":"output_limit"`) {
		t.Fatalf("policy output limit changed: %d %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(sandbox, "started")); err != nil {
		t.Fatal("actual worker startup not acknowledged")
	}
	time.Sleep(500 * time.Millisecond)
	if _, err := os.Stat(filepath.Join(sandbox, "after_limit")); err == nil {
		t.Fatal("worker executed after hard output boundary")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
