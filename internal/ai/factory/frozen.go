package factory

import (
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/chatgpt"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/config"
)

type frozenConfiguration struct {
	env         map[string]string
	anthropic   func() (ai.AnthropicCredential, error)
	chatPath    string
	chatPathErr error
	vertex      gemini.ClientOption
}

// FreezeConfiguration snapshots environment configuration and credential source
// selection for a deferred client. It reads no credential file, invokes no ADC
// discovery and performs no network I/O. Secrets stay in memory, never logs.
// Eager callers omitting this option retain existing dynamic environment behavior.
func FreezeConfiguration(extraVars ...string) Option {
	f := &frozenConfiguration{env: make(map[string]string), anthropic: ai.PrepareAnthropicCredential(), vertex: gemini.WithPreparedConfiguration()}
	names := append([]string{
		config.EnvOpenAIAPIKey, config.EnvOpenAIBaseURL, config.EnvOpenRouterAPIKey,
		config.EnvGeminiAPIKey, config.EnvGoogleAPIKey, config.EnvAnthropicAPIKey,
		config.EnvAINoADC, config.EnvLyceumBaseURL, config.EnvZAIBaseURL,
		config.EnvOllamaHost, "LYCEUM_API_KEY", "ZAI_API_KEY",
	}, extraVars...)
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			f.env[name] = config.Raw(name)
		}
	}
	f.chatPath, f.chatPathErr = chatgpt.AuthPath()
	if f.chatPathErr == nil {
		f.chatPath, f.chatPathErr = filepath.Abs(f.chatPath)
	}
	return func(o *options) { o.frozen = f }
}

func (o *options) env(name string) string {
	if o.frozen != nil {
		return o.frozen.env[name]
	}
	return config.Raw(name)
}
func (o *options) envTrim(name string) string { return strings.TrimSpace(o.env(name)) }
func (o *options) endpoint(name, fallback string) string {
	if v := o.envTrim(name); v != "" {
		return v
	}
	return fallback
}
