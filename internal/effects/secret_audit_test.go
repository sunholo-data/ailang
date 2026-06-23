package effects

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// traceDump renders all collected trace events as a single string for leak
// assertions.
func traceDump(t *testing.T, c *trace.Collector) string {
	t.Helper()
	var b strings.Builder
	for _, ev := range c.Events() {
		b.WriteString(fmt.Sprintf("%+v", ev))
		if ev.Effect != nil {
			b.WriteString(fmt.Sprintf(" effect=%+v", *ev.Effect))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestSecretRead_AuditEmittedWithoutValue(t *testing.T) {
	const secretValue = "SUPER-SECRET-VALUE"
	r := &fakeResolver{value: secretValue}
	ctx := newSecretTestCtx(r)
	ctx.Secret.Approver = allowApprover{}

	var events []SecretAuditEvent
	ctx.Secret.Audit = func(ev SecretAuditEvent) { events = append(events, ev) }

	_, err := Call(ctx, "Secret", "read", []eval.Value{&eval.StringValue{Value: "op://Prod/stripe/api-key"}, &eval.StringValue{Value: "test purpose"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect an "approved" then a "resolved" event.
	if len(events) != 2 {
		t.Fatalf("got %d audit events, want 2: %+v", len(events), events)
	}
	if events[0].Decision != "approved" || events[1].Decision != "resolved" {
		t.Fatalf("decisions = %q,%q want approved,resolved", events[0].Decision, events[1].Decision)
	}
	for _, ev := range events {
		if ev.Ref != "op://Prod/stripe/api-key" {
			t.Fatalf("audit ref = %q", ev.Ref)
		}
		if strings.Contains(ev.Ref+ev.Purpose+ev.Decision+ev.Err, secretValue) {
			t.Fatalf("audit event leaked the secret value: %+v", ev)
		}
	}
}

// TestSecretRead_TraceRedactsValue is the M3 integration acceptance: a resolved
// secret value must NOT appear in the emitted trace.
func TestSecretRead_TraceRedactsValue(t *testing.T) {
	const secretValue = "SUPER-SECRET-VALUE"
	r := &fakeResolver{value: secretValue}
	ctx := newSecretTestCtx(r)
	ctx.Trace = trace.NewCollector()

	_, err := Call(ctx, "Secret", "read", []eval.Value{&eval.StringValue{Value: "op://Prod/stripe/api-key"}, &eval.StringValue{Value: "test purpose"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dump := traceDump(t, ctx.Trace)
	if strings.Contains(dump, secretValue) {
		t.Fatalf("trace leaked the secret value:\n%s", dump)
	}
	if !strings.Contains(dump, redactedSecretMarker) {
		t.Fatalf("trace missing redaction marker %q:\n%s", redactedSecretMarker, dump)
	}
}
