package eval_harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMaintenanceToolchainVersionHelper(t *testing.T) {
	if os.Getenv("MAINTENANCE_TOOLCHAIN_TEST_CHILD") == "1" {
		fmt.Print("observed-runtime-control/1\n")
		os.Exit(0)
	}
}

func TestMaintenanceToolchainRejectsFrozenRuntimeDrift(t *testing.T) {
	t.Setenv("MAINTENANCE_TOOLCHAIN_TEST_CHILD", "1")
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		t.Fatal(err)
	}
	binaryHash, err := maintenanceFileSHA(bin)
	if err != nil {
		t.Fatal(err)
	}
	command := []string{bin}
	evidence := MaintenanceToolchainEvidence{Schema: "maintenance.toolchain/1", Command: command, VersionCommand: []string{bin, "-test.run=^TestMaintenanceToolchainVersionHelper$"}, Version: "observed-runtime-control/1", BinarySHA256: binaryHash}
	data, _ := json.Marshal(evidence)
	h := sha256.Sum256(data)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "tool.json")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	tc := MaintenanceToolchain{Language: "fixture", Command: command, Version: evidence.Version, Evidence: MaintenanceFileRef{Path: path, SHA256: hex.EncodeToString(h[:])}}
	e := &MaintenanceExecution{Manifest: MaintenanceExecutionManifest{Toolchains: []MaintenanceToolchain{tc}, Limits: MaintenanceLimits{MaxStageSeconds: 5}}}
	tr := MaintenanceTrajectoryPlan{Language: "fixture", NativeToolDigest: tc.Evidence.SHA256}
	got, err := verifyMaintenanceToolchain(context.Background(), e, tr)
	if err != nil {
		t.Fatal(err)
	}
	if got != binaryHash {
		t.Fatal("lost observed binary identity")
	}
	e.Manifest.Toolchains[0].Version = "different"
	if _, err = verifyMaintenanceToolchain(context.Background(), e, tr); err == nil {
		t.Fatal("declared version drift accepted")
	}
	e.Manifest.Toolchains[0] = tc
	evidence.BinarySHA256 = hex.EncodeToString(make([]byte, 32))
	data, _ = json.Marshal(evidence)
	h = sha256.Sum256(data)
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	e.Manifest.Toolchains[0].Evidence.SHA256 = hex.EncodeToString(h[:])
	tr.NativeToolDigest = e.Manifest.Toolchains[0].Evidence.SHA256
	if _, err = verifyMaintenanceToolchain(context.Background(), e, tr); err == nil {
		t.Fatal("actual binary differs from frozen evidence but accepted")
	}
}
