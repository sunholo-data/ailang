package config

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// clearCloudEnv isolates a test from the machine: every variable this package
// reads is unset, the home is a fixture, and the metadata cache is empty.
func clearCloudEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	for _, v := range []string{EnvCloudProject, EnvGoogleCloudProject, EnvCloudRegion,
		EnvGoogleCloudRegion, EnvNoMetadata, EnvConfigFile, EnvStrict} {
		t.Setenv(v, "")
	}
	// Off GCE the metadata host does not resolve; do not depend on that, point
	// the lookup at a server that answers 404 unless a test says otherwise.
	srv := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(srv.Close)
	prev := metadataBaseURL
	metadataBaseURL = srv.URL
	t.Cleanup(func() { metadataBaseURL = prev })
	resetMetadataForTest()
	t.Cleanup(resetMetadataForTest)
	return home
}

func TestCloudProjectPrecedence(t *testing.T) {
	home := clearCloudEnv(t)
	ctx := context.Background()

	if _, err := CloudProject(ctx); !errors.Is(err, ErrNoCloudProject) {
		t.Fatalf("empty env: err = %v, want ErrNoCloudProject", err)
	}

	// 3. config file
	if err := os.MkdirAll(filepath.Join(home, ".ailang"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ailang", "config.yaml"),
		[]byte("pubsub:\n  enabled: true\n  project_id: from-yaml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p, src, err := CloudProjectSource(ctx)
	if err != nil || p != "from-yaml" || src != SourceConfigFile {
		t.Fatalf("yaml: got %q via %q, err %v", p, src, err)
	}

	// 2. GOOGLE_CLOUD_PROJECT beats the file
	t.Setenv(EnvGoogleCloudProject, "from-google")
	p, src, err = CloudProjectSource(ctx)
	if err != nil || p != "from-google" || src != SourceGoogleEnv {
		t.Fatalf("google env: got %q via %q, err %v", p, src, err)
	}

	// 1. AILANG_CLOUD_PROJECT beats everything
	t.Setenv(EnvCloudProject, " from-ailang ")
	p, src, err = CloudProjectSource(ctx)
	if err != nil || p != "from-ailang" || src != SourceAilangEnv {
		t.Fatalf("ailang env: got %q via %q, err %v", p, src, err)
	}
}

func TestCloudProjectHonoursAilangConfigOverride(t *testing.T) {
	clearCloudEnv(t)
	alt := filepath.Join(t.TempDir(), "alt.yaml")
	if err := os.WriteFile(alt, []byte("pubsub:\n  project_id: from-alt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvConfigFile, alt)
	p, err := CloudProject(context.Background())
	if err != nil || p != "from-alt" {
		t.Fatalf("got %q, %v", p, err)
	}
}

func TestCloudProjectMetadataIsLastAndCached(t *testing.T) {
	clearCloudEnv(t)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path != "/computeMetadata/v1/project/project-id" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte("from-metadata\n"))
	}))
	t.Cleanup(srv.Close)
	metadataBaseURL = srv.URL
	resetMetadataForTest()

	ctx := context.Background()
	p, src, err := CloudProjectSource(ctx)
	if err != nil || p != "from-metadata" || src != SourceMetadata {
		t.Fatalf("metadata: got %q via %q, err %v", p, src, err)
	}
	if _, err := CloudProject(ctx); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("metadata server called %d times; the outcome must be cached per process", calls)
	}

	// AILANG_NO_METADATA short-circuits even a cached answer's source.
	t.Setenv(EnvNoMetadata, "1")
	if _, err := CloudProject(ctx); !errors.Is(err, ErrNoCloudProject) {
		t.Fatalf("with %s set: err = %v, want ErrNoCloudProject", EnvNoMetadata, err)
	}
}

func TestCloudProjectNeverDefaults(t *testing.T) {
	clearCloudEnv(t)
	t.Setenv(EnvNoMetadata, "1")
	p, err := CloudProject(context.Background())
	if p != "" || !errors.Is(err, ErrNoCloudProject) {
		t.Fatalf("got %q, %v; the resolver must not invent a project", p, err)
	}
	if !strings.Contains(err.Error(), EnvCloudProject) {
		t.Fatalf("error must name what to set: %v", err)
	}
}

