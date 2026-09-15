// Package factory builds an ai.Provider from a provider name, resolving the
// credential, base URL and auth lane in ONE place.
//
// Before M-V1-SIMPLIFY-S3 M4 five callers (cmd/ailang exec.go and
// ai_handlers.go ×2, eval_harness/ai_provider.go, mission/quorum/call.go,
// coordinator_lifecycle.go) each carried their own env-key → NewClient switch,
// and they had drifted: exec.go read GEMINI_API_KEY where the others read
// GOOGLE_API_KEY, pinned its own ollama endpoint, and only two of the five
// honoured OPENAI_BASE_URL. Those switches are now calls to New.
//
// It lives beside internal/ai rather than inside it because every provider
// package imports ai (they implement ai.Provider); ai importing them back
// would be a cycle.
package factory

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/anthropic"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/ai/ollama"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/ai/openrouter"
)

// Lane names how the provider was authenticated. Safe to log — never the secret.
type Lane string

const (
	LaneAPIKey          Lane = "api-key"         // a metered key from the environment or WithAPIKey
	LaneOAuth           Lane = "oauth"           // an Anthropic subscription token (ai.ResolveAnthropicCredential)
	LaneADC             Lane = "adc"             // Google Application Default Credentials via Vertex AI
	LaneLocal           Lane = "local"           // ollama: no credential, the daemon is local (or proxies with its device key)
	LaneUnauthenticated Lane = "unauthenticated" // an OpenAI-compatible endpoint that takes no key (OPENAI_BASE_URL set, no key)
	LaneConfigDriven    Lane = "config-driven"   // an [[ai_provider]] block resolved by the registry hook
)

// Client is a constructed provider plus how it was built.
type Client struct {
	Provider ai.Provider
	Type     ai.ProviderType
	Lane     Lane
}

// Option configures construction.
type Option func(*options)

type options struct {
	apiKey       string
	apiKeyEnv    string
	baseURL      string
	gcpProject   string
	configDriven func(name string) ai.Provider
}

// WithAPIKey supplies the credential explicitly; the environment is not read.
func WithAPIKey(key string) Option { return func(o *options) { o.apiKey = key } }

// WithAPIKeyEnv names the environment variable holding the key (a models.yml
// env_var) instead of the provider's standard one. Empty is a no-op.
func WithAPIKeyEnv(name string) Option {
	return func(o *options) { o.apiKeyEnv = strings.TrimSpace(name) }
}

// WithEndpoint overrides the endpoint: an OpenAI-compatible base URL for
// openai/lyceum/zai/openrouter, the daemon endpoint for ollama.
func WithEndpoint(u string) Option { return func(o *options) { o.baseURL = u } }

// WithGCPProject pins the Vertex AI project for Google's ADC lane and makes
// that lane the ONLY one tried — a caller that names a project has chosen
// Vertex, and falling back to an AI Studio key would change which project
// is billed (mission/quorum reviewers). Empty means ADC with the gcloud
// default project, then the key from the environment.
func WithGCPProject(p string) Option { return func(o *options) { o.gcpProject = p } }

// WithConfigDriven supplies the registry lookup for names that are not
// built-in (M-AI-PROVIDER-CONFIG [[ai_provider]] blocks). Built-ins win on a
// name collision (D4). Without it an unknown name is an error.
func WithConfigDriven(lookup func(name string) ai.Provider) Option {
	return func(o *options) { o.configDriven = lookup }
}

// NewProvider is New without the lane.
func NewProvider(name string, opts ...Option) (ai.Provider, error) {
	c, err := New(name, opts...)
	if err != nil {
		return nil, err
	}
	return c.Provider, nil
}

