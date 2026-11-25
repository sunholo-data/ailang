package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

// docsToolCases is one valid call per tool in mcp_tools/. The Anthropic and
// OpenAI directories require every listed tool to succeed on valid input, and
// six of these failed on every call for months (#2271) while
// `ailang mcp check` passed, because that checks the listing, not the calls.
// A tool missing from this table fails the test, so a new tool needs a case.
var docsToolCases = map[string]string{
	"ailang_versions":       `{}`,
	"docs_search":           `{"query": "effects", "forVersion": ""}`,
	"docs_nav":              `{"forVersion": ""}`,
	"stdlib_modules":        `{"forVersion": ""}`,
	"stdlib_module":         `{"name": "std/list", "forVersion": ""}`,
	"stdlib_search":         `{"query": "map", "forVersion": ""}`,
	"effects_catalog":       `{"forVersion": ""}`,
	"limitations_list":      `{"category": "", "forVersion": ""}`,
	"examples_list":         `{"features": "", "status": "", "forVersion": ""}`,
	"example_get":           `{"path": "runnable/hello.ail", "forVersion": ""}`,
	"example_for_concept":   `{"concept": "effects", "forVersion": ""}`,
	"prompt_get":            `{"forVersion": "", "kind": ""}`,
	"changelog_for_version": `{"version": "VERSION"}`,
	"install_guide":         `{"harness": "", "platform": ""}`,
	"onboarding_guide":      `{"role": ""}`,
	"roadmap":               `{}`,
	"design_docs_list":      `{"state": "", "version": ""}`,
	"design_doc":            `{"slug": "SLUG"}`,
	"benchmarks_list":       `{"sinceVersion": ""}`,
	"benchmarks_for_model":  `{"model": "MODEL", "version": ""}`,
	"benchmarks_compare":    `{"modelA": "MODEL", "modelB": "MODEL"}`,
}

// docsToolRefusals are calls that must come back as tool errors (isError),
// not as a successful result carrying an error message.
var docsToolRefusals = map[string]string{
	"design_doc":         `{"slug": "no-such-design-doc"}`,
	"example_get":        `{"path": "../go.mod", "forVersion": ""}`,
	"prompt_get":         `{"forVersion": "", "kind": "compact"}`,
	"benchmarks_compare": `{"modelA": "no-such-model", "modelB": "MODEL"}`,
	"examples_list":      `{"features": "", "status": "", "forVersion": "not-a-version"}`,
	"design_docs_list":   `{"state": "drafted", "version": ""}`,
	"onboarding_guide":   `{"role": "manager"}`,
}

// fakeRepo serves the working tree as raw.githubusercontent.com serves the
// repository: /<ref>/<path>. The ref is recorded and otherwise ignored.
func fakeRepo(t *testing.T, root string) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref, rel, ok := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if !ok || rel == "" { // not a file fetch: a stray local probe of a reused port hit "/" once
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		seen = append(seen, ref+" "+rel)
		mu.Unlock()
		if strings.Contains(rel, "..") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, filepath.FromSlash(rel)))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), seen...) }
}

