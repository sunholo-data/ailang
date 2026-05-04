// Schema for `[[ai_provider]]` blocks in ailang.toml.
// See design_docs/planned/v0_15_0/m-ai-provider-config.md.
//
// A package may declare zero or more [[ai_provider]] blocks. Each block
// registers a config-driven AI provider that integrates with the AI effect,
// budget tracking, AI cap gating, and trace system identically to the
// hardcoded built-in providers (openai, anthropic, gemini, ollama, openrouter).

package pkg

import (
	"fmt"
	"regexp"
	"strings"
)

// AIProviderSchemaV1 is the only schema version recognised in v0.15.0.
// v2 will add optional fields (tool-use templating, image input, batch);
// v1 packages remain valid against v2+ runtimes via additive evolution.
const AIProviderSchemaV1 = 1

// validRequestShapes enumerates the v1 request-shape catalog.
// Adding a new shape is a v2 schema bump.
var validRequestShapes = map[string]bool{
	"openai_chat":        true, // OpenAI / OpenRouter / Together / Groq / Anyscale / Fireworks / vLLM / llama.cpp openai-compat
	"anthropic_messages": true, // Anthropic native (messages with content blocks)
	"simple_completion":  true, // Ollama-style single-string prompt
	"custom":             true, // request_template escape hatch (Go template)
}

// validAuthTypes enumerates the v1 auth-shape catalog.
// auth_headers escape hatch covers anything else.
var validAuthTypes = map[string]bool{
	"bearer":      true, // Authorization: Bearer ${env}
	"x-api-key":   true, // x-api-key: ${env}
	"query-param": true, // ?<name>=${env}
	"none":        true, // no auth (local endpoints)
}

// envInterpolationPattern restricts ${VAR} syntax to uppercase-snake-case
// identifiers. Literal substitution only — no shell expansion, no command
// substitution, no nested fallbacks. Documented as v1 scope; v2 may expand.
var envInterpolationPattern = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// AIProviderSpec is a single [[ai_provider]] block in ailang.toml.
type AIProviderSpec struct {
	SchemaVersion int    `toml:"schema_version"` // required; must be 1
	Name          string `toml:"name"`           // routing prefix (e.g. "vllm" → call("vllm/...", ...))
	Endpoint      string `toml:"endpoint"`       // request URL
	RequestShape  string `toml:"request_shape"`  // openai_chat | anthropic_messages | simple_completion | custom
	ResponseShape string `toml:"response_shape"` // optional; if absent, mirrors request_shape conventions
	ResponsePath  string `toml:"response_path"`  // JSONPath to extract result text on 2xx
	ErrorPath     string `toml:"error_path"`     // JSONPath to extract error message on 4xx/5xx

	// request_template is consulted only when request_shape == "custom".
	// Go template string with {{prompt}}/{{model}}/{{messages}} variables.
	RequestTemplate string `toml:"request_template"`

	Auth         AIProviderAuth         `toml:"auth"`
	AuthHeaders  map[string]string      `toml:"auth_headers"` // escape hatch; literal ${ENV_VAR} interpolation
	Cost         AIProviderCost         `toml:"cost"`
	Capabilities AIProviderCapabilities `toml:"capabilities"`
	Streaming    AIProviderStreaming    `toml:"streaming"`
	Models       AIProviderModels       `toml:"models"`
}

// AIProviderAuth declares one of the named auth shapes. Either Type+Env
// (named shape) or top-level AuthHeaders on the parent spec (escape hatch);
// not both.
type AIProviderAuth struct {
	Type string `toml:"type"`           // bearer | x-api-key | query-param | none
	Env  string `toml:"env"`            // env var name holding the secret (uppercase-snake-case)
	Name string `toml:"name,omitempty"` // for query-param: query string parameter name
}

// AIProviderCost declares per-token and/or per-call pricing for the budget
// tracker. Either or both may be set; runtime uses what's available.
type AIProviderCost struct {
	InputPer1MUSD  float64 `toml:"input_per_1m_usd"`  // USD per 1M input tokens
	OutputPer1MUSD float64 `toml:"output_per_1m_usd"` // USD per 1M output tokens
	PerCallUSD     float64 `toml:"per_call_usd"`      // flat USD per call (e.g. fixed-fee endpoints)
	Currency       string  `toml:"currency"`          // currency code; defaults to USD
}

// AIProviderCapabilities flags features this provider supports. TOML keys
// match the stable wire identifiers in internal/ai/routing.go (AICapability)
// so registration-time declarations and request-time AIRoutingPolicy.Require
// share one vocabulary. Calls requiring unsupported capabilities fail with
// AIError{code: "CapabilityNotSupported"}.
type AIProviderCapabilities struct {
	ToolCalling       bool `toml:"tool_calling"`       // function/tool calling (CapToolCalling)
	JSONMode          bool `toml:"json_mode"`          // schema-enforced JSON output (CapJSONMode)
	Streaming         bool `toml:"streaming"`          // SSE token streaming (CapStreaming) — consumed by M-AI-STREAMING-HELPER
	Vision            bool `toml:"vision"`             // multimodal image input (CapVision)
	StructuredOutputs bool `toml:"structured_outputs"` // schema-enforced structured output (CapStructuredOutputs)
}

