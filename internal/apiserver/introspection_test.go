package apiserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-SERVEAPI-OPERATOR-SURFACE D1/D2: introspection lists only the authorized
// surface, and --no-introspection removes it.

const surfaceModule = `module talk

-- Health for the proxy.
@route("GET", "/healthz")
export func healthz() -> string = "ok"

-- A helper that --routes-only must hide everywhere, docs included.
export func secretHelper(x: int) -> int = x + 1
`

// newSurfaceServer writes files (relative path -> source) into a temp
// project, loads the project directory and serves buildRoutes.
func newSurfaceServer(t *testing.T, cfg Config, files map[string]string) (*Server, *httptest.Server) {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(repoRoot, "std"))
	root := t.TempDir()
	for rel, src := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if cfg.Port == "" {
		cfg.Port = "0"
	}
	srv := New(root, cfg)
	t.Cleanup(func() { _ = srv.Close() })
	if err := srv.LoadModules([]string{root}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	ts := httptest.NewServer(srv.buildRoutes())
	t.Cleanup(ts.Close)
	return srv, ts
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

var introspectionPaths = []string{
	"/api/_meta/modules",
	"/api/_meta/modules/talk",
	"/api/_meta/openapi.json",
	"/api/_meta/docs",
	"/api/_meta/redoc",
	"/api/_health",
}

func TestIntrospection_RoutesOnlyListsOnlyRoutes(t *testing.T) {
	_, ts := newSurfaceServer(t, Config{RoutesOnly: true}, map[string]string{"talk.ail": surfaceModule})

	for _, p := range []string{"/api/_meta/modules", "/api/_meta/modules/talk"} {
		code, body := get(t, ts.URL+p)
		if code != http.StatusOK {
			t.Fatalf("%s: HTTP %d", p, code)
		}
		if !strings.Contains(body, "healthz") {
			t.Fatalf("instrument check: %s should list the @route export:\n%s", p, body)
		}
		if strings.Contains(body, "secretHelper") || strings.Contains(body, "must hide") {
			t.Fatalf("%s leaks an export --routes-only hides:\n%s", p, body)
		}
	}
}

func TestIntrospection_DefaultStillListsEverything(t *testing.T) {
	_, ts := newSurfaceServer(t, Config{}, map[string]string{"talk.ail": surfaceModule})
	code, body := get(t, ts.URL+"/api/_meta/modules")
	if code != http.StatusOK || !strings.Contains(body, "secretHelper") {
		t.Fatalf("without --routes-only every exposed export is listed; HTTP %d:\n%s", code, body)
	}
	var resp ModulesListResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil || resp.Count != 1 {
		t.Fatalf("want exactly the one local module, got %d (%v):\n%s", resp.Count, err, body)
	}
}

func TestIntrospection_NoIntrospectionRemovesMetaAndHealth(t *testing.T) {
	_, ts := newSurfaceServer(t, Config{RoutesOnly: true, NoIntrospection: true}, map[string]string{"talk.ail": surfaceModule})

	// Positive control: the route still answers.
	if code, body := get(t, ts.URL+"/healthz"); code != http.StatusOK || !strings.Contains(body, "ok") {
		t.Fatalf("route /healthz: HTTP %d %s", code, body)
	}
	for _, p := range introspectionPaths {
		code, body := get(t, ts.URL+p)
		if code != http.StatusNotFound {
			t.Errorf("%s: HTTP %d, want 404 with --no-introspection:\n%.200s", p, code, body)
		}
		if strings.Contains(body, "healthz") || strings.Contains(body, "googleapis") {
			t.Errorf("%s: body still carries introspection content:\n%.200s", p, body)
		}
	}
}

func TestIntrospection_ReservedPathsStayReserved(t *testing.T) {
	// With introspection off, an app still cannot claim /api/_health: the
	// flag removes surface, it does not free paths.
	mod := "module talk\n\n@route(\"GET\", \"/api/_health\")\nexport func mine() -> string = \"mine\"\n"
	_, ts := newSurfaceServer(t, Config{NoIntrospection: true}, map[string]string{"talk.ail": mod})
	code, body := get(t, ts.URL+"/api/_health")
	if strings.Contains(body, "mine") {
		t.Fatalf("a @route claimed a reserved introspection path (HTTP %d): %s", code, body)
	}
}
