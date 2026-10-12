//go:build unix

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
)

func policyFixturePIDs(t *testing.T, root string) []int {
	t.Helper()
	out, err := exec.Command("ps", "-axo", "pid,ppid,pgid,args").Output()
	if err != nil {
		t.Fatalf("policy fixture inventory: %v", err)
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, root) || !strings.Contains(line, "run --policy") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			t.Fatalf("invalid inventory row %q", line)
		}
		pid, err := strconv.Atoi(fields[0])
		group, groupErr := strconv.Atoi(fields[2])
		if err != nil || groupErr != nil || pid <= 1 || group != pid {
			t.Fatalf("fixture does not own group: %q", line)
		}
		pids = append(pids, pid)
	}
	return pids
}

func policySleepFixture(t *testing.T, root string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		dir := filepath.Join(root, fmt.Sprintf("worker%d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		writeAil(t, dir, "worker.ail", `module worker
import std/fs (writeFile)
import std/io (println)
import std/clock (sleep)
export func main() -> () ! {FS, IO, Clock} {
 writeFile("started", "actual worker entered main");
 println("worker ready");
 sleep(60000)
}
`)
		body := fmt.Sprintf("security_mode=\"restricted\"\nallowed_caps=[\"FS\",\"IO\",\"Clock\"]\nentry=\"main\"\ntimeout_ms=90000\nfs_sandbox=%q\nmax_output_bytes=8192\n", filepath.ToSlash(dir))
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("worker%d-policy.toml", i)), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunPolicy_OwnedCancellation(t *testing.T) {
	if _, err := exec.LookPath("ps"); err != nil {
		t.Skip("positive-PID process inventory requires ps")
	}
	bin := buildAilang(t)
	for _, kind := range []string{"managed", "async"} {
		for _, scenario := range []string{"cancel", "eof", "return", "error", "exit", "sigint", "sigterm"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				root := t.TempDir()
				count := 1
				if kind == "async" {
					count = 2
				}
				policySleepFixture(t, root, count)
				imports := "import std/io (println, readLineOpt, exit)\n"
				spawn := ""
				cancelCalls := ""
				for i := 0; i < count; i++ {
					args := fmt.Sprintf(`["run","--policy",%q,"--chdir",%q,"worker.ail"]`, filepath.ToSlash(filepath.Join(root, fmt.Sprintf("worker%d-policy.toml", i))), filepath.ToSlash(filepath.Join(root, fmt.Sprintf("worker%d", i))))
					if kind == "managed" {
						imports += "import std/process (spawnProcess,cancelProcess)\n"
						spawn += fmt.Sprintf("let child%d=spawnProcess(%q,%s);\n", i, filepath.ToSlash(bin), args)
						cancelCalls += fmt.Sprintf("println(show(cancelProcess(child%d)));\n", i)
					} else {
						if i == 0 {
							imports += "import std/stream (asyncExecProcess,cancelProcessSource)\n"
						}
						spawn += fmt.Sprintf("let child%d=asyncExecProcess(%q,%s,\"worker%d\",1,16);\n", i, filepath.ToSlash(bin), args, i)
						cancelCalls += fmt.Sprintf("println(show(cancelProcessSource(child%d)));\n", i)
					}
				}
				action := "()"
				if scenario == "cancel" || scenario == "eof" {
					action = cancelCalls + "()"
				}
				if scenario == "exit" {
					action = "exit(7)"
				}
				if scenario == "error" {
					action = "println(show(1 / 0))"
				}
				host := writeAil(t, root, "host.ail", "module host\n"+imports+"export func main() -> () ! {IO, Process, Stream} {\n"+spawn+"let key=readLineOpt();\n"+action+"\n}\n")
				bounded, done := context.WithTimeout(t.Context(), 20*time.Second)
				defer done()
				cmd := exec.CommandContext(bounded, bin, "run", "--caps", "IO,Process,Stream", "--process-allowlist", bin, host)
				cmd.Dir = root
				proctree.Configure(cmd)
				var stdout, stderr bytes.Buffer
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				stdin, err := cmd.StdinPipe()
				if err != nil {
					t.Fatal(err)
				}
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				waited := make(chan error, 1)
				go func() { waited <- cmd.Wait() }()
				finished := false
				pids := []int{}
				defer func() {
					_ = stdin.Close()
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
					ready := true
					for i := 0; i < count; i++ {
						_, err := os.Stat(filepath.Join(root, fmt.Sprintf("worker%d", i), "started"))
						ready = ready && err == nil
					}
					if ready {
						break
					}
					if time.Now().After(startup) {
						t.Fatal("actual workers did not acknowledge startup")
					}
					time.Sleep(10 * time.Millisecond)
				}
				pids = policyFixturePIDs(t, root)
				if len(pids) != 2*count {
					t.Fatalf("expected supervisor and worker groups: %v", pids)
				}
				started := time.Now()
				switch scenario {
				case "sigint":
					err = cmd.Process.Signal(os.Interrupt)
				case "sigterm":
					err = cmd.Process.Signal(syscall.SIGTERM)
				case "eof":
					err = stdin.Close()
				default:
					_, err = stdin.Write([]byte("quit\n"))
				}
				if err != nil {
					t.Fatal(err)
				}
				select {
				case <-waited:
					finished = true
				case <-time.After(2500 * time.Millisecond):
					t.Fatal("shared shutdown exceeded 2s + observation allowance")
				}
				expected := 0
				if scenario == "error" {
					expected = 1
				}
				if scenario == "exit" {
					expected = 7
				}
				if scenario == "sigint" {
					expected = 130
				}
				if scenario == "sigterm" {
					expected = 143
				}
				if cmd.ProcessState.ExitCode() != expected {
					t.Fatalf("exit %d want %d stdout=%s stderr=%s", cmd.ProcessState.ExitCode(), expected, stdout.String(), stderr.String())
				}
				if scenario == "cancel" || scenario == "eof" {
					if strings.Count(stdout.String(), "Ok(())") != count {
						t.Fatalf("missing acknowledged cancellation: %s %s", stdout.String(), stderr.String())
					}
				}
				for _, pid := range pids {
					if !processGone(pid) {
						t.Fatalf("acknowledged process or zombie %d survived successful host cleanup after %s", pid, time.Since(started))
					}
				}
			})
		}
	}
}
