package agentrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// LLMCLIConfig configures an LLM CLI handler
type LLMCLIConfig struct {
	// CLI command name (e.g., "claude", "gemini", "openai")
	CLICommand string

	// Model name (e.g., "claude-sonnet-4-5", "gemini-2.5-pro", "gpt-4")
	Model string

	// Arguments template for the CLI
	// Use placeholders: {{prompt}}, {{model}}, {{format}}
	//
	// Examples:
	//   Claude: ["--prompt", "{{prompt}}", "--model", "{{model}}"]
	//   Gemini: ["--prompt", "{{prompt}}", "--model", "{{model}}", "--output-format", "{{format}}"]
	//   OpenAI: ["api", "chat.completions.create", "-m", "{{model}}", "-g", "user", "{{prompt}}"]
	ArgsTemplate []string

	// Output format (e.g., "json", "text")
	OutputFormat string

	// Working directory for execution
	WorkDir string

	// Agent prompt file (optional)
	AgentFile string

	// Provider name for logging (e.g., "anthropic", "google", "openai")
	Provider string
}

// LLMCLIHandler is a generic handler for LLM CLI tools
type LLMCLIHandler struct {
	Config *LLMCLIConfig
}

// NewLLMCLIHandler creates a generic LLM CLI handler
func NewLLMCLIHandler(config *LLMCLIConfig) *LLMCLIHandler {
	return &LLMCLIHandler{Config: config}
}

// HandleMessage processes messages using the configured LLM CLI
func (h *LLMCLIHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	// Build prompt
	prompt := h.buildPrompt(msg)

	// Build command arguments from template
	args := h.buildArgs(prompt)

	// Execute CLI
	cmd := exec.Command(h.Config.CLICommand, args...)
	cmd.Dir = h.Config.WorkDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s CLI execution failed: %w (output: %s)",
			h.Config.Provider, err, string(output))
	}

	return map[string]interface{}{
		"status":   "completed",
		"output":   string(output),
		"model":    h.Config.Model,
		"provider": h.Config.Provider,
	}, nil
}

// buildArgs replaces placeholders in args template
func (h *LLMCLIHandler) buildArgs(prompt string) []string {
	args := make([]string, len(h.Config.ArgsTemplate))
	for i, arg := range h.Config.ArgsTemplate {
		switch arg {
		case "{{prompt}}":
			args[i] = prompt
		case "{{model}}":
			args[i] = h.Config.Model
		case "{{format}}":
			args[i] = h.Config.OutputFormat
		default:
			args[i] = arg
		}
	}
	return args
}

// buildPrompt builds the prompt from agent file and message
func (h *LLMCLIHandler) buildPrompt(msg *agentprotocol.Envelope) string {
	var prompt string

	// If agent file specified, read it
	if h.Config.AgentFile != "" {
		content, err := os.ReadFile(h.Config.AgentFile)
		if err == nil {
			prompt = string(content) + "\n\n"
		}
	}

	// Add message context
	payload, _ := json.Marshal(msg.Payload)
	prompt += fmt.Sprintf("Message from: %s\nMessage ID: %s\nCorrelation ID: %s\n\nPayload:\n%s",
		msg.FromAgent, msg.MessageID, msg.CorrelationID, string(payload))

	return prompt
}

// --- Convenience constructors for specific providers ---

// NewClaudeCLIHandler creates a handler for Claude CLI (simple prompt-response).
//
// This is DIFFERENT from ClaudeAgentHandler (in claude_bridge.go):
// - NewClaudeCLIHandler: Simple prompt-response via "claude" CLI command
// - ClaudeAgentHandler: Executes .claude/agents/*.md files with full agent SDK
//
// Use NewClaudeCLIHandler when:
// - You just need a quick LLM response
// - Simple prompt-response pattern
// - No full agent execution needed
//
// Use ClaudeAgentHandler when:
// - You have a .claude/agents/*.md file
// - Agent needs tools, MCP, or state
func NewClaudeCLIHandler(model, agentFile, workDir string) *LLMCLIHandler {
	if model == "" {
		model = "claude-sonnet-4-5"
	}
	return NewLLMCLIHandler(&LLMCLIConfig{
		CLICommand: "claude",
		Model:      model,
		ArgsTemplate: []string{
			"--prompt", "{{prompt}}",
			"--model", "{{model}}",
		},
		OutputFormat: "json",
		WorkDir:      workDir,
		AgentFile:    agentFile,
		Provider:     "anthropic",
	})
}

// NewGeminiCLIHandler creates a handler for Gemini CLI
func NewGeminiCLIHandler(model, agentFile, workDir string) *LLMCLIHandler {
	if model == "" {
		model = "gemini-2.5-pro"
	}
	return NewLLMCLIHandler(&LLMCLIConfig{
		CLICommand: "gemini",
		Model:      model,
		ArgsTemplate: []string{
			"--prompt", "{{prompt}}",
			"--model", "{{model}}",
			"--output-format", "{{format}}",
		},
		OutputFormat: "json",
		WorkDir:      workDir,
		AgentFile:    agentFile,
		Provider:     "google",
	})
}

// NewOpenAICLIHandler creates a handler for OpenAI CLI (Codex)
func NewOpenAICLIHandler(model, agentFile, workDir string) *LLMCLIHandler {
	if model == "" {
		model = "gpt-4"
	}
	return NewLLMCLIHandler(&LLMCLIConfig{
		CLICommand: "openai",
		Model:      model,
		ArgsTemplate: []string{
			"api", "chat.completions.create",
			"-m", "{{model}}",
			"-g", "user", "{{prompt}}",
		},
		OutputFormat: "json",
		WorkDir:      workDir,
		AgentFile:    agentFile,
		Provider:     "openai",
	})
}

// Example usage:
//
// // Claude (Anthropic)
// claudeHandler := NewClaudeCLIHandler("claude-sonnet-4-5", ".claude/agents/eval-analyzer.md", ".")
//
// // Gemini (Google)
// geminiHandler := NewGeminiCLIHandler("gemini-2.5-pro", ".claude/agents/eval-analyzer.md", ".")
//
// // OpenAI (Codex)
// openaiHandler := NewOpenAICLIHandler("gpt-4", ".claude/agents/eval-analyzer.md", ".")
//
// // Custom CLI (e.g., local model)
// customHandler := NewLLMCLIHandler(&LLMCLIConfig{
//     CLICommand:   "ollama",
//     Model:        "llama3.1:70b",
//     ArgsTemplate: []string{"run", "{{model}}", "{{prompt}}"},
//     WorkDir:      ".",
//     Provider:     "ollama",
// })
//
// // Use in runner
// runner, _ := NewRunner(&AgentConfig{
//     AgentID: "my-agent",
//     Handler: claudeHandler,
// })
// runner.Run()
