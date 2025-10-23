package agentrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// ClaudeAgentHandler bridges Claude agents (.claude/agents/*.md) with the agent protocol.
// It executes full Claude agents using the Anthropic Agent SDK.
//
// This is DIFFERENT from NewClaudeCLIHandler (in llm_cli_handler.go):
// - ClaudeAgentHandler: Executes .claude/agents/*.md files with full agent context (tools, MCP, state)
// - NewClaudeCLIHandler: Simple prompt-response via "claude" CLI (no agent file required)
//
// Use ClaudeAgentHandler when:
// - You have a .claude/agents/*.md file defining the agent
// - Agent needs tools, MCP servers, or persistent state
// - You want full agent execution (not just LLM chat)
//
// Use NewClaudeCLIHandler when:
// - You just need a quick LLM response
// - No agent file required
// - Simple prompt-response pattern
type ClaudeAgentHandler struct {
	AgentFile string // Path to .claude/agents/agent-name.md
	WorkDir   string // Working directory for the agent
	Model     string // Model to use (e.g., "claude-sonnet-4-5")
}

// NewClaudeAgentHandler creates a handler that executes a Claude agent.
func NewClaudeAgentHandler(agentFile, workDir string) *ClaudeAgentHandler {
	return &ClaudeAgentHandler{
		AgentFile: agentFile,
		WorkDir:   workDir,
	}
}

// HandleMessage executes the Claude agent with the message payload.
func (h *ClaudeAgentHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	// Convert message to a prompt for the Claude agent
	prompt := h.buildPrompt(msg)

	// Execute Claude agent
	output, err := h.executeClaudeAgent(prompt)
	if err != nil {
		return nil, fmt.Errorf("claude agent execution failed: %w", err)
	}

	// Parse response
	response := map[string]interface{}{
		"status": "completed",
		"output": output,
	}

	return response, nil
}

// buildPrompt creates a prompt for the Claude agent from the message.
func (h *ClaudeAgentHandler) buildPrompt(msg *agentprotocol.Envelope) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("You are processing a message from agent '%s'\n\n", msg.FromAgent))
	sb.WriteString(fmt.Sprintf("Message ID: %s\n", msg.MessageID))
	sb.WriteString(fmt.Sprintf("Correlation ID: %s\n", msg.CorrelationID))
	sb.WriteString(fmt.Sprintf("Message Type: %s\n\n", msg.MessageType))

	// Add payload as JSON
	if msg.Payload != nil {
		payloadJSON, _ := json.MarshalIndent(msg.Payload, "", "  ")
		sb.WriteString("Payload:\n")
		sb.WriteString(string(payloadJSON))
		sb.WriteString("\n\n")
	}

	sb.WriteString("Please process this message and provide your response.\n")

	return sb.String()
}

// executeClaudeAgent runs the Claude agent as a subprocess.
// This assumes the Anthropic Agent SDK is available and configured.
func (h *ClaudeAgentHandler) executeClaudeAgent(prompt string) (string, error) {
	// For now, we'll simulate Claude agent execution by reading the agent file
	// and returning a mock response. In production, this would use the Anthropic Agent SDK.

	// Read agent file to understand what it does
	agentContent, err := os.ReadFile(h.AgentFile)
	if err != nil {
		return "", fmt.Errorf("failed to read agent file: %w", err)
	}

	// TODO: Integrate with Anthropic Agent SDK
	// For now, return a simulated response
	response := fmt.Sprintf("Agent executed successfully.\nAgent file: %s\nPrompt length: %d chars\n\nNOTE: This is a mock response. Full integration with Anthropic Agent SDK pending.",
		filepath.Base(h.AgentFile),
		len(prompt))

	_ = agentContent // Will be used when SDK integration is complete

	return response, nil
}

// SkillHandler bridges AILANG skills (.claude/skills/*) with the agent protocol.
type SkillHandler struct {
	SkillName string // Name of the skill to execute
	WorkDir   string // Working directory
}

// NewSkillHandler creates a handler that executes an AILANG skill.
func NewSkillHandler(skillName, workDir string) *SkillHandler {
	return &SkillHandler{
		SkillName: skillName,
		WorkDir:   workDir,
	}
}

// HandleMessage executes the skill with the message payload.
func (h *SkillHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	// Skills are invoked via Claude Code's Skill tool
	// For now, we'll simulate by executing any associated scripts

	skillDir := filepath.Join(h.WorkDir, ".claude", "skills", h.SkillName)

	// Check if skill has scripts
	scriptsDir := filepath.Join(skillDir, "scripts")
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return map[string]interface{}{
			"status": "no_scripts",
			"message": fmt.Sprintf("Skill '%s' has no scripts directory", h.SkillName),
		}, nil
	}

	// List available scripts
	scripts, err := os.ReadDir(scriptsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scripts directory: %w", err)
	}

	var scriptNames []string
	for _, script := range scripts {
		if !script.IsDir() && filepath.Ext(script.Name()) == ".sh" {
			scriptNames = append(scriptNames, script.Name())
		}
	}

	response := map[string]interface{}{
		"status":  "skill_info",
		"skill":   h.SkillName,
		"scripts": scriptNames,
		"message": "Skill scripts listed. Full execution integration pending.",
	}

	return response, nil
}

// CommandHandler executes a shell command in response to a message.
type CommandHandler struct {
	Command string   // Command to execute
	Args    []string // Command arguments
	WorkDir string   // Working directory
}

// NewCommandHandler creates a handler that executes a shell command.
func NewCommandHandler(command string, args []string, workDir string) *CommandHandler {
	return &CommandHandler{
		Command: command,
		Args:    args,
		WorkDir: workDir,
	}
}

// HandleMessage executes the command with message payload as input.
func (h *CommandHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	// Serialize message to JSON for command input
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	// Execute command
	cmd := exec.Command(h.Command, h.Args...)
	cmd.Dir = h.WorkDir
	cmd.Stdin = strings.NewReader(string(msgJSON))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w\nOutput: %s", err, string(output))
	}

	// Try to parse output as JSON, otherwise return as string
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err == nil {
		return result, nil
	}

	// Return raw output
	return map[string]interface{}{
		"status": "completed",
		"output": string(output),
	}, nil
}

// FunctionHandler wraps a Go function as a message handler.
type FunctionHandler struct {
	Fn func(msg *agentprotocol.Envelope) (map[string]interface{}, error)
}

// NewFunctionHandler creates a handler from a Go function.
func NewFunctionHandler(fn func(*agentprotocol.Envelope) (map[string]interface{}, error)) *FunctionHandler {
	return &FunctionHandler{Fn: fn}
}

// HandleMessage delegates to the wrapped function.
func (h *FunctionHandler) HandleMessage(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
	return h.Fn(msg)
}
