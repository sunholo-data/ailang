package effects

// Fixture-driven --ai-stub (#1497): `ailang run --ai-stub --ai-stub-fixtures
// FILE` replays recorded, provider-shaped responses keyed by the sha256 of
// the request, so an adapter's parsing can be tested headless, with no key,
// no network and no spend. A request with no fixture is a loud error — never
// a quiet fall back to the default `{"kind":"Wait"}` stub.
//
// File format (JSON, version 1) — documented in docs/docs/guides/ai-effect.mdx:
//
//	{
//	  "version": 1,
//	  "fixtures": [
//	    {"kind": "text",  "prompt": "hello", "response": "Hi there"},
//	    {"kind": "text",  "sha256": "<64 hex>", "response": "...",
//	     "tool_calls": [{"id": "c1", "name": "lookup", "arguments": "{}"}]},
//	    {"kind": "text",  "prompt": "flaky", "error": {"code": "RateLimit", "message": "slow down", "retryable": true}},
//	    {"kind": "image", "prompt": "a red cube", "base64": "iVBOR...", "mime_type": "image/png"},
//	    {"kind": "speech", "request": {"text": "Well met.", "voice": "Kore", "style": ""}, "pcm_base64": "AAABAA=="}
//	  ]
//	}
//
// Keys: an entry names its request either literally ("prompt") or by
// "sha256" (lowercase hex of the canonical request bytes, below) — exactly
// one of the two. Canonical request bytes per kind:
//
//	text  : the prompt string (call, callJson, callJsonSimple and their
//	        *Result variants; the schema is not part of the key) or, for
//	        step / stepWithCache / stepWithStream, the content of the LAST
//	        message in the conversation
//	image : the prompt string (callImage, callImageBase64; options ignored)
//	speech: the compact JSON object {"style":S,"text":T,"voice":V} — keys
//	        sorted, no spaces, UTF-8 unescaped (Python:
//	        json.dumps(d, sort_keys=True, separators=(",",":"), ensure_ascii=False));
//	        options are not part of the key. A speech entry may give the
//	        literal "request" object instead of "sha256".

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
)

// Fixture kinds.
const (
	fixtureKindText   = "text"
	fixtureKindImage  = "image"
	fixtureKindSpeech = "speech"
)

// fixtureFile is the on-disk shape.
type fixtureFile struct {
	Version  int            `json:"version"`
	Fixtures []fixtureEntry `json:"fixtures"`
}

