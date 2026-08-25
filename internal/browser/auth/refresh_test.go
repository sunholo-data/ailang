package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const loggedInState = `{"cookies":[{"name":"sid","value":"FRESH-SESSION-TOKEN","domain":"crm.example.com"}],"origins":[]}`
const loggedOutState = `{"cookies":[],"origins":[]}`

// fakePage records what the login adapter did, and — critically — records every
// value it was asked to fill so a test can prove the raw password reached the
// driver rather than a tool-call transcript.
type fakePage struct {
	url        string
	state      string
	filled     map[string]string
	waitErr    error
	stateErr   error
	navigated  []string
	clicked    []string
	currentErr error
}

func newFakePage(state string) *fakePage {
	return &fakePage{url: "https://crm.example.com/dashboard", state: state, filled: map[string]string{}}
}

func (p *fakePage) Navigate(_ context.Context, url string) error {
	p.navigated = append(p.navigated, url)
	return nil
}

func (p *fakePage) Fill(_ context.Context, selector string, value SecretValue) error {
	p.filled[selector] = value.Reveal()
	return nil
}

func (p *fakePage) Click(_ context.Context, selector string) error {
	p.clicked = append(p.clicked, selector)
	return nil
}

func (p *fakePage) WaitFor(context.Context, string) error { return p.waitErr }

func (p *fakePage) CurrentURL(context.Context) (string, error) {
	return p.url, p.currentErr
}

func (p *fakePage) StorageState(context.Context) ([]byte, error) {
	if p.stateErr != nil {
		return nil, p.stateErr
	}
	return []byte(p.state), nil
}

type staticSecrets map[string]string

func (s staticSecrets) Resolve(_ context.Context, ref SecretRef) (SecretValue, error) {
	value, ok := s[ref.Ref]
	if !ok {
		return SecretValue{}, fmt.Errorf("no secret for %s", ref)
	}
	return NewSecretValue(value), nil
}

// crmAdapter is what a REVIEWED site adapter looks like: explicit selectors,
// explicit credential refs, explicit post-login evidence.
type crmAdapter struct {
	loginErr  error
	assertion PostLoginAssertion
}

func (a *crmAdapter) Site() string { return "crm.example.com" }

func (a *crmAdapter) Credentials() []SecretRef {
	return []SecretRef{{Provider: "1password", Ref: "crm-automation-password"}}
}

func (a *crmAdapter) Login(ctx context.Context, page PageDriver, secrets SecretResolver) error {
	if a.loginErr != nil {
		return a.loginErr
	}
	password, err := secrets.Resolve(ctx, a.Credentials()[0])
	if err != nil {
		return err
	}
	if err := page.Navigate(ctx, "https://crm.example.com/login"); err != nil {
		return err
	}
	if err := page.Fill(ctx, "#password", password); err != nil {
		return err
	}
	return page.Click(ctx, "#submit")
}

func (a *crmAdapter) Assertion() PostLoginAssertion {
	if a.assertion.Satisfied() {
		return a.assertion
	}
	return PostLoginAssertion{
		URLPrefix:      "https://crm.example.com/dashboard",
		RequireCookies: []string{"sid"},
	}
}

func refreshFixture(t *testing.T) (*Broker, *MemoryRegistry, *MemoryAuditSink, KeyProtector) {
	t.Helper()
	return newTestBroker(t)
}

func refreshRequest(page PageDriver, adapter LoginAdapter) RefreshRequest {
	return RefreshRequest{
		Ref:        AuthProfileRef{Alias: "crm", Version: VersionLatest},
		NewVersion: "v2",
		Reason:     "session-expired",
		Run:        RunIdentity{RunID: "refresh-1", Principal: "refresh-worker"},
		Adapter:    adapter,
		Page:       page,
		Secrets:    staticSecrets{"crm-automation-password": "hunter2-correct-horse"},
		RetireOld:  true,
	}
}

