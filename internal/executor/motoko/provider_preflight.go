package motoko

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/chatgpt"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
)

// D1 (M-MOTOKO-FMT-REMEASUREMENT-INSTRUMENT, design doc §12.2/§12.4): the
// motoko health check can no longer make a provider-credential decision,
// because HealthCheck sees no task and therefore no lane model (it is the
// once-cached shared executor method; e.model is a single process-global
// default that is never the 17 distinct lanes' models). The refusal moved
// HERE, to the per-task choke point: ExecuteStreaming, through which ALL
// motoko work passes (CanaryCheck and agent_runner_multi both reach it via
// Execute). Placed there, the check is strictly DOWNSTREAM of repo discovery
// (HealthCheck's `motoko --version` query already populated e.motokoRepo),
// which the design doc §12.3 requires so the profile cannot degrade to
// extensions.order=[] and drop the fmt extension that is the treatment.
//
// The condition is expressed on the RESOLVED PROVIDER, never on a literal
// env-var name: ai.GuessProvider(model) returns the provider, and
// ai.EnvVarForProvider(provider) returns that provider's required key or ""
// for providers that need none (ollama). One check covers ollama / OpenAI /
// Anthropic / Google / OpenRouter.
func requireProviderCredential(model string) error {
	provider := ai.GuessProvider(model)

	// Unresolvable-model guard (plan §3.2, T-B6). Measured: ai.GuessProvider
	// returns the EMPTY ProviderType (`""`) for any model it cannot classify —
	// ai.GuessProvider("") yields "" (internal/ai/config.go: the terminal
	// `return ""`), and the vendor/model + prefix checks all miss. KEYING ON
	// THIS OBSERVED VALUE: an empty provider string can only mean "could not
	// resolve", and never "a real provider that happens to need no key",
	// because the only no-key provider (ollama) is returned as the NON-empty
	// ProviderType "ollama" by the explicit ollama-prefix branch above the
	// fallthrough. So `provider == ""` is a contradiction by construction and
	// must be a loud refusal, not a silent pass: an admitted unresolvable
	// model could later bind to OpenRouter at runtime and burn wall-clock
	// before failing — exactly the false fail-fast O4 set D1 up to kill.
	if model == "" || provider == "" {
		return fmt.Errorf("could not resolve provider for motoko model %q — refusing run (no provider to bind a credential to)",
			model)
	}

	// The ChatGPT subscription lane carries no env key: its credential is the
	// codex CLI's login, which the child reads from $HOME. Refuse here rather
	// than 401 inside motoko.
	if provider == ai.ProviderChatGPT || provider == ai.ProviderCodex {
		if _, err := chatgpt.LoadCredential(); err != nil {
			return fmt.Errorf("motoko model %q runs on the ChatGPT subscription lane: %w — refusing run", model, err)
		}
		return nil
	}

	envVar := ai.EnvVarForProvider(provider)
	if envVar == "" {
		// Provider needs no credential (example: ollama local). Admit.
		return nil
	}
	if config.Raw(envVar) == "" {
		return fmt.Errorf("motoko model %q resolves to provider %q, which requires %s (unset) — refusing run",
			model, provider.String(), envVar)
	}
	return nil
}

// motokoAuthLane is the billing lane the run's model actually used. The
// ChatGPT lane is a subscription, so its priced cost is list-price-equivalent,
// never metered spend.
func motokoAuthLane(model string) executor.AuthLane {
	provider := ai.GuessProvider(model)
	if provider == ai.ProviderChatGPT || provider == ai.ProviderCodex {
		return executor.AuthLaneSubscription
	}
	return executor.AuthLaneForModel(model)
}
