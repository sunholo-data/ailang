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

	fs.Parse(os.Args[2:])

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
	var agent interface {
		GenerateCode(ctx context.Context, prompt string) (*eval_harness.GenerateResult, error)
	}

	if *mock {
		// Use mock agent for testing
		mockCode := generateMockCode(*benchmarkID, targetLangs[0])
		agent = eval_harness.NewMockAIAgent(*model, mockCode)
		fmt.Printf("%s Using mock AI agent\n", yellow("⚠"))
	} else {
		aiAgent, err := eval_harness.NewAIAgent(*model, *seed)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to create AI agent: %v\n", red("Error"), err)
			os.Exit(1)
		}
		agent = aiAgent
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

		// Generate code
		ctx := context.Background()
		result, err := agent.GenerateCode(ctx, prompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: code generation failed: %v\n", red("✗"), err)
			continue
		}

		fmt.Printf("  %s Generated %d tokens\n", green("✓"), result.Tokens)

		// Get runner
		runner, err := eval_harness.GetRunner(lang, spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to get runner: %v\n", red("✗"), err)
			continue
		}

		// Execute code
		fmt.Printf("  Running code...\n")
		runResult, err := runner.Run(result.Code, *timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: execution failed: %v\n", red("✗"), err)
			continue
		}

		// Check output
		runResult.StdoutOk = eval_harness.CompareOutput(spec.ExpectedOut, runResult.Stdout)

		// Categorize error
		errorCategory := eval_harness.CategorizeError(
			runResult.CompileOk,
			runResult.RuntimeOk,
			runResult.StdoutOk,
		)

		// Print results
		if runResult.CompileOk {
			fmt.Printf("  %s Compile: OK\n", green("✓"))
		} else {
			fmt.Printf("  %s Compile: FAILED\n", red("✗"))
		}

		if runResult.RuntimeOk {
			fmt.Printf("  %s Runtime: OK\n", green("✓"))
		} else {
			fmt.Printf("  %s Runtime: FAILED\n", red("✗"))
		}

		if runResult.StdoutOk {
			fmt.Printf("  %s Output: MATCH\n", green("✓"))
		} else {
			fmt.Printf("  %s Output: MISMATCH\n", red("✗"))
			if runResult.Stdout != "" {
				fmt.Printf("    Expected: %s\n", truncateOutput(spec.ExpectedOut))
				fmt.Printf("    Got:      %s\n", truncateOutput(runResult.Stdout))
			}
		}

		fmt.Printf("  Duration: %dms\n", runResult.Duration.Milliseconds())

		// Create metrics
		metrics := eval_harness.NewRunMetrics(spec.ID, lang, *model, *seed)
		metrics.Tokens = result.Tokens
		metrics.CostUSD = eval_harness.CalculateCost(*model, result.Tokens)
		metrics.CompileOk = runResult.CompileOk
		metrics.RuntimeOk = runResult.RuntimeOk
		metrics.StdoutOk = runResult.StdoutOk
		metrics.DurationMs = runResult.Duration.Milliseconds()
		metrics.ErrorCategory = errorCategory
		metrics.Stderr = runResult.Stderr
		metrics.Code = result.Code

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
			return `let fizzbuzz = \i.
  if i % 15 == 0 then "FizzBuzz"
  else if i % 3 == 0 then "Fizz"
  else if i % 5 == 0 then "Buzz"
  else show(i)
in
let range = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10] in
map(\i. print(fizzbuzz(i)), range)`
		default:
			return `print("Hello from AILANG")`
		}
	default:
		return fmt.Sprintf("// Mock code for %s in %s\n", benchmarkID, lang)
	}
}