type fixtureError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type fixtureToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type fixtureEntry struct {
	Kind   string  `json:"kind"`
	SHA256 string  `json:"sha256,omitempty"`
	Prompt *string `json:"prompt,omitempty"`
	// speech: the literal request, instead of "sha256"
	Request *SpeechFixtureRequest `json:"request,omitempty"`

	// text
	Response  *string           `json:"response,omitempty"`
	ToolCalls []fixtureToolCall `json:"tool_calls,omitempty"`
	// image
	Base64   string `json:"base64,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	// speech: s16le mono 24 kHz PCM, standard base64
	PCMBase64 string `json:"pcm_base64,omitempty"`
	// any kind: replay a typed provider failure
	Error *fixtureError `json:"error,omitempty"`

	decoded []byte // image or PCM bytes, decoded at load
}

// SpeechFixtureRequest is a callSpeech request as a fixture names it.
type SpeechFixtureRequest struct {
	Text  string `json:"text"`
	Voice string `json:"voice"`
	Style string `json:"style"`
}

// SpeechFixtureCanonical is the canonical request string for a callSpeech:
// compact JSON with sorted keys and no HTML escaping.
func SpeechFixtureCanonical(text, voice, style string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	// A map marshals with sorted keys; Encode cannot fail on strings.
	_ = enc.Encode(map[string]string{"style": style, "text": text, "voice": voice})
	return strings.TrimSuffix(buf.String(), "\n")
}

// StubFixtures is a loaded, validated fixture file.
type StubFixtures struct {
	path    string
	entries map[string]map[string]*fixtureEntry // kind → sha256 → entry
}

// FixtureKey is the sha256 hex of a canonical request — what a fixture's
// "sha256" field holds and what a miss error prints.
func FixtureKey(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

// LoadStubFixtures reads and validates a fixture file. Every problem is
// reported with the entry index so a broken file never half-loads.
func LoadStubFixtures(path string) (*StubFixtures, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("--ai-stub-fixtures: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var f fixtureFile
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("--ai-stub-fixtures %s: invalid JSON: %w", path, err)
	}
	if f.Version != 1 {
		return nil, fmt.Errorf("--ai-stub-fixtures %s: \"version\" must be 1, got %d", path, f.Version)
	}
	sf := &StubFixtures{path: path, entries: map[string]map[string]*fixtureEntry{}}
	for i := range f.Fixtures {
		e := &f.Fixtures[i]
		key, err := sf.validate(e)
		if err != nil {
			return nil, fmt.Errorf("--ai-stub-fixtures %s: fixtures[%d]: %w", path, i, err)
		}
		if sf.entries[e.Kind] == nil {
			sf.entries[e.Kind] = map[string]*fixtureEntry{}
		}
		if _, dup := sf.entries[e.Kind][key]; dup {
			return nil, fmt.Errorf("--ai-stub-fixtures %s: fixtures[%d]: duplicate %s fixture for sha256 %s", path, i, e.Kind, key)
		}
		sf.entries[e.Kind][key] = e
	}
	return sf, nil
}

// validate checks one entry and returns its key.
func (sf *StubFixtures) validate(e *fixtureEntry) (string, error) {
	key, err := e.key()
	if err != nil {
		return "", err
	}
	if e.Error != nil {
		if strings.TrimSpace(e.Error.Code) == "" {
			return "", fmt.Errorf("\"error\" needs a non-empty \"code\"")
		}
		if e.Response != nil || len(e.ToolCalls) > 0 || e.Base64 != "" || e.MimeType != "" || e.PCMBase64 != "" {
			return "", fmt.Errorf("an \"error\" fixture carries no response payload")
		}
		return key, nil
	}
	switch e.Kind {
	case fixtureKindText:
		if e.Response == nil {
			return "", fmt.Errorf("text fixture needs \"response\" (or \"error\")")
		}
		if e.Base64 != "" || e.MimeType != "" || e.PCMBase64 != "" {
			return "", fmt.Errorf("text fixture cannot carry \"base64\"/\"mime_type\"/\"pcm_base64\"")
		}
	case fixtureKindImage:
		if e.Base64 == "" || e.MimeType == "" {
			return "", fmt.Errorf("image fixture needs \"base64\" and \"mime_type\" (or \"error\")")
		}
		if e.Response != nil || len(e.ToolCalls) > 0 || e.PCMBase64 != "" {
			return "", fmt.Errorf("image fixture cannot carry \"response\"/\"tool_calls\"/\"pcm_base64\"")
		}
		b, err := base64.StdEncoding.DecodeString(e.Base64)
		if err != nil {
			return "", fmt.Errorf("image \"base64\" does not decode: %w", err)
		}
		e.decoded = b
	case fixtureKindSpeech:
		if e.PCMBase64 == "" {
			return "", fmt.Errorf("speech fixture needs \"pcm_base64\" (or \"error\")")
		}
		if e.Response != nil || len(e.ToolCalls) > 0 || e.Base64 != "" || e.MimeType != "" {
			return "", fmt.Errorf("speech fixture carries only \"pcm_base64\"")
		}
		b, err := base64.StdEncoding.DecodeString(e.PCMBase64)
		if err != nil {
			return "", fmt.Errorf("speech \"pcm_base64\" does not decode: %w", err)
		}
		if len(b)%2 != 0 {
			return "", fmt.Errorf("speech \"pcm_base64\" is %d bytes, not whole 16-bit samples", len(b))
		}
		e.decoded = b
	default:
		return "", fmt.Errorf("unknown \"kind\" %q (want text, image or speech)", e.Kind)
	}
	return key, nil
}

// key resolves the entry's sha256: given directly, or computed from the
// literal request.
func (e *fixtureEntry) key() (string, error) {
	hasSHA := e.SHA256 != ""
	hasPrompt := e.Prompt != nil
	if e.Request != nil {
		if e.Kind != fixtureKindSpeech {
			return "", fmt.Errorf("\"request\" is for speech fixtures; use \"prompt\"")
		}
		if hasSHA || hasPrompt {
			return "", fmt.Errorf("give exactly one of \"sha256\" or \"request\"")
		}
		return FixtureKey(SpeechFixtureCanonical(e.Request.Text, e.Request.Voice, e.Request.Style)), nil
	}
	if hasPrompt && e.Kind == fixtureKindSpeech {
		return "", fmt.Errorf("a speech fixture is keyed by \"request\" {text, voice, style} or \"sha256\", not \"prompt\"")
	}
	switch {
	case hasSHA && hasPrompt:
		return "", fmt.Errorf("give exactly one of \"sha256\" or \"prompt\", not both")
	case hasSHA:
		if len(e.SHA256) != 64 || strings.ToLower(e.SHA256) != e.SHA256 {
			return "", fmt.Errorf("\"sha256\" must be 64 lowercase hex characters")
		}
		if _, err := hex.DecodeString(e.SHA256); err != nil {
			return "", fmt.Errorf("\"sha256\" is not hex: %w", err)
		}
		return e.SHA256, nil
	case hasPrompt:
		return FixtureKey(*e.Prompt), nil
	}
	return "", fmt.Errorf("needs \"sha256\" or \"prompt\"")
}

// lookup returns the entry for canonical, or the miss error.
func (sf *StubFixtures) lookup(kind, canonical string) (*fixtureEntry, error) {
	key := FixtureKey(canonical)
	if e, ok := sf.entries[kind][key]; ok {
		if e.Error != nil {
			return nil, ai.NewAIError(e.Error.Code, e.Error.Message, e.Error.Retryable)
		}
		return e, nil
	}
	return nil, ai.NewAIError(ai.CodeInternal, fmt.Sprintf(
		"E_AI_STUB_FIXTURE_MISS: no %s fixture in %s for sha256 %s (request %q); add {\"kind\":%q,\"sha256\":%q,...}",
		kind, sf.path, key, truncateForMiss(canonical), kind, key), false)
}

func truncateForMiss(s string) string {
	const max = 80
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// FixtureAIHandler is the --ai-stub handler when --ai-stub-fixtures is set.
// It embeds the default stub only so the AIHandler interface stays satisfied
// if a method is added; every method the fixture format covers is overridden
// and misses loudly.
type FixtureAIHandler struct {
	*StubAIHandler
	fixtures *StubFixtures
}

// NewFixtureAIHandler wraps loaded fixtures as an AIHandler.
func NewFixtureAIHandler(f *StubFixtures) *FixtureAIHandler {
	return &FixtureAIHandler{StubAIHandler: NewStubAIHandler(), fixtures: f}
}

func (h *FixtureAIHandler) text(canonical string) (string, error) {
	e, err := h.fixtures.lookup(fixtureKindText, canonical)
	if err != nil {
		return "", err
	}
	return *e.Response, nil
}

// Call replays the text fixture keyed by the prompt.
func (h *FixtureAIHandler) Call(input string) (string, error) { return h.text(input) }

// CallJson replays the text fixture keyed by the prompt (schema not keyed).
func (h *FixtureAIHandler) CallJson(input, _ string) (string, error) { return h.text(input) }

// CallImage writes the image fixture keyed by the prompt to outputPath.
func (h *FixtureAIHandler) CallImage(prompt, outputPath, _ string) (string, error) {
	e, err := h.fixtures.lookup(fixtureKindImage, prompt)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", fmt.Errorf("stub: failed to create directory: %w", err)
	}
	if err := os.WriteFile(outputPath, e.decoded, 0o644); err != nil {
		return "", fmt.Errorf("stub: failed to write image: %w", err)
	}
	return outputPath, nil
}

// CallImageBase64 returns the image fixture keyed by the prompt.
func (h *FixtureAIHandler) CallImageBase64(prompt, _ string) (string, error) {
	e, err := h.fixtures.lookup(fixtureKindImage, prompt)
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(map[string]string{"base64": e.Base64, "mime_type": e.MimeType})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Step replays the text fixture keyed by the last message's content.
func (h *FixtureAIHandler) Step(model string, messages []ai.Message, _ []ai.ToolSchema) (*ai.Response, error) {
	last := ""
	if len(messages) > 0 {
		last = messages[len(messages)-1].Content
	}
	e, err := h.fixtures.lookup(fixtureKindText, last)
	if err != nil {
		return nil, err
	}
	resp := &ai.Response{Text: *e.Response, FinishReason: "stop", Model: model}
	for _, tc := range e.ToolCalls {
		resp.ToolCalls = append(resp.ToolCalls, ai.ToolCall{ID: tc.ID, Name: tc.Name, Arguments: tc.Arguments})
	}
	if len(resp.ToolCalls) > 0 {
		resp.FinishReason = "tool_calls"
	}
	return resp, nil
}

// CallSpeech replays the speech fixture keyed by {style, text, voice}.
func (h *FixtureAIHandler) CallSpeech(text, voice, style, options string) ([]byte, error) {
	if _, err := ai.ParseSpeechOptions(options); err != nil {
		return nil, err
	}
	e, err := h.fixtures.lookup(fixtureKindSpeech, SpeechFixtureCanonical(text, voice, style))
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), e.decoded...), nil
}

// StepWithCache ignores the cache hints, as the default stub does.
func (h *FixtureAIHandler) StepWithCache(model string, messages []ai.Message, tools []ai.ToolSchema, _ []ai.CacheBreakpoint) (*ai.Response, error) {
	return h.Step(model, messages, tools)
}

// StepWithStream replays Step and fires one ContentDelta + Usage.
func (h *FixtureAIHandler) StepWithStream(model string, messages []ai.Message, tools []ai.ToolSchema, _ []ai.CacheBreakpoint, onChunk func(ai.StreamChunk)) (*ai.Response, error) {
	resp, err := h.Step(model, messages, tools)
	if err != nil {
		return nil, err
	}
	if onChunk != nil {
		if resp.Text != "" {
			onChunk(ai.StreamContentDelta{Text: resp.Text})
		}
		onChunk(ai.StreamUsage{})
	}
	return resp, nil
}