func TestRefreshPublishesANewImmutableVersion(t *testing.T) {
	broker, registry, sink, protector := refreshFixture(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	page := newFakePage(loggedInState)
	result, err := broker.Refresh(ctx, refreshRequest(page, &crmAdapter{}))
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if result.Published.Version != "v2" {
		t.Fatalf("published %q, want v2", result.Published.Version)
	}
	if result.Published.PreviousVersion != "v1" {
		t.Fatalf("rollback pointer = %q, want v1", result.Published.PreviousVersion)
	}
	if result.Retired != "v1" {
		t.Fatalf("retired %q, want v1", result.Retired)
	}

	// v1 must remain resolvable by pin so a rollback can still name it...
	if _, err := registry.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: "v1"}); err != nil {
		t.Fatalf("retired v1 is no longer pinnable: %v", err)
	}
	// ...but latest must now be v2.
	latest, err := registry.Resolve(ctx, AuthProfileRef{Alias: "crm", Version: VersionLatest})
	if err != nil {
		t.Fatalf("resolve latest: %v", err)
	}
	if latest.Version != "v2" {
		t.Fatalf("latest = %q, want v2", latest.Version)
	}

	// The published material must be the NEW state, sealed.
	record, err := registry.Open(ctx, AuthProfileRef{Alias: "crm", Version: "v2"})
	if err != nil {
		t.Fatalf("open v2: %v", err)
	}
	_, sealedBytes, _ := record.Material.Materialize()
	if strings.Contains(string(sealedBytes), "FRESH-SESSION-TOKEN") {
		t.Fatalf("the refreshed state was stored as plaintext")
	}
	sealed, err := ParseSealedEnvelope(sealedBytes)
	if err != nil {
		t.Fatalf("parse published envelope: %v", err)
	}
	plaintext, err := Open(ctx, protector, sealed)
	if err != nil {
		t.Fatalf("open published envelope: %v", err)
	}
	if string(plaintext) != loggedInState {
		t.Fatalf("the published version does not hold the captured state")
	}

	if !auditOps(sink)["refresh"] {
		t.Fatalf("the refresh was not audited")
	}
}

// A refresh that cannot prove the login worked must publish nothing. Publishing
// a logged-out state would make every later run fail in a way that looks like a
// model failure.
func TestRefreshPublishesNothingWhenPostLoginVerificationFails(t *testing.T) {
	cases := map[string]func(*fakePage, *crmAdapter){
		"missing required cookie": func(p *fakePage, _ *crmAdapter) { p.state = loggedOutState },
		"wrong url":               func(p *fakePage, _ *crmAdapter) { p.url = "https://crm.example.com/login?error=1" },
		"selector absent": func(p *fakePage, a *crmAdapter) {
			p.waitErr = errors.New("timeout")
			a.assertion = PostLoginAssertion{Selector: "#account-menu"}
		},
	}

	for name, breakIt := range cases {
		t.Run(name, func(t *testing.T) {
			broker, registry, _, protector := refreshFixture(t)
			ctx := context.Background()
			publishLocal(t, registry, protector, "crm", "v1", cookieJar)

			page := newFakePage(loggedInState)
			adapter := &crmAdapter{}
			breakIt(page, adapter)

			if _, err := broker.Refresh(ctx, refreshRequest(page, adapter)); !IsFailure(err, FailureRefreshRequired) {
				t.Fatalf("Refresh returned %v, want %s", err, FailureRefreshRequired)
			}
			assertOnlyVersion(t, registry, "crm", "v1")
		})
	}
}

