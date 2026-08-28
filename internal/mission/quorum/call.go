package quorum

import (
	"context"
	"errors"
	"fmt"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"os"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/ai/ollama"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// ErrUnknownModel is returned by ResolveCaller when the model id is not present
// in models.yml. Callers use errors.Is to report a semantically correct
// absence reason ("unknown-model") rather than lumping it under "auth".
var ErrUnknownModel = errors.New("model not in models.yml")

// googleEnvMu serializes the process-global os.Setenv of GOOGLE_CLOUD_PROJECT
// below. RunQuorum resolves reviewers in PARALLEL, so two Google reviewers with
// different gcp_project values could otherwise race on this shared env var (a
// latent data race + a wrong-project mutation). The mutation is process-global
// because the Vertex ADC client reads GOOGLE_CLOUD_PROJECT from the environment;
// we serialize rather than restructure that contract.
var googleEnvMu sync.Mutex

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

	var provider ai.Provider
	switch ai.ProviderFromString(mc.Provider) {
	case ai.ProviderOpenAI:
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, nil, fmt.Errorf("reviewer %q needs OPENAI_API_KEY (openai provider) — not set", modelID)
		}
		provider = openai.NewClient(apiKey)

	case ai.ProviderGoogle:
		// Vertex ADC path — the design doc's flagged-but-mitigated route.
		// Export the model's gcp_project so NewVertexAIClient/ADC resolves the
		// right project on a rig where GOOGLE_CLOUD_PROJECT is unset. We do
		// NOT read GEMINI_API_KEY (absent on the rig) — that is the whole point.
		if mc.GCPProject != "" {
			// Set for this process so the ADC client picks up the project.
			// Serialized: RunQuorum fans out reviewers in parallel and this env
			// var is process-global (see googleEnvMu). Re-check inside the lock
			// so we only mutate when still unset.
			googleEnvMu.Lock()
			if os.Getenv("GOOGLE_CLOUD_PROJECT") == "" {
				_ = os.Setenv("GOOGLE_CLOUD_PROJECT", mc.GCPProject)
			}
			googleEnvMu.Unlock()
		}
		client, gerr := gemini.NewVertexAIClient(mc.GCPProject)
		if gerr != nil {
			return nil, nil, fmt.Errorf("reviewer %q needs Vertex ADC (gemini provider, gcp_project=%q) — %w", modelID, mc.GCPProject, gerr)
		}
		provider = client

	case ai.ProviderOllama:
		// Ollama Cloud reviewer (M-OLLAMA-CLOUD-PROVIDER). A third vendor for the
		// quorum: gpt5-6-sol is OpenAI and gemini-3-1-pro is Google, so a
		// `-cloud`-suffixed ollama model adds an independent prior without
		// touching the local GPU — the daemon proxies it to ollama.com.
		//
		// There is NO API key to check here: inference rides the DEVICE key
		// registered by `ollama signin`, and OLLAMA_API_KEY is only for
		// ollama.com's own /api/usage, which the local daemon does not proxy.
		// Same Principle-2 posture as the other lanes — refuse loudly if the
		// daemon is unreachable rather than letting a reviewer silently vanish
		// (an absent reviewer degrades the quorum to N-1, which must be a
		// reported fact, not an accident).
		client, oerr := ollama.NewClient()
		if oerr != nil {
			return nil, nil, fmt.Errorf("reviewer %q needs a reachable ollama daemon (ollama provider) — %w", modelID, oerr)
		}
		provider = client

	default:
		return nil, nil, fmt.Errorf("reviewer %q provider %q unsupported for quorum (want openai, google, or ollama)", modelID, mc.Provider)
	}

	maxTokens, err := reviewerMaxTokens(mc)
	if err != nil {
		return nil, nil, err
	}
	handler := ai.NewHandler(provider, mc.APIName, ai.WithMaxTokens(maxTokens))
	return &handlerCaller{handler: handler, maxTokens: maxTokens}, mc, nil
}
