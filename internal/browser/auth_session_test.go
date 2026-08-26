package browser

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/browser/auth"
)

const authTestState = `{"cookies":[{"name":"sid","value":"SUPER-SECRET-SESSION"}]}`

// fakeAuthProvider records whether a browser was ever provisioned. Tests that
// assert "the preflight ran first" fail loudly if it was.
type fakeAuthProvider struct {
	fakeProvider
	mu                sync.Mutex
	authenticated     int
	lastAttachment    AuthAttachment
	createAuthErr     error
	forbidProvisionin bool
	t                 *testing.T
}

func (f *fakeAuthProvider) Name() string { return "local-playwright" }

func (f *fakeAuthProvider) CreateAuthenticated(_ context.Context, _ SessionSpec, attachment AuthAttachment) (Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.forbidProvisionin {
		f.t.Errorf("a browser was provisioned even though the request should have been denied first")
	}
	f.authenticated++
	f.lastAttachment = attachment
	if f.createAuthErr != nil {
		return Session{}, f.createAuthErr
	}
	return Session{ID: "session-auth", Provider: f.Name(), CreatedAt: time.Unix(10, 0)}, nil
}

func (f *fakeAuthProvider) provisionCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.authenticated
}

type authFixture struct {
	broker   *auth.Broker
	registry *auth.MemoryRegistry
	sink     *auth.MemoryAuditSink
	provider *fakeAuthProvider
	control  *Controller
}

