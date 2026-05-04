package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test the full happy path: a manifest with a valid [[ai_provider]] block parses
// and validates with all fields readable on the resulting struct.
func TestLoadManifest_AIProvider_Valid(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/ai_vllm"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/ai_vllm/core"]

[[ai_provider]]
schema_version = 1
name = "vllm"
endpoint = "http://localhost:8000/v1/chat/completions"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
error_path = "$.error.message"
auth = { type = "bearer", env = "VLLM_API_KEY" }
cost = { input_per_1m_usd = 0.0, output_per_1m_usd = 0.0, currency = "USD" }
capabilities = { tool_calling = false, json_mode = true, streaming = true, vision = false, structured_outputs = true }

[ai_provider.streaming]
enabled = true
endpoint = "http://localhost:8000/v1/chat/completions"
delta_path = "$.choices[0].delta.content"
reasoning_path = "$.choices[0].delta.reasoning_content"
done_sentinel = "[DONE]"

[ai_provider.models]
allowed = ["llama-3.1-70b", "llama-3.1-8b"]
`
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}

	if len(m.AIProviders) != 1 {
		t.Fatalf("expected 1 ai_provider, got %d", len(m.AIProviders))
	}

	p := m.AIProviders[0]
	if p.Name != "vllm" {
		t.Errorf("name = %q, want vllm", p.Name)
	}
	if p.SchemaVersion != 1 {
		t.Errorf("schema_version = %d, want 1", p.SchemaVersion)
	}
	if p.RequestShape != "openai_chat" {
		t.Errorf("request_shape = %q, want openai_chat", p.RequestShape)
	}
	if p.Auth.Type != "bearer" || p.Auth.Env != "VLLM_API_KEY" {
		t.Errorf("auth = %+v, want {bearer, VLLM_API_KEY}", p.Auth)
	}
	if !p.Capabilities.Streaming {
		t.Errorf("capabilities.streaming = false, want true")
	}
	if !p.Capabilities.StructuredOutputs {
		t.Errorf("capabilities.structured_outputs = false, want true")
	}
	if p.Capabilities.ToolCalling {
		t.Errorf("capabilities.tool_calling = true, want false")
	}
	if p.Capabilities.Vision {
		t.Errorf("capabilities.vision = true, want false")
	}
	if !p.Streaming.Enabled || p.Streaming.DoneSentinel != "[DONE]" {
		t.Errorf("streaming = %+v, want enabled with [DONE] sentinel", p.Streaming)
	}
	if len(p.Models.Allowed) != 2 {
		t.Errorf("models.allowed = %d, want 2", len(p.Models.Allowed))
	}
	if p.EffectiveStreamingEndpoint() != "http://localhost:8000/v1/chat/completions" {
		t.Errorf("EffectiveStreamingEndpoint = %q", p.EffectiveStreamingEndpoint())
	}
	if p.Cost.EffectiveCurrency() != "USD" {
		t.Errorf("EffectiveCurrency = %q, want USD", p.Cost.EffectiveCurrency())
	}
}

// Test that streaming endpoint defaults to the main endpoint when omitted.
func TestAIProvider_StreamingEndpointFallback(t *testing.T) {
	p := AIProviderSpec{
		Endpoint: "https://api.openai.com/v1/chat/completions",
		Streaming: AIProviderStreaming{
			Enabled:   true,
			DeltaPath: "$.choices[0].delta.content",
			// Endpoint deliberately empty
		},
	}
	if got := p.EffectiveStreamingEndpoint(); got != p.Endpoint {
		t.Errorf("EffectiveStreamingEndpoint = %q, want %q", got, p.Endpoint)
	}
}

// Test currency default.
func TestAIProvider_CurrencyDefault(t *testing.T) {
	c := AIProviderCost{}
	if got := c.EffectiveCurrency(); got != "USD" {
		t.Errorf("EffectiveCurrency = %q, want USD", got)
	}
	c.Currency = "EUR"
	if got := c.EffectiveCurrency(); got != "EUR" {
		t.Errorf("EffectiveCurrency = %q, want EUR", got)
	}
}

// Test the auth_headers escape hatch with ${ENV_VAR} interpolation.
func TestLoadManifest_AIProvider_AuthHeaders(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/ai_custom"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/ai_custom/core"]

[[ai_provider]]
schema_version = 1
name = "custom"
endpoint = "https://api.example.com/v1/chat"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth_headers = { Authorization = "Bearer ${CUSTOM_TOKEN}", X-Org = "${CUSTOM_ORG}" }
`
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}
	p := m.AIProviders[0]
	if got := p.AuthHeaders["Authorization"]; got != "Bearer ${CUSTOM_TOKEN}" {
		t.Errorf("Authorization header = %q", got)
	}
	if p.HasNamedAuth() {
		t.Errorf("HasNamedAuth() = true, want false (only auth_headers set)")
	}
}

