package main

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

type docsConfinementFixture struct {
	bin, home, work, helperLog string
	requests                   atomic.Int32
}

// Only loopback is reachable: the proxy always refuses and never forwards.
func newDocsConfinementFixture(t *testing.T) *docsConfinementFixture {
	t.Helper()
	f := &docsConfinementFixture{bin: cliTestBin(t), home: t.TempDir(), work: t.TempDir(), helperLog: filepath.Join(t.TempDir(), "helpers")}
	helperDir := t.TempDir()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	helper := filepath.Join(helperDir, "gh"+suffix)
	out, stderr, code := testutil.RunBounded(t, "", 60*time.Second, "go", "build", "-o", helper, "../../internal/docsearch/github/testdata/credentialhelper")
	if code != 0 {
		t.Fatalf("helper build: %s%s", out, stderr)
	}
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helperDir, "git"+suffix), data, 0700); err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		http.Error(w, "DOCS_TEST_PROXY_DENIED", http.StatusForbidden)
	}))
	t.Cleanup(proxy.Close)
	clearDocsExportEnv(t)
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		t.Setenv(key, proxy.URL)
	}
	for _, key := range []string{"NO_PROXY", "no_proxy", "GITHUB_TOKEN", "AILANG_AGENT_POLICY", "AILANG_NO_GITHUB_SEARCH"} {
		t.Setenv(key, "")
	}
	testutil.SetHomeDir(t, f.home)
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "XDG_DATA_HOME"} {
		t.Setenv(key, filepath.Join(f.home, key))
	}
	t.Setenv("PATH", helperDir)
	t.Setenv("DOCSEARCH_HELPER_LOG", f.helperLog)
	return f
}

func clearDocsExportEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "OTEL_") || strings.HasPrefix(upper, "OTLP_") || strings.HasPrefix(upper, "GOOGLE_") || strings.HasPrefix(upper, "GCLOUD_") || strings.HasPrefix(upper, "GCP_") || strings.Contains(upper, "PROXY") || strings.HasPrefix(upper, "AILANG_TRACE") {
			t.Setenv(key, "")
		}
	}
}

func docsState(t *testing.T, root string) map[string]string {
	t.Helper()
	state := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += fmt.Sprintf(":%x", sha256.Sum256(data))
		}
		state[filepath.ToSlash(rel)] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func (f *docsConfinementFixture) run(t *testing.T, args ...string) cliResult {
	t.Helper()
	r := runCLIIn(t, f.bin, f.home, f.work, args...)
	if r.timedOut {
		t.Fatal("docs CLI exceeded existing30s bound")
	}
	return r
}

func (f *docsConfinementFixture) assertUntouched(t *testing.T, home, work map[string]string, requests int32, helpers string) {
	t.Helper()
	data, err := os.ReadFile(f.helperLog)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if f.requests.Load() != requests || string(data) != helpers {
		t.Fatalf("side effect: requests%d helpers%q", f.requests.Load(), data)
	}
	if !reflect.DeepEqual(home, docsState(t, f.home)) || !reflect.DeepEqual(work, docsState(t, f.work)) {
		t.Fatal("private HOME/XDG/corpus changed")
	}
}

func (f *docsConfinementFixture) proveInstruments(t *testing.T) {
	t.Helper()
	r := f.run(t, "docs", "search", "needle")
	if r.exitCode == 0 || !strings.Contains(r.stderr, "requires authentication") {
		t.Fatalf("auth positive: %+v", r)
	}
	data, err := os.ReadFile(f.helperLog)
	if err != nil || !strings.Contains(string(data), "gh\n") {
		t.Fatalf("gh instrument: %q %v", data, err)
	}
	t.Setenv("GITHUB_TOKEN", "dummy-docs-test-token")
	r = f.run(t, "docs", "search", "needle")
	data, err = os.ReadFile(f.helperLog)
	if r.exitCode == 0 || f.requests.Load() != 1 || err != nil || !strings.Contains(string(data), "git\n") {
		t.Fatalf("git/proxy instruments: %+v requests%d helpers%q %v", r, f.requests.Load(), data, err)
	}
}
