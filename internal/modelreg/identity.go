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
