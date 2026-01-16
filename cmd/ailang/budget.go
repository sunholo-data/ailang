package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/sunholo/ailang/internal/coordinator"
	ailembed "github.com/sunholo/ailang/internal/embed"
)

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// BudgetConfig mirrors the AILANG type
type BudgetConfig struct {
	WorkspaceBudget  float64                    `json:"workspaceBudget"`
	DailyBudget      float64                    `json:"dailyBudget"`
	TaskMaxCost      float64                    `json:"taskMaxCost"`
	WarningThreshold float64                    `json:"warningThreshold"`
	ProviderBudgets  map[string]*ProviderBudget `json:"providerBudgets,omitempty"`
}

// ProviderBudget defines per-provider budget limits
type ProviderBudget struct {
	DailyBudget      float64 `json:"dailyBudget"`
	TaskMaxCost      float64 `json:"taskMaxCost"`
	HardLimit        bool    `json:"hardLimit"`
	WarningThreshold float64 `json:"warningThreshold"`
}

// BudgetStatus mirrors the AILANG type
type BudgetStatus struct {
	Allowed            bool    `json:"allowed"`
	RemainingWorkspace float64 `json:"remainingWorkspace"`
	RemainingDaily     float64 `json:"remainingDaily"`
	WarningLevel       string  `json:"warningLevel"`
	Message            string  `json:"message"`
}

