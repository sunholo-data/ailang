package agentrunner

import (
	"fmt"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// --- Type aliases for clarity (all use generic LLMCLIHandler) ---

// AnthropicAgentHandler executes Claude agents via Claude CLI
// Docs: https://docs.anthropic.com/claude/docs/claude-cli
type AnthropicAgentHandler = LLMCLIHandler

// GeminiAgentHandler executes Gemini agents via Gemini CLI
// Docs: https://developers.google.com/gemini-code-assist/docs/gemini-cli
type GeminiAgentHandler = LLMCLIHandler

// OpenAIAgentHandler executes OpenAI agents via OpenAI CLI (Codex)
// Docs: https://developers.openai.com/codex/cli/
type OpenAIAgentHandler = LLMCLIHandler

// --- Convenience constructors ---

// NewAnthropicAgentHandler creates a handler for Anthropic Claude models
func NewAnthropicAgentHandler(model, agentFile, workDir string) *AnthropicAgentHandler {
	return NewClaudeCLIHandler(model, agentFile, workDir)
}

// NewGeminiAgentHandler creates a handler for Google Gemini models
func NewGeminiAgentHandler(model, agentFile, workDir string) *GeminiAgentHandler {
	return NewGeminiCLIHandler(model, agentFile, workDir)
}

// NewOpenAIAgentHandler creates a handler for OpenAI models
func NewOpenAIAgentHandler(model, agentFile, workDir string) *OpenAIAgentHandler {
	return NewOpenAICLIHandler(model, agentFile, workDir)
}

// MultiModelAgentHandler routes to different providers based on model name
type MultiModelAgentHandler struct {
	Handlers map[string]MessageHandler
}

// NewMultiModelAgentHandler creates a router for multiple LLM providers
func NewMultiModelAgentHandler() *MultiModelAgentHandler {
	return &MultiModelAgentHandler{
		Handlers: make(map[string]MessageHandler),
	}
}

// RegisterHandler registers a handler for a specific model
func (m *MultiModelAgentHandler) RegisterHandler(modelName string, handler MessageHandler) {
	m.Handlers[modelName] = handler
}

// HandleMessage routes to appropriate handler based on message payload
func (m *MultiModelAgentHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	// Check if message specifies a model
	modelName, ok := msg.Payload["model"].(string)
	if !ok {
		modelName = "default"
	}

	handler, ok := m.Handlers[modelName]
	if !ok {
		return nil, fmt.Errorf("no handler registered for model: %s", modelName)
	}

	return handler.HandleMessage(msg)
}

// Example usage:
//
// // Create multi-model handler
// multiHandler := NewMultiModelAgentHandler()
//
// // Register Claude
// multiHandler.RegisterHandler("claude-sonnet-4-5",
//     NewAnthropicAgentHandler("claude-sonnet-4-5", ".claude/agents/eval-analyzer.md", "."))
//
// // Register Gemini
// multiHandler.RegisterHandler("gemini-2-5-pro",
//     NewGeminiAgentHandler("gemini-2.5-pro", ".claude/agents/eval-analyzer.md", "."))
//
// // Register OpenAI
// multiHandler.RegisterHandler("gpt-4",
//     NewOpenAIAgentHandler("gpt-4", ".claude/agents/eval-analyzer.md", "."))
//
// // Register default (Claude)
// multiHandler.RegisterHandler("default",
//     NewAnthropicAgentHandler("claude-sonnet-4-5", ".claude/agents/eval-analyzer.md", "."))
//
// // Use in runner
// runner, _ := NewRunner(&AgentConfig{
//     AgentID:      "multi-model-agent",
//     Handler:      multiHandler,
//     PollInterval: 3 * time.Second,
// })
// runner.Run()
//
// // Send message with model choice:
// ./bin/send-message multi-model-agent '{"model": "gemini-2-5-pro", "action": "analyze"}'
// ./bin/send-message multi-model-agent '{"model": "gpt-4", "action": "analyze"}'
// ./bin/send-message multi-model-agent '{"action": "analyze"}'  // Uses default (Claude)
