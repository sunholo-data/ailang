package effects

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
)

// fakeResolver is a test double for secrets.Resolver.
type fakeResolver struct {
	value     string
	err       error
	lastRef   string
	callCount int
}

func (f *fakeResolver) Read(_ context.Context, ref string) (string, error) {
	f.callCount++
	f.lastRef = ref
	if f.err != nil {
		return "", f.err
	}
	return f.value, nil
}

func newSecretTestCtx(r *fakeResolver) *EffContext {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("Secret"))
	ctx.Secret = &SecretContext{Resolver: r}
	return ctx
}

func TestSecretRead_Success(t *testing.T) {
	r := &fakeResolver{value: "s3cr3t"}
	ctx := newSecretTestCtx(r)
	got, err := Call(ctx, "Secret", "read", []eval.Value{&eval.StringValue{Value: "op://Prod/stripe/api-key"}})
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	sv, ok := got.(*eval.StringValue)
	if !ok {
		t.Fatalf("result type = %T, want *eval.StringValue", got)
	}
	if sv.Value != "s3cr3t" {
		t.Fatalf("result = %q, want %q", sv.Value, "s3cr3t")
	}
	if r.lastRef != "op://Prod/stripe/api-key" {
		t.Fatalf("resolver got ref %q", r.lastRef)
	}
}

func TestSecretRead_CapabilityDenied(t *testing.T) {
	r := &fakeResolver{value: "s3cr3t"}
	ctx := NewEffContext(nil) // no Secret capability granted
	ctx.Secret = &SecretContext{Resolver: r}
	_, err := Call(ctx, "Secret", "read", []eval.Value{&eval.StringValue{Value: "op://Prod/stripe/api-key"}})
	if err == nil {
		t.Fatal("expected capability error, got nil")
	}
	if r.callCount != 0 {
		t.Fatal("resolver was called despite missing Secret capability")
	}
}

func TestSecretRead_ResolverFailure(t *testing.T) {
	r := &fakeResolver{err: errors.New("op: item not found")}
	ctx := newSecretTestCtx(r)
	_, err := Call(ctx, "Secret", "read", []eval.Value{&eval.StringValue{Value: "op://Prod/stripe/api-key"}})
	if err == nil {
		t.Fatal("expected error from resolver failure")
	}
	if !strings.Contains(err.Error(), "E_SECRET_UNAVAILABLE") {
		t.Fatalf("error = %v, want E_SECRET_UNAVAILABLE", err)
	}
}

func TestSecretRead_ApproverDenies(t *testing.T) {
	r := &fakeResolver{value: "s3cr3t"}
	ctx := newSecretTestCtx(r)
	ctx.Secret.Approver = denyApprover{}
	_, err := Call(ctx, "Secret", "read", []eval.Value{&eval.StringValue{Value: "op://Prod/stripe/api-key"}})
	if err == nil {
		t.Fatal("expected denial error")
	}
	if !strings.Contains(err.Error(), "E_SECRET_DENIED") {
		t.Fatalf("error = %v, want E_SECRET_DENIED", err)
	}
	if r.callCount != 0 {
		t.Fatal("resolver was called despite approval denial — gate must run BEFORE resolution")
	}
}

func TestSecretRead_ApproverAllowsThenResolves(t *testing.T) {
	r := &fakeResolver{value: "s3cr3t"}
	ctx := newSecretTestCtx(r)
	ctx.Secret.Approver = allowApprover{}
	got, err := Call(ctx, "Secret", "read", []eval.Value{&eval.StringValue{Value: "op://Prod/stripe/api-key"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.(*eval.StringValue).Value != "s3cr3t" {
		t.Fatal("value not resolved after approval")
	}
}

func TestSecretRead_WrongArgType(t *testing.T) {
	ctx := newSecretTestCtx(&fakeResolver{value: "x"})
	_, err := Call(ctx, "Secret", "read", []eval.Value{&eval.IntValue{Value: 7}})
	if err == nil || !strings.Contains(err.Error(), "E_SECRET_TYPE_ERROR") {
		t.Fatalf("error = %v, want E_SECRET_TYPE_ERROR", err)
	}
}

type denyApprover struct{}

func (denyApprover) Approve(_ context.Context, _, _ string) error {
	return errors.New("operator denied")
}

type allowApprover struct{}

func (allowApprover) Approve(_ context.Context, _, _ string) error { return nil }
