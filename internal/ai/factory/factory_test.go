package factory

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/anthropic"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/ai/ollama"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/ai/openrouter"
)

// isolate clears every credential the factory can read so a developer's
// environment (keys, ~/.claude/.credentials.json, gcloud on PATH) cannot leak
// into the assertions. Nothing here talks to a provider.
func isolate(t *testing.T) {
	t.Helper()
	for _, v := range []string{
		"OPENAI_API_KEY", "OPENAI_BASE_URL", "ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN",
		"CLAUDE_CODE_OAUTH_TOKEN", "GOOGLE_API_KEY", "GEMINI_API_KEY", "OPENROUTER_API_KEY",
		"LYCEUM_API_KEY", "ZAI_API_KEY", "OLLAMA_HOST", "AILANG_CLOUD_PROJECT",
		"GOOGLE_CLOUD_PROJECT", "GOOGLE_APPLICATION_CREDENTIALS",
	} {
		t.Setenv(v, "")
	}
	t.Setenv("HOME", t.TempDir()) // no ~/.claude/.credentials.json, no ~/.ailang/config.yaml
	t.Setenv("PATH", t.TempDir()) // no gcloud
}

func TestNew_OpenAICompatibleLanes(t *testing.T) {
	isolate(t)

	if _, err := New("openai"); err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("no key, no base URL: %v", err)
	}
	t.Setenv("OPENAI_BASE_URL", "http://localhost:1/v1")
	c, err := New("openai")
	if err != nil || c.Lane != LaneUnauthenticated || c.Type != ai.ProviderOpenAI {
		t.Fatalf("base URL only: %+v %v", c, err)
	}
	if _, ok := c.Provider.(*openai.Client); !ok {
		t.Fatalf("provider = %T", c.Provider)
	}
	t.Setenv("OPENAI_API_KEY", "sk-x")
	if c, err = New("openai"); err != nil || c.Lane != LaneAPIKey {
		t.Fatalf("key: %+v %v", c, err)
	}

	for _, name := range []string{"lyceum", "zai", "z-ai", "openrouter"} {
		if _, err := New(name); err == nil || !strings.Contains(err.Error(), "_API_KEY environment variable required") {
			t.Errorf("%s without key: %v", name, err)
		}
	}
	t.Setenv("LYCEUM_API_KEY", "l")
	t.Setenv("ZAI_API_KEY", "z")
	t.Setenv("OPENROUTER_API_KEY", "o")
	if c, err := New("lyceum"); err != nil || c.Type != ai.ProviderLyceum || c.Lane != LaneAPIKey {
		t.Errorf("lyceum: %+v %v", c, err)
	}
	if c, err := New("z-ai"); err != nil || c.Type != ai.ProviderZAI {
		t.Errorf("z-ai: %+v %v", c, err)
	}
	c, err = New("openrouter")
	if err != nil || c.Type != ai.ProviderOpenRouter {
		t.Fatalf("openrouter: %+v %v", c, err)
	}
	if _, ok := c.Provider.(*openrouter.Client); !ok {
		t.Fatalf("openrouter provider = %T", c.Provider)
	}
}

func TestNew_AnthropicLanes(t *testing.T) {
	isolate(t)

	if _, err := New("anthropic"); err == nil || !strings.Contains(err.Error(), "no Anthropic credential") {
		t.Fatalf("no credential: %v", err)
	}
	c, err := New("anthropic", WithAPIKey("sk-ant-explicit"))
	if err != nil || c.Lane != LaneAPIKey {
		t.Fatalf("explicit key: %+v %v", c, err)
	}
	if _, ok := c.Provider.(*anthropic.Client); !ok {
		t.Fatalf("provider = %T", c.Provider)
	}
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "oauth-token")
	if c, err = New("anthropic"); err != nil || c.Lane != LaneOAuth {
		t.Fatalf("oauth token: %+v %v", c, err)
	}
	// The metered key outbids the OAuth token — the SDK precedence, on purpose.
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-metered")
	if c, err = New("anthropic"); err != nil || c.Lane != LaneAPIKey {
		t.Fatalf("both set: %+v %v", c, err)
	}
	// A models.yml env_var is honoured verbatim, and its absence is an error
	// rather than a silent slide onto another lane.
	t.Setenv("MY_ANTHROPIC_KEY", "")
	if _, err := New("anthropic", WithAPIKeyEnv("MY_ANTHROPIC_KEY")); err == nil || !strings.Contains(err.Error(), "MY_ANTHROPIC_KEY") {
		t.Fatalf("custom env unset: %v", err)
	}
}

