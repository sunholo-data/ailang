package messaging

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-V1-SIMPLIFY-S4 M1: the OpenAI and Gemini embedding models used to be
// hard-coded when nothing named them. The model fixes the VECTOR DIMENSION,
// so an unpinned one is a silent search-corruption hazard, not a convenience:
// served now with a warning, refused under AILANG_STRICT_CONFIG=1.
func TestNewEmbedderFromConfig_ModelIsTheDeprecatedDefaultThenStrict(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))
	for _, v := range []string{config.EnvStrict, EnvEmbedOpenAIModel, EnvEmbedGeminiModel} {
		t.Setenv(v, "")
	}

	cases := []struct {
		provider, env, deprecated string
		cfg                       func(model string) EmbedConfig
	}{
		{"openai", EnvEmbedOpenAIModel, deprecatedOpenAIEmbedModel, func(m string) EmbedConfig {
			return EmbedConfig{Provider: "openai", OpenAI: OpenAIEmbedConfig{APIKey: "k", Model: m}}
		}},
		{"gemini", EnvEmbedGeminiModel, deprecatedGeminiEmbedModel, func(m string) EmbedConfig {
			return EmbedConfig{Provider: "gemini", Gemini: GeminiEmbedConfig{APIKey: "k", Model: m}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			t.Setenv(config.EnvStrict, "")
			emb, err := NewEmbedderFromConfig(tc.cfg(""))
			if err != nil || emb == nil || emb.ModelName() != tc.provider+":"+tc.deprecated {
				t.Fatalf("unset: (%v, %v), want the deprecated %q served", emb, err, tc.deprecated)
			}

			t.Setenv(config.EnvStrict, "1")
			if emb, err = NewEmbedderFromConfig(tc.cfg("")); emb != nil || !errors.Is(err, config.ErrDeprecatedDefault) {
				t.Fatalf("strict: (%v, %v), want config.ErrDeprecatedDefault", emb, err)
			}

			// A configured model (yaml or explicit) is never deprecated.
			if emb, err = NewEmbedderFromConfig(tc.cfg("pinned-model")); err != nil || emb.ModelName() != tc.provider+":pinned-model" {
				t.Fatalf("strict with a configured model: (%v, %v)", emb, err)
			}

			// The env var reaches the config through LoadEmbedConfigFromEnv.
			t.Setenv("AILANG_EMBED_PROVIDER", tc.provider)
			t.Setenv(tc.env, "env-model")
			loaded := LoadEmbedConfigFromEnv()
			var got string
			switch tc.provider {
			case "openai":
				loaded.OpenAI.APIKey = "k"
				got = loaded.OpenAI.Model
			case "gemini":
				loaded.Gemini.APIKey = "k"
				got = loaded.Gemini.Model
			}
			if got != "env-model" {
				t.Fatalf("%s=env-model: LoadEmbedConfigFromEnv model = %q", tc.env, got)
			}
			if emb, err = NewEmbedderFromConfig(loaded); err != nil || emb.ModelName() != tc.provider+":env-model" {
				t.Fatalf("strict with %s set: (%v, %v)", tc.env, emb, err)
			}
		})
	}
}

// LoadEmbedConfigFromEnv cannot return an error, so it must NOT fill the
// model: the deprecation decision belongs to NewEmbedderFromConfig, which can.
func TestLoadEmbedConfigFromEnv_LeavesCloudModelsEmpty(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))
	t.Setenv(EnvEmbedOpenAIModel, "")
	t.Setenv(EnvEmbedGeminiModel, "")
	cfg := LoadEmbedConfigFromEnv()
	if cfg.OpenAI.Model != "" || cfg.Gemini.Model != "" {
		t.Fatalf("models pre-filled (openai=%q gemini=%q): a value here bypasses the deprecation gate", cfg.OpenAI.Model, cfg.Gemini.Model)
	}
}
