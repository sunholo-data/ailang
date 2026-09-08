package modelreg

import (
	"fmt"
	"strings"
)

// OriginVendor resolves model origin, never treating a router as an independent
// model vendor. Unknown origin needs an explicit registry fact before judging.
func (c *ModelsConfig) OriginVendor(name string) (string, error) {
	m, err := c.GetModel(name)
	if err != nil {
		return "", err
	}
	v := strings.ToLower(strings.TrimSpace(m.ModelVendor))
	if v == "" {
		switch strings.ToLower(m.Provider) {
		case "openai", "anthropic", "google", "gemini", "vertex", "zai":
			v = strings.ToLower(m.Provider)
		case "openrouter":
			prefix, model, ok := strings.Cut(strings.TrimPrefix(m.APIName, "openrouter/"), "/")
			if ok && model != "" && prefix != "openrouter" {
				v = strings.ToLower(prefix)
			}
		}
	}
	switch v {
	case "gemini", "vertex":
		v = "google"
	case "zai", "z.ai":
		v = "z-ai"
	}
	if v == "" || v == "openrouter" || v == "ollama" {
		return "", fmt.Errorf("model %q has unknown origin vendor; declare model_vendor in the registry", name)
	}
	for _, ch := range v {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return "", fmt.Errorf("model %q has invalid model_vendor %q", name, v)
		}
	}
	return v, nil
}

// DispatchOriginVendor rejects known wire-route contradictions before a role
// can use registry identity as independence evidence.
func (c *ModelsConfig) DispatchOriginVendor(name string) (string, error) {
	vendor, err := c.OriginVendor(name)
	if err != nil {
		return "", err
	}
	m, err := c.GetModel(name)
	if err != nil {
		return "", err
	}
	if m.AgentCLI == nil || *m.AgentCLI == "" {
		return vendor, nil
	}
	cli, wire, err := c.GetExecutorForModel(name)
	if err != nil {
		return "", err
	}
	var wireVendor string
	switch cli {
	case "claude":
		wireVendor = "anthropic"
	case "codex":
		wireVendor = "openai"
	case "pi":
		provider, id, ok := strings.Cut(wire, "/")
		if !ok || id == "" {
			return "", fmt.Errorf("model %q requires a qualified Pi wire route", name)
		}
		probe := &ModelsConfig{Models: map[string]ModelConfig{"wire": {Provider: provider, APIName: id}}}
		wireVendor, err = probe.OriginVendor("wire")
		if err != nil {
			return "", fmt.Errorf("model %q wire origin unresolved: %w", name, err)
		}
	default:
		return vendor, nil // Unsupported harnesses are rejected by dispatch admission.
	}
	if vendor != wireVendor {
		return "", fmt.Errorf("model %q origin %q conflicts with %s wire origin %q", name, vendor, cli, wireVendor)
	}
	return vendor, nil
}
