package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SecretRef names a credential in the trusted vault. It is a POINTER, never a
// value: a ref is safe to store in config, log, and audit, and resolving it is a
// separate, audited control-plane action.
type SecretRef struct {
	Provider string `json:"provider"`
	Ref      string `json:"ref"`
}

func (r SecretRef) String() string { return r.Provider + ":" + r.Ref }

// SecretValue is a resolved credential. Like SensitiveProfileMaterial it has no
// exported fields and redacts under every presentation, so it cannot be
// stringified into a prompt, a tool argument, an error, or a log line by
// accident. Reveal is the single deliberate extraction point, and only
// PageDriver.Fill is expected to call it.
type SecretValue struct {
	value string
}

func NewSecretValue(value string) SecretValue { return SecretValue{value: value} }

func (v SecretValue) String() string   { return Redacted }
func (v SecretValue) GoString() string { return Redacted }
func (v SecretValue) Error() string    { return Redacted }
func (v SecretValue) Empty() bool      { return v.value == "" }

func (v SecretValue) MarshalJSON() ([]byte, error) { return json.Marshal(Redacted) }

// Reveal returns the raw credential. Every caller is trusted control-plane code
// running outside the model/MCP loop.
func (v SecretValue) Reveal() string { return v.value }

// SecretResolver resolves refs to values. The AILANG 1Password/Secret Manager
// resolver satisfies this; the model never holds one.
type SecretResolver interface {
	Resolve(ctx context.Context, ref SecretRef) (SecretValue, error)
}

// PageDriver is the minimal browser surface a login adapter needs.
//
// It is deliberately NOT the Playwright MCP tool surface. MCP tool inputs are
// part of the AI/tool transcript, so a password passed as an MCP `browser_type`
// argument would be captured there. Fill takes a SecretValue and hands the raw
// string straight to the driver, so the credential never becomes a tool
// argument that anything transcribes.
type PageDriver interface {
	Navigate(ctx context.Context, url string) error
	Fill(ctx context.Context, selector string, value SecretValue) error
	Click(ctx context.Context, selector string) error
	WaitFor(ctx context.Context, selector string) error
	CurrentURL(ctx context.Context) (string, error)
	// StorageState captures the authenticated state. The bytes are
	// credential-grade and must be sealed before they touch durable storage.
	StorageState(ctx context.Context) ([]byte, error)
}

// PostLoginAssertion is the deterministic evidence that a login actually
// succeeded. Without it a refresh would happily publish a logged-out state, and
// every downstream run would fail in a way that looks like a model failure.
type PostLoginAssertion struct {
	// URLPrefix, when set, must prefix the post-login URL.
	URLPrefix string `json:"url_prefix,omitempty"`
	// Selector, when set, must be present after login.
	Selector string `json:"selector,omitempty"`
	// RequireCookies are cookie names that must appear in the captured state.
	RequireCookies []string `json:"require_cookies,omitempty"`
}

// Satisfied reports whether the assertion says anything at all. An assertion
// that asserts nothing is treated as absent, not as satisfied.
func (a PostLoginAssertion) Satisfied() bool {
	return a.URLPrefix != "" || a.Selector != "" || len(a.RequireCookies) > 0
}

// LoginAdapter is site-specific, TRUSTED control-plane code.
//
// There is deliberately no generic, model-authored implementation: a general
// "fill the login form" step driven by a model is exactly the capability this
// design refuses to build. Each site gets a reviewed adapter or it gets manual
// bootstrap.
type LoginAdapter interface {
	Site() string
	Credentials() []SecretRef
	Login(ctx context.Context, page PageDriver, secrets SecretResolver) error
	Assertion() PostLoginAssertion
}

// RefreshRequest publishes a new profile version from a live login.
type RefreshRequest struct {
	Ref        AuthProfileRef
	NewVersion string
	Reason     string
	Run        RunIdentity
	Adapter    LoginAdapter
	Page       PageDriver
	Secrets    SecretResolver

	// RetireOld retires the previous version once the new one is published, so
	// `latest` moves forward while pinned rollback references keep working.
	RetireOld bool
}

// RefreshResult is the safe record of a published refresh.
type RefreshResult struct {
	Published SafeProfile `json:"published"`
	Retired   string      `json:"retired,omitempty"`
}

