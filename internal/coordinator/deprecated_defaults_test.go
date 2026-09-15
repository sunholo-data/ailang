package coordinator

// M-V1-SIMPLIFY-S4 M1: the coordinator's remaining silent fallbacks are
// deprecated defaults (ruling D3). Each test proves BOTH halves of the
// contract: the default is still served with AILANG_STRICT_CONFIG unset, and
// AILANG_STRICT_CONFIG=1 yields an error wrapping config.ErrDeprecatedDefault
// (or the named strict behaviour). The once-per-process stderr line itself is
// config's own contract, proven in internal/config/cloud_test.go; here the
// assertion is on the VALUE and the ERROR, which is what a caller sees.

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pubsub"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// noDeprecatedEnv clears every variable these sites read, so a developer's
// shell cannot make a "default applies" test pass by accident.
func noDeprecatedEnv(t *testing.T) {
	t.Helper()
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))
	for _, v := range []string{config.EnvStrict, EnvDefaultProvider, EnvBudgetUnlimited, EnvWorkspace, EnvApprovalTimeout} {
		t.Setenv(v, "")
	}
}

func quietDaemon() *Daemon {
	return &Daemon{ctx: context.Background(), logger: log.New(io.Discard, "", 0), taskStore: NewMockStore()}
}

// --- Site 1: provider "claude" ---------------------------------------------

func TestTaskProvider_DeprecatedDefaultThenStrict(t *testing.T) {
	noDeprecatedEnv(t)
	d := quietDaemon()

	got, err := d.taskProvider(nil)
	if err != nil || got != "claude" {
		t.Fatalf("unset: (%q, %v), want the deprecated default \"claude\" served", got, err)
	}

	t.Setenv(EnvDefaultProvider, "codex")
	if got, err = d.taskProvider(nil); err != nil || got != "codex" {
		t.Fatalf("%s=codex: (%q, %v)", EnvDefaultProvider, got, err)
	}
	t.Setenv(EnvDefaultProvider, "")

	t.Setenv(config.EnvStrict, "1")
	if got, err = d.taskProvider(nil); got != "" || !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("strict: (%q, %v), want config.ErrDeprecatedDefault", got, err)
	}
	// Declared providers are never deprecated, strict or not.
	if got, err = d.taskProvider(&AgentConfig{Provider: "pi"}); err != nil || got != "pi" {
		t.Fatalf("strict with a declared provider: (%q, %v)", got, err)
	}
	d.coordConfig = &CoordinatorConfig{DefaultProvider: "opencode"}
	if got, err = d.taskProvider(nil); err != nil || got != "opencode" {
		t.Fatalf("strict with coordinator default_provider: (%q, %v)", got, err)
	}
}

// --- Site 9: budget cap that disappears ------------------------------------

// A budgets config that cannot be read used to mean "no enforcement". Now it
// is the deprecated "unlimited" default: served with a warning, acknowledged
// by AILANG_BUDGET_UNLIMITED=1, refused under strict.
func TestCheckBudget_UnreadableConfigIsTheDeprecatedUnlimitedDefault(t *testing.T) {
	noDeprecatedEnv(t)
	bad := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(bad, []byte("budgets: [unclosed\n  - :\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_CONFIG", bad)
	if _, err := LoadBudgetsConfig(); err == nil {
		t.Fatal("instrument check: the malformed config must fail to load, or this test proves nothing")
	}

	d := quietDaemon()
	task := &TaskRecord{ID: "task-uncapped"}

	blocked, err := d.checkBudgetBeforeExecution(context.Background(), task, &AgentConfig{Provider: "claude"})
	if blocked || err != nil {
		t.Fatalf("unset: (blocked=%v, %v), want the task allowed on the deprecated unlimited default", blocked, err)
	}

	t.Setenv(config.EnvStrict, "1")
	blocked, err = d.checkBudgetBeforeExecution(context.Background(), task, &AgentConfig{Provider: "claude"})
	if !blocked || !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("strict: (blocked=%v, %v), want blocked with config.ErrDeprecatedDefault", blocked, err)
	}
	if n := d.taskStore.(*MockStore).calls["MarkTaskFailed"]; n != 1 {
		t.Fatalf("strict refusal must mark the task failed exactly once, got %d", n)
	}

	// The explicit acknowledgement is not a deprecated default.
	t.Setenv(EnvBudgetUnlimited, "1")
	blocked, err = d.checkBudgetBeforeExecution(context.Background(), task, &AgentConfig{Provider: "claude"})
	if blocked || err != nil {
		t.Fatalf("strict + %s=1: (blocked=%v, %v), want allowed", EnvBudgetUnlimited, blocked, err)
	}
}

