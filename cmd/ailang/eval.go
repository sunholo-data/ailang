package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/eval_harness"
)

func runEval() {
	// Parse eval subcommand flags
	fs := flag.NewFlagSet("eval", flag.ExitOnError)
	benchmarkID := fs.String("benchmark", "", "Benchmark ID to run")
	langs := fs.String("langs", "python,ailang", "Comma-separated list of languages")
	model := fs.String("model", "claude-sonnet-4-5", "LLM model to use (gpt5, claude-sonnet-4-5, gemini-2-5-pro)")
	seed := fs.Int64("seed", 42, "Random seed for deterministic runs")
	outputDir := fs.String("output", "eval_results", "Output directory for results")
	timeout := fs.Duration("timeout", 30*time.Second, "Timeout for code execution")
	mock := fs.Bool("mock", false, "Use mock AI agent (for testing)")
	listModels := fs.Bool("list-models", false, "List available models and exit")
	selfRepair := fs.Bool("self-repair", false, "Enable single-shot self-repair on errors")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Handle --list-models
	if *listModels {
		printAvailableModels()
		return
	}

	if *benchmarkID == "" {
		fmt.Fprintf(os.Stderr, "%s: --benchmark flag is required\n", red("Error"))
		fmt.Println("Usage: ailang eval --benchmark <id> [--langs python,ailang] [--model gpt-4]")
		os.Exit(1)
	}

	// Load benchmark spec
	specPath := filepath.Join("benchmarks", *benchmarkID+".yml")
	spec, err := eval_harness.LoadSpec(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load benchmark: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Running benchmark: %s\n", cyan("→"), spec.Description)
	fmt.Printf("  Languages: %s\n", *langs)
	fmt.Printf("  Model: %s\n", *model)
	fmt.Printf("  Seed: %d\n", *seed)

	// Parse languages
	targetLangs := strings.Split(*langs, ",")
	for i := range targetLangs {
		targetLangs[i] = strings.TrimSpace(targetLangs[i])
	}

	// Create metrics logger
	logger := eval_harness.NewMetricsLogger(*outputDir)

	// Create AI agent (or mock)
	var agent *eval_harness.AIAgent

	if *mock {
		fmt.Fprintf(os.Stderr, "%s: --self-repair not supported with --mock\n", red("Error"))
		if *selfRepair {
			os.Exit(1)
		}
		// Mock agent doesn't support self-repair currently
	}

	// Create real AI agent (required for self-repair)
	aiAgent, err := eval_harness.NewAIAgent(*model, *seed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to create AI agent: %v\n", red("Error"), err)
		os.Exit(1)
	}
	agent = aiAgent

	if *selfRepair {
		fmt.Printf("  %s Self-repair enabled (will retry on errors)\n", cyan("ℹ"))
	}

	// Run benchmark for each language
	for _, lang := range targetLangs {
		// Check if language is supported
		if !spec.SupportsLanguage(lang) {
			fmt.Printf("%s Language %s not supported by benchmark %s\n",
				yellow("⚠"), lang, spec.ID)
			continue
		}

		fmt.Printf("\n%s Testing %s...\n", cyan("→"), lang)

		// Generate prompt
		prompt := spec.PromptForLanguage(lang)
		fmt.Printf("  Prompt: %s...\n", truncatePrompt(prompt, 60))

		// Get runner
		runner, err := eval_harness.GetRunner(lang, spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to get runner: %v\n", red("✗"), err)
			continue
		}

		// Create RepairRunner and execute with optional self-repair
		ctx := context.Background()
		repairRunner := eval_harness.NewRepairRunner(agent, runner, spec, *timeout, *selfRepair)
		metrics, err := repairRunner.Run(ctx, prompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: benchmark execution failed: %v\n", red("✗"), err)
			continue
		}

		// Print first attempt results
		fmt.Printf("  %s Generated %d output tokens (%d input + %d output)\n",
			green("✓"), metrics.OutputTokens, metrics.InputTokens, metrics.OutputTokens)

		fmt.Printf("  Running code...\n")

		if metrics.CompileOk {
			fmt.Printf("  %s Compile: OK\n", green("✓"))
		} else {
			fmt.Printf("  %s Compile: FAILED\n", red("✗"))
		}

		if metrics.RuntimeOk {
			fmt.Printf("  %s Runtime: OK\n", green("✓"))
		} else {
			fmt.Printf("  %s Runtime: FAILED\n", red("✗"))
		}

		if metrics.StdoutOk {
			fmt.Printf("  %s Output: MATCH\n", green("✓"))
		} else {
			fmt.Printf("  %s Output: MISMATCH\n", red("✗"))
			if metrics.Stderr != "" {
				fmt.Printf("    Expected: %s\n", truncateOutput(spec.ExpectedOut))
			}
		}

		// Display timing breakdown
		if metrics.CompileMs > 0 || metrics.ExecuteMs > 0 {
			fmt.Printf("  Duration: %dms (compile: %dms, execute: %dms)\n",
				metrics.DurationMs, metrics.CompileMs, metrics.ExecuteMs)
		} else {
			fmt.Printf("  Duration: %dms (total process time)\n", metrics.DurationMs)
		}

		// If self-repair was used, show repair results
		if metrics.RepairUsed {
			fmt.Printf("\n  %s Self-repair triggered (error: %s)\n", yellow("↻"), metrics.ErrCode)
			fmt.Printf("  %s Repair tokens: %d input + %d output\n",
				cyan("ℹ"), metrics.RepairTokensIn, metrics.RepairTokensOut)
			if metrics.RepairOk {
				fmt.Printf("  %s Repair: SUCCESS\n", green("✓"))
			} else {
				fmt.Printf("  %s Repair: FAILED\n", red("✗"))
			}
		}

		// Log metrics
		if err := logger.Log(metrics); err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to log metrics: %v\n", yellow("⚠"), err)
		}
	}

	fmt.Printf("\n%s Benchmark complete. Results saved to %s/\n", green("✓"), *outputDir)
}