// AIProviderStreaming carries SSE streaming params consumed by the
// M-AI-STREAMING-HELPER milestone (v0.17.0). Optional in v0.15.0; runtime
// hook lands when the streaming milestone ships.
type AIProviderStreaming struct {
	Enabled       bool   `toml:"enabled"`
	Endpoint      string `toml:"endpoint"`       // optional; defaults to AIProviderSpec.Endpoint
	DeltaPath     string `toml:"delta_path"`     // JSONPath into delta event for content text
	ReasoningPath string `toml:"reasoning_path"` // JSONPath for reasoning_content / thinking (optional)
	DoneSentinel  string `toml:"done_sentinel"`  // OpenAI-style "[DONE]"; Anthropic uses event types instead
}

// AIProviderModels optionally constrains which model identifiers route to
// this provider. If empty, prefix-match accepts any model under <name>/.
type AIProviderModels struct {
	Allowed []string `toml:"allowed"`
}

// HasNamedAuth reports whether this provider declares an auth shape from the
// named catalog (vs only auth_headers escape hatch).
func (s *AIProviderSpec) HasNamedAuth() bool {
	return s.Auth.Type != ""
}

// EffectiveStreamingEndpoint returns the streaming endpoint URL, falling back
// to the main endpoint if streaming.endpoint is unset.
func (s *AIProviderSpec) EffectiveStreamingEndpoint() string {
	if s.Streaming.Endpoint != "" {
		return s.Streaming.Endpoint
	}
	return s.Endpoint
}

// EffectiveCurrency returns the currency code, defaulting to USD when unset.
func (c *AIProviderCost) EffectiveCurrency() string {
	if c.Currency == "" {
		return "USD"
	}
	return c.Currency
}

