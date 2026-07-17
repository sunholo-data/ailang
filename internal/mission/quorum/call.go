package quorum

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
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
	handler *ai.Handler
}

func (c *handlerCaller) CallJSON(sysPrompt, userPrompt, schema string) (string, *ai.Response, error) {
	// We drive the provider directly (not Handler.CallJson) so we can attach
	// the reviewer system prompt per-call AND recover the full *ai.Response
	// for cost accounting in one round-trip.
	resp, err := c.handler.Provider().Generate(context.Background(), &ai.Request{
		Model:          c.handler.Model(),
		SystemPrompt:   sysPrompt,
		UserPrompt:     userPrompt,
		MaxTokens:      reviewMaxTokens,
		ResponseFormat: "json",
		ResponseSchema: schema,
	})
	if err != nil {
		return "", nil, err
	}
	return strings.TrimSpace(resp.Text), resp, nil
}

// reviewMaxTokens caps reviewer output. Reviews are short structured JSON, but
// the frontier reviewers are REASONING models whose thinking tokens count
// against maxOutputTokens (Gemini 3.x especially — "2x reasoning"). At the old
// 4096 cap the thinking trace could consume the whole budget and truncate the
// JSON answer mid-object, so the review was silently dropped as
// "malformed JSON" and the quorum degraded to N-1 (observed live, mission iter
// 42: gemini-3-1-pro finishReason=MAX_TOKENS on a substantive objection). 16384
// leaves genuine thinking headroom above the ~1-2K structured verdict. Cost is
// billed per ACTUAL token emitted (a short verdict stays in cents) and the
// pre-flight budget cap uses a fixed expectedOutputTokens estimate, so raising
// this ceiling does not change budget gating. A residual truncation now fails
// LOUDLY via resp.FinishReason == "length" (see runReviewerWith), never silently.
const reviewMaxTokens = 16384

// ResolveCaller builds a JSONCaller for a models.yml model id, wiring the
// correct provider + auth from the shipped registry:
//   - openai  → OPENAI_API_KEY (the rig has it)
//   - google  → Vertex ADC (env_var is "" in models.yml; GEMINI_API_KEY is NOT
//     consulted — that is the rig-absent var the design doc flagged), with the
//     model's gcp_project exported so ADC resolves the right project.
//
// It refuses (hard error, Principle 2) when the required auth is unavailable
// rather than silently falling back to a different provider/model.
func ResolveCaller(modelID string) (JSONCaller, *eval_harness.ModelConfig, error) {
	if err := eval_harness.InitModelsConfig(); err != nil {
		return nil, nil, fmt.Errorf("load models.yml: %w", err)
	}
	mc, err := eval_harness.GlobalModelsConfig.GetModel(modelID)
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

	default:
		return nil, nil, fmt.Errorf("reviewer %q provider %q unsupported for quorum (want openai or google)", modelID, mc.Provider)
	}

	handler := ai.NewHandler(provider, mc.APIName, ai.WithMaxTokens(reviewMaxTokens))
	return &handlerCaller{handler: handler}, mc, nil
}
