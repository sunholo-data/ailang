package quorum

import (
	"context"
	"errors"
	"fmt"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/factory"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// ErrUnknownModel is returned by ResolveCaller when the model id is not present
// in models.yml. Callers use errors.Is to report a semantically correct
// absence reason ("unknown-model") rather than lumping it under "auth".
var ErrUnknownModel = errors.New("model not in models.yml")

// JSONCaller is the minimal provider surface the reviewer needs: a single
// structured-JSON call plus token/cost details for budget accounting. Both
// *ai.Handler (production) and the test stub satisfy it. Keeping the surface
// this small is what lets us reuse the shipped handler without dragging in the
// full effects.AIHandler machinery.
type JSONCaller interface {
	// CallJSON sends the prompt configured for JSON structured output and
	// returns the raw JSON text plus the resolved response (for token/cost).
	CallJSON(systemPrompt, userPrompt, schema string) (string, *ai.Response, error)
}

// handlerCaller adapts an *ai.Handler to JSONCaller. It uses
// GenerateWithDetails so we recover token counts + provider-reported CostUSD
// for the budget ledger.
type handlerCaller struct {
	handler   *ai.Handler
	maxTokens int
}

func (c *handlerCaller) CallJSON(sysPrompt, userPrompt, schema string) (string, *ai.Response, error) {
	// We drive the provider directly (not Handler.CallJson) so we can attach
	// the reviewer system prompt per-call AND recover the full *ai.Response
	// for cost accounting in one round-trip.
	resp, err := c.handler.Provider().Generate(context.Background(), &ai.Request{
		Model:          c.handler.Model(),
		SystemPrompt:   sysPrompt,
		UserPrompt:     userPrompt,
		MaxTokens:      c.maxTokens,
		ResponseFormat: "json",
		ResponseSchema: schema,
	})
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(resp.Text), resp, nil
}

// reviewerMaxTokens returns the reviewer's output budget: the model's FULL
// declared strength from models.yml, never a policy cap of our own.
//
// History, because this number has been wrong twice in the same direction.
// Reviews are short structured JSON, but the frontier reviewers are REASONING
// models whose thinking tokens count against maxOutputTokens (Gemini 3.x
// especially — "2x reasoning"). At 4096 the thinking trace consumed the whole
// budget and truncated the JSON mid-object, so the review was dropped as
// "malformed JSON" and the quorum silently degraded to N-1 (mission iter 42:
// gemini-3-1-pro finishReason=MAX_TOKENS on a substantive objection). Raising
// it to a hardcoded 16384 fixed that case and left the same trap set one octave
// up — a deeper thinker still hits a ceiling nobody chose per-model.
//
// Policy (2026-08-13): outside the eval harness, models run at full ability.
// Restricting thinking is an EVAL decision — evals equalise headroom so token
// counts are comparable between models (TestModels_CloudHeadroomEqualised).
// Nothing else has that excuse: a throttled reviewer is just a worse reviewer.
// Cost is billed per ACTUAL token emitted (a short verdict stays in cents) and
// the pre-flight cap uses a fixed expectedOutputTokens estimate, so this
// ceiling does not change budget gating.
//
// Fails LOUDLY (Principle 2) when the registry declares no budget: 0 would fall
// back to the ai.Handler's 4096 default and silently re-create the iter-42 bug.
func reviewerMaxTokens(mc *eval_harness.ModelConfig) (int, error) {
	if mc.MaxOutputTokens <= 0 {
		return 0, fmt.Errorf("reviewer %q declares no max_output_tokens in models.yml — refusing to fall back to the 4096 handler default, which truncates reasoning models mid-verdict", mc.APIName)
	}
	return mc.MaxOutputTokens, nil
}

// ResolveCaller builds a JSONCaller for a models.yml model id, wiring the
// correct provider + auth from the shipped registry:
//   - openai  → OPENAI_API_KEY (the rig has it)
//   - google  → Vertex ADC (env_var is "" in models.yml; GEMINI_API_KEY is NOT
//     consulted — that is the rig-absent var the design doc flagged), with the
//     model's gcp_project exported so ADC resolves the right project.
//   - ollama  → the local daemon, which proxies `-cloud`-suffixed models to
//     ollama.com signed with the device key. No API key is involved.
//
// It refuses (hard error, Principle 2) when the required auth is unavailable
// rather than silently falling back to a different provider/model.
func ResolveCaller(modelID string) (JSONCaller, *eval_harness.ModelConfig, error) {
	if err := eval_harness.InitModelsConfig(); err != nil {
		return nil, nil, fmt.Errorf("load models.yml: %w", err)
	}
	mc, err := modelreg.GlobalModelsConfig.GetModel(modelID)
	if err != nil {
		return nil, nil, fmt.Errorf("model %q not in models.yml: %w", modelID, ErrUnknownModel)
	}

	// Three vendors are admitted so the quorum's priors stay independent:
	// openai, google (Vertex ADC — the model's gcp_project is handed to the
	// factory EXPLICITLY, which makes that lane the only one tried; GEMINI_API_KEY,
	// absent on the rig, is never consulted), and ollama (the local daemon
	// proxies `-cloud` models to ollama.com on the device key from `ollama
	// signin`; there is no API key to check — M-OLLAMA-CLOUD-PROVIDER). Each
	// lane refuses loudly when its credential is missing rather than letting a
	// reviewer silently vanish: an absent reviewer degrades the quorum to N-1,
	// which must be a reported fact, not an accident.
	switch ai.ProviderFromString(mc.Provider) {
	case ai.ProviderOpenAI, ai.ProviderGoogle, ai.ProviderOllama:
	default:
		return nil, nil, fmt.Errorf("reviewer %q provider %q unsupported for quorum (want openai, google, or ollama)", modelID, mc.Provider)
	}
	provider, err := factory.NewProvider(mc.Provider, factory.WithGCPProject(mc.GCPProject))
	if err != nil {
		return nil, nil, fmt.Errorf("reviewer %q (%s provider) — %w", modelID, mc.Provider, err)
	}

	maxTokens, err := reviewerMaxTokens(mc)
	if err != nil {
		return nil, nil, err
	}
	handler := ai.NewHandler(provider, mc.APIName, ai.WithMaxTokens(maxTokens))
	return &handlerCaller{handler: handler, maxTokens: maxTokens}, mc, nil
}
