package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Audit H-6 P0.3 (M-EXECUTOR-UID-SPLIT): the agent CLI runs as the image user,
// so anything that user owns, the agent can rewrite. /plugins and the
// pre-baked ailang_bootstrap hooks must stay root-owned; only the runtime
// clone root is handed to ailang.

func agentBaseRunText(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "docker", "Dockerfile.agent-base"))
	if err != nil {
		t.Fatalf("read Dockerfile.agent-base: %v", err)
	}
	// Join continuation lines so one RUN is one line.
	return strings.ReplaceAll(string(b), "\\\n", " ")
}

var chownArgs = regexp.MustCompile(`chown\s+(?:-R\s+)?\S+\s+([^&;\n]+)`)

func TestAgentBaseKeepsPluginsRootOwned(t *testing.T) {
	text := agentBaseRunText(t)
	chowned := map[string]bool{}
	for _, m := range chownArgs.FindAllStringSubmatch(text, -1) {
		for _, p := range strings.Fields(m[1]) {
			chowned[strings.TrimRight(p, "/")] = true
		}
	}
	for _, forbidden := range []string{"/plugins", "/plugins/ailang_bootstrap"} {
		if chowned[forbidden] {
			t.Errorf("Dockerfile.agent-base chowns %s to the agent user: the agent could rewrite its own hooks (audit H-6 P0.3)", forbidden)
		}
	}
	if !chowned[runtimePluginRoot] {
		t.Errorf("Dockerfile.agent-base does not chown %s: execute-job's runtime plugin clone would fail with EACCES", runtimePluginRoot)
	}
}

// The seam: the path the code clones into and the path the image makes
// writable are two copies of one fact.
func TestRuntimePluginRootMatchesImage(t *testing.T) {
	if !strings.HasPrefix(runtimePluginRoot, "/plugins/") {
		t.Fatalf("runtimePluginRoot = %q, want a subdirectory of /plugins", runtimePluginRoot)
	}
	if !strings.Contains(agentBaseRunText(t), "mkdir -p /workspace "+runtimePluginRoot) {
		t.Errorf("Dockerfile.agent-base does not create %s", runtimePluginRoot)
	}
}