// validateAIProviders checks all [[ai_provider]] blocks in a manifest for
// required fields, schema version, enum membership, and conflict-free names
// (within a single manifest). Cross-package conflict detection happens at
// load time after dependency resolution — see internal/pkg/loader.go.
func validateAIProviders(providers []AIProviderSpec) error {
	seen := make(map[string]int) // name → index of first declaration

	for i, p := range providers {
		idx := i + 1 // 1-indexed for human-readable error messages

		if p.SchemaVersion == 0 {
			return fmt.Errorf("[[ai_provider]] #%d: schema_version is required (current: %d)", idx, AIProviderSchemaV1)
		}
		if p.SchemaVersion > AIProviderSchemaV1 {
			return fmt.Errorf("[[ai_provider]] #%d: schema_version %d is newer than this AILANG runtime supports (max: %d) — upgrade AILANG to use this package",
				idx, p.SchemaVersion, AIProviderSchemaV1)
		}
		if p.SchemaVersion < AIProviderSchemaV1 {
			return fmt.Errorf("[[ai_provider]] #%d: schema_version %d is older than the minimum supported version %d",
				idx, p.SchemaVersion, AIProviderSchemaV1)
		}

		if p.Name == "" {
			return fmt.Errorf("[[ai_provider]] #%d: name is required (used as routing prefix)", idx)
		}
		if strings.Contains(p.Name, "/") {
			return fmt.Errorf("[[ai_provider]] #%d: name %q must be a single segment (no slashes) — slashes separate provider from model in call(\"%s/<model>\", ...)",
				idx, p.Name, p.Name)
		}
		if firstIdx, exists := seen[p.Name]; exists {
			return fmt.Errorf("[[ai_provider]] #%d: provider name %q already declared in [[ai_provider]] #%d of this manifest",
				idx, p.Name, firstIdx+1)
		}
		seen[p.Name] = i

		if p.Endpoint == "" {
			return fmt.Errorf("[[ai_provider]] #%d (%q): endpoint is required", idx, p.Name)
		}
		if !strings.HasPrefix(p.Endpoint, "http://") && !strings.HasPrefix(p.Endpoint, "https://") {
			return fmt.Errorf("[[ai_provider]] #%d (%q): endpoint must be an http:// or https:// URL, got %q",
				idx, p.Name, p.Endpoint)
		}

		if p.RequestShape == "" {
			return fmt.Errorf("[[ai_provider]] #%d (%q): request_shape is required (one of: openai_chat, anthropic_messages, simple_completion, custom)",
				idx, p.Name)
		}
		if !validRequestShapes[p.RequestShape] {
			return fmt.Errorf("[[ai_provider]] #%d (%q): unknown request_shape %q — valid: openai_chat, anthropic_messages, simple_completion, custom",
				idx, p.Name, p.RequestShape)
		}
		if p.RequestShape == "custom" && p.RequestTemplate == "" {
			return fmt.Errorf("[[ai_provider]] #%d (%q): request_shape=\"custom\" requires request_template",
				idx, p.Name)
		}

		if p.ResponsePath == "" {
			return fmt.Errorf("[[ai_provider]] #%d (%q): response_path is required (JSONPath to extract result text)",
				idx, p.Name)
		}

		// Auth: must declare exactly one of (named auth.type) or (top-level auth_headers).
		// Both is allowed (auth_headers extends headers from the named shape).
		// Neither is an error — even local "none" auth requires an explicit declaration
		// to avoid silent insecure defaults.
		hasNamedAuth := p.Auth.Type != ""
		hasAuthHeaders := len(p.AuthHeaders) > 0
		if !hasNamedAuth && !hasAuthHeaders {
			return fmt.Errorf("[[ai_provider]] #%d (%q): must declare auth.type (bearer|x-api-key|query-param|none) or auth_headers", idx, p.Name)
		}

		if hasNamedAuth {
			if !validAuthTypes[p.Auth.Type] {
				return fmt.Errorf("[[ai_provider]] #%d (%q): unknown auth.type %q — valid: bearer, x-api-key, query-param, none",
					idx, p.Name, p.Auth.Type)
			}
			// bearer/x-api-key/query-param need an env var pointing at the secret
			if p.Auth.Type != "none" && p.Auth.Env == "" {
				return fmt.Errorf("[[ai_provider]] #%d (%q): auth.type=%q requires auth.env (the env var holding the secret)",
					idx, p.Name, p.Auth.Type)
			}
			// query-param needs a parameter name
			if p.Auth.Type == "query-param" && p.Auth.Name == "" {
				return fmt.Errorf("[[ai_provider]] #%d (%q): auth.type=\"query-param\" requires auth.name (the query string parameter name)",
					idx, p.Name)
			}
			// env var name must follow the same pattern we accept for ${...} interpolation
			if p.Auth.Env != "" && !envInterpolationPattern.MatchString("${"+p.Auth.Env+"}") {
				return fmt.Errorf("[[ai_provider]] #%d (%q): auth.env %q must match [A-Z_][A-Z0-9_]*",
					idx, p.Name, p.Auth.Env)
			}
		}

		// auth_headers: every value may contain ${VAR} interpolation; verify the
		// pattern is well-formed (no unmatched braces, no lowercase identifiers).
		for headerName, headerVal := range p.AuthHeaders {
			if headerName == "" {
				return fmt.Errorf("[[ai_provider]] #%d (%q): auth_headers contains empty header name", idx, p.Name)
			}
			if err := validateInterpolation(headerVal); err != nil {
				return fmt.Errorf("[[ai_provider]] #%d (%q): auth_headers[%q]: %w",
					idx, p.Name, headerName, err)
			}
		}

		// Streaming sub-block: if enabled, delta_path is required.
		if p.Streaming.Enabled && p.Streaming.DeltaPath == "" {
			return fmt.Errorf("[[ai_provider]] #%d (%q): streaming.enabled = true requires streaming.delta_path (JSONPath into delta event for content text)",
				idx, p.Name)
		}

		// Cost: if all three pricing fields are zero AND no per-call price set,
		// log nothing — cost = 0 is a valid declaration (e.g. local llama.cpp).
		// Currency code, if set, must be a 3-letter uppercase code.
		if p.Cost.Currency != "" {
			if !isUpperASCIIAlpha3(p.Cost.Currency) {
				return fmt.Errorf("[[ai_provider]] #%d (%q): cost.currency %q must be a 3-letter uppercase ISO 4217 code (e.g. USD, EUR)",
					idx, p.Name, p.Cost.Currency)
			}
		}
	}

	return nil
}

// validateInterpolation checks a string for well-formed ${VAR} references.
// Catches common mistakes: unmatched ${, lowercase identifiers, empty braces.
func validateInterpolation(s string) error {
	for i := 0; i < len(s); i++ {
		if s[i] == '$' && i+1 < len(s) && s[i+1] == '{' {
			end := strings.Index(s[i:], "}")
			if end == -1 {
				return fmt.Errorf("unclosed ${...} at position %d", i)
			}
			ref := s[i : i+end+1]
			if !envInterpolationPattern.MatchString(ref) {
				return fmt.Errorf("malformed env reference %q (must match ${[A-Z_][A-Z0-9_]*})", ref)
			}
			i += end
		}
	}
	return nil
}

// isUpperASCIIAlpha3 returns true for exactly 3 uppercase ASCII letters
// (matches ISO 4217 currency codes like USD, EUR, GBP).
func isUpperASCIIAlpha3(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