// truncatePrompt truncates a prompt for display
func truncatePrompt(prompt string, maxLen int) string {
	// Take first line or maxLen chars
	lines := strings.Split(prompt, "\n")
	firstLine := strings.TrimSpace(lines[0])
	if len(firstLine) > maxLen {
		return firstLine[:maxLen] + "..."
	}
	return firstLine
}

// truncateOutput truncates output for display
func truncateOutput(output string) string {
	output = strings.TrimSpace(output)
	if len(output) > 50 {
		return output[:50] + "..."
	}
	return output
}

// printAvailableModels prints configured models from models.yml
func printAvailableModels() {
	// Try to load models config
	if err := eval_harness.InitModelsConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load models config: %v\n", yellow("⚠"), err)
		fmt.Println("\nFalling back to built-in model list:")
		fmt.Println("\nOpenAI:")
		fmt.Println("  gpt-5          - GPT-5 full model (default)")
		fmt.Println("  gpt-5-mini     - GPT-5 mini (faster, cheaper)")
		fmt.Println("\nAnthropic:")
		fmt.Println("  claude-sonnet-4-5  - Claude Sonnet 4.5 (best for coding)")
		fmt.Println("\nGoogle:")
		fmt.Println("  gemini-2-5-pro  - Gemini 2.5 Pro with thinking")
		return
	}

	config := eval_harness.GlobalModelsConfig

	fmt.Println(bold("Available Models"))
	fmt.Println()
	fmt.Printf("Default: %s\n", green(config.GetDefaultModel()))
	fmt.Println()

	// Group by provider
	providers := map[string][]string{
		"openai":    {},
		"anthropic": {},
		"google":    {},
	}

	for name, model := range config.Models {
		if list, ok := providers[model.Provider]; ok {
			providers[model.Provider] = append(list, name)
		}
	}

	// Print each provider
	if len(providers["openai"]) > 0 {
		fmt.Println(cyan("OpenAI:"))
		for _, name := range providers["openai"] {
			model, _ := config.GetModel(name)
			fmt.Printf("  %-20s %s\n", name, model.Description)
			fmt.Printf("  %-20s API: %s\n", "", model.APIName)
			fmt.Printf("  %-20s Env: %s\n", "", model.EnvVar)
			fmt.Println()
		}
	}

	if len(providers["anthropic"]) > 0 {
		fmt.Println(cyan("Anthropic:"))
		for _, name := range providers["anthropic"] {
			model, _ := config.GetModel(name)
			fmt.Printf("  %-20s %s\n", name, model.Description)
			fmt.Printf("  %-20s API: %s\n", "", model.APIName)
			fmt.Printf("  %-20s Env: %s\n", "", model.EnvVar)
			fmt.Println()
		}
	}

	if len(providers["google"]) > 0 {
		fmt.Println(cyan("Google:"))
		for _, name := range providers["google"] {
			model, _ := config.GetModel(name)
			fmt.Printf("  %-20s %s\n", name, model.Description)
			fmt.Printf("  %-20s API: %s\n", "", model.APIName)
			fmt.Printf("  %-20s Env: %s\n", "", model.EnvVar)
			fmt.Println()
		}
	}

	fmt.Println("Benchmark Suite (recommended):")
	for _, name := range config.GetBenchmarkSuite() {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	fmt.Println("Usage:")
	fmt.Printf("  %s\n", cyan("ailang eval --benchmark fizzbuzz --model claude-sonnet-4-5"))
	fmt.Printf("  %s\n", cyan("ailang eval --benchmark fizzbuzz --model gpt5"))
}

// generateMockCode generates simple mock code for testing
func generateMockCode(benchmarkID, lang string) string {
	switch lang {
	case "python":
		switch benchmarkID {
		case "fizzbuzz":
			return `for i in range(1, 101):
    if i % 15 == 0:
        print("FizzBuzz")
    elif i % 3 == 0:
        print("Fizz")
    elif i % 5 == 0:
        print("Buzz")
    else:
        print(i)`
		default:
			return `print("Hello from Python")`
		}
	case "ailang":
		switch benchmarkID {
		case "fizzbuzz":
			// Note: This mock uses a simplified version since recursion not yet implemented
			return `module benchmark/solution

import std/io (println)

export func fizzbuzzValue(n: int) -> string {
  if n % 15 == 0 then "FizzBuzz"
  else if n % 3 == 0 then "Fizz"
  else if n % 5 == 0 then "Buzz"
  else show(n)
}

export func main() -> () ! {IO} {
  println("1");
  println("2");
  println("Fizz");
  println("4");
  println("Buzz")
}`
		default:
			return `module benchmark/solution

import std/io (println)

export func main() -> () ! {IO} {
  println("Hello from AILANG")
}`
		}
	default:
		return fmt.Sprintf("// Mock code for %s in %s\n", benchmarkID, lang)
	}
}
