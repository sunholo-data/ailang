package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func TestRunEmbedderExplicitAndDerived(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_EMBED_PROVIDER", "")
	for _, bound := range []string{"anthropic", "openrouter", "custom"} {
		e, err := resolveRunEmbedder(aiSetup{}, bound)
		if err != nil || e != nil {
			t.Fatalf("%s: %v %v", bound, e, err)
		}
	}
	t.Setenv("AILANG_EMBED_PROVIDER", "invalid")
	if _, err := resolveRunEmbedder(aiSetup{}, "openai"); err == nil {
		t.Fatal("invalid explicit provider accepted")
	}
	t.Setenv("AILANG_EMBED_PROVIDER", "none")
	if e, err := resolveRunEmbedder(aiSetup{}, "google"); err != nil || e != nil {
		t.Fatalf("explicit disable ignored: %v %v", e, err)
	}
	t.Setenv("AILANG_EMBED_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "fake")
	t.Setenv("AILANG_EMBED_OPENAI_MODEL", "text-embedding-3-small")
	e, err := resolveRunEmbedder(aiSetup{}, "google")
	if err != nil || e.ModelName() != "openai:text-embedding-3-small" {
		t.Fatalf("%v %v", e, err)
	}
}
func TestRunEmbeddingPersistentAndCapability(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	path := filepath.Join(t.TempDir(), "cache.db")
	denied := effects.NewEffContext(nil)
	if err := setupRunEmbedder(denied, aiSetup{Stub: true, EmbedCacheDB: path}, ""); err == nil {
		t.Fatal("missing capability accepted")
	}
	first := effects.NewEffContext(nil)
	first.Grant(effects.NewCapability("AI"))
	first.AI = effects.NewAIContext(effects.NewStubAIHandler())
	if err := setupRunEmbedder(first, aiSetup{Stub: true, EmbedCacheDB: path}, ""); err != nil {
		t.Fatal(err)
	}
	v, err := first.AI.Embed("same text")
	if err != nil {
		t.Fatal(err)
	}
	first.AI.Close()
	second := effects.NewEffContext(nil)
	second.Grant(effects.NewCapability("AI"))
	second.AI = effects.NewAIContext(effects.NewStubAIHandler())
	if err := setupRunEmbedder(second, aiSetup{Stub: true, EmbedCacheDB: path}, ""); err != nil {
		t.Fatal(err)
	}
	defer second.AI.Close()
	w, err := second.AI.Embed("same text")
	if err != nil || !reflect.DeepEqual(v, w) {
		t.Fatalf("%v %v", w, err)
	}
}

type embedTransport func(*http.Request) (*http.Response, error)

func (f embedTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestRunCloudEmbeddersWithoutOllama(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))
	t.Setenv("GOOGLE_API_KEY", "fake")
	t.Setenv("OPENAI_API_KEY", "fake")
	t.Setenv("AILANG_EMBED_PROVIDER", "")
	t.Setenv("AILANG_EMBED_GEMINI_MODEL", "text-embedding-004")
	t.Setenv("AILANG_EMBED_OPENAI_MODEL", "text-embedding-3-small")
	previous := http.DefaultTransport
	defer func() { http.DefaultTransport = previous }()
	count := 0
	http.DefaultTransport = embedTransport(func(r *http.Request) (*http.Response, error) {
		count++
		body := `{"data":[{"embedding":[1,2]}]}`
		switch r.URL.Host {
		case "generativelanguage.googleapis.com":
			body = `{"embedding":{"values":[1,2]}}`
		case "api.openai.com":
		default:
			t.Fatalf("unexpected provider host %s", r.URL.Host)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	for _, provider := range []string{"google", "openai"} {
		e, err := resolveRunEmbedder(aiSetup{}, provider)
		if err != nil {
			t.Fatal(err)
		}
		v, err := e.Embed("sample")
		if err != nil || len(v) != 2 {
			t.Fatalf("%s: %v %v", provider, v, err)
		}
	}
	if count != 2 {
		t.Fatalf("calls=%d", count)
	}
}
func TestUnsupportedEmbedDoesNotInitializeCompletion(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))
	t.Setenv("AILANG_EMBED_PROVIDER", "")
	initialized := false
	ctx := effects.NewEffContext(nil)
	ctx.AI = effects.NewLazyAIContext(func() (*effects.AIContext, error) {
		initialized = true
		return nil, fmt.Errorf("completion credentials unavailable")
	})
	if err := setupRunEmbedder(ctx, aiSetup{}, "openrouter"); err != nil {
		t.Fatal(err)
	}
	_, err := ctx.AI.Embed("hello")
	if initialized || !strings.Contains(err.Error(), "ProviderNotFound") {
		t.Fatalf("initialized=%v err=%v", initialized, err)
	}
}

func TestLazyEmbeddingFreezesCredentialsAndDefersKeyFile(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))
	t.Setenv("AILANG_EMBED_PROVIDER", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("AILANG_AI_KEY_FILE", "")
	ctx := effects.NewEffContext(nil)
	ctx.AI = effects.NewAIContext(effects.NewStubAIHandler())
	if err := setupRunEmbedder(ctx, aiSetup{KeyFile: filepath.Join(t.TempDir(), "missing.key")}, "openai", true); err != nil {
		t.Fatalf("eager credential access: %v", err)
	}
	if _, err := ctx.AI.Embed("hello"); err == nil {
		t.Fatal("missing key file ignored")
	}
	frozen := effects.NewEffContext(nil)
	frozen.AI = effects.NewAIContext(effects.NewStubAIHandler())
	if err := setupRunEmbedder(frozen, aiSetup{}, "openai", true); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENAI_API_KEY", "too-late")
	if _, err := frozen.AI.Embed("hello"); err == nil {
		t.Fatal("late environment changed frozen credentials")
	}
}

func TestRunEmbeddingConfigErrors(t *testing.T) {
	t.Setenv("AILANG_EMBED_PROVIDER", "")
	t.Setenv("AILANG_AI_KEY_FILE", "")
	for _, body := range []string{"embeddings: {provider: bogus}", "embeddings: {openai: {model: x}}", "embeddings: {provider: openai, openai: {timeout: bad}}", "embeddings: {provider: openai, openai: {timeout: '-1s'}}"} {
		path := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AILANG_CONFIG", path)
		if _, err := prepareRunEmbeddingPlan(aiSetup{}, "google"); err == nil {
			t.Fatalf("invalid config accepted: %s", body)
		}
	}
}
func TestRunEmbeddingCacheFlagOverridesEnvironment(t *testing.T) {
	t.Setenv("AILANG_EMBED_CACHE_DB", filepath.Join(t.TempDir(), "env.db"))
	explicit := filepath.Join(t.TempDir(), "flag.db")
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("AI"))
	ctx.AI = effects.NewAIContext(effects.NewStubAIHandler())
	if err := setupRunEmbedder(ctx, aiSetup{Stub: true, EmbedCacheDB: explicit}, ""); err != nil {
		t.Fatal(err)
	}
	defer ctx.AI.Close()
	if _, err := os.Stat(explicit); err != nil {
		t.Fatal("flag cache not created", err)
	}
	if _, err := os.Stat(os.Getenv("AILANG_EMBED_CACHE_DB")); !os.IsNotExist(err) {
		t.Fatal("environment cache opened despite flag")
	}
}