// Multiple providers in one manifest are supported.
func TestLoadManifest_AIProvider_Multiple(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/ai_multi"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/ai_multi/core"]

[[ai_provider]]
schema_version = 1
name = "vllm"
endpoint = "http://localhost:8000/v1/chat/completions"
request_shape = "openai_chat"
response_path = "$.choices[0].message.content"
auth = { type = "none" }

[[ai_provider]]
schema_version = 1
name = "anthropic_proxy"
endpoint = "https://proxy.example.com/v1/messages"
request_shape = "anthropic_messages"
response_path = "$.content[0].text"
auth = { type = "x-api-key", env = "ANTHROPIC_PROXY_KEY" }
`
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest failed: %v", err)
	}
	if len(m.AIProviders) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(m.AIProviders))
	}
	if m.AIProviders[0].Name != "vllm" || m.AIProviders[1].Name != "anthropic_proxy" {
		t.Errorf("provider names = %q, %q", m.AIProviders[0].Name, m.AIProviders[1].Name)
	}
}

// Each row exercises one validation rule. errSubstr is checked against the
// resulting error message; empty errSubstr means the manifest should validate.
func TestValidateAIProviders_Errors(t *testing.T) {
	cases := []struct {
		name      string
		providers []AIProviderSpec
		errSubstr string
	}{
		{
			name: "missing schema_version",
			providers: []AIProviderSpec{{
				Name:         "x",
				Endpoint:     "https://e",
				RequestShape: "openai_chat",
				ResponsePath: "$.x",
				Auth:         AIProviderAuth{Type: "none"},
			}},
			errSubstr: "schema_version is required",
		},
		{
			name: "future schema_version",
			providers: []AIProviderSpec{{
				SchemaVersion: 99,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
			}},
			errSubstr: "newer than this AILANG runtime supports",
		},
		{
			name: "name with slash",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "vendor/name",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
			}},
			errSubstr: "must be a single segment",
		},
		{
			name: "missing endpoint",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
			}},
			errSubstr: "endpoint is required",
		},
		{
			name: "non-http endpoint",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "ftp://server",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
			}},
			errSubstr: "must be an http:// or https:// URL",
		},
		{
			name: "unknown request_shape",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "cohere_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
			}},
			errSubstr: "unknown request_shape",
		},
		{
			name: "custom shape without template",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "custom",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
				// RequestTemplate empty
			}},
			errSubstr: "requires request_template",
		},
		{
			name: "missing response_path",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				Auth:          AIProviderAuth{Type: "none"},
			}},
			errSubstr: "response_path is required",
		},
		{
			name: "no auth declared",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				// Auth empty, AuthHeaders empty
			}},
			errSubstr: "must declare auth.type",
		},
		{
			name: "unknown auth.type",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "oauth2"},
			}},
			errSubstr: "unknown auth.type",
		},
		{
			name: "bearer without env",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "bearer"},
			}},
			errSubstr: "requires auth.env",
		},
		{
			name: "query-param without name",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "query-param", Env: "API_KEY"},
			}},
			errSubstr: "requires auth.name",
		},
		{
			name: "lowercase env name",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "bearer", Env: "api_key"},
			}},
			errSubstr: "must match",
		},
		{
			name: "malformed auth_headers interpolation",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				AuthHeaders:   map[string]string{"X-Foo": "Bearer ${unclosed"},
			}},
			errSubstr: "unclosed",
		},
		{
			name: "lowercase env in auth_headers",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				AuthHeaders:   map[string]string{"X-Foo": "Bearer ${api_key}"},
			}},
			errSubstr: "malformed env reference",
		},
		{
			name: "streaming enabled without delta_path",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
				Streaming:     AIProviderStreaming{Enabled: true},
			}},
			errSubstr: "requires streaming.delta_path",
		},
		{
			name: "duplicate provider names in same manifest",
			providers: []AIProviderSpec{
				{
					SchemaVersion: 1,
					Name:          "vllm",
					Endpoint:      "http://localhost:8000/v1",
					RequestShape:  "openai_chat",
					ResponsePath:  "$.x",
					Auth:          AIProviderAuth{Type: "none"},
				},
				{
					SchemaVersion: 1,
					Name:          "vllm", // duplicate
					Endpoint:      "http://localhost:9000/v1",
					RequestShape:  "openai_chat",
					ResponsePath:  "$.x",
					Auth:          AIProviderAuth{Type: "none"},
				},
			},
			errSubstr: "already declared",
		},
		{
			name: "invalid currency code",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "x",
				Endpoint:      "https://e",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
				Cost:          AIProviderCost{Currency: "usd"}, // lowercase
			}},
			errSubstr: "ISO 4217",
		},
		{
			name: "valid minimal provider",
			providers: []AIProviderSpec{{
				SchemaVersion: 1,
				Name:          "minimal",
				Endpoint:      "https://e.com/v1",
				RequestShape:  "openai_chat",
				ResponsePath:  "$.x",
				Auth:          AIProviderAuth{Type: "none"},
			}},
			errSubstr: "", // expect validation success
		},
		{
			name:      "empty provider list",
			providers: nil,
			errSubstr: "", // expect success — packages without [[ai_provider]] are normal
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAIProviders(tc.providers)
			if tc.errSubstr == "" {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.errSubstr)
			}
			if !strings.Contains(err.Error(), tc.errSubstr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.errSubstr)
			}
		})
	}
}

// Test that a manifest validation propagates ai_provider errors.
func TestLoadManifest_AIProvider_ValidationFailsWholeLoad(t *testing.T) {
	dir := t.TempDir()
	content := `