func newAuthFixture(t *testing.T, policy auth.AuthProfilePolicy) authFixture {
	t.Helper()

	protector, err := auth.NewRandomStaticKeyProtector("test-key")
	if err != nil {
		t.Fatalf("protector: %v", err)
	}
	registry := auth.NewMemoryRegistry()
	sink := auth.NewMemoryAuditSink()
	broker, err := auth.NewBroker(auth.BrokerOptions{
		Registry:    registry,
		Leases:      auth.NewLeaseManager(30 * time.Minute),
		Protector:   protector,
		Audit:       sink,
		SessionRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("broker: %v", err)
	}

	sealed, err := auth.Seal(context.Background(), protector, []byte(authTestState))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := registry.Publish(context.Background(), auth.Record{
		Alias: "crm", Version: "v1", Provider: "local-playwright",
		Policy: policy, Material: auth.NewStorageStateMaterial(sealed.Bytes()),
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	provider := &fakeAuthProvider{t: t}
	provider.fakeProvider.connection = NewSensitiveConnection(MCPServerSpec{Name: "playwright", Command: "fake"}, nil)

	return authFixture{
		broker:   broker,
		registry: registry,
		sink:     sink,
		provider: provider,
		control: NewController(provider, ControllerOptions{
			CleanupTimeout: time.Second,
			Now:            func() time.Time { return time.Unix(20, 0) },
		}),
	}
}

func runnablePolicy() auth.AuthProfilePolicy {
	return auth.AuthProfilePolicy{
		AllowedOrigins: []string{"https://crm.example.com"},
		AccountClass:   auth.AccountReadonly,
		MaxConcurrent:  1,
		AllowArtifacts: []string{},
		EgressBoundary: auth.EgressOperatorAcknowledged,
	}
}

func authConfig() AuthConfig {
	return AuthConfig{
		Request: auth.ProvisionRequest{
			Ref:      auth.AuthProfileRef{Alias: "crm", Version: auth.VersionLatest},
			Run:      auth.RunIdentity{RunID: "run-1", Principal: "eval-harness"},
			Mode:     auth.LeaseRead,
			Provider: "local-playwright",
		},
	}
}

// This is the acceptance criterion in its sharpest form: the fake provider fails
// the test if it is ever reached.
func TestDeniedProfileNeverProvisionsABrowser(t *testing.T) {
	policy := runnablePolicy()
	policy.EgressBoundary = auth.EgressAbsent // fail closed
	fixture := newAuthFixture(t, policy)
	fixture.provider.forbidProvisionin = true

	cfg := authConfig()
	cfg.Broker = fixture.broker

	_, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-1"}, cfg)
	if !auth.IsFailure(err, auth.FailureScopeDenied) {
		t.Fatalf("StartAuthenticated returned %v, want %s", err, auth.FailureScopeDenied)
	}
	if got := fixture.provider.provisionCount(); got != 0 {
		t.Fatalf("%d browser(s) were provisioned for a denied request", got)
	}
	if calls := fixture.provider.recorded(); len(calls) != 0 {
		t.Fatalf("the provider was called at all: %v", calls)
	}
}

func TestStartAuthenticatedStampsTheResolvedIdentity(t *testing.T) {
	fixture := newAuthFixture(t, runnablePolicy())
	cfg := authConfig()
	cfg.Broker = fixture.broker

	run, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-1", ArtifactDir: t.TempDir()}, cfg)
	if err != nil {
		t.Fatalf("StartAuthenticated: %v", err)
	}

	attachment := fixture.provider.lastAttachment
	if attachment.StorageStatePath == "" {
		t.Fatalf("the provider received no storage-state path")
	}
	contents, err := os.ReadFile(attachment.StorageStatePath)
	if err != nil {
		t.Fatalf("read materialized state: %v", err)
	}
	if string(contents) != authTestState {
		t.Fatalf("the provider received the wrong storage state")
	}
	if attachment.Mode != auth.LeaseRead || attachment.Persist {
		t.Fatalf("an ordinary run was attached in %q mode with persist=%t", attachment.Mode, attachment.Persist)
	}

	manifest, err := run.Finish(context.Background(), TerminationCompleted)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if manifest.AuthProfileAlias != "crm" || manifest.AuthProfileVersion != "v1" {
		t.Fatalf("manifest identity = %s@%s, want crm@v1", manifest.AuthProfileAlias, manifest.AuthProfileVersion)
	}
	if manifest.AuthProfileVersion == auth.VersionLatest {
		t.Fatalf("the manifest recorded %q instead of a concrete version", auth.VersionLatest)
	}
	if manifest.AuthLeaseID == "" || manifest.AuthMode != string(auth.LeaseRead) {
		t.Fatalf("manifest lease identity is incomplete: %+v", manifest)
	}
	if manifest.ProfileHash == "" {
		t.Fatalf("the manifest recorded no profile hash")
	}
}

func TestStartAuthenticatedReleasesEverythingWhenTheProviderFails(t *testing.T) {
	fixture := newAuthFixture(t, runnablePolicy())
	fixture.provider.createAuthErr = errors.New("chromium would not launch")

	cfg := authConfig()
	cfg.Broker = fixture.broker

	if _, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-1"}, cfg); err == nil {
		t.Fatalf("StartAuthenticated succeeded despite a provider failure")
	}
	assertNothingHeld(t, fixture)
}

func TestFinishReleasesTheLeaseAndDestroysThePlaintext(t *testing.T) {
	fixture := newAuthFixture(t, runnablePolicy())
	cfg := authConfig()
	cfg.Broker = fixture.broker

	run, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-1", ArtifactDir: t.TempDir()}, cfg)
	if err != nil {
		t.Fatalf("StartAuthenticated: %v", err)
	}
	path := fixture.provider.lastAttachment.StorageStatePath

	if _, err := run.Finish(context.Background(), TerminationCompleted); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Finish left decrypted state at %s", path)
	}
	assertNothingHeld(t, fixture)

	// Finish runs on paths that may already have finished.
	if _, err := run.Finish(context.Background(), TerminationCompleted); err != nil {
		t.Fatalf("second Finish: %v", err)
	}
}

// Cancellation and deadline are the two ways a run ends without the caller
// reaching its own cleanup. Both must still release.
func TestUseReleasesOnCancellationAndDeadline(t *testing.T) {
	for _, name := range []string{"cancelled", "deadline"} {
		t.Run(name, func(t *testing.T) {
			fixture := newAuthFixture(t, runnablePolicy())
			cfg := authConfig()
			cfg.Broker = fixture.broker

			run, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-1", ArtifactDir: t.TempDir()}, cfg)
			if err != nil {
				t.Fatalf("StartAuthenticated: %v", err)
			}
			path := fixture.provider.lastAttachment.StorageStatePath

			ctx, cancel := context.WithCancel(context.Background())
			if name == "deadline" {
				ctx, cancel = context.WithTimeout(context.Background(), time.Millisecond)
			}
			outcome := run.Use(ctx, TerminationCancelled, func(ctx context.Context) error {
				cancel()
				<-ctx.Done()
				return ctx.Err()
			})
			cancel()

			if outcome.Err == nil {
				t.Fatalf("Use reported no error for a %s run", name)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("a %s run left decrypted state at %s", name, path)
			}
			assertNothingHeld(t, fixture)
		})
	}
}

// A panic must still destroy the plaintext and free the lease, and must still
// reach the caller — swallowing it would hide the failure.
func TestUseReleasesOnPanicAndRepanics(t *testing.T) {
	fixture := newAuthFixture(t, runnablePolicy())
	cfg := authConfig()
	cfg.Broker = fixture.broker

	run, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-1", ArtifactDir: t.TempDir()}, cfg)
	if err != nil {
		t.Fatalf("StartAuthenticated: %v", err)
	}
	path := fixture.provider.lastAttachment.StorageStatePath

	panicked := func() (recovered any) {
		defer func() { recovered = recover() }()
		run.Use(context.Background(), TerminationExecutorFailed, func(context.Context) error {
			panic("agent exploded mid-run")
		})
		return nil
	}()

	if panicked == nil {
		t.Fatalf("Use swallowed the panic")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("a panicking run left decrypted state at %s", path)
	}
	assertNothingHeld(t, fixture)
}

func TestStartAuthenticatedRefusesAProviderThatCannotAuthenticate(t *testing.T) {
	fixture := newAuthFixture(t, runnablePolicy())
	// A plain provider has no CreateAuthenticated method.
	plain := &fakeProvider{connection: NewSensitiveConnection(MCPServerSpec{Name: "playwright", Command: "fake"}, nil)}
	control := NewController(plain, ControllerOptions{CleanupTimeout: time.Second, Now: func() time.Time { return time.Unix(20, 0) }})

	cfg := authConfig()
	cfg.Broker = fixture.broker
	cfg.Request.Provider = "" // skip the provider-match check so we reach the capability check

	_, err := control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-1"}, cfg)
	if !auth.IsFailure(err, auth.FailureScopeDenied) {
		t.Fatalf("StartAuthenticated returned %v, want %s", err, auth.FailureScopeDenied)
	}
	if calls := plain.recorded(); len(calls) != 0 {
		t.Fatalf("an unauthenticated provider was called anyway: %v", calls)
	}
	assertNothingHeld(t, fixture)
}

func TestStartAuthenticatedRequiresABroker(t *testing.T) {
	fixture := newAuthFixture(t, runnablePolicy())
	if _, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-1"}, AuthConfig{}); err == nil {
		t.Fatalf("StartAuthenticated accepted a nil broker")
	}
	if got := fixture.provider.provisionCount(); got != 0 {
		t.Fatalf("%d browser(s) provisioned without a broker", got)
	}
}