func TestDocsTools_EveryToolSucceeds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tools/build-snapshot links versioned/latest with a symlink")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	snap := t.TempDir()
	build := exec.Command("go", "run", "./tools/build-snapshot", "--repo", root, "--out", snap)
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build-snapshot: %v\n%s", err, out)
	}
	repo, requests := fakeRepo(t, root)
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	t.Setenv("AILANG_RELAX_MODULES", "1")
	t.Setenv("MCP_SNAPSHOT_DIR", snap)
	t.Setenv("MCP_SOURCE_BASE", repo.URL)

	eff := effects.NewEffContext(nil)
	for _, c := range []string{"FS", "Env", "Net"} {
		eff.Grant(effects.NewCapability(c))
	}
	eff.Net.AllowHTTP, eff.Net.AllowLocalhost = true, true // the fake repository
	srv := New(root, Config{Port: "0", MCP: true, RoutesOnly: true, EffCtx: eff})
	t.Cleanup(func() { _ = srv.Close() })
	t.Chdir(root) // imports resolve from the working directory, as in the image (/srv)
	if err := srv.LoadModules([]string{"mcp_tools/"}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}

	version := strings.TrimPrefix(strings.TrimSpace(mustRead(t, filepath.Join(root, "std", "VERSION"))), "v")
	fill := strings.NewReplacer("VERSION", version, "SLUG", "m-agent-mcp-website", "MODEL", anyModel(t, snap))
	tools := map[string]struct {
		mod string
		exp ExportInfo
	}{}
	for path, mod := range srv.modules {
		for _, e := range mod.Exports {
			if e.MCPName != "" && !e.IsNoMCP {
				tools[e.MCPName] = struct {
					mod string
					exp ExportInfo
				}{path, e}
			}
		}
	}
	ms := NewMCPServer(srv)
	call := func(name, args string) (string, bool) {
		t.Helper()
		tool := tools[name]
		res, err := ms.makeToolHandler(tool.mod, tool.exp)(context.Background(), mcpCallReq(name, fill.Replace(args)))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return mcpResultText(t, res), res.IsError
	}

	for name := range tools {
		if name == "submit_feedback" {
			continue // posts to the feedback inbox; covered by its own tests
		}
		args, ok := docsToolCases[name]
		if !ok {
			t.Errorf("tool %s has no case in docsToolCases: add a valid call", name)
			continue
		}
		text, isErr := call(name, args)
		if isErr {
			t.Errorf("%s(%s) failed: %s", name, args, clipText(text, 300))
			continue
		}
		for _, bad := range []string{`"snapshot_read_failed"`, `"unknown_version"`, `"missing_field"`, `"examples": []`} {
			if strings.Contains(text, bad) {
				t.Errorf("%s(%s) succeeded with %s in it: %s", name, args, bad, clipText(text, 300))
			}
		}
	}
	for name := range docsToolCases {
		if _, ok := tools[name]; !ok {
			t.Errorf("docsToolCases names %s, which mcp_tools/ no longer has", name)
		}
	}
	for name, args := range docsToolRefusals {
		if text, isErr := call(name, args); !isErr {
			t.Errorf("%s(%s) should be a tool error, got success: %s", name, args, clipText(text, 300))
		}
	}

	// Filters filter (#2271 X2: design_docs_list returned every doc whatever
	// state said), and the guide names only tools that exist.
	var all, rejected struct {
		Total int `json:"total"`
		Docs  []struct {
			State string `json:"state"`
		} `json:"docs"`
	}
	for _, c := range []struct {
		args string
		into any
	}{{`{"state": "", "version": ""}`, &all}, {`{"state": "rejected", "version": ""}`, &rejected}} {
		text, _ := call("design_docs_list", c.args)
		if err := json.Unmarshal([]byte(text), c.into); err != nil {
			t.Fatalf("design_docs_list(%s): %v: %s", c.args, err, clipText(text, 300))
		}
	}
	if rejected.Total == 0 || rejected.Total >= all.Total {
		t.Errorf("design_docs_list state=rejected matched %d of %d docs", rejected.Total, all.Total)
	}
	for _, d := range rejected.Docs {
		if d.State != "rejected" {
			t.Errorf("design_docs_list state=rejected listed a %q doc", d.State)
			break
		}
	}
	guide, _ := call("onboarding_guide", `{"role": ""}`)
	for _, m := range regexp.MustCompile("`([a-z_]+)\\(").FindAllStringSubmatch(guide, -1) {
		if _, ok := tools[m[1]]; !ok && m[1] != "submit_feedback" { // the server's own tool, not in mcp_tools/
			t.Errorf("onboarding_guide tells the agent to call %s, which is not a tool", m[1])
		}
	}
	if agent, _ := call("onboarding_guide", `{"role": "agent"}`); strings.Contains(agent, `"developer"`) {
		t.Errorf("onboarding_guide role=agent returned the developer guide too")
	}

	// The live tools went to the release tag or dev, never anywhere else.
	for _, r := range requests() {
		ref, _, _ := strings.Cut(r, " ")
		if ref != "dev" && ref != "v"+version {
			t.Errorf("fetched at ref %q (%s); want dev or v%s", ref, r, version)
		}
	}
	if len(requests()) == 0 {
		t.Error("no live fetch reached the fake repository")
	}
}

func anyModel(t *testing.T, snap string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(snap, "unscoped", "benchmarks", "by_model"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("snapshot has no per-model benchmarks: %v", err)
	}
	return strings.TrimSuffix(entries[0].Name(), ".json")
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