func TestNew_GoogleLanes(t *testing.T) {
	isolate(t)

	c, err := New("gemini", WithAPIKey("aistudio"))
	if err != nil || c.Lane != LaneAPIKey || c.Type != ai.ProviderGoogle {
		t.Fatalf("explicit key: %+v %v", c, err)
	}
	if _, ok := c.Provider.(*gemini.Client); !ok {
		t.Fatalf("provider = %T", c.Provider)
	}
	// A pinned project is Vertex ADC — constructed without contacting anything.
	if c, err = New("google", WithGCPProject("proj")); err != nil || c.Lane != LaneADC {
		t.Fatalf("pinned project: %+v %v", c, err)
	}
	// No ADC project resolvable (no env, no config file, no gcloud): the key
	// from the environment is next, GEMINI_API_KEY accepted as the alias.
	if _, err := New("google"); err == nil || !strings.Contains(err.Error(), "GOOGLE_API_KEY") {
		t.Fatalf("nothing set: %v", err)
	}
	t.Setenv("GEMINI_API_KEY", "g")
	if c, err = New("vertex"); err != nil || c.Lane != LaneAPIKey {
		t.Fatalf("GEMINI_API_KEY alias: %+v %v", c, err)
	}
	// ...but a models.yml env_var replaces the alias search entirely.
	if _, err := New("google", WithAPIKeyEnv("GOOGLE_STUDIO_KEY")); err == nil {
		t.Fatal("custom env unset should not fall back to GEMINI_API_KEY")
	}
}

func TestNew_Ollama(t *testing.T) {
	isolate(t)
	c, err := New("ollama")
	if err != nil || c.Lane != LaneLocal || c.Type != ai.ProviderOllama {
		t.Fatalf("%+v %v", c, err)
	}
	if _, ok := c.Provider.(*ollama.Client); !ok {
		t.Fatalf("provider = %T", c.Provider)
	}
	if _, ok := c.Provider.(interface{ CheckConnection(context.Context) error }); !ok {
		t.Fatal("ollama client should expose CheckConnection for the pre-flight probe")
	}
}

// stubProvider is a config-driven stand-in; it is never called.
type stubProvider struct{}

func (*stubProvider) Generate(context.Context, *ai.Request) (*ai.Response, error) {
	return nil, errors.New("stub")
}
func (*stubProvider) Step(context.Context, *ai.Request) (*ai.Response, error) {
	return nil, errors.New("stub")
}
func (*stubProvider) Name() string { return "stub" }

func TestNew_ConfigDrivenAndUnknown(t *testing.T) {
	isolate(t)
	if _, err := New("acme"); err == nil || !strings.Contains(err.Error(), `unsupported AI provider: "acme"`) {
		t.Fatalf("unknown: %v", err)
	}
	stub := &stubProvider{}
	lookup := func(name string) ai.Provider {
		if name == "acme" {
			return stub
		}
		return nil
	}
	c, err := New("acme", WithConfigDriven(lookup))
	if err != nil || c.Lane != LaneConfigDriven || c.Provider != stub {
		t.Fatalf("config-driven: %+v %v", c, err)
	}
	if _, err := New("other", WithConfigDriven(lookup)); err == nil {
		t.Fatal("lookup miss must error")
	}
	// Built-ins win on a name collision (D4): the hook is never consulted.
	t.Setenv("OPENAI_API_KEY", "k")
	c, err = New("openai", WithConfigDriven(func(string) ai.Provider { return stub }))
	if err != nil || c.Lane == LaneConfigDriven {
		t.Fatalf("collision: %+v %v", c, err)
	}
}
