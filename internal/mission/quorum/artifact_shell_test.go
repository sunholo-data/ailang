package quorum_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestQuorumArtifactShellSuite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runs /bin/bash on tools/launchd/test_quorum_artifacts.sh; the launchd quorum driver it covers is macOS-only")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	script, err := filepath.Abs("../../../tools/launchd/test_quorum_artifacts.sh")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/bash", script, "--writer", writerWrapper(t, home))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell suite: %v\n%s", err, out)
	}
	t.Log(string(out))
}

// Keep the integration proofs executable independently when a protected mirror
// prevents the aggregate suite from running. The aggregate remains a fatal gate.
func TestQuorumArtifactWriterArms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runs /bin/bash on tools/launchd/test_quorum_artifacts.sh; the launchd quorum driver it covers is macOS-only")
	}
	for _, arm := range []string{"cross_tree_durable", "attended_no_exports", "cross_mode_fixed_order"} {
		t.Run(arm, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			script, err := filepath.Abs("../../../tools/launchd/test_quorum_artifacts.sh")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "/bin/bash", script, "--writer", writerWrapper(t, home), "--arm", arm)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("writer arm: %v\n%s", err, out)
			}
		})
	}
}
