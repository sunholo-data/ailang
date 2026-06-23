package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

func TestAttachCloudSecretApprover_CloudMode(t *testing.T) {
	t.Setenv("AILANG_STORAGE", "gcp")
	t.Setenv("AILANG_COORDINATOR_URL", "https://coord.example")
	ctx := effects.NewEffContext(nil)
	attachCloudSecretApprover(ctx)
	if ctx.Secret == nil || ctx.Secret.Approver == nil {
		t.Fatal("expected a networked approver to be attached in cloud mode")
	}
}

func TestAttachCloudSecretApprover_LocalMode_NoApprover(t *testing.T) {
	t.Setenv("AILANG_STORAGE", "")
	t.Setenv("AILANG_COORDINATOR_URL", "https://coord.example")
	ctx := effects.NewEffContext(nil)
	attachCloudSecretApprover(ctx)
	if ctx.Secret != nil && ctx.Secret.Approver != nil {
		t.Fatal("expected NO approver outside gcp storage mode (local runs stay un-gated)")
	}
}

func TestAttachCloudSecretApprover_ApprovalURLPrimary(t *testing.T) {
	t.Setenv("AILANG_STORAGE", "gcp")
	t.Setenv("AILANG_APPROVAL_URL", "https://dash.example")
	t.Setenv("AILANG_COORDINATOR_URL", "")
	ctx := effects.NewEffContext(nil)
	attachCloudSecretApprover(ctx)
	if ctx.Secret == nil || ctx.Secret.Approver == nil {
		t.Fatal("expected approver attached via AILANG_APPROVAL_URL")
	}
}

func TestAttachCloudSecretApprover_GcpButNoURL_NoApprover(t *testing.T) {
	t.Setenv("AILANG_STORAGE", "gcp")
	t.Setenv("AILANG_APPROVAL_URL", "")
	t.Setenv("AILANG_COORDINATOR_URL", "")
	ctx := effects.NewEffContext(nil)
	attachCloudSecretApprover(ctx)
	if ctx.Secret != nil && ctx.Secret.Approver != nil {
		t.Fatal("expected NO approver when no approval URL is configured")
	}
}
