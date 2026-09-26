package coordinator

// The variant -> executor-CLI table.
//
// This is the coordinator's copy of the dispatcher's providersForVariant.
// ProviderForVariant reads it to derive an agent's executor from its image, so
// the two must not drift; a drift arm on the cloudrun side compares them.
//
// The copy exists because internal/dispatch/cloudrun already imports this
// package's types, and the reverse edge would be a cycle.
//
// A runtime audit of provider/variant mismatches lived here briefly. It was
// deleted when the provider became derived: a mismatch is no longer
// constructible, so the audit could never fire, and a control that cannot trip
// is worse than none — it reads as coverage.

var variantProviders = map[string][]string{
	"":          {"claude"},
	"default":   {"claude"},
	"go":        {"claude"},
	"codex":     {"codex"},
	"codex-go":  {"codex"},
	"gemini":    {"gemini"},
	"gemini-go": {"gemini"},
	"opencode":  {"opencode"},
	"pi":        {"pi"},
	"pi-go":     {"pi"},
	"motoko":    {"motoko"},
	"eval":      nil, // agent-eval carries every CLI
	"eval-go":   nil, //   ditto (FROM agent-eval)
}

// VariantProviders exposes this package's copy of the variant/provider table so
// the dispatcher's own test can prove the two have not drifted. The comparison
// lives on that side because internal/dispatch/cloudrun already imports this
// package; the reverse edge would be a cycle, which is also why the copy exists.
func VariantProviders() map[string][]string {
	out := make(map[string][]string, len(variantProviders))
	for k, v := range variantProviders {
		if v == nil {
			out[k] = nil
			continue
		}
		out[k] = append([]string(nil), v...)
	}
	return out
}
