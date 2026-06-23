package effects

import (
	"context"
	"fmt"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/secrets"
)

// redactedSecretMarker replaces a resolved secret value anywhere it would
// otherwise be rendered (traces, audit events). See effects.Call.
const redactedSecretMarker = "«secret»"

// init registers Secret effect operations.
func init() {
	RegisterOp("Secret", "read", secretRead)
}

// SecretAuditEvent records one secret-access decision for the audit trail. It
// deliberately carries the reference and outcome but NEVER the resolved value.
type SecretAuditEvent struct {
	Ref      string // the op:// reference requested (safe to log)
	Purpose  string // human-readable intent shown at the approval surface
	Decision string // "approved", "denied", "resolved", or "unavailable"
	Err      string // redacted error detail when Decision is denied/unavailable
}

// SecretContext holds the state for the Secret effect: a backend resolver that
// turns an op:// reference into a plaintext value. In trust model A (v0.26.0)
// the resolver shells out to the 1Password CLI with a service-account token.
//
// M3 will extend this context with an approval gate (a callback to the
// coordinator's ApprovalCheckpoint) injected by the runtime to avoid an
// effects→coordinator import cycle.
type SecretContext struct {
	// Resolver turns a reference into a value. Required; secretRead errors if nil.
	Resolver secrets.Resolver

	// Approver, if set, is consulted BEFORE the resolver runs. It returns nil to
	// allow resolution or an error to deny it. The runtime injects this (M3) so
	// the effects package need not import the coordinator. nil = no gate.
	Approver SecretApprover

	// Audit, if set, receives one event per access decision. The value is never
	// included. nil = no audit sink.
	Audit func(SecretAuditEvent)
}

// emitAudit sends an event to the audit sink if one is configured.
func (sc *SecretContext) emitAudit(ev SecretAuditEvent) {
	if sc != nil && sc.Audit != nil {
		sc.Audit(ev)
	}
}

// SecretApprover gates secret resolution behind a (possibly remote, possibly
// human) approval decision. Implementations block until a decision is made.
// Returning a non-nil error denies resolution; the value is never read.
type SecretApprover interface {
	// Approve is called with the reference and a human-readable purpose. It
	// returns nil to permit resolution, or an error describing the denial.
	Approve(ctx context.Context, ref, purpose string) error
}

// NewSecretContext returns a Secret context backed by the real 1Password CLI.
func NewSecretContext() *SecretContext {
	return &SecretContext{Resolver: secrets.NewOnePasswordResolver()}
}

// secretRead implements Secret.read(ref: string) -> string.
//
// It validates the reference, runs the (optional) approval gate, then resolves
// the value. The resolved value is returned as a StringValue; M5 will label it
// <secret> so the type system forbids it from reaching {not secret} sinks, and
// M3 redacts it from traces.
func secretRead(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("E_SECRET_TYPE_ERROR: read: expected 2 arguments (ref, purpose), got %d", len(args))
	}
	ref, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_SECRET_TYPE_ERROR: read: expected String reference, got %T", args[0])
	}
	purposeArg, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_SECRET_TYPE_ERROR: read: expected String purpose, got %T", args[1])
	}
	if ctx.Secret == nil || ctx.Secret.Resolver == nil {
		return nil, fmt.Errorf("E_SECRET_NO_RESOLVER: no secret resolver configured for the Secret effect")
	}

	goctx := ctx.GoCtx
	if goctx == nil {
		goctx = context.Background()
	}

	// Approval gate (M3): block on a human decision before any value is read.
	// The purpose is the human-readable reason shown in the approval request.
	purpose := purposeArg.Value
	if purpose == "" {
		purpose = ref.Value // empty purpose falls back to the ref
	}
	if ctx.Secret.Approver != nil {
		if err := ctx.Secret.Approver.Approve(goctx, ref.Value, purpose); err != nil {
			ctx.Secret.emitAudit(SecretAuditEvent{Ref: ref.Value, Purpose: purpose, Decision: "denied", Err: err.Error()})
			return nil, fmt.Errorf("E_SECRET_DENIED: %w", err)
		}
		ctx.Secret.emitAudit(SecretAuditEvent{Ref: ref.Value, Purpose: purpose, Decision: "approved"})
	}

	val, err := ctx.Secret.Resolver.Read(goctx, ref.Value)
	if err != nil {
		// err carries the ref (safe) but never the value.
		ctx.Secret.emitAudit(SecretAuditEvent{Ref: ref.Value, Purpose: purpose, Decision: "unavailable", Err: err.Error()})
		return nil, fmt.Errorf("E_SECRET_UNAVAILABLE: %w", err)
	}
	ctx.Secret.emitAudit(SecretAuditEvent{Ref: ref.Value, Purpose: purpose, Decision: "resolved"})
	return &eval.StringValue{Value: val}, nil
}
