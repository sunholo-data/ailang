package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// $AILANG_CONFIG names the ailang config, which is not necessarily a registry.
//
// Daneel's send path sets it to a pubsub-only file because a send publishes its
// notification only when the sender's config has a pubsub section. The resolver
// then accepted that file as the agent registry — zero agents, labelled
// authoritative — and every dispatch guard that checks "am I looking at the
// shared plane?" correctly refused. Measured 2026-09-17 across daneel v0.2.5 →
// v0.2.11: two design requests deferred seven hours.
func TestLoadAgentRegistryFromDeclared_PubsubOnlyConfigDeclaresNothing(t *testing.T) {
	dir := t.TempDir()
	pubsubOnly := filepath.Join(dir, "ailang-messaging.yaml")
	if err := os.WriteFile(pubsubOnly, []byte(`pubsub:
  enabled: true
  project_id: ailang-multivac
  topic: ailang-messages
`), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, declared, err := coordinator.LoadAgentRegistryFromDeclared(pubsubOnly)
	if err != nil {
		t.Fatalf("a pubsub-only config must LOAD without error (it is a valid config): %v", err)
	}
	if declared {
		t.Fatal("a config with no coordinator section was reported as declaring agents")
	}
	// The dangerous part, and the reason "declared" has to be a separate fact:
	// the registry handed back is NOT empty. It is AILANG's built-in default
	// fleet, which reads exactly like a real deployment.
	if len(reg.ListAgents()) == 0 {
		t.Log("note: the built-in default fleet is empty on this build; the guard still holds")
	} else if reg.GetAgentByID("sprint-planner") == nil {
		t.Logf("built-in defaults changed shape: %d agents", len(reg.ListAgents()))
	}

	// A file that really does declare agents still wins, or the fix would take
	// the flag's own purpose away.
	withAgents := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(withAgents, []byte(`coordinator:
  agents:
    - id: pkg-sunholo-auth
      inbox: "pkg:sunholo/auth"
      workspace: sunholo-data/ailang-packages
`), 0o644); err != nil {
		t.Fatal(err)
	}
	reg2, declared2, err := coordinator.LoadAgentRegistryFromDeclared(withAgents)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !declared2 {
		t.Fatalf("a config declaring an agent must be accepted, got %d agents", len(reg2.ListAgents()))
	}
	if reg2.GetAgentForInbox("pkg:sunholo/auth") == nil {
		t.Error("the declared agent is missing from the registry")
	}
}

// `--registry cloud` is a plane, not a path: a caller that knows it means the
// shared plane must not have to arrange for an env var to be absent to say so.
func TestResolveInboxRegistry_CloudIsNotTreatedAsAPath(t *testing.T) {
	_, label, err := resolveInboxRegistry("cloud")
	if err != nil {
		// Offline or unauthorised is fine — what must NOT happen is git-style
		// "cannot load the agent registry cloud", which would mean it looked
		// for a FILE called cloud.
		if strings.Contains(err.Error(), "cannot load the agent registry cloud") {
			t.Fatalf("--registry cloud was treated as a file path: %v", err)
		}
		if !strings.Contains(err.Error(), "--registry cloud") {
			t.Errorf("the error should name the flag that failed: %v", err)
		}
		return
	}
	if !strings.HasPrefix(label, "gs://") {
		t.Errorf("--registry cloud resolved to %q, want the plane's gs:// registry", label)
	}
}