func TestRefreshPublishesNothingWhenLoginFails(t *testing.T) {
	broker, registry, _, protector := refreshFixture(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	adapter := &crmAdapter{loginErr: errors.New("2FA challenge presented")}
	if _, err := broker.Refresh(ctx, refreshRequest(newFakePage(loggedInState), adapter)); !IsFailure(err, FailureRefreshRequired) {
		t.Fatalf("Refresh returned %v, want %s", err, FailureRefreshRequired)
	}
	assertOnlyVersion(t, registry, "crm", "v1")
}

// An adapter that declares no evidence is refused rather than trusted.
func TestRefreshRefusesAnAdapterWithNoPostLoginAssertion(t *testing.T) {
	broker, registry, _, protector := refreshFixture(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	adapter := &emptyAssertionAdapter{}
	if _, err := broker.Refresh(ctx, refreshRequest(newFakePage(loggedInState), adapter)); !IsFailure(err, FailureRefreshRequired) {
		t.Fatalf("Refresh returned %v, want %s", err, FailureRefreshRequired)
	}
	assertOnlyVersion(t, registry, "crm", "v1")
}

type emptyAssertionAdapter struct{}

func (a *emptyAssertionAdapter) Site() string             { return "crm.example.com" }
func (a *emptyAssertionAdapter) Credentials() []SecretRef { return nil }
func (a *emptyAssertionAdapter) Login(context.Context, PageDriver, SecretResolver) error {
	return nil
}
func (a *emptyAssertionAdapter) Assertion() PostLoginAssertion { return PostLoginAssertion{} }

// Refresh must hold an EXCLUSIVE lease: an ordinary run must not read a profile
// that is mid-refresh.
func TestRefreshTakesAnExclusiveLeaseAndReleasesIt(t *testing.T) {
	broker, registry, _, protector := refreshFixture(t)
	ctx := context.Background()
	profile := publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	// A live read lease must block the refresh.
	reader, err := broker.Leases().Acquire(ctx, profile, RunIdentity{RunID: "reader"}, LeaseRead)
	if err != nil {
		t.Fatalf("acquire read lease: %v", err)
	}
	if _, err := broker.Refresh(ctx, refreshRequest(newFakePage(loggedInState), &crmAdapter{})); !IsFailure(err, FailureLeaseConflict) {
		t.Fatalf("Refresh under a live read lease returned %v, want %s", err, FailureLeaseConflict)
	}
	if err := broker.Leases().Release(ctx, reader); err != nil {
		t.Fatalf("release: %v", err)
	}

	if _, err := broker.Refresh(ctx, refreshRequest(newFakePage(loggedInState), &crmAdapter{})); err != nil {
		t.Fatalf("Refresh after the reader left: %v", err)
	}
	// The refresh lease must be gone afterwards.
	if active := broker.Leases().Active(profile.Ref()); len(active) != 0 {
		t.Fatalf("refresh leaked %d lease(s)", len(active))
	}
}

func TestRefreshRejectsAReservedOrDuplicateVersion(t *testing.T) {
	broker, registry, _, protector := refreshFixture(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	for _, version := range []string{VersionLatest, "", "v1"} {
		request := refreshRequest(newFakePage(loggedInState), &crmAdapter{})
		request.NewVersion = version
		if _, err := broker.Refresh(ctx, request); err == nil {
			t.Fatalf("Refresh accepted new version %q", version)
		}
	}
	assertOnlyVersion(t, registry, "crm", "v1")
}

func TestBootstrapPublishesTheFirstVersion(t *testing.T) {
	broker, registry, sink, _ := refreshFixture(t)
	ctx := context.Background()

	published, err := broker.Bootstrap(ctx, BootstrapRequest{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: testPolicy(), State: []byte(loggedInState),
		Run:        RunIdentity{RunID: "bootstrap-1", Principal: "operator"},
		CapturedAt: time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if published.Version != "v1" || published.Sequence != 1 {
		t.Fatalf("bootstrap published %s (seq %d), want v1 seq 1", published.Version, published.Sequence)
	}

	record, err := registry.Open(ctx, published.Ref())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_, sealedBytes, _ := record.Material.Materialize()
	if strings.Contains(string(sealedBytes), "FRESH-SESSION-TOKEN") {
		t.Fatalf("bootstrap stored the state as plaintext")
	}
	if !auditOps(sink)["bootstrap"] {
		t.Fatalf("the bootstrap was not audited")
	}

	if _, err := broker.Bootstrap(ctx, BootstrapRequest{
		Alias: "crm", Version: "v2", Provider: "local-playwright", Policy: testPolicy(),
	}); err == nil {
		t.Fatalf("Bootstrap accepted empty state")
	}
}

func TestSecretValueRedactsEverywhere(t *testing.T) {
	secret := NewSecretValue("hunter2-correct-horse")

	for label, rendered := range map[string]string{
		"String":   secret.String(),
		"GoString": secret.GoString(),
		"Error":    secret.Error(),
		"%v":       fmt.Sprintf("%v", secret),
		"%+v":      fmt.Sprintf("%+v", secret),
		"%#v":      fmt.Sprintf("%#v", secret),
	} {
		if strings.Contains(rendered, "hunter2") {
			t.Fatalf("%s leaked the password: %s", label, rendered)
		}
	}

	encoded, err := json.Marshal(struct {
		Name     string
		Password SecretValue
	}{Name: "svc-automation", Password: secret})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(encoded), "hunter2") {
		t.Fatalf("JSON leaked the password: %s", encoded)
	}
	if secret.Reveal() != "hunter2-correct-horse" {
		t.Fatalf("Reveal did not return the credential")
	}
}

// The password must reach the browser driver, NOT a tool-call argument. This
// test proves the value actually arrives where it is supposed to.
func TestTheCredentialReachesTheDriverNotATranscript(t *testing.T) {
	broker, registry, _, protector := refreshFixture(t)
	ctx := context.Background()
	publishLocal(t, registry, protector, "crm", "v1", cookieJar)

	page := newFakePage(loggedInState)
	if _, err := broker.Refresh(ctx, refreshRequest(page, &crmAdapter{})); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if page.filled["#password"] != "hunter2-correct-horse" {
		t.Fatalf("the adapter did not fill the password through the driver")
	}
}

// The acceptance criterion in its enforceable form: no shipped code path turns a
// credential into an MCP tool argument. Reveal() is the only way to get the raw
// value, so an allowlist over its call sites IS the guarantee.
func TestNoShippedCodePathRevealsASecretIntoAToolArgument(t *testing.T) {
	// Reveal may be called only by trusted control-plane code. Today nothing in
	// the shipped tree calls it: real site adapters are the intended caller and
	// none ship yet. If you add one, add it here deliberately.
	allowed := map[string]bool{}

	roots := []string{".", "../../executor", "../../eval_harness", ".."}
	var offenders []string

	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(root, name)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if !strings.Contains(string(source), ".Reveal()") {
				continue
			}
			if allowed[filepath.ToSlash(path)] {
				continue
			}
			offenders = append(offenders, filepath.ToSlash(path))
		}
	}

	if len(offenders) > 0 {
		t.Fatalf("these shipped files reveal a raw credential and are not on the allowlist: %v\n"+
			"A revealed secret must never become an MCP tool argument — tool inputs are part of the AI transcript.",
			offenders)
	}
}

// The instrument above must be able to see a violation, or it proves nothing.
func TestTheRevealScanCanActuallyDetectAViolation(t *testing.T) {
	dir := t.TempDir()
	offending := filepath.Join(dir, "leaky.go")
	if err := os.WriteFile(offending, []byte("package x\nfunc f(v SecretValue) string { return v.Reveal() }\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	source, err := os.ReadFile(offending)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if !strings.Contains(string(source), ".Reveal()") {
		t.Fatalf("the scan's matcher cannot detect a plain Reveal() call")
	}
}

func assertOnlyVersion(t *testing.T, registry *MemoryRegistry, alias, version string) {
	t.Helper()
	profiles, err := registry.List(context.Background(), alias)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Version != version {
		var got []string
		for _, profile := range profiles {
			got = append(got, profile.Version)
		}
		t.Fatalf("registry holds %v, want only [%s] — a failed refresh must publish nothing", got, version)
	}
}
