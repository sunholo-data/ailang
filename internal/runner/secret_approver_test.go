package runner

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
)

// clearPlaneEnv isolates a test from the plane variables and the retired
// selectors a developer's shell may still export.
func clearPlaneEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{config.EnvStorage, config.EnvStorageMessaging, config.EnvStorageCoordinator, config.EnvStorageObservatory} {
		t.Setenv(v, "")
	}
	for _, v := range config.RemovedEnvNames() {
		t.Setenv(v, "")
	}
}

func TestAttachCloudSecretApprover_CloudMode(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_STORAGE", "gcp")
	t.Setenv("AILANG_COORDINATOR_URL", "https://coord.example")
	ctx := effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Secret == nil || ctx.Secret.Approver == nil {
		t.Fatal("expected a networked approver to be attached in cloud mode")
	}
}

// hybrid is the shared plane too: it silently lacked the approver before
// M-V1-SIMPLIFY-S3 M3.
func TestAttachCloudSecretApprover_HybridIsShared(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_STORAGE", "hybrid")
	t.Setenv("AILANG_COORDINATOR_URL", "https://coord.example")
	ctx := effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Secret == nil || ctx.Secret.Approver == nil {
		t.Fatal("expected a networked approver to be attached on the hybrid plane")
	}
}

// A malformed plane must not fail open: a retired selector is an error, not
// "not cloud".
func TestAttachCloudSecretApprover_UnresolvablePlaneIsAnError(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_MESSAGES_STORE", "gcp")
	t.Setenv("AILANG_COORDINATOR_URL", "https://coord.example")
	ctx := effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); !errors.Is(err, config.ErrRemovedEnv) {
		t.Fatalf("err = %v, want config.ErrRemovedEnv", err)
	}
}

func TestAttachCloudSecretApprover_LocalMode_NoApprover(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_COORDINATOR_URL", "https://coord.example")
	ctx := effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Secret != nil && ctx.Secret.Approver != nil {
		t.Fatal("expected NO approver outside gcp storage mode (local runs stay un-gated)")
	}
}

func TestAttachCloudSecretApprover_ApprovalURLPrimary(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_STORAGE", "gcp")
	t.Setenv("AILANG_APPROVAL_URL", "https://dash.example")
	t.Setenv("AILANG_COORDINATOR_URL", "")
	ctx := effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Secret == nil || ctx.Secret.Approver == nil {
		t.Fatal("expected approver attached via AILANG_APPROVAL_URL")
	}
}

func TestAttachCloudSecretApprover_GcpButNoURL_NoApprover(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_STORAGE", "gcp")
	t.Setenv("AILANG_APPROVAL_URL", "")
	t.Setenv("AILANG_COORDINATOR_URL", "")
	ctx := effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Secret != nil && ctx.Secret.Approver != nil {
		t.Fatal("expected NO approver when no approval URL is configured")
	}
}
