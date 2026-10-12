//go:build unix

package main

import (
	"bytes"
	"fmt"
	"github.com/sunholo-data/ailang/internal/proctree"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Losing an outer supervisor must join an inner policy supervisor through its
// private receipt, then stop the orphaned outer worker's own isolated group.
func TestRunPolicy_NestedSupervisorLifetime(t *testing.T) {
	if _, err := exec.LookPath("ps"); err != nil {
		t.Skip("fixture process inventory requires ps")
	}
	bin := buildAilang(t)
	root := t.TempDir()
	inner := filepath.Join(root, "inner")
	policySleepFixture(t, inner, 1)
	outer := filepath.Join(root, "outer")
	if err := os.MkdirAll(outer, 0755); err != nil {
		t.Fatal(err)
	}
	source := fmt.Sprintf(`module worker
import std/fs (writeFile)
import std/io (readLineOpt)
import std/process (spawnProcess)
export func main() -> () ! {FS, IO, Process} {
 let child=spawnProcess(%q,["run","--policy",%q,"--chdir",%q,"worker.ail"]);
 writeFile("started", "outer worker entered main");
 let key=readLineOpt();
 ()
}
`, filepath.ToSlash(bin), filepath.ToSlash(filepath.Join(inner, "worker0-policy.toml")), filepath.ToSlash(filepath.Join(inner, "worker0")))
	writeAil(t, outer, "worker.ail", source)
	body := fmt.Sprintf("security_mode=\"trusted_host\"\nallowed_caps=[\"FS\",\"IO\",\"Process\"]\nprocess_allow=[%q]\nentry=\"main\"\ntimeout_ms=90000\nfs_sandbox=%q\n", filepath.ToSlash(bin), filepath.ToSlash(outer))
	policyPath := filepath.Join(root, "outer-policy.toml")
	if err := os.WriteFile(policyPath, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "run", "--policy", policyPath, "--chdir", outer, "worker.ail")
	proctree.SetGroup(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	finished := false
	pids := []int{}
	defer func() {
		if !finished {
			_ = cmd.Process.Kill()
			<-waited
		}
		for _, pid := range pids {
			if pid > 1 && pid != cmd.Process.Pid && !processGone(pid) {
				killProcess(pid)
			}
		}
	}()
	startup := time.Now().Add(10 * time.Second)
	for {
		_, outerErr := os.Stat(filepath.Join(outer, "started"))
		_, innerErr := os.Stat(filepath.Join(inner, "worker0", "started"))
		if outerErr == nil && innerErr == nil {
			break
		}
		select {
		case err := <-waited:
			finished = true
			t.Fatalf("nested startup failed: %v %s %s", err, stdout.String(), stderr.String())
		default:
		}
		if time.Now().After(startup) {
			t.Fatal("nested worker did not acknowledge startup")
		}
		time.Sleep(10 * time.Millisecond)
	}
	pids = policyFixturePIDs(t, root)
	if len(pids) != 4 {
		t.Fatalf("expected two supervisors and two worker groups: %v", pids)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	<-waited
	finished = true
	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		allGone := true
		for _, pid := range pids {
			allGone = allGone && processGone(pid)
		}
		if allGone {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("nested policy process or zombie survived lifetime cleanup: %v", pids)
}
