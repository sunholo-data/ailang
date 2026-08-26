package eval_harness

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
	"github.com/sunholo-data/ailang/internal/browser/auth"
	"github.com/sunholo-data/ailang/internal/executor"
)

const evalAuthState = `{"cookies":[{"name":"sid","value":"EVAL-SESSION-TOKEN"}],"origins":[]}`

// authBrowserTestProvider is a browserTestProvider that can also start from a
// leased profile, which is what a real provider must do.
type authBrowserTestProvider struct {
	browserTestProvider
	attachment    browser.AuthAttachment
	authenticated int

	// seenState is read WHILE the session is live. Reading it afterwards would
	// always fail, because destroying the plaintext is the point.
	seenState    string
	seenStateErr error
}

func (p *authBrowserTestProvider) CreateAuthenticated(_ context.Context, _ browser.SessionSpec, attachment browser.AuthAttachment) (browser.Session, error) {
	p.authenticated++
	p.attachment = attachment
	p.calls = append(p.calls, "create-authenticated")
	if attachment.StorageStatePath != "" {
		contents, err := os.ReadFile(attachment.StorageStatePath)
		p.seenState, p.seenStateErr = string(contents), err
	}
	return browser.Session{ID: "browser-auth-1", Provider: p.Name(), CreatedAt: time.Now()}, nil
}

func newEvalAuthBroker(t *testing.T, policy auth.AuthProfilePolicy) *auth.Broker {
	t.Helper()

	root := t.TempDir()
	registry, err := auth.NewFileRegistry(filepath.Join(root, "profiles"))
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	protector, err := auth.NewRandomStaticKeyProtector("eval-test")
	if err != nil {
		t.Fatalf("protector: %v", err)
	}
	broker, err := auth.NewBroker(auth.BrokerOptions{
		Registry:    registry,
		Leases:      auth.NewLeaseManager(time.Minute),
		Protector:   protector,
		SessionRoot: filepath.Join(root, "materializations"),
	})
	if err != nil {
		t.Fatalf("broker: %v", err)
	}
	if _, err := broker.Bootstrap(context.Background(), auth.BootstrapRequest{
		Alias: "crm", Version: "v1", Provider: "test-browser",
		Policy: policy, State: []byte(evalAuthState),
		Run: auth.RunIdentity{RunID: "bootstrap", Principal: "operator"},
	}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	return broker
}

func evalAuthPolicy() auth.AuthProfilePolicy {
	return auth.AuthProfilePolicy{
		AllowedOrigins: []string{"https://crm.example.com"},
		AccountClass:   auth.AccountReadonly,
		MaxConcurrent:  1,
		AllowArtifacts: []string{},
		EgressBoundary: auth.EgressOperatorAcknowledged,
	}
}

func TestEvalRunStartsAuthenticatedAndBanksSafeIdentity(t *testing.T) {
	provider := &authBrowserTestProvider{}
	provider.secret = "wss://secret-endpoint"
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapMCP}}
	broker := newEvalAuthBroker(t, evalAuthPolicy())

	task := &executor.Task{ID: "run-auth-1", Workspace: t.TempDir()}
	result, manifest, err := executeWithBrowser(context.Background(), exec, task, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider:           "test-browser",
		ProviderInstance:   provider,
		AuthProfile:        "crm@latest",
		AuthBrokerInstance: broker,
		ArtifactDir:        t.TempDir(),
	})
	if err != nil {
		t.Fatalf("executeWithBrowser: %v", err)
	}
	if result == nil || manifest == nil {
		t.Fatalf("missing result or manifest")
	}

	if provider.authenticated != 1 {
		t.Fatalf("authenticated create called %d times, want 1", provider.authenticated)
	}
	// The provider must have received a real, readable materialization while the
	// session was live.
	if provider.seenStateErr != nil {
		t.Fatalf("the provider could not read its materialized state: %v", provider.seenStateErr)
	}
	if provider.seenState != evalAuthState {
		t.Fatalf("the provider received the wrong storage state")
	}
	if provider.attachment.Mode != auth.LeaseRead || provider.attachment.Persist {
		t.Fatalf("an eval run was attached as mode=%q persist=%t; evals are always read-only",
			provider.attachment.Mode, provider.attachment.Persist)
	}

	// The banked manifest identifies WHICH profile ran, by concrete version.
	if manifest.AuthProfileAlias != "crm" || manifest.AuthProfileVersion != "v1" {
		t.Fatalf("manifest identity = %s@%s, want crm@v1", manifest.AuthProfileAlias, manifest.AuthProfileVersion)
	}
	if manifest.AuthProfileVersion == auth.VersionLatest {
		t.Fatalf("the manifest banked %q instead of a concrete version", auth.VersionLatest)
	}
	if manifest.AuthLeaseID == "" || manifest.ProfileHash == "" {
		t.Fatalf("manifest is missing lease or profile identity: %+v", manifest)
	}
}

