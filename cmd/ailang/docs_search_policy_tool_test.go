package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/policytool"
)

// This host-Go boundary requires no restricted runtime/symlink fixture. In
// particular, the real child success/refusal assertions also run on Windows.
func TestDocsSearch_ConfinementActualPolicyTool(t *testing.T) {
	f := newDocsConfinementFixture(t)
	f.proveInstruments(t)
	// Only the host --policy argument supplies policy; the parent marker is absent.
	t.Setenv("AILANG_AGENT_POLICY", "")
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.toml")
	if err := os.WriteFile(policy, []byte("allowed_caps = []\nentry = \"main\"\ncli_allow = [\"docs:search\"]\nmax_output_bytes = 16384\n"), 0600); err != nil {
		t.Fatal(err)
	}
	request := filepath.Join(dir, "request.json")
	if err := os.WriteFile(request, []byte(`{"op":"docs_search","query":"needle","flags":{"json":"","limit":"1"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	helpers, err := os.ReadFile(f.helperLog)
	if err != nil {
		t.Fatal(err)
	}
	home, work := docsState(t, f.home), docsState(t, f.work)
	count := f.requests.Load()
	for _, token := range []string{"", "dummy-docs-test-token"} {
		t.Setenv("GITHUB_TOKEN", token)
		resp := f.tool(t, policy, request)
		if resp.OK || resp.ExitCode == 0 || resp.Refused != "" || len(resp.Argv) == 0 {
			t.Fatalf("child refusal envelope: %+v", resp)
		}
		assertDocsRefusal(t, cliResult{exitCode: resp.ExitCode, stderr: resp.Stderr})
		f.assertUntouched(t, home, work, count, string(helpers))
	}
	corpus := filepath.Join(f.work, "design_docs")
	if err := os.Mkdir(corpus, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "needle.md"), []byte("# Needle contracts\nneedle typed contracts and effects\n"), 0600); err != nil {
		t.Fatal(err)
	}
	home, work = docsState(t, f.home), docsState(t, f.work)
	resp := f.tool(t, policy, request)
	if !resp.OK || resp.ExitCode != 0 || resp.Refused != "" {
		t.Fatalf("local child envelope: %+v", resp)
	}
	docsStableResults(t, cliResult{stdout: resp.Stdout, stderr: resp.Stderr, exitCode: resp.ExitCode})
	f.assertUntouched(t, home, work, count, string(helpers))
	for _, flag := range []string{"neural", "path"} {
		body := `{"op":"docs_search","query":"needle","flags":{"` + flag + `":"true"}}`
		if err := os.WriteFile(request, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		resp = f.tool(t, policy, request)
		if resp.OK || !strings.Contains(resp.Refused, "does not admit flag") || len(resp.Argv) != 0 {
			t.Fatalf("flag schema: %+v", resp)
		}
	}
}

func (f *docsConfinementFixture) tool(t *testing.T, policy, request string) policytool.Response {
	t.Helper()
	r := f.run(t, "policy-tool", "--policy", policy, "--request-file", request)
	var resp policytool.Response
	if r.exitCode != 0 || json.Unmarshal([]byte(r.stdout), &resp) != nil {
		t.Fatalf("tool process must produce existing JSON at exit0: %+v", r)
	}
	return resp
}