func budgetCommand() {
	args := flag.Args()[1:] // Skip "budget"

	// Extract subcommand first (status, check, help)
	subcommand := "status"
	var remaining []string
	if len(args) > 0 && !startsWithDash(args[0]) {
		subcommand = args[0]
		remaining = args[1:]
	} else {
		remaining = args
	}

	fs := flag.NewFlagSet("budget", flag.ExitOnError)
	costFlag := fs.Float64("cost", 0, "Estimated task cost for checking")
	workspaceSpend := fs.Float64("workspace-spend", 0, "Current workspace spend (USD)")
	dailySpend := fs.Float64("daily-spend", 0, "Current daily spend (USD)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	helpFlag := fs.Bool("help", false, "Show help")

	// Per-provider flags (NEW)
	providerFlag := fs.String("provider", "", "Check budget for specific provider (claude, gemini)")
	allProviders := fs.Bool("all-providers", false, "Show all provider budgets")

	// Custom budget config flags (used as fallback if no config file)
	workspaceBudget := fs.Float64("workspace-budget", 100.0, "Workspace budget limit (USD)")
	dailyBudgetLimit := fs.Float64("daily-budget", 50.0, "Daily budget limit (USD)")
	taskMaxCost := fs.Float64("task-max", 25.0, "Maximum single task cost (USD)")
	warningThreshold := fs.Float64("warning-threshold", 0.8, "Warning threshold (0-1)")

	_ = fs.Parse(remaining)

	if *helpFlag || subcommand == "help" {
		printBudgetHelp()
		return
	}

	// Try to load config from ~/.ailang/config.yaml
	config := loadBudgetConfigFromYAML()

	// Override with CLI flags if specified (non-default values)
	if *workspaceBudget != 100.0 {
		config.WorkspaceBudget = *workspaceBudget
	}
	if *dailyBudgetLimit != 50.0 {
		config.DailyBudget = *dailyBudgetLimit
	}
	if *taskMaxCost != 25.0 {
		config.TaskMaxCost = *taskMaxCost
	}
	if *warningThreshold != 0.8 {
		config.WarningThreshold = *warningThreshold
	}

	switch subcommand {
	case "status":
		runBudgetStatus(config, *workspaceSpend, *dailySpend, *providerFlag, *allProviders, *jsonOutput)
	case "check":
		// ailang budget check --cost 5.0 --daily-spend 40 --provider claude
		estimatedCost := *costFlag
		// Also try to parse positional if cost flag not set
		if estimatedCost == 0 && fs.NArg() > 0 {
			if cost, err := parseFloat(fs.Arg(0)); err == nil {
				estimatedCost = cost
			}
		}
		if estimatedCost == 0 {
			fmt.Fprintf(os.Stderr, "%s: estimated cost required\n", red("Error"))
			fmt.Println("Usage: ailang budget check --cost 5.0 [--daily-spend N] [--provider claude]")
			os.Exit(1)
		}
		runBudgetCheck(config, estimatedCost, *workspaceSpend, *dailySpend, *providerFlag, *jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown budget subcommand '%s'\n", red("Error"), subcommand)
		printBudgetHelp()
		os.Exit(1)
	}
}

// loadBudgetConfigFromYAML loads budget configuration from ~/.ailang/config.yaml
func loadBudgetConfigFromYAML() BudgetConfig {
	budgetsCfg, err := coordinator.LoadBudgetsConfig()
	if err != nil || budgetsCfg == nil {
		// Return defaults if no config file
		return BudgetConfig{
			WorkspaceBudget:  100.0,
			DailyBudget:      50.0,
			TaskMaxCost:      25.0,
			WarningThreshold: 0.8,
		}
	}

	config := BudgetConfig{
		ProviderBudgets: make(map[string]*ProviderBudget),
	}

	if budgetsCfg.Global != nil {
		config.WorkspaceBudget = budgetsCfg.Global.WorkspaceBudget
		config.DailyBudget = budgetsCfg.Global.DailyBudget
		config.TaskMaxCost = budgetsCfg.Global.TaskMaxCost
		config.WarningThreshold = budgetsCfg.Global.WarningThreshold
	} else {
		// Fallback to defaults
		config.WorkspaceBudget = 100.0
		config.DailyBudget = 50.0
		config.TaskMaxCost = 25.0
		config.WarningThreshold = 0.8
	}

	// Load per-provider configs
	for provider, limit := range budgetsCfg.Providers {
		config.ProviderBudgets[provider] = &ProviderBudget{
			DailyBudget:      limit.DailyBudget,
			TaskMaxCost:      limit.TaskMaxCost,
			HardLimit:        limit.HardLimit,
			WarningThreshold: limit.WarningThreshold,
		}
	}

	return config
}

func startsWithDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

// BurnRateInfo for CLI output
type BurnRateInfo struct {
	CostPerHour          float64 `json:"costPerHour"`
	HoursUntilExhaustion int     `json:"hoursUntilExhaustion"`
	WindowHours          int     `json:"windowHours"`
}

func runBudgetStatus(config BudgetConfig, workspaceSpend, dailySpend float64, provider string, allProviders, jsonOutput bool) {
	// Use AILANG to check budget status
	engine := ailembed.New(".")
	defer engine.Close()

	result, err := engine.Call(
		"internal/dashboard_transforms/budget_checker",
		"checkTaskBudget",
		config,
		0.0, // estimated cost of 0 for status check
		workspaceSpend,
		dailySpend,
	)

	var status BudgetStatus
	if err != nil {
		// Fall back to Go implementation
		status = goCheckTaskBudget(config, 0, workspaceSpend, dailySpend)
	} else {
		goResult, _ := ailembed.ToGo(result)
		resultMap := goResult.(map[string]interface{})
		status = BudgetStatus{
			Allowed:            getBoolVal(resultMap, "allowed"),
			RemainingWorkspace: getFloatVal(resultMap, "remainingWorkspace"),
			RemainingDaily:     getFloatVal(resultMap, "remainingDaily"),
			WarningLevel:       getStringVal(resultMap, "warningLevel"),
			Message:            getStringVal(resultMap, "message"),
		}
	}

	// Calculate burn rate from daily spend (simplified: assume 8-hour window)
	windowHours := 8
	burnRate := calculateBurnRateCLI(engine, dailySpend, windowHours)
	exhaustionHours := calculateExhaustionCLI(engine, config.DailyBudget-dailySpend, burnRate.CostPerHour)

	if jsonOutput {
		output := map[string]interface{}{
			"config":         config,
			"status":         status,
			"workspaceSpend": workspaceSpend,
			"dailySpend":     dailySpend,
			"usagePercent":   (dailySpend / config.DailyBudget) * 100,
			"usingAilang":    err == nil,
			"burnRate": map[string]interface{}{
				"costPerHour":          burnRate.CostPerHour,
				"hoursUntilExhaustion": exhaustionHours,
				"windowHours":          windowHours,
			},
		}
		// Add per-provider info if available
		if len(config.ProviderBudgets) > 0 {
			output["providerBudgets"] = config.ProviderBudgets
		}
		if provider != "" {
			output["provider"] = provider
			if pb, ok := config.ProviderBudgets[provider]; ok {
				output["providerConfig"] = pb
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(output)
		return
	}

	// Human-readable output
	fmt.Println(bold("Budget Status"))
	fmt.Println()

	// Show specific provider or global config
	if provider != "" {
		// Show provider-specific status
		fmt.Printf("  Provider: %s\n\n", cyan(provider))
		if pb, ok := config.ProviderBudgets[provider]; ok {
			fmt.Printf("  Provider Config:\n")
			fmt.Printf("    Daily Budget:     $%.2f\n", pb.DailyBudget)
			fmt.Printf("    Task Max Cost:    $%.2f\n", pb.TaskMaxCost)
			fmt.Printf("    Hard Limit:       %v\n", pb.HardLimit)
			if pb.WarningThreshold > 0 {
				fmt.Printf("    Warning at:       %.0f%%\n", pb.WarningThreshold*100)
			}
		} else {
			fmt.Printf("  (using global limits - no provider-specific config)\n")
		}
		fmt.Println()
	}

	fmt.Printf("  Global Config:\n")
	fmt.Printf("    Workspace Budget: $%.2f\n", config.WorkspaceBudget)
	fmt.Printf("    Daily Budget:     $%.2f\n", config.DailyBudget)
	fmt.Printf("    Task Max Cost:    $%.2f\n", config.TaskMaxCost)
	fmt.Printf("    Warning at:       %.0f%%\n", config.WarningThreshold*100)
	fmt.Println()
	fmt.Printf("  Current Usage:\n")
	fmt.Printf("    Workspace Spend: $%.2f / $%.2f (%.1f%%)\n",
		workspaceSpend, config.WorkspaceBudget, (workspaceSpend/config.WorkspaceBudget)*100)
	fmt.Printf("    Daily Spend:     $%.2f / $%.2f (%.1f%%)\n",
		dailySpend, config.DailyBudget, (dailySpend/config.DailyBudget)*100)
	fmt.Println()

	// Show all provider budgets if requested
	if allProviders && len(config.ProviderBudgets) > 0 {
		fmt.Printf("  Per-Provider Budgets:\n")
		// Sort providers for consistent output
		providers := make([]string, 0, len(config.ProviderBudgets))
		for p := range config.ProviderBudgets {
			providers = append(providers, p)
		}
		sort.Strings(providers)

		for _, p := range providers {
			pb := config.ProviderBudgets[p]
			hardStr := ""
			if pb.HardLimit {
				hardStr = " " + red("[HARD]")
			}
			fmt.Printf("    %s: $%.2f/day, max $%.2f/task%s\n",
				cyan(p), pb.DailyBudget, pb.TaskMaxCost, hardStr)
		}
		fmt.Println()
	}

	fmt.Printf("  Status: %s\n", colorWarningLevel(status.WarningLevel))

	// Show burn rate
	fmt.Println()
	fmt.Printf("  Burn Rate:\n")
	if burnRate.CostPerHour > 0 {
		fmt.Printf("    Rate:      $%.2f/hour (last %d hours)\n", burnRate.CostPerHour, windowHours)
		if exhaustionHours >= 0 {
			if exhaustionHours == 0 {
				fmt.Printf("    Forecast:  %s\n", red("Budget exhausted"))
			} else if exhaustionHours < 4 {
				fmt.Printf("    Forecast:  %s hours until exhaustion\n", red(fmt.Sprintf("%d", exhaustionHours)))
			} else if exhaustionHours < 12 {
				fmt.Printf("    Forecast:  %s hours until exhaustion\n", yellow(fmt.Sprintf("%d", exhaustionHours)))
			} else {
				fmt.Printf("    Forecast:  %d hours until exhaustion\n", exhaustionHours)
			}
		} else {
			fmt.Printf("    Forecast:  Budget will not be exhausted at current rate\n")
		}
	} else {
		fmt.Printf("    Rate:      No recent spend (last %d hours)\n", windowHours)
		fmt.Printf("    Forecast:  N/A\n")
	}

	if err == nil {
		fmt.Printf("\n  %s\n", cyan("(using AILANG budget_checker)"))
	}
}

func runBudgetCheck(config BudgetConfig, estimatedCost, workspaceSpend, dailySpend float64, provider string, jsonOutput bool) {
	engine := ailembed.New(".")
	defer engine.Close()

	// Get effective limits for this provider
	effectiveDailyBudget := config.DailyBudget
	effectiveTaskMax := config.TaskMaxCost
	effectiveWarningThreshold := config.WarningThreshold
	hardLimit := false

	if provider != "" {
		if pb, ok := config.ProviderBudgets[provider]; ok {
			if pb.DailyBudget > 0 {
				effectiveDailyBudget = pb.DailyBudget
			}
			if pb.TaskMaxCost > 0 {
				effectiveTaskMax = pb.TaskMaxCost
			}
			if pb.WarningThreshold > 0 {
				effectiveWarningThreshold = pb.WarningThreshold
			}
			hardLimit = pb.HardLimit
		}
	}

	// Create effective config for AILANG
	effectiveConfig := BudgetConfig{
		WorkspaceBudget:  config.WorkspaceBudget,
		DailyBudget:      effectiveDailyBudget,
		TaskMaxCost:      effectiveTaskMax,
		WarningThreshold: effectiveWarningThreshold,
	}

	result, err := engine.Call(
		"internal/dashboard_transforms/budget_checker",
		"checkTaskBudget",
		effectiveConfig,
		estimatedCost,
		workspaceSpend,
		dailySpend,
	)

	var status BudgetStatus
	if err != nil {
		status = goCheckTaskBudget(effectiveConfig, estimatedCost, workspaceSpend, dailySpend)
	} else {
		goResult, _ := ailembed.ToGo(result)
		resultMap := goResult.(map[string]interface{})
		status = BudgetStatus{
			Allowed:            getBoolVal(resultMap, "allowed"),
			RemainingWorkspace: getFloatVal(resultMap, "remainingWorkspace"),
			RemainingDaily:     getFloatVal(resultMap, "remainingDaily"),
			WarningLevel:       getStringVal(resultMap, "warningLevel"),
			Message:            getStringVal(resultMap, "message"),
		}
	}

	if jsonOutput {
		output := map[string]interface{}{
			"status":      status,
			"hardLimit":   hardLimit,
			"usingAilang": err == nil,
		}
		if provider != "" {
			output["provider"] = provider
			output["effectiveDailyBudget"] = effectiveDailyBudget
			output["effectiveTaskMax"] = effectiveTaskMax
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(output)
		return
	}

	// Human-readable output
	fmt.Printf("Checking task with estimated cost: $%.2f\n", estimatedCost)
	if provider != "" {
		fmt.Printf("Provider: %s\n", cyan(provider))
		fmt.Printf("Effective limits: $%.2f/day, max $%.2f/task\n", effectiveDailyBudget, effectiveTaskMax)
		if hardLimit {
			fmt.Printf("Limit type: %s\n", red("HARD (blocks task)"))
		}
	}
	fmt.Println()

	if status.Allowed {
		fmt.Printf("  %s Task approved\n", green("✓"))
	} else {
		fmt.Printf("  %s Task blocked\n", red("✗"))
	}
	fmt.Printf("  Level:     %s\n", colorWarningLevel(status.WarningLevel))
	fmt.Printf("  Message:   %s\n", status.Message)
	fmt.Printf("  Remaining: $%.2f workspace, $%.2f daily\n",
		status.RemainingWorkspace, status.RemainingDaily)

	if err == nil {
		fmt.Printf("\n  %s\n", cyan("(validated by AILANG contracts)"))
	}

	if !status.Allowed {
		os.Exit(1)
	}
}

func colorWarningLevel(level string) string {
	switch level {
	case "ok":
		return green("OK")
	case "warning":
		return yellow("WARNING")
	case "critical":
		return red("CRITICAL")
	case "exceeded":
		return red("EXCEEDED")
	default:
		return level
	}
}

// Go fallback implementation
func goCheckTaskBudget(config BudgetConfig, estimatedCost, workspaceSpend, dailySpend float64) BudgetStatus {
	remainingWorkspace := config.WorkspaceBudget - workspaceSpend
	remainingDaily := config.DailyBudget - dailySpend
	minRemaining := remainingWorkspace
	if remainingDaily < minRemaining {
		minRemaining = remainingDaily
	}

	if estimatedCost > minRemaining {
		return BudgetStatus{
			Allowed:            false,
			RemainingWorkspace: remainingWorkspace,
			RemainingDaily:     remainingDaily,
			WarningLevel:       "exceeded",
			Message:            "Task cost exceeds remaining budget",
		}
	}

	if estimatedCost > config.TaskMaxCost {
		return BudgetStatus{
			Allowed:            false,
			RemainingWorkspace: remainingWorkspace,
			RemainingDaily:     remainingDaily,
			WarningLevel:       "exceeded",
			Message:            "Task exceeds maximum single-task cost",
		}
	}

	usageRatio := (dailySpend + estimatedCost) / config.DailyBudget
	level := "ok"
	if usageRatio > 0.9 {
		level = "critical"
	} else if usageRatio > config.WarningThreshold {
		level = "warning"
	}

	return BudgetStatus{
		Allowed:            true,
		RemainingWorkspace: remainingWorkspace - estimatedCost,
		RemainingDaily:     remainingDaily - estimatedCost,
		WarningLevel:       level,
		Message:            "Task approved",
	}
}

// Helper functions
func getBoolVal(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getFloatVal(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0.0
}

func getStringVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// calculateBurnRateCLI calculates burn rate using AILANG with Go fallback
func calculateBurnRateCLI(engine *ailembed.Engine, dailySpend float64, windowHours int) BurnRateInfo {
	info := BurnRateInfo{
		WindowHours: windowHours,
	}

	// For CLI, we estimate burn rate from daily spend divided by assumed active hours
	// This is a simplification since we don't have access to task history
	if dailySpend > 0 && windowHours > 0 {
		info.CostPerHour = dailySpend / float64(windowHours)
	}

	return info
}

// calculateExhaustionCLI calculates hours until budget exhaustion using AILANG with Go fallback
func calculateExhaustionCLI(engine *ailembed.Engine, remainingBudget, burnRate float64) int {
	if burnRate <= 0 {
		return -1 // No burn, infinite time
	}

	// Try AILANG first
	result, err := engine.Call(
		"internal/dashboard_transforms/budget_checker",
		"forecastExhaustion",
		remainingBudget,
		burnRate,
	)
	if err != nil {
		// Go fallback
		return int(remainingBudget / burnRate)
	}

	// Parse Option[int] result
	goResult, err := ailembed.ToGo(result)
	if err != nil {
		return int(remainingBudget / burnRate)
	}

	if resultMap, ok := goResult.(map[string]interface{}); ok {
		if _, exists := resultMap["value"]; exists {
			switch v := resultMap["value"].(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			}
		}
		// Check for tag-based ADT representation
		if tag, exists := resultMap["_tag"]; exists && tag == "Some" {
			switch v := resultMap["value"].(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			}
		}
	}

	return -1 // None case
}

func printBudgetHelp() {
	fmt.Println("Usage: ailang budget [command] [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  status    Show current budget status (default)")
	fmt.Println("  check     Check if a task can proceed within budget")
	fmt.Println("  help      Show this help")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --cost <amount>       Estimated task cost for checking (required for check)")
	fmt.Println("  --provider <name>     Check budget for specific provider (claude, gemini)")
	fmt.Println("  --all-providers       Show all provider budgets in status")
	fmt.Println("  --workspace-spend     Current workspace spend (USD)")
	fmt.Println("  --daily-spend         Current daily spend (USD)")
	fmt.Println("  --workspace-budget    Workspace budget limit (default: from config or $100)")
	fmt.Println("  --daily-budget        Daily budget limit (default: from config or $50)")
	fmt.Println("  --task-max            Max single task cost (default: from config or $25)")
	fmt.Println("  --warning-threshold   Warning threshold 0-1 (default: from config or 0.8)")
	fmt.Println("  --json                Output as JSON")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang budget status")
	fmt.Println("  ailang budget status --all-providers")
	fmt.Println("  ailang budget status --provider claude")
	fmt.Println("  ailang budget check --cost 5.0 --provider claude")
	fmt.Println("  ailang budget check --cost 10 --daily-spend 45 --json")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Per-provider budgets are loaded from ~/.ailang/config.yaml:")
	fmt.Println()
	fmt.Println("  budgets:")
	fmt.Println("    global:")
	fmt.Println("      daily_budget: 50.0")
	fmt.Println("      task_max_cost: 25.0")
	fmt.Println("    providers:")
	fmt.Println("      claude:")
	fmt.Println("        daily_budget: 30.0")
	fmt.Println("        hard_limit: true")
	fmt.Println("      gemini:")
	fmt.Println("        daily_budget: 20.0")
	fmt.Println()
	fmt.Println("This command demonstrates AILANG dogfooding:")
	fmt.Println("  - Calls AILANG budget_checker.ail with contracts")
	fmt.Println("  - Falls back to Go implementation on error")
	fmt.Println("  - AILANG uses 'requires' contracts for validation")
}
