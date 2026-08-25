package browserbase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
	"github.com/sunholo-data/ailang/internal/browser/auth"
)

// DefaultContextSyncDelay is how long the refresh workflow waits for Browserbase
// to finish publishing an updated Context before the new profile version is
// treated as readable. Browserbase documents that a Context is persisted
// asynchronously after the session ends, so a refresh that publishes
// immediately can hand the next run the pre-refresh state.
//
// It is a configured parameter, not a literal sleep, so the wait is testable
// without spending it and tunable without a code change.
const DefaultContextSyncDelay = 5 * time.Second

const contextsPath = "/v1/contexts"

// ContextAttachment binds one hosted Context to one session under one lease.
//
// Persist is the caller's *request* to write back, kept separate from Mode on
// purpose: the pairing of "read lease" with "wants to persist" is exactly the
// mistake that must be refused, and a design that derived one from the other
// could not express it. The refusal happens before any HTTP request is issued.
type ContextAttachment struct {
	Material auth.SensitiveProfileMaterial
	Mode     auth.LeaseMode
	Persist  bool
}

// ContextStatus is the safe projection of a hosted Context. It deliberately
// carries no identifier: the caller already holds the opaque material, and a
// status value travels further than the material does.
type ContextStatus struct {
	Present   bool      `json:"present"`
	ExpiresAt time.Time `json:"expires_at,omitzero"`
}

// contextBinding is the provider-private record of an attached Context. It
// follows the SensitiveProfileMaterial redaction discipline: no exported
// fields, and every presentation redacts. The persist flag is safe to show —
// it is a policy decision, not a credential.
type contextBinding struct {
	material auth.SensitiveProfileMaterial
	persist  bool
}

func (b contextBinding) String() string {
	return fmt.Sprintf("browserbase context %s (persist=%t)", browser.Redacted, b.persist)
}

func (b contextBinding) GoString() string { return b.String() }

func (b contextBinding) Error() string { return b.String() }

func (b contextBinding) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ContextID string `json:"context_id"`
		Persist   bool   `json:"persist"`
	}{ContextID: browser.Redacted, Persist: b.persist})
}

// contextResponse is the provider's Context representation.
//
// ExpiresAt is decoded defensively: Browserbase does not currently document an
// expiry field on Contexts, and the authoritative expiry for a profile is the
// policy recorded in the registry. Honoring it when present means a provider
// that starts reporting expiry fails closed rather than silently handing back a
// dead Context.
type contextResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	ExpiresAt string `json:"expiresAt"`
}

// CreateContext provisions a new hosted Context and returns its identifier as
// opaque material. The raw ID is never returned as a plain string.
func (p *Provider) CreateContext(ctx context.Context) (auth.SensitiveProfileMaterial, error) {
	body := map[string]any{}
	if p.projectID != "" {
		body["projectId"] = p.projectID
	}
	var response contextResponse
	status, err := p.doJSONWithStatus(ctx, http.MethodPost, contextsPath, body, &response, browser.FailureProvision)
	if err != nil {
		if malformedResponse(status) {
			return auth.SensitiveProfileMaterial{},
				auth.NewFailureReason(auth.FailureMaterializeFailed, "create_context", "malformed_provider_response")
		}
		return auth.SensitiveProfileMaterial{}, err
	}
	if response.ID == "" {
		return auth.SensitiveProfileMaterial{},
			auth.NewFailureReason(auth.FailureMaterializeFailed, "create_context", "missing_context_id")
	}
	return auth.NewProviderContextMaterial(response.ID), nil
}

// GetContext reports whether a hosted Context is still usable. A Context the
// provider no longer knows about is a terminal profile failure, not a transport
// error, because retrying cannot bring it back.
func (p *Provider) GetContext(ctx context.Context, material auth.SensitiveProfileMaterial) (ContextStatus, error) {
	contextID, err := contextIDFrom(material, "get_context")
	if err != nil {
		return ContextStatus{}, err
	}
	var response contextResponse
	status, err := p.doJSONWithStatus(ctx, http.MethodGet, contextPath(contextID), nil, &response, browser.FailureProvision)
	if err != nil {
		switch {
		case status == http.StatusNotFound:
			return ContextStatus{}, auth.NewFailureReason(auth.FailureProfileNotFound, "get_context", "context_absent")
		case malformedResponse(status):
			return ContextStatus{}, auth.NewFailureReason(auth.FailureMaterializeFailed, "get_context", "malformed_provider_response")
		default:
			return ContextStatus{}, err
		}
	}
	expiresAt := parseTime(response.ExpiresAt)
	if !expiresAt.IsZero() && !p.clock().Before(expiresAt) {
		return ContextStatus{}, auth.NewFailureReason(auth.FailureProfileExpired, "get_context", "context_expired")
	}
	return ContextStatus{Present: true, ExpiresAt: expiresAt}, nil
}

