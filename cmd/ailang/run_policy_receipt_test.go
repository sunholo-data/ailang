package main

import (
	"encoding/json"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/runlife"
)

// Test-only same-executable worker for an admitted process that exits zero.
// stdout cannot supply the missing or failed private completion proof.
func policyReceiptFixtureWorker() bool {
	mode := os.Getenv("AILANG_TEST_POLICY_RECEIPT")
	if mode == "" || len(os.Args) < 4 || os.Args[1] != "run" || os.Args[2] != "--policy-worker" {
		return false
	}
	parts := strings.Split(os.Getenv(runlife.MetadataEnv), ",")
	if len(parts) != 3 {
		os.Exit(9)
	}
	fd, err := strconv.Atoi(parts[1])
	if err != nil || fd < 4 {
		os.Exit(9)
	}
	receipt := os.NewFile(uintptr(fd), "private-test-receipt")
	encoder := json.NewEncoder(receipt)
	_ = encoder.Encode(map[string]any{"version": 1, "kind": "ready"})
	if mode != "missing" {
		frame := map[string]any{"version": 1, "kind": "receipt"}
		if mode == "failed" {
			frame["error"] = "fixture cleanup failed"
		}
		_ = encoder.Encode(frame)
	}
	_ = receipt.Close()
	control := os.NewFile(3, "private-test-control")
	_ = json.NewEncoder(control).Encode(controlMessage{Kind: "admitted", Admission: &admissionLine{OK: true}})
	_ = control.Close()
	os.Exit(0)
	return true
}

func TestRunPolicy_RequiresCleanupReceiptBeforeSuccess(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("supported runtime ownership platform")
	}
	dir := t.TempDir()
	policy := writePolicy(t, dir, "security_mode=\"trusted_host\"\nallowed_caps=[]\nentry=\"main\"\ntimeout_ms=10000\n")
	for _, mode := range []string{"missing", "failed", "good"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("AILANG_TEST_POLICY_RECEIPT", mode)
			code := supervisePolicyRun(policy, "", runPolicyWidening{set: map[string]bool{}}, nil)
			want := 1
			if mode == "good" {
				want = 0
			}
			if code != want {
				t.Fatalf("ordinary supervisor exit %d, expected %d for %s cleanup receipt", code, want, mode)
			}
		})
	}
}
