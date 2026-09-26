package runner

import (
	"context"
	"errors"
	"strings"
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

// A malformed plane must not fail open: a retired selector denies every
// secret() with the resolution error. But it must not fail CLOSED for
// programs that never call secret(): attach succeeds, the denial is deferred
// into the approver (a shell still exporting the old selector broke
// `ailang run hello.ail` before this — measured 2026-09-15).
func TestAttachCloudSecretApprover_UnresolvablePlaneDeniesSecretsOnly(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_MESSAGES_STORE", "gcp")
	t.Setenv("AILANG_COORDINATOR_URL", "https://coord.example")
	ctx := effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatalf("attach must not fail a run that may never call secret(): %v", err)
	}
	if ctx.Secret == nil || ctx.Secret.Approver == nil {
		t.Fatal("expected a deferred-denial approver to be installed")
	}
	err := ctx.Secret.Approver.Approve(context.Background(), "op://v/i/f", "test")
	if !errors.Is(err, config.ErrRemovedEnv) {
		t.Fatalf("Approve err = %v, want config.ErrRemovedEnv", err)
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

// The shared plane with no approval endpoint is un-gated — the deprecated
// default (M-V1-SIMPLIFY-S4 M1, closing the M-SECRET-REMOTE-APPROVAL-WIRING
// M2 note): still served with AILANG_STRICT_CONFIG unset, but under strict a
// DENYING approver is installed, deferred so that only a program that calls
// secret() sees the refusal.
func TestAttachCloudSecretApprover_GcpButNoURL_DeprecatedUngatedThenStrictDenies(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv("AILANG_STORAGE", "gcp")
	t.Setenv("AILANG_APPROVAL_URL", "")
	t.Setenv("AILANG_COORDINATOR_URL", "")
	t.Setenv(config.EnvStrict, "")
	ctx := effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Secret != nil && ctx.Secret.Approver != nil {
		t.Fatal("unset: expected NO approver (the deprecated un-gated default is still served)")
	}

	t.Setenv(config.EnvStrict, "1")
	ctx = effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatalf("strict: attach must not fail a run that may never call secret(): %v", err)
	}
	if ctx.Secret == nil || ctx.Secret.Approver == nil {
		t.Fatal("strict: expected a denying approver to be installed")
	}
	err := ctx.Secret.Approver.Approve(context.Background(), "op://v/i/f", "test")
	if !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("strict Approve err = %v, want config.ErrDeprecatedDefault", err)
	}
	if !strings.Contains(err.Error(), EnvApprovalURL) {
		t.Fatalf("the denial must name %s; got %v", EnvApprovalURL, err)
	}

	// With the endpoint set, strict installs the real networked gate.
	t.Setenv(EnvApprovalURL, "https://dash.example")
	ctx = effects.NewEffContext(nil)
	if err := attachCloudSecretApprover(ctx); err != nil {
		t.Fatal(err)
	}
	if _, denying := ctx.Secret.Approver.(deferredPlaneError); denying || ctx.Secret.Approver == nil {
		t.Fatalf("strict with %s set: approver = %T, want the networked approver", EnvApprovalURL, ctx.Secret.Approver)
	}
}