// New builds the provider for a name as ai.ProviderFromString reads it
// ("gemini"/"google"/"vertex", "z-ai", an ai.ProviderType, or a config-driven
// name). It fails loudly when the credential the lane needs is absent —
// there is no safe default lane to fall back to (CLAUDE.md §2).
func New(name string, opts ...Option) (*Client, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	typ := ai.ProviderFromString(name)

	switch typ {
	case ai.ProviderOpenAI:
		baseURL := o.baseURL
		if baseURL == "" {
			baseURL = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
		}
		key := o.key(typ)
		if key == "" && baseURL == "" {
			return nil, fmt.Errorf("%s environment variable required (or set OPENAI_BASE_URL for a custom unauthenticated endpoint)", o.keyEnv(typ))
		}
		var copts []openai.ClientOption
		if baseURL != "" {
			copts = append(copts, openai.WithBaseURL(baseURL))
		}
		lane := LaneAPIKey
		if key == "" {
			lane = LaneUnauthenticated
		}
		return &Client{Provider: openai.NewClient(key, copts...), Type: typ, Lane: lane}, nil

	case ai.ProviderLyceum, ai.ProviderZAI:
		// Same openai transport, a different endpoint (ai.LyceumBaseURL /
		// ai.ZAIBaseURL honour their *_BASE_URL overrides).
		key, err := o.requireKey(typ)
		if err != nil {
			return nil, err
		}
		baseURL := o.baseURL
		if baseURL == "" {
			if typ == ai.ProviderLyceum {
				baseURL = ai.LyceumBaseURL()
			} else {
				baseURL = ai.ZAIBaseURL()
			}
		}
		return &Client{Provider: openai.NewClient(key, openai.WithBaseURL(baseURL)), Type: typ, Lane: LaneAPIKey}, nil

	case ai.ProviderAnthropic:
		// The credential's lane decides the header shape: an OAuth token in
		// x-api-key is a 401, not a degraded run. ai.ResolveAnthropicCredential
		// is the single source of truth, so this cannot disagree with the
		// cost classifier (ai.AnthropicLaneIsOAuth).
		if key := o.key(typ); key != "" {
			return &Client{Provider: anthropic.NewClient(key, o.anthropicOpts()...), Type: typ, Lane: LaneAPIKey}, nil
		}
		if o.apiKeyEnv != "" {
			return nil, fmt.Errorf("%s environment variable required", o.apiKeyEnv)
		}
		cred, err := ai.ResolveAnthropicCredential()
		if err != nil {
			return nil, err
		}
		copts := o.anthropicOpts()
		lane := LaneAPIKey
		if cred.OAuth {
			copts = append(copts, anthropic.WithOAuth())
			lane = LaneOAuth
		}
		return &Client{Provider: anthropic.NewClient(cred.Value, copts...), Type: typ, Lane: lane}, nil

	case ai.ProviderGoogle:
		// An explicit key means AI Studio. Otherwise ADC first — many machines
		// carry a GOOGLE_API_KEY for other tools but run Vertex on ADC — then
		// the key from the environment: the models.yml env_var if given, else
		// GOOGLE_API_KEY, else GEMINI_API_KEY (Google's SDKs accept either
		// name; `ailang exec --provider gemini` read only the latter before).
		// A pinned project is Vertex-only, see WithGCPProject.
		if o.apiKey != "" {
			return &Client{Provider: gemini.NewClient(o.apiKey, o.geminiOpts()...), Type: typ, Lane: LaneAPIKey}, nil
		}
		client, adcErr := gemini.NewVertexAIClient(o.gcpProject, o.geminiOpts()...)
		if adcErr == nil {
			return &Client{Provider: client, Type: typ, Lane: LaneADC}, nil
		}
		if o.gcpProject != "" {
			return nil, fmt.Errorf("Vertex ADC (gcp_project=%q) unavailable: %w", o.gcpProject, adcErr)
		}
		key := o.key(typ)
		if key == "" && o.apiKeyEnv == "" {
			key = os.Getenv("GEMINI_API_KEY")
		}
		if key != "" {
			return &Client{Provider: gemini.NewClient(key, o.geminiOpts()...), Type: typ, Lane: LaneAPIKey}, nil
		}
		return nil, fmt.Errorf("Gemini auth failed: Application Default Credentials (ADC) not configured, and %s is not set.\n"+
			"  Option 1: gcloud auth application-default login  (recommended, for Vertex AI)\n"+
			"  Option 2: export %s=<key>  (get one at https://aistudio.google.com/apikey; GEMINI_API_KEY is accepted too)\n"+
			"  ADC error: %w", o.keyEnv(typ), o.keyEnv(typ), adcErr)

	case ai.ProviderOllama:
		// No credential: the daemon is local, or proxies `-cloud` models to
		// ollama.com with the device key from `ollama signin`. The endpoint is
		// decided ONCE, in ollama.NewClient: OLLAMA_HOST, else 127.0.0.1:11434
		// (IPv4-pinned so the harness cannot silently reach an uncapped
		// Ollama.app listener over ::1 — see defaultEndpoint there).
		var copts []ollama.ClientOption
		if o.baseURL != "" {
			copts = append(copts, ollama.WithEndpoint(o.baseURL))
		}
		client, err := ollama.NewClient(copts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama client: %w", err)
		}
		return &Client{Provider: client, Type: typ, Lane: LaneLocal}, nil

	case ai.ProviderOpenRouter:
		key, err := o.requireKey(typ)
		if err != nil {
			return nil, err
		}
		var copts []openrouter.ClientOption
		if o.baseURL != "" {
			copts = append(copts, openrouter.WithBaseURL(o.baseURL))
		}
		return &Client{Provider: openrouter.NewClient(key, copts...), Type: typ, Lane: LaneAPIKey}, nil
	}

	if o.configDriven != nil {
		if p := o.configDriven(name); p != nil {
			return &Client{Provider: p, Type: typ, Lane: LaneConfigDriven}, nil
		}
	}
	return nil, fmt.Errorf("unsupported AI provider: %q (built-in: openai, anthropic, gemini, ollama, openrouter, lyceum, zai)", name)
}

// keyEnv is the variable the key is read from: WithAPIKeyEnv, else the
// provider's standard variable.
func (o *options) keyEnv(typ ai.ProviderType) string {
	if o.apiKeyEnv != "" {
		return o.apiKeyEnv
	}
	return ai.EnvVarForProvider(typ)
}

// key returns the explicit key, else the environment's ("" when unset).
func (o *options) key(typ ai.ProviderType) string {
	if o.apiKey != "" {
		return o.apiKey
	}
	if env := o.keyEnv(typ); env != "" {
		return os.Getenv(env)
	}
	return ""
}

func (o *options) requireKey(typ ai.ProviderType) (string, error) {
	if key := o.key(typ); key != "" {
		return key, nil
	}
	return "", fmt.Errorf("%s environment variable required", o.keyEnv(typ))
}

func (o *options) anthropicOpts() []anthropic.ClientOption {
	if o.baseURL != "" {
		return []anthropic.ClientOption{anthropic.WithBaseURL(o.baseURL)}
	}
	return nil
}

func (o *options) geminiOpts() []gemini.ClientOption {
	if o.baseURL != "" {
		return []gemini.ClientOption{gemini.WithBaseURL(o.baseURL)}
	}
	return nil
}