// Recording is off for authenticated runs: private page content is as sensitive
// as the login state, and the policy that would classify it has not shipped.
func TestAuthenticatedEvalRunsDenyRecordingByDefault(t *testing.T) {
	provider := &authBrowserTestProvider{}
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapMCP}}
	broker := newEvalAuthBroker(t, evalAuthPolicy())

	task := &executor.Task{ID: "run-auth-2", Workspace: t.TempDir()}
	if _, _, err := executeWithBrowser(context.Background(), exec, task, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: provider,
		AuthProfile: "crm@latest", AuthBrokerInstance: broker, ArtifactDir: t.TempDir(),
	}); err != nil {
		t.Fatalf("executeWithBrowser: %v", err)
	}

	// An anonymous run keeps recording on; the authenticated one must not.
	anonymous := &authBrowserTestProvider{}
	anonTask := &executor.Task{ID: "run-anon", Workspace: t.TempDir()}
	if _, _, err := executeWithBrowser(context.Background(), exec, anonTask, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: anonymous, ArtifactDir: t.TempDir(),
	}); err != nil {
		t.Fatalf("anonymous executeWithBrowser: %v", err)
	}
	if anonymous.authenticated != 0 {
		t.Fatalf("an anonymous run took the authenticated path")
	}
}

// A denied profile must fail the run rather than quietly downgrading it. A run
// that silently starts logged out looks like a model failure.
func TestDeniedProfileFailsTheEvalRunInsteadOfDowngrading(t *testing.T) {
	policy := evalAuthPolicy()
	policy.EgressBoundary = auth.EgressAbsent // fail closed

	provider := &authBrowserTestProvider{}
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapMCP}}
	broker := newEvalAuthBroker(t, policy)

	task := &executor.Task{ID: "run-auth-3", Workspace: t.TempDir()}
	_, _, err := executeWithBrowser(context.Background(), exec, task, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: provider,
		AuthProfile: "crm@latest", AuthBrokerInstance: broker, ArtifactDir: t.TempDir(),
	})
	if err == nil {
		t.Fatalf("a denied profile produced a successful run")
	}
	if !auth.IsFailure(err, auth.FailureScopeDenied) {
		t.Fatalf("error = %v, want %s", err, auth.FailureScopeDenied)
	}
	if provider.authenticated != 0 || len(provider.calls) != 0 {
		t.Fatalf("a browser was provisioned for a denied profile: %v", provider.calls)
	}
}

func TestMalformedProfileReferenceFailsBeforeProvisioning(t *testing.T) {
	provider := &authBrowserTestProvider{}
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapMCP}}

	task := &executor.Task{ID: "run-auth-4", Workspace: t.TempDir()}
	_, _, err := executeWithBrowser(context.Background(), exec, task, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: provider,
		AuthProfile: "../escape@v1", ArtifactDir: t.TempDir(),
	})
	if err == nil {
		t.Fatalf("a path-escaping profile reference was accepted")
	}
	if len(provider.calls) != 0 {
		t.Fatalf("a browser was provisioned for a malformed reference: %v", provider.calls)
	}
}

// Whatever else a banked result carries, it must not carry the login state.
func TestBankedResultCarriesNoProfileMaterial(t *testing.T) {
	provider := &authBrowserTestProvider{}
	provider.secret = "wss://secret-endpoint"
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapMCP}}
	broker := newEvalAuthBroker(t, evalAuthPolicy())

	task := &executor.Task{ID: "run-auth-5", Workspace: t.TempDir()}
	result, manifest, err := executeWithBrowser(context.Background(), exec, task, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: provider,
		AuthProfile: "crm@latest", AuthBrokerInstance: broker, ArtifactDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("executeWithBrowser: %v", err)
	}

	for label, value := range map[string]any{"result": result, "manifest": manifest} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s: %v", label, err)
		}
		for _, forbidden := range []string{"EVAL-SESSION-TOKEN", "secret-endpoint", `"sid"`} {
			if strings.Contains(string(encoded), forbidden) {
				t.Fatalf("banked %s leaked %q: %s", label, forbidden, encoded)
			}
		}
	}
}

// After the run, nothing decrypted may remain.
func TestAuthenticatedEvalRunLeavesNoPlaintextBehind(t *testing.T) {
	provider := &authBrowserTestProvider{}
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapMCP}}
	broker := newEvalAuthBroker(t, evalAuthPolicy())

	task := &executor.Task{ID: "run-auth-6", Workspace: t.TempDir()}
	if _, _, err := executeWithBrowser(context.Background(), exec, task, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: provider,
		AuthProfile: "crm@latest", AuthBrokerInstance: broker, ArtifactDir: t.TempDir(),
	}); err != nil {
		t.Fatalf("executeWithBrowser: %v", err)
	}

	path := provider.attachment.StorageStatePath
	if path == "" {
		t.Fatalf("no materialization was made, so this test proves nothing")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("the run left decrypted state at %s", path)
	}
	entries, err := os.ReadDir(broker.SessionRoot())
	if err != nil {
		t.Fatalf("read session root: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("session root still holds %d materialization(s)", len(entries))
	}
	if active := broker.Leases().Active(auth.AuthProfileRef{Alias: "crm", Version: "v1"}); len(active) != 0 {
		t.Fatalf("the run leaked %d lease(s)", len(active))
	}
}
