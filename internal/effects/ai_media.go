package effects

// AI.callMediaResult (M-STD-AI-GEMINI-FILE-SLICING, #2458): single-shot media
// call returning Result[MediaCallResult, AIError] with text + token usage.

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// AIMediaHandler is implemented by AI handlers that support single-shot typed
// media calls with token usage reporting.
type AIMediaHandler interface {
	CallMediaResult(prompt string, media []ai.MediaPart, options string) (*ai.Response, error)
}

// CallMediaResult dispatches a single-shot typed media call to the configured
// AI handler.
func (c *AIContext) CallMediaResult(prompt string, media []ai.MediaPart, options string) (*ai.Response, error) {
	resolved, initErr := c.ready()
	if initErr != nil {
		return nil, initErr
	}
	if resolved != c {
		return resolved.CallMediaResult(prompt, media, options)
	}
	if c.handler == nil {
		return nil, ErrNoAIHandler
	}
	mh, ok := c.handler.(AIMediaHandler)
	if !ok {
		return nil, ai.NewAIError(ai.CodeCapabilityNotSupported,
			fmt.Sprintf("callMediaResult: the configured AI handler (%T) does not support typed media calls", c.handler), false)
	}
	return mh.CallMediaResult(prompt, media, options)
}

// CallMediaResult provides a deterministic stub response for --ai-stub
// (validating media parts and options first).
func (h *StubAIHandler) CallMediaResult(prompt string, media []ai.MediaPart, options string) (*ai.Response, error) {
	parts := make([]ai.MediaPart, len(media))
	copy(parts, media)
	if err := ai.ValidateMediaParts(parts); err != nil {
		return nil, err
	}
	if _, err := ai.ParseMediaCallOptions(options); err != nil {
		return nil, err
	}
	return &ai.Response{
		Text:         h.defaultResponse,
		InputTokens:  10 + len(parts)*5,
		OutputTokens: 5,
		TotalTokens:  15 + len(parts)*5,
	}, nil
}

func init() {
	RegisterOp("AI", "callMediaResult", aiCallMediaResult)
}

// aiCallMediaResult implements AI.callMediaResult(prompt, media, options)
// -> Result[MediaCallResult, AIError].
func aiCallMediaResult(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callMediaResult: expected 3 arguments, got %d", len(args))
	}
	promptVal, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callMediaResult: expected string prompt, got %T", args[0])
	}
	mediaList, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callMediaResult: expected [MediaPart] media, got %T", args[1])
	}
	optionsVal, ok := args[2].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("E_AI_TYPE_ERROR: callMediaResult: expected string options, got %T", args[2])
	}
	if ctx.AI == nil {
		return makeAIErrorResultRecord(ai.NewAIError(ai.CodeProviderNotFound, ErrNoAIHandler.Error(), false)), nil
	}

	media, decErr := decodeMediaParts(mediaList)
	if decErr != nil {
		aiErr := classifyOpError(decErr)
		ctx.RecordAIEffect("callMediaResult", []string{truncateForTrace(promptVal.Value)}, fmt.Sprintf(errResultPrefix, aiErr.Code), ctx.AI.LastRoutingMetadata())
		return makeAIErrorResultRecord(aiErr), nil
	}
	if _, optErr := ai.ParseMediaCallOptions(optionsVal.Value); optErr != nil {
		aiErr := classifyOpError(optErr)
		ctx.RecordAIEffect("callMediaResult", []string{truncateForTrace(promptVal.Value)}, fmt.Sprintf(errResultPrefix, aiErr.Code), ctx.AI.LastRoutingMetadata())
		return makeAIErrorResultRecord(aiErr), nil
	}

	resp, err := ctx.AI.CallMediaResult(promptVal.Value, media, optionsVal.Value)
	if err != nil {
		aiErr := classifyOpError(err)
		ctx.RecordAIEffect("callMediaResult", []string{truncateForTrace(promptVal.Value)}, fmt.Sprintf(errResultPrefix, aiErr.Code), ctx.AI.LastRoutingMetadata())
		return makeAIErrorResultRecord(aiErr), nil
	}

	ctx.RecordAIEffect("callMediaResult", []string{truncateForTrace(promptVal.Value)}, truncateForTrace(resp.Text), ctx.AI.LastRoutingMetadata())
	return makeOkMediaCallResult(resp), nil
}
