package main

import (
	"log"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
	"github.com/sunholo/ailang/internal/agentrunner"
)

func main() {
	log.Println("🌐 Multi-Model Agent Starting...")
	log.Println("   Supports: Claude (Anthropic), Gemini (Google), GPT (OpenAI)")
	log.Println("")

	// Create multi-model handler
	multiHandler := agentrunner.NewMultiModelAgentHandler()

	// Register Claude handler (Anthropic)
	log.Println("Registering Claude (Anthropic) handler...")
	claudeHandler := agentrunner.NewAnthropicAgentHandler(
		"claude-sonnet-4-5",
		".claude/agents/eval-analyzer.md",
		".",
	)
	multiHandler.RegisterHandler("claude-sonnet-4-5", claudeHandler)
	multiHandler.RegisterHandler("claude", claudeHandler) // Alias

	// Register Gemini handler (Google)
	log.Println("Registering Gemini (Google) handler...")
	geminiHandler := agentrunner.NewGeminiAgentHandler(
		"gemini-2.5-pro",
		".claude/agents/eval-analyzer.md",
		".",
	)
	multiHandler.RegisterHandler("gemini-2.5-pro", geminiHandler)
	multiHandler.RegisterHandler("gemini", geminiHandler) // Alias

	// Register GPT handler (OpenAI)
	log.Println("Registering GPT (OpenAI) handler...")
	gptHandler := agentrunner.NewOpenAIAgentHandler(
		"gpt-4",
		".claude/agents/eval-analyzer.md",
		".",
	)
	multiHandler.RegisterHandler("gpt-4", gptHandler)
	multiHandler.RegisterHandler("gpt", gptHandler) // Alias

	// Register o1 models (OpenAI reasoning models)
	log.Println("Registering o1 (OpenAI reasoning) handler...")
	o1Handler := agentrunner.NewOpenAIAgentHandler(
		"o1-preview",
		".claude/agents/eval-analyzer.md",
		".",
	)
	multiHandler.RegisterHandler("o1-preview", o1Handler)
	multiHandler.RegisterHandler("o1", o1Handler) // Alias

	// Set default to Claude
	multiHandler.RegisterHandler("default", claudeHandler)

	log.Println("")
	log.Println("✓ All handlers registered")
	log.Println("")
	log.Println("Usage examples:")
	log.Println("  # Use Claude (default)")
	log.Println("  ailang agent send multi-model-agent '{\"action\": \"analyze\"}'")
	log.Println("")
	log.Println("  # Use Gemini explicitly")
	log.Println("  ailang agent send multi-model-agent '{\"model\": \"gemini\", \"action\": \"analyze\"}'")
	log.Println("")
	log.Println("  # Use GPT-4")
	log.Println("  ailang agent send multi-model-agent '{\"model\": \"gpt-4\", \"action\": \"analyze\"}'")
	log.Println("")
	log.Println("  # Use o1 (reasoning model)")
	log.Println("  ailang agent send multi-model-agent '{\"model\": \"o1\", \"action\": \"analyze\"}'")
	log.Println("")

	// Create runner
	runner, err := agentrunner.NewRunner(&agentrunner.AgentConfig{
		AgentID:      "multi-model-agent",
		StateDir:     ".ailang/state",
		PollInterval: 3 * time.Second,
		Handler:      multiHandler,
		Capabilities: map[string]interface{}{
			"models": []string{
				"claude-sonnet-4-5",
				"gemini-2.5-pro",
				"gpt-4",
				"o1-preview",
			},
			"description": "Multi-model agent supporting Claude, Gemini, and OpenAI",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	log.Println("🚀 Multi-model agent started. Press Ctrl+C to stop.")
	log.Println("")

	// Run agent
	if err := runner.Run(); err != nil {
		log.Fatalf("Runner error: %v", err)
	}
}