// Refresh publishes a NEW immutable profile version from a fresh login.
//
// The order is the safety property:
//
//  1. resolve the current version and take an EXCLUSIVE refresh lease
//  2. drive the site-specific login outside the model/MCP loop
//  3. assert the post-login state actually looks logged in
//  4. capture, seal, and publish a NEW version — never overwrite
//  5. retire the old version and record a rollback pointer
//
// A failure at any step publishes nothing. That is the whole point: a refresh
// that half-succeeds would replace a working identity with a broken one.
func (b *Broker) Refresh(ctx context.Context, request RefreshRequest) (*RefreshResult, error) {
	if request.Adapter == nil {
		return nil, errors.New("refresh requires a site login adapter")
	}
	if request.Page == nil {
		return nil, errors.New("refresh requires a page driver")
	}
	if err := validateVersion(request.NewVersion); err != nil {
		return nil, err
	}

	current, err := b.Resolve(ctx, request.Ref)
	if err != nil {
		return nil, err
	}

	assertion := request.Adapter.Assertion()
	if !assertion.Satisfied() {
		return nil, NewFailureReason(FailureRefreshRequired, "refresh", "adapter_declares_no_post_login_assertion")
	}

	lease, err := b.Acquire(ctx, current, request.Run, LeaseRefresh)
	if err != nil {
		return nil, err
	}
	defer func() { _ = b.leases.Release(ctx, lease) }()

	if err := request.Adapter.Login(ctx, request.Page, request.Secrets); err != nil {
		failure := NewFailure(FailureRefreshRequired, "site login", err)
		b.record(ctx, NewAuditEvent("refresh", current, lease, request.Run, b.now(), failure))
		return nil, failure
	}

	state, err := request.Page.StorageState(ctx)
	if err != nil {
		failure := NewFailure(FailureRefreshRequired, "capture storage state", err)
		b.record(ctx, NewAuditEvent("refresh", current, lease, request.Run, b.now(), failure))
		return nil, failure
	}

	if err := verifyPostLogin(ctx, request.Page, assertion, state); err != nil {
		b.record(ctx, NewAuditEvent("refresh", current, lease, request.Run, b.now(), err))
		return nil, err
	}

	sealed, err := Seal(ctx, b.protector, state)
	if err != nil {
		failure := NewFailure(FailureMaterializeFailed, "seal refreshed state", err)
		b.record(ctx, NewAuditEvent("refresh", current, lease, request.Run, b.now(), failure))
		return nil, failure
	}

	published, err := b.registry.Publish(ctx, Record{
		Alias:    current.Alias,
		Version:  request.NewVersion,
		Provider: current.Provider,
		Policy:   current.Policy,
		Material: NewStorageStateMaterial(sealed.Bytes()),
	})
	if err != nil {
		b.record(ctx, NewAuditEvent("refresh", current, lease, request.Run, b.now(), err))
		return nil, err
	}

	result := &RefreshResult{Published: published}
	if request.RetireOld {
		if err := b.registry.Retire(ctx, current.Ref()); err != nil {
			// The new version is already published and usable; failing the whole
			// refresh here would be worse than reporting a stale predecessor.
			b.record(ctx, NewAuditEvent("retire", current, lease, request.Run, b.now(), err))
		} else {
			result.Retired = current.Version
		}
	}

	event := NewAuditEvent("refresh", published, lease, request.Run, b.now(), nil)
	event.Decision = DecisionPublished
	event.Reason = request.Reason
	b.record(ctx, event)
	return result, nil
}

// verifyPostLogin turns "the adapter said it logged in" into evidence.
func verifyPostLogin(ctx context.Context, page PageDriver, assertion PostLoginAssertion, state []byte) error {
	if assertion.URLPrefix != "" {
		current, err := page.CurrentURL(ctx)
		if err != nil {
			return NewFailure(FailureRefreshRequired, "read post-login url", err)
		}
		if len(current) < len(assertion.URLPrefix) || current[:len(assertion.URLPrefix)] != assertion.URLPrefix {
			return NewFailureReason(FailureRefreshRequired, "verify post-login", "url_prefix_mismatch")
		}
	}
	if assertion.Selector != "" {
		if err := page.WaitFor(ctx, assertion.Selector); err != nil {
			return NewFailureReason(FailureRefreshRequired, "verify post-login", "selector_absent")
		}
	}
	for _, name := range assertion.RequireCookies {
		present, err := storageStateHasCookie(state, name)
		if err != nil {
			return NewFailureReason(FailureRefreshRequired, "verify post-login", "unreadable_storage_state")
		}
		if !present {
			return NewFailureReason(FailureRefreshRequired, "verify post-login", "required_cookie_absent: "+name)
		}
	}
	return nil
}

// storageStateHasCookie inspects the captured state WITHOUT returning any of it.
// The cookie's presence is the only thing that leaves this function.
func storageStateHasCookie(state []byte, name string) (bool, error) {
	var parsed struct {
		Cookies []struct {
			Name string `json:"name"`
		} `json:"cookies"`
	}
	if err := json.Unmarshal(state, &parsed); err != nil {
		return false, fmt.Errorf("storage state is not valid JSON")
	}
	for _, cookie := range parsed.Cookies {
		if cookie.Name == name {
			return true, nil
		}
	}
	return false, nil
}

// BootstrapRequest publishes the FIRST version of a profile from a trusted,
// headful, operator-driven login. The password may never enter AILANG at all:
// the operator types it into the browser themselves.
type BootstrapRequest struct {
	Alias      string
	Version    string
	Provider   string
	Policy     AuthProfilePolicy
	State      []byte
	Run        RunIdentity
	CapturedAt time.Time
}

// Bootstrap seals and publishes an operator-captured authenticated state.
//
// This is explicitly NOT an eval: the session that produced State ran headful
// with recording disabled and is marked non-comparable by the caller.
func (b *Broker) Bootstrap(ctx context.Context, request BootstrapRequest) (SafeProfile, error) {
	if len(request.State) == 0 {
		return SafeProfile{}, NewFailureReason(FailureMaterializeFailed, "bootstrap", "empty_storage_state")
	}
	sealed, err := Seal(ctx, b.protector, request.State)
	if err != nil {
		return SafeProfile{}, NewFailure(FailureMaterializeFailed, "seal bootstrap state", err)
	}
	published, err := b.registry.Publish(ctx, Record{
		Alias:    request.Alias,
		Version:  request.Version,
		Provider: request.Provider,
		Policy:   request.Policy,
		Material: NewStorageStateMaterial(sealed.Bytes()),
	})
	if err != nil {
		return SafeProfile{}, err
	}
	event := NewAuditEvent("bootstrap", published, ProfileLease{}, request.Run, b.now(), nil)
	event.Decision = DecisionPublished
	b.record(ctx, event)
	return published, nil
}
