package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

type streamRecordPolicy struct {
	record   bool
	failLoud bool
}

// aiStreamCore is the single validation, decode, dispatch, delivery, and trace
// implementation shared by both streaming operations. Public wrappers retain
// responsibility for constructing their intentionally different return shapes.
func aiStreamCore(ctx *EffContext, args []eval.Value, opName string, policy streamRecordPolicy) ([]eval.Value, *ai.Response, *ai.AIError, error) {
	model, messagesArg, toolsArg, breakpointsArg, onChunkFn, err := validateStreamArgs(args, opName)
	if err != nil {
		return nil, nil, nil, err
	}
	if ctx.AI == nil {
		return nil, nil, ai.NewAIError(ai.CodeProviderNotFound, ErrNoAIHandler.Error(), false), nil
	}
	if ctx.FnCaller == nil {
		return nil, nil, ai.NewAIError(ai.CodeInternal, opName+": FnCaller not wired (evaluator integration missing)", false), nil
	}

	messages, conversionErr := decodeMessages(messagesArg)
	if conversionErr != nil {
		return nil, nil, schemaAIError(conversionErr), nil
	}
	tools, conversionErr := decodeToolSchemas(toolsArg)
	if conversionErr != nil {
		return nil, nil, schemaAIError(conversionErr), nil
	}
	breakpoints, conversionErr := decodeCacheBreakpoints(breakpointsArg)
	if conversionErr != nil {
		return nil, nil, schemaAIError(conversionErr), nil
	}

	var recorded []eval.Value
	if policy.record {
		recorded = make([]eval.Value, 0, 16)
	}
	providerChunks := 0
	onChunk := func(chunk ai.StreamChunk) {
		providerChunks++
		encoded := encodeStreamChunk(chunk)
		if encoded == nil {
			return
		}
		if policy.record {
			recorded = append(recorded, encoded)
		}
		if _, callErr := ctx.FnCaller(onChunkFn, encoded); callErr != nil {
			ctx.RecordAIEffect(opName+".callback",
				[]string{fmt.Sprintf("chunk:%d", providerChunks)},
				fmt.Sprintf(errResultPrefix, callErr.Error()), nil)
		}
	}

	resp, stepErr := ctx.AI.StepWithStream(model.Value, messages, tools, breakpoints, onChunk)
	var aiErr *ai.AIError
	if stepErr != nil {
		aiErr = classifyOpError(stepErr)
	}
	recordStreamTerminalTrace(ctx, opName, model.Value, len(messages), len(tools), len(breakpoints), providerChunks, len(recorded), false, resp, aiErr)
	return recorded, resp, aiErr, nil
}

func validateStreamArgs(args []eval.Value, opName string) (*eval.StringValue, *eval.ListValue, *eval.ListValue, *eval.ListValue, eval.Value, error) {
	if len(args) < 5 {
		return nil, nil, nil, nil, nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected 5 arguments, got %d", opName, len(args))
	}
	model, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, nil, nil, nil, nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected string model, got %T", opName, args[0])
	}
	messages, ok := args[1].(*eval.ListValue)
	if !ok {
		return nil, nil, nil, nil, nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected list[Message] messages, got %T", opName, args[1])
	}
	tools, ok := args[2].(*eval.ListValue)
	if !ok {
		return nil, nil, nil, nil, nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected list[ToolSchema] tools, got %T", opName, args[2])
	}
	breakpoints, ok := args[3].(*eval.ListValue)
	if !ok {
		return nil, nil, nil, nil, nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: expected list[CacheBreakpoint] cache_breakpoints, got %T", opName, args[3])
	}
	if args[4] == nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("E_AI_TYPE_ERROR: %s: on_chunk callback is nil", opName)
	}
	return model, messages, tools, breakpoints, args[4], nil
}

func schemaAIError(err error) *ai.AIError {
	return ai.NewAIError(ai.CodeSchemaValidation, err.Error(), false)
}

func recordStreamTerminalTrace(ctx *EffContext, opName, model string, messageCount, toolCount, cacheCount, providerChunks, deliveredChunks int, drainExhausted bool, resp *ai.Response, aiErr *ai.AIError) {
	counts := fmt.Sprintf("messages:%d tools:%d cache:%d provider_chunks:%d delivered_chunks:%d", messageCount, toolCount, cacheCount, providerChunks, deliveredChunks)
	if drainExhausted {
		counts += " drain_exhausted:true"
	}
	result := ""
	if aiErr != nil {
		result = fmt.Sprintf(errResultPrefix, aiErr.Code)
	} else {
		result = fmt.Sprintf("text:%s tool_calls:%d finish:%s cache_read:%d cache_create:%d", truncateForTrace(resp.Text), len(resp.ToolCalls), resp.FinishReason, resp.CacheReadInputTokens, resp.CacheCreationInputTokens)
	}
	ctx.RecordAIEffect(opName, []string{truncateForTrace(model), counts}, result, ctx.AI.LastRoutingMetadata())
}
