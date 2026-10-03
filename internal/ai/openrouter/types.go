// Package openrouter provides an OpenRouter API client implementing the ai.Provider interface.
//
// OpenRouter (https://openrouter.ai) is a unified gateway that fronts ~100 LLMs
// from many vendors behind an OpenAI-compatible Chat Completions API. This client
// is a thin Chat Completions adapter — it does not attempt to support OpenAI's
// Responses API or any vendor-specific extensions other than the OpenRouter-specific
// `cached_tokens` and `cost` fields surfaced in the response.
//
// The wire types are openai's Chat Completions types (the step path already
// reused openai's builders; M-V1-SIMPLIFY-S3 M4 made the Generate path do the
// same) extended by composition with OpenRouter's own fields. Embedding keeps
// the JSON field order of the OpenAI shape and appends the extensions, so a
// request with none of them set marshals byte-identically to a plain OpenAI
// body — the property the golden-body tests defend.
package openrouter

import "github.com/sunholo-data/ailang/internal/ai/openai"

// chatRequest is openai.ChatRequest plus OpenRouter's routing, reasoning and
// Broadcast-correlation fields. OpenRouter normalizes max_tokens across
// providers, so only MaxTokens is set (never MaxCompletionTokens).
type chatRequest struct {
	openai.ChatRequest
	// Provider carries OpenRouter's dynamic routing config. Translated from
	// ai.AIRoutingPolicy by translatePolicy; nil when the caller did not
	// supply a routing policy.
	Provider *providerField `json:"provider,omitempty"`
	// Reasoning caps hidden thinking tokens for reasoning models
	// (https://openrouter.ai/docs/use-cases/reasoning-tokens). Nil = provider
	// default. Set from ai.Request.Options["reasoning_max_tokens"] so
	// always-thinking models (e.g. z-ai/glm-5.2) keep content headroom inside
	// max_tokens instead of burning the whole budget on thought.
	Reasoning *reasoningField `json:"reasoning,omitempty"`

	// User, SessionID and Trace are OpenRouter Broadcast correlation fields
	// (https://openrouter.ai/docs/guides/features/broadcast). They do not
	// affect the completion; they travel with the trace OpenRouter pushes to
	// its configured destinations, which is what makes a broadcast span
	// joinable to the eval run that caused it.
	//
	// Every field is omitempty: a request that sets none must produce
	// byte-identical wire output to before they existed. Every OpenRouter call
	// in the project flows through these structs, so that is the property the
	// golden-body tests defend.
	//
	// Length caps are OpenRouter's: user <= 128 chars, session_id <= 256.
	// Over-cap values are rejected at construction rather than truncated —
	// a silently shortened correlation id joins to nothing.
	User      string         `json:"user,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Trace     map[string]any `json:"trace,omitempty"`

	// Modalities and ImageConfig are OpenRouter's image-output extensions
	// (M-OPENROUTER-IMAGE-OUTPUT, #1500): modalities ["image","text"] asks an
	// image-output model for an image; image_config carries the caller's
	// aspect_ratio / output_format. Both are set only on image requests and
	// are omitempty, so text bodies stay byte-identical.
	Modalities  []string     `json:"modalities,omitempty"`
	ImageConfig *imageConfig `json:"image_config,omitempty"`
}

// imageConfig is the subset of OpenRouter's provider-specific image_config
// this adapter forwards — only keys the caller set.
type imageConfig struct {
	AspectRatio  string `json:"aspect_ratio,omitempty"`
	OutputFormat string `json:"output_format,omitempty"`
}

// chatRequestWithParts is chatRequest with the messages array replaced by
// multimodal messages, used only when an image request carries reference
// images (#1496). The shallower Messages field shadows the embedded string-
// content one; key order moves, which no golden test pins for this shape.
type chatRequestWithParts struct {
	chatRequest
	Messages []partsMessage `json:"messages"`
}

// partsMessage is a chat message whose content is either a plain string
// (system) or an OpenAI-style content-parts array (user with images).
type partsMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// contentPart is one OpenAI-style multimodal content part.
type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

// reasoningField configures OpenRouter's normalized reasoning controls.
// MaxTokens and Effort are mutually exclusive per the OpenRouter contract;
// when both are requested, Effort wins (it is the vendor-documented dial —
// e.g. Kimi K3's Low/Standard/High/Max — while max_tokens is best-effort).
type reasoningField struct {
	MaxTokens int    `json:"max_tokens,omitempty"`
	Effort    string `json:"effort,omitempty"`
}

// Message, choice and structured-output shapes are OpenAI's verbatim.
type (
	chatMessage        = openai.ChatMessage
	chatResponseFormat = openai.ChatResponseFormat
	chatJSONSchema     = openai.ChatJSONSchema
)

// chatResponse is openai.ChatResponse with OpenRouter's extensions.
//
// Provider reports which underlying vendor served the request when routing
// is engaged (e.g. "Anthropic", "OpenAI"). Not all responses include it;
// absent → empty string. Usage shadows the embedded field so the extended
// usage block (cost, cached tokens) decodes.
type chatResponse struct {
	openai.ChatResponse
	Provider string    `json:"provider,omitempty"`
	Usage    chatUsage `json:"usage"`
}

// chatChoice is a completion choice — OpenAI's shape verbatim.
type chatChoice = openai.ChatChoice

// chatResponseDecoded is the decode target for Generate responses: the
// extended chatResponse with Choices shadowed by a shape whose message also
// carries OpenRouter's generated images (choices[].message.images[], #1500).
// chatResponse itself stays OpenAI-shaped so test servers marshal it as before.
type chatResponseDecoded struct {
	chatResponse
	Choices []decodedChoice `json:"choices"`
}

type decodedChoice struct {
	Index        int              `json:"index"`
	Message      assistantMessage `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

// assistantMessage is openai.ChatMessage plus generated images. Each image is
// {"type":"image_url","image_url":{"url":"data:<mime>;base64,..."}}.
type assistantMessage struct {
	Role    string           `json:"role"`
	Content string           `json:"content"`
	Images  []assistantImage `json:"images,omitempty"`
}

type assistantImage struct {
	Type     string   `json:"type,omitempty"`
	ImageURL imageURL `json:"image_url"`
}

// chatUsage is openai.ChatUsage plus OpenRouter's cost reporting:
//   - cost — total inference cost in USD as a float (sum of upstream + markup)
//   - cost_details.upstream_inference_cost — upstream-only portion (informational)
//
// prompt_tokens_details.cached_tokens (input tokens served from prompt cache)
// is already on the OpenAI shape.
type chatUsage struct {
	openai.ChatUsage
	Cost        float64 `json:"cost,omitempty"` // OpenRouter reports total cost as a float
	CostDetails struct {
		UpstreamInferenceCost float64 `json:"upstream_inference_cost,omitempty"`
	} `json:"cost_details,omitempty"`
}
