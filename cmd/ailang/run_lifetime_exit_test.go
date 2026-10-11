//go:build unix

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/runlife"
)

// A runtime that acknowledged the private channel must publish cleanup even
// when flag parsing or policy admission exits before a runner owner exists.
func TestRunLifetime_EarlyExitReceipt(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	policy := writePolicy(t, dir, "security_mode=\"trusted_host\"\nallowed_caps=[]\nentry=\"main\"\ntimeout_ms=10000\nfs_sandbox=\"${SANDBOX}\"\n")
	sandbox := filepath.Join(dir, "sandbox")
	writeAil(t, sandbox, "prog.ail", "module prog\nimport std/io (println)\nexport func main() -> () ! {IO} = println(\"denied\")\n")
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"help", []string{"--help"}, 0},
		{"parse", []string{"--unknown-run-flag"}, 2},
		{"chdir", []string{"--chdir", filepath.Join(dir, "missing"), "prog.ail"}, 1},
		{"missing-file", nil, 1},
		{"policy-refusal", []string{"--policy", policy, "--caps", "IO", "prog.ail"}, 1},
		{"policy-denial", []string{"--policy", policy, "prog.ail"}, 2},
		{"startup-error", []string{"missing.ail"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, ownerRequest, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			ownerReceipt, receipt, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer request.Close()
			defer ownerRequest.Close()
			defer ownerReceipt.Close()
			defer receipt.Close()
			cmd := exec.Command(bin, append([]string{"run"}, tc.args...)...)
			cmd.Dir = sandbox
			cmd.Env = append(runlife.StripEnvironment(os.Environ()), runlife.MetadataEnv+"=3,4")
			cmd.ExtraFiles = []*os.File{request, receipt}
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			request.Close()
			receipt.Close()
			frames := make(chan []map[string]any, 1)
			go func() {
				decoder := json.NewDecoder(ownerReceipt)
				var got []map[string]any
				for {
					var frame map[string]any
					if decoder.Decode(&frame) != nil {
						break
					}
					got = append(got, frame)
				}
				frames <- got
			}()
			select {
			case got := <-frames:
				_ = cmd.Wait()
				if cmd.ProcessState.ExitCode() != tc.code {
					t.Fatalf("exit %d, expected %d: %s", cmd.ProcessState.ExitCode(), tc.code, output.String())
				}
				if len(got) != 2 || got[0]["kind"] != "ready" || got[1]["kind"] != "receipt" || got[1]["error"] != nil {
					t.Fatalf("missing successful private cleanup receipt: %v; output: %s", got, output.String())
				}
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				t.Fatal("early CLI exit did not close private receipt channel")
			}
		})
	}
}
