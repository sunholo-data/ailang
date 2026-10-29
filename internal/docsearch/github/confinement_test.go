package github

import (
	"context"
	"crypto/sha256"
	"errors"
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

	"github.com/sunholo-data/ailang/internal/docsearch"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func requireConfinement(t *testing.T, err error, authority pkg.Authority) {
	t.Helper()
	var ce *pkg.ConfinedError
	if !errors.As(err, &ce) || ce.Authority != authority || !strings.Contains(ce.Remedy, "--path") {
		t.Fatalf("want typed %s docs refusal, got %v", authority, err)
	}
}

func privateDocsHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv("AILANG_AGENT_POLICY", "")
	for _, key := range []string{"XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME"} {
		t.Setenv(key, filepath.Join(home, key))
	}
	return home
}

func homeSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
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
		out[filepath.ToSlash(rel)] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDocsConfinementConstructorBeforeHelpers(t *testing.T) {
	privateDocsHome(t)
	dir := t.TempDir()
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	helper := filepath.Join(dir, "gh"+suffix)
	out, stderr, code := testutil.RunBounded(t, "", 60*time.Second, "go", "build", "-o", helper, "./testdata/credentialhelper")
	if code != 0 {
		t.Fatalf("helper build: %s%s", out, stderr)
	}
	data, err := os.ReadFile(helper)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "git"+suffix), data, 0700); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "helpers")
	t.Setenv("DOCSEARCH_HELPER_LOG", log)
	t.Setenv("PATH", dir)
	t.Setenv("GITHUB_TOKEN", "")
	if _, err := NewBackend(context.Background()); err == nil {
		t.Fatal("empty fake gh must fail auth")
	}
	t.Setenv("GITHUB_TOKEN", "dummy")
	if _, err := NewBackend(context.Background()); err != nil {
		t.Fatal(err)
	}
	calls, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(calls), "gh\n") || !strings.Contains(string(calls), "git\n") {
		t.Fatalf("inactive helpers: %q", calls)
	}
	if err := os.WriteFile(log, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_AGENT_POLICY", "private-policy")
	for _, token := range []string{"", "dummy"} {
		t.Setenv("GITHUB_TOKEN", token)
		_, err := NewBackend(context.Background())
		requireConfinement(t, err, pkg.AuthNetwork)
	}
	calls, err = os.ReadFile(log)
	if err != nil || len(calls) != 0 {
		t.Fatalf("confined helper work: %q %v", calls, err)
	}
}

func TestDocsConfinementSearchAndCache(t *testing.T) {
	home := privateDocsHome(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"total_count":1,"items":[{"path":"guide.md","html_url":"https://example.invalid/guide"}]}`)
	}))
	defer srv.Close()
	b := NewBackendForTest(srv.URL, "acme/docs", "dummy", srv.Client())
	opts := docOptions("cached", "")
	got, _, err := b.Search(context.Background(), opts)
	if err != nil || len(got) != 1 || calls.Load() != 1 {
		t.Fatalf("HTTP positive: %v %v calls%d", got, err, calls.Load())
	}
	if _, _, err := b.Search(context.Background(), opts); err != nil || calls.Load() != 1 {
		t.Fatalf("cache positive: %v calls%d", err, calls.Load())
	}
	before := homeSnapshot(t, home)
	t.Setenv("AILANG_AGENT_POLICY", "private-policy")
	for _, backend := range []*Backend{b, {}, {Repo: "invalid", Token: ""}, NewBackendForTest(srv.URL, "acme/docs", "dummy", srv.Client())} {
		_, _, err := backend.Search(context.Background(), opts)
		requireConfinement(t, err, pkg.AuthNetwork)
	}
	err = writeCache("acme/docs", "new", "", 10, nil, docsearch.SearchStats{}, time.Now())
	requireConfinement(t, err, pkg.AuthHomeWrite)
	if calls.Load() != 1 || !reflect.DeepEqual(before, homeSnapshot(t, home)) {
		t.Fatal("confined request or state mutation")
	}
	testutil.SetHomeDir(t, "")
	err = writeCache("acme/docs", "new", "", 10, nil, docsearch.SearchStats{}, time.Now())
	requireConfinement(t, err, pkg.AuthHomeWrite)
}

func TestDocsConfinementHTTPAndWriteTransitions(t *testing.T) {
	for _, phase := range []string{"pagination", "cache-write", "ordinary-write-error"} {
		t.Run(phase, func(t *testing.T) {
			home := privateDocsHome(t)
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if phase != "ordinary-write-error" {
					os.Setenv("AILANG_AGENT_POLICY", "late-policy")
				}
				fmt.Fprint(w, `{"total_count":101,"items":[`)
				count := 1
				if phase == "pagination" {
					count = 100
				}
				for i := 0; i < count; i++ {
					if i > 0 {
						fmt.Fprint(w, ",")
					}
					fmt.Fprintf(w, `{"path":"%d.md","html_url":"https://example.invalid/%d"}`, i, i)
				}
				fmt.Fprint(w, `]}`)
			}))
			defer srv.Close()
			if phase == "ordinary-write-error" {
				blocked := filepath.Join(home, ".ailang")
				if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := homeSnapshot(t, home)
			b := NewBackendForTest(srv.URL, "acme/docs", "dummy", srv.Client())
			opts := docOptions(phase, "")
			if phase == "pagination" {
				opts.Limit = 101
			}
			_, _, err := b.Search(context.Background(), opts)
			switch phase {
			case "pagination":
				requireConfinement(t, err, pkg.AuthNetwork)
			case "cache-write":
				requireConfinement(t, err, pkg.AuthHomeWrite)
			default:
				if err != nil {
					t.Fatalf("optional cache failure: %v", err)
				}
			}
			if calls.Load() != 1 || !reflect.DeepEqual(before, homeSnapshot(t, home)) {
				t.Fatalf("requests=%d or state changed", calls.Load())
			}
		})
	}
}

func TestDocsUnconfinedPagination(t *testing.T) {
	privateDocsHome(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		fmt.Fprint(w, `{"total_count":101,"items":[`)
		count := 100
		if calls.Load() == 2 {
			count = 1
		}
		for i := 0; i < count; i++ {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"path":"%d.md"}`, i)
		}
		fmt.Fprint(w, `]}`)
	}))
	defer srv.Close()
	b := NewBackendForTest(srv.URL, "acme/docs", "dummy", srv.Client())
	opts := docOptions("pages", "")
	opts.Limit = 101
	got, _, err := b.Search(context.Background(), opts)
	if err != nil || len(got) != 101 || calls.Load() != 2 {
		t.Fatalf("pagination positive: results%d calls%d err%v", len(got), calls.Load(), err)
	}
}