// DeleteContext revokes a hosted Context. It is idempotent: a Context that is
// already gone is the state the caller asked for, and cleanup paths run on
// exit routes that may already have deleted it. Any other failure is reported
// as a cleanup failure so it cannot be mistaken for the primary error.
func (p *Provider) DeleteContext(ctx context.Context, material auth.SensitiveProfileMaterial) error {
	contextID, err := contextIDFrom(material, "delete_context")
	if err != nil {
		return err
	}
	status, err := p.doJSONWithStatus(ctx, http.MethodDelete, contextPath(contextID), nil, nil, browser.FailureCleanup)
	if err != nil {
		if status == http.StatusNotFound {
			return nil
		}
		return auth.NewFailure(auth.FailureCleanupFailed, "delete_context", err)
	}
	return nil
}

// AwaitContextSync observes the provider's documented publish delay. The
// refresh workflow calls it after a persisting session ends and before the new
// profile version is published, so a subsequent run cannot read pre-refresh
// state. Cancellation during the wait is a timeout rather than a silent
// early return: skipping the delay is exactly the failure it exists to prevent.
func (p *Provider) AwaitContextSync(ctx context.Context) error {
	if p.contextSyncDelay <= 0 {
		return nil
	}
	waiter := p.sleep
	if waiter == nil {
		waiter = defaultSleep
	}
	if err := waiter(ctx, p.contextSyncDelay); err != nil {
		return browser.NewFailure(browser.FailureSessionTimeout, "await Browserbase context sync", err)
	}
	return nil
}

// CreateWithContext provisions a session that loads a stored hosted Context.
//
// The Context ID is read out of the opaque material, placed directly into the
// request body, and retained only in the provider-private binding. It is never
// written back into the SessionSpec, so it cannot reach a manifest, a result
// record, or a log line by way of the spec.
func (p *Provider) CreateWithContext(ctx context.Context, spec browser.SessionSpec, attach ContextAttachment) (browser.Session, error) {
	persist, contextID, err := attach.resolve()
	if err != nil {
		return browser.Session{}, err
	}
	body := p.sessionBody(spec)
	browserSettings(body)["context"] = map[string]any{"id": contextID, "persist": persist}
	binding := contextBinding{material: attach.Material, persist: persist}
	return p.createSession(ctx, spec, body, &binding)
}

// resolve validates the attachment and returns the EFFECTIVE persistence flag.
// Every check here runs before the caller touches the network.
func (a ContextAttachment) resolve() (bool, string, error) {
	if !a.Mode.Valid() {
		return false, "", auth.NewFailureReason(auth.FailureScopeDenied, "attach_context", "unknown_lease_mode")
	}
	// The fail-closed rule: only an exclusive refresh lease may write back.
	if a.Persist && !a.Mode.Writes() {
		return false, "", auth.NewFailureReason(auth.FailureWritebackDenied, "attach_context", "persist_requires_refresh_lease")
	}
	contextID, err := contextIDFrom(a.Material, "attach_context")
	if err != nil {
		return false, "", err
	}
	return a.Persist && a.Mode.Writes(), contextID, nil
}

// contextIDFrom extracts the hosted Context ID from opaque material, refusing
// material of the wrong shape rather than sending an empty id to the provider.
func contextIDFrom(material auth.SensitiveProfileMaterial, op string) (string, error) {
	kind, _, contextID := material.Materialize()
	if kind != auth.MaterialProviderContext {
		return "", auth.NewFailureReason(auth.FailureMaterializeFailed, op, "wrong_material_kind")
	}
	if contextID == "" {
		return "", auth.NewFailureReason(auth.FailureMaterializeFailed, op, "empty_material")
	}
	return contextID, nil
}

func contextPath(contextID string) string {
	return contextsPath + "/" + url.PathEscape(contextID)
}

// malformedResponse reports whether an error accompanied a success status,
// which can only mean the body did not decode.
func malformedResponse(status int) bool {
	return status >= 200 && status < 300
}

// defaultSleep is the production waiter: a timer that still honors cancellation.
func defaultSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