// With the compiled-in defaults every provider has a cap, so strict mode must
// NOT refuse a normally-configured task — the refusal is for the cap that
// disappeared, not for budgets in general.
func TestCheckBudget_ConfiguredCapIsUnaffectedByStrict(t *testing.T) {
	noDeprecatedEnv(t)
	t.Setenv(config.EnvStrict, "1")
	d := quietDaemon()
	blocked, err := d.checkBudgetBeforeExecution(context.Background(), &TaskRecord{ID: "task-capped"}, &AgentConfig{Provider: "claude"})
	if blocked || err != nil {
		t.Fatalf("strict with default caps: (blocked=%v, %v), want allowed", blocked, err)
	}
}

// --- Site 8: approval timeout 0 → 1h ---------------------------------------

func TestApprovalCheckpoint_ZeroTimeoutIsTheDeprecatedDefault(t *testing.T) {
	noDeprecatedEnv(t)

	ac := NewApprovalCheckpoint(0)
	if ac.timeoutErr != nil || ac.defaultTimeout != time.Hour {
		t.Fatalf("unset: default=%v err=%v, want 1h served", ac.defaultTimeout, ac.timeoutErr)
	}

	t.Setenv(EnvApprovalTimeout, "36h")
	if ac = NewApprovalCheckpoint(0); ac.timeoutErr != nil || ac.defaultTimeout != 36*time.Hour {
		t.Fatalf("%s=36h: default=%v err=%v", EnvApprovalTimeout, ac.defaultTimeout, ac.timeoutErr)
	}

	// A set-but-broken value is an error, never silently 1h.
	t.Setenv(EnvApprovalTimeout, "soon")
	if ac = NewApprovalCheckpoint(0); ac.timeoutErr == nil || ac.defaultTimeout != 0 {
		t.Fatalf("%s=soon: default=%v err=%v, want an error and no default", EnvApprovalTimeout, ac.defaultTimeout, ac.timeoutErr)
	}
	t.Setenv(EnvApprovalTimeout, "")

	// An explicit constructor timeout never consults the environment.
	t.Setenv(config.EnvStrict, "1")
	if ac = NewApprovalCheckpoint(24 * time.Hour); ac.timeoutErr != nil || ac.defaultTimeout != 24*time.Hour {
		t.Fatalf("explicit 24h under strict: default=%v err=%v", ac.defaultTimeout, ac.timeoutErr)
	}

	ac = NewApprovalCheckpoint(0)
	if !errors.Is(ac.timeoutErr, config.ErrDeprecatedDefault) {
		t.Fatalf("strict: timeoutErr = %v, want config.ErrDeprecatedDefault", ac.timeoutErr)
	}
	// The refusal is deferred to the request that needs the default …
	status, err := ac.RequestApproval(context.Background(), &ApprovalRequest{ID: "needs-default", Type: ApprovalTypeMerge})
	if status != ApprovalStatusRejected || !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("strict request without Timeout: (%s, %v), want rejected with config.ErrDeprecatedDefault", status, err)
	}
	if ac.GetRequest("needs-default") != nil {
		t.Fatal("a refused request must not be registered as pending")
	}
	// … and a request carrying its own Timeout is unaffected.
	go func() {
		for i := 0; i < 200; i++ {
			if ac.Approve("has-timeout", "test") == nil {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	status, err = ac.RequestApproval(context.Background(), &ApprovalRequest{ID: "has-timeout", Type: ApprovalTypeMerge, Timeout: time.Minute})
	if err != nil || status != ApprovalStatusApproved {
		t.Fatalf("strict request with its own Timeout: (%s, %v), want approved", status, err)
	}
}

// --- Site 3: workspace "default" (daemon broadcaster) -----------------------

func TestInitPubSubBroadcaster_WorkspaceIsTheDeprecatedDefault(t *testing.T) {
	noDeprecatedEnv(t)
	newDaemon := func() *Daemon {
		d := quietDaemon()
		d.pubsubPublisher = &pubsub.Publisher{} // non-nil so initPubSub (network) is never reached
		return d
	}

	d := newDaemon()
	if err := d.initPubSubBroadcaster(); err != nil {
		t.Fatalf("unset: %v, want the deprecated \"default\" workspace served", err)
	}
	if d.eventBroadcaster == nil {
		t.Fatal("unset: broadcaster not installed")
	}

	d = newDaemon()
	t.Setenv(config.EnvStrict, "1")
	if err := d.initPubSubBroadcaster(); !errors.Is(err, config.ErrDeprecatedDefault) {
		t.Fatalf("strict: %v, want config.ErrDeprecatedDefault", err)
	}
	if d.eventBroadcaster != nil {
		t.Fatal("strict: broadcaster must not be installed on a refused workspace")
	}

	d = newDaemon()
	t.Setenv(EnvWorkspace, "sunholo-data/ailang")
	if err := d.initPubSubBroadcaster(); err != nil || d.eventBroadcaster == nil {
		t.Fatalf("strict with %s set: err=%v broadcaster=%v", EnvWorkspace, err, d.eventBroadcaster != nil)
	}
}