func TestRegionPrecedenceAndDeprecatedDefault(t *testing.T) {
	clearCloudEnv(t)
	resetWarningsForTest()
	var buf bytes.Buffer
	warnOutput = &buf
	t.Cleanup(func() { warnOutput = os.Stderr })

	r, err := Region()
	if err != nil || r != "europe-west1" {
		t.Fatalf("default: got %q, %v", r, err)
	}
	if !strings.Contains(buf.String(), EnvCloudRegion+" is unset; using deprecated default europe-west1") {
		t.Fatalf("expected a deprecation warning, got %q", buf.String())
	}

	t.Setenv(EnvGoogleCloudRegion, "us-central1")
	if r, _ = Region(); r != "us-central1" {
		t.Fatalf("google region: %q", r)
	}
	t.Setenv(EnvCloudRegion, "europe-west3")
	if r, _ = Region(); r != "europe-west3" {
		t.Fatalf("ailang region: %q", r)
	}

	t.Setenv(EnvCloudRegion, "")
	t.Setenv(EnvGoogleCloudRegion, "")
	t.Setenv(EnvStrict, "1")
	if r, err = Region(); r != "" || !errors.Is(err, ErrDeprecatedDefault) {
		t.Fatalf("strict: got %q, %v", r, err)
	}
}

func TestDeprecatedDefaultWarnsOncePerName(t *testing.T) {
	clearCloudEnv(t)
	resetWarningsForTest()
	t.Cleanup(resetWarningsForTest)
	var buf bytes.Buffer
	warnOutput = &buf
	t.Cleanup(func() { warnOutput = os.Stderr })

	for i := 0; i < 3; i++ {
		v, err := DeprecatedDefault("AILANG_CLOUD_PROJECT", "ailang-multivac")
		if err != nil || v != "ailang-multivac" {
			t.Fatalf("call %d: got %q, %v", i, v, err)
		}
	}
	if v, _ := DeprecatedDefault("AILANG_OTHER", "x"); v != "x" {
		t.Fatalf("other name: %q", v)
	}

	out := buf.String()
	want := "AILANG_CLOUD_PROJECT is unset; using deprecated default ailang-multivac. Set AILANG_CLOUD_PROJECT. v1.0.0 will refuse to start without it.\n"
	if n := strings.Count(out, want); n != 1 {
		t.Fatalf("project warning printed %d times, want exactly once:\n%s", n, out)
	}
	if n := strings.Count(out, "AILANG_OTHER is unset"); n != 1 {
		t.Fatalf("other warning printed %d times, want once:\n%s", n, out)
	}
}

func TestDeprecatedDefaultStrictRefuses(t *testing.T) {
	clearCloudEnv(t)
	resetWarningsForTest()
	var buf bytes.Buffer
	warnOutput = &buf
	t.Cleanup(func() { warnOutput = os.Stderr })

	for _, on := range []string{"1", "true"} {
		t.Setenv(EnvStrict, on)
		v, err := DeprecatedDefault("AILANG_CLOUD_PROJECT", "ailang-multivac")
		if v != "" {
			t.Fatalf("%s=%s: returned %q, must return empty", EnvStrict, on, v)
		}
		if !errors.Is(err, ErrDeprecatedDefault) {
			t.Fatalf("%s=%s: err = %v, want ErrDeprecatedDefault", EnvStrict, on, err)
		}
		if !strings.Contains(err.Error(), "AILANG_CLOUD_PROJECT") {
			t.Fatalf("error must name the variable: %v", err)
		}
	}
	if buf.Len() != 0 {
		t.Fatalf("strict mode must not also warn: %q", buf.String())
	}
	t.Setenv(EnvStrict, "0")
	if Strict() {
		t.Fatal("0 must not be strict")
	}
}