[package]
name = "sunholo/ai_bad"
version = "0.1.0"
edition = "1"

[exports]
modules = ["sunholo/ai_bad/core"]

[[ai_provider]]
schema_version = 1
name = "bad"
endpoint = "not-a-url"
request_shape = "openai_chat"
response_path = "$.x"
auth = { type = "none" }
`
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadManifest(dir)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "must be an http:// or https:// URL") {
		t.Errorf("error %q does not mention URL validation", err)
	}
}

// Validate the env-interpolation regex directly.
func TestEnvInterpolationPattern(t *testing.T) {
	cases := []struct {
		s    string
		ok   bool
		desc string
	}{
		{"${API_KEY}", true, "uppercase snake"},
		{"${API}", true, "single uppercase"},
		{"${A}", true, "single letter"},
		{"${_PRIVATE}", true, "leading underscore"},
		{"${API_KEY_2}", true, "with digits"},
		{"${api_key}", false, "lowercase rejected"},
		{"${1KEY}", false, "leading digit rejected"},
		{"${API-KEY}", false, "hyphen rejected"},
		{"${}", false, "empty rejected"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := envInterpolationPattern.MatchString(tc.s)
			if got != tc.ok {
				t.Errorf("MatchString(%q) = %v, want %v", tc.s, got, tc.ok)
			}
		})
	}
}
