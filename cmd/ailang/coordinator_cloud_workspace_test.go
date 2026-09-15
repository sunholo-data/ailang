package main

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The execute-job's completion workspace is the partition its result is
// published under. Unset, it is the deprecated "default" (D3, M-V1-SIMPLIFY-S4
// M1): served with one stderr warning, refused under AILANG_STRICT_CONFIG=1.
func TestExecuteJobWorkspace_DeprecatedDefaultThenStrict(t *testing.T) {
	noCloudIdentity(t)
	t.Setenv(coordinator.EnvWorkspace, "")

	ws, err := executeJobWorkspace()
	if err != nil || ws != coordinator.DeprecatedWorkspaceDefault {
		t.Fatalf("unset: (%q, %v), want %q served", ws, err, coordinator.DeprecatedWorkspaceDefault)
	}

	t.Setenv("AILANG_STRICT_CONFIG", "1")
	if ws, err = executeJobWorkspace(); ws != "" || !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("strict: (%q, %v), want config.ErrDeprecatedDefault", ws, err)
	}

	t.Setenv(coordinator.EnvWorkspace, "sunholo-data/ailang")
	if ws, err = executeJobWorkspace(); err != nil || ws != "sunholo-data/ailang" {
		t.Fatalf("strict with %s set: (%q, %v)", coordinator.EnvWorkspace, ws, err)
	}
}

// Under strict with no workspace the job must refuse BEFORE any Pub/Sub or
// executor work: there is no partition to publish a failure to.
func TestCoordinatorExecuteJob_StrictNoWorkspaceRefusesFirst(t *testing.T) {
	noCloudIdentity(t)
	t.Setenv(coordinator.EnvWorkspace, "")
	t.Setenv("AILANG_STRICT_CONFIG", "1")
	err := coordinatorExecuteJob(nil)
	if !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("err = %v, want config.ErrDeprecatedDefault", err)
	}
}
