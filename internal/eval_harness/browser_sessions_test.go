package eval_harness

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
	"github.com/sunholo-data/ailang/internal/executor"
)

type browserTestProvider struct {
	calls  []string
	secret string
}

func (p *browserTestProvider) Name() string { return "test-browser" }
func (p *browserTestProvider) Create(context.Context, browser.SessionSpec) (browser.Session, error) {
	p.calls = append(p.calls, "create")
	return browser.Session{ID: "browser-1", Provider: p.Name(), CreatedAt: time.Now()}, nil
}
func (p *browserTestProvider) Connection(context.Context, browser.Session) (browser.SensitiveConnection, error) {
	p.calls = append(p.calls, "connection")
	return browser.NewSensitiveConnection(browser.MCPServerSpec{Name: "playwright", Command: "npx", EnvVars: []string{"BROWSER_ENDPOINT"}, Required: true}, map[string]string{"BROWSER_ENDPOINT": p.secret}), nil
}
func (p *browserTestProvider) Inspect(context.Context, browser.Session) (browser.InspectionRef, error) {
	p.calls = append(p.calls, "inspect")
	return browser.InspectionRef{Available: true, Ref: "safe-ref"}, nil
}
func (p *browserTestProvider) Export(context.Context, browser.Session, string) (browser.ArtifactManifest, error) {
	p.calls = append(p.calls, "export")
	return browser.ArtifactManifest{Complete: true}, nil
}
func (p *browserTestProvider) Stop(context.Context, browser.Session) (browser.Usage, error) {
	p.calls = append(p.calls, "stop")
	return browser.Usage{ActionCount: 3}, nil
}

type browserTestExecutor struct {
	caps []executor.Capability
	err  error
	seen *executor.Task
}

func (e *browserTestExecutor) Name() string { return "test-executor" }
func (e *browserTestExecutor) Execute(ctx context.Context, task *executor.Task) (*executor.Result, error) {
	return e.ExecuteStreaming(ctx, task, &executor.NoOpEventHandler{})
}
func (e *browserTestExecutor) ExecuteStreaming(_ context.Context, task *executor.Task, _ executor.EventHandler) (*executor.Result, error) {
	e.seen = task
	return &executor.Result{Success: e.err == nil, ProviderData: map[string]any{}}, e.err
}
func (e *browserTestExecutor) Capabilities() []executor.Capability { return e.caps }
func (e *browserTestExecutor) CostModel() *executor.CostModel      { return &executor.CostModel{} }
func (e *browserTestExecutor) HealthCheck(context.Context) error   { return nil }
func (e *browserTestExecutor) Close() error                        { return nil }

func TestExecuteWithBrowserInjectsMCPAndBanksSafeManifest(t *testing.T) {
	secret := "wss://browser.example?token=never-bank"
	provider := &browserTestProvider{secret: secret}
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapMCP}}
	task := &executor.Task{ID: "run-1", ExtraEnv: map[string]string{"EXISTING": "kept"}}
	result, manifest, err := executeWithBrowser(context.Background(), exec, task, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: provider, ChainID: "chain", StageID: "stage",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(exec.seen.MCPServers) != 1 || exec.seen.ExtraEnv["BROWSER_ENDPOINT"] != secret || exec.seen.ExtraEnv["EXISTING"] != "kept" {
		t.Fatalf("browser task not injected correctly: %#v", exec.seen)
	}
	if manifest == nil || manifest.Provider != provider.Name() || manifest.Usage.ActionCount != 3 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if !manifest.Comparable || manifest.ManagedVessel || manifest.NonComparableReason != "" {
		t.Fatalf("neutral browser manifest has wrong comparison identity: %#v", manifest)
	}
	b, _ := json.Marshal(result.ProviderData)
	if strings.Contains(string(b), secret) || strings.Contains(string(b), "never-bank") {
		t.Fatalf("banked provider data leaked connection: %s", b)
	}
	wantCalls := "create,connection,inspect,export,stop"
	if got := strings.Join(provider.calls, ","); got != wantCalls {
		t.Fatalf("calls = %s, want %s", got, wantCalls)
	}
}

func TestExecuteWithBrowserCleansUpOnExecutorError(t *testing.T) {
	provider := &browserTestProvider{secret: "secret"}
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapMCP}, err: errors.New("executor failed")}
	_, manifest, err := executeWithBrowser(context.Background(), exec, &executor.Task{ID: "run"}, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: provider,
	})
	if err == nil || manifest == nil || manifest.Termination != browser.TerminationExecutorFailed {
		t.Fatalf("err=%v manifest=%#v", err, manifest)
	}
	if got := strings.Join(provider.calls, ","); !strings.HasSuffix(got, "export,stop") {
		t.Fatalf("cleanup missing: %s", got)
	}
}

func TestExecuteWithBrowserRejectsUnsupportedExecutorBeforeProvision(t *testing.T) {
	provider := &browserTestProvider{secret: "secret"}
	exec := &browserTestExecutor{caps: []executor.Capability{executor.CapStreaming}}
	_, _, err := executeWithBrowser(context.Background(), exec, &executor.Task{ID: "run"}, &executor.NoOpEventHandler{}, BrowserSessionConfig{
		Provider: "test-browser", ProviderInstance: provider,
	})
	if err == nil || len(provider.calls) != 0 {
		t.Fatalf("unsupported executor should fail before provision: err=%v calls=%v", err, provider.calls)
	}
}