// The run id is validated before anything is leased or decrypted.
func TestStartAuthenticatedValidatesTheRunIDFirst(t *testing.T) {
	fixture := newAuthFixture(t, runnablePolicy())
	fixture.provider.forbidProvisionin = true
	cfg := authConfig()
	cfg.Broker = fixture.broker

	if _, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{}, cfg); err == nil {
		t.Fatalf("StartAuthenticated accepted an empty run id")
	}
	assertNothingHeld(t, fixture)
}

func TestConcurrentAuthenticatedRunsLeakNoLeases(t *testing.T) {
	policy := runnablePolicy()
	policy.MaxConcurrent = 4
	fixture := newAuthFixture(t, policy)

	const workers = 16
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cfg := authConfig()
			cfg.Broker = fixture.broker
			cfg.Request.Run = auth.RunIdentity{RunID: "run-" + string(rune('a'+i)), Principal: "eval-harness"}

			run, err := fixture.control.StartAuthenticated(context.Background(), SessionSpec{RunID: "run-x", ArtifactDir: t.TempDir()}, cfg)
			if err != nil {
				if !auth.IsFailure(err, auth.FailureLeaseConflict) {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if _, err := run.Finish(context.Background(), TerminationCompleted); err != nil {
				t.Errorf("Finish: %v", err)
			}
		}(i)
	}
	wg.Wait()

	assertNothingHeld(t, fixture)
}

func assertNothingHeld(t *testing.T, fixture authFixture) {
	t.Helper()

	ref := auth.AuthProfileRef{Alias: "crm", Version: "v1"}
	if active := fixture.broker.Leases().Active(ref); len(active) != 0 {
		t.Fatalf("%d lease(s) still held", len(active))
	}
	entries, err := os.ReadDir(fixture.broker.SessionRoot())
	if err != nil {
		t.Fatalf("read session root: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("session root still holds %d materialization(s)", len(entries))
	}
}
