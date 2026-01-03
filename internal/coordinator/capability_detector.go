package coordinator

import (
	"strings"
)

// CapabilityType represents the type of capability required
type CapabilityType string

const (
	// AILANG effect capabilities (map to language effects)
	CapabilityIO    CapabilityType = "IO"    // Console input/output
	CapabilityFS    CapabilityType = "FS"    // File system access
	CapabilityNet   CapabilityType = "Net"   // Network access
	CapabilityClock CapabilityType = "Clock" // Time/sleep operations
	CapabilityEnv   CapabilityType = "Env"   // Environment variables
	CapabilityAI    CapabilityType = "AI"    // AI/LLM operations
	CapabilityDebug CapabilityType = "Debug" // Debugging operations

	// Coordinator-specific capabilities (not language effects)
	CapabilityShell  CapabilityType = "Shell"  // Shell/bash execution (high risk)
	CapabilityBudget CapabilityType = "Budget" // High-cost operations
)

// Capability represents a detected capability requirement
type Capability struct {
	Type        CapabilityType `json:"type"`
	Paths       []string       `json:"paths,omitempty"`       // Affected paths or URLs
	BudgetDelta float64        `json:"budget_delta,omitempty"` // Estimated cost if Budget type
}

// CapabilityDetector analyzes task content and determines required capabilities
type CapabilityDetector struct{}

// NewCapabilityDetector creates a new capability detector
func NewCapabilityDetector() *CapabilityDetector {
	return &CapabilityDetector{}
}

// DetectCapabilities analyzes content and returns required capabilities
// Returns nil if no special capabilities are needed (safe execution)
func (cd *CapabilityDetector) DetectCapabilities(content string) []Capability {
	var caps []Capability
	lowerContent := strings.ToLower(content)

	// AILANG effect capabilities
	if cd.needsIO(lowerContent) {
		caps = append(caps, Capability{Type: CapabilityIO})
	}

	if cd.needsFileSystem(lowerContent) {
		caps = append(caps, Capability{
			Type:  CapabilityFS,
			Paths: cd.extractPaths(content),
		})
	}

	if cd.needsNetwork(lowerContent) {
		caps = append(caps, Capability{
			Type:  CapabilityNet,
			Paths: cd.extractURLs(content),
		})
	}

	if cd.needsClock(lowerContent) {
		caps = append(caps, Capability{Type: CapabilityClock})
	}

	if cd.needsEnv(lowerContent) {
		caps = append(caps, Capability{Type: CapabilityEnv})
	}

	if cd.needsAI(lowerContent) {
		caps = append(caps, Capability{Type: CapabilityAI})
	}

	if cd.needsDebug(lowerContent) {
		caps = append(caps, Capability{Type: CapabilityDebug})
	}

	// Coordinator-specific capabilities
	if cd.needsShell(lowerContent) {
		caps = append(caps, Capability{
			Type:  CapabilityShell,
			Paths: []string{"*"}, // Shell has access to everything
		})
	}

	if cd.needsHighCost(lowerContent) {
		caps = append(caps, Capability{
			Type:        CapabilityBudget,
			BudgetDelta: cd.estimateCost(content),
		})
	}

	return caps
}

// needsFileSystem checks if content requires file system access
func (cd *CapabilityDetector) needsFileSystem(content string) bool {
	fsKeywords := []string{
		"file", "write", "create", "delete", "modify", "edit",
		"save", "read", "directory", "folder", "path",
		"refactor", "install", // Refactoring and installing typically modify files
	}

	for _, keyword := range fsKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// needsNetwork checks if content requires network access
func (cd *CapabilityDetector) needsNetwork(content string) bool {
	netKeywords := []string{
		"http", "https", "api", "fetch", "download",
		"upload", "request", "curl", "webhook",
	}

	for _, keyword := range netKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// needsShell checks if content requires shell/bash execution
func (cd *CapabilityDetector) needsShell(content string) bool {
	shellKeywords := []string{
		"bash", "shell", "command", "execute", "run script",
		"install", "npm", "git", "docker", "make",
	}

	for _, keyword := range shellKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// needsHighCost checks if content might be expensive
func (cd *CapabilityDetector) needsHighCost(content string) bool {
	highCostKeywords := []string{
		"refactor", "analyze entire", "comprehensive", "all files",
		"migrate", "rewrite", "complete overhaul",
	}

	for _, keyword := range highCostKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// needsIO checks if content requires console I/O
func (cd *CapabilityDetector) needsIO(content string) bool {
	ioKeywords := []string{
		"print", "console", "output", "display", "show",
		"log", "stdin", "stdout", "readline", "input",
	}

	for _, keyword := range ioKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// needsClock checks if content requires time/clock operations
func (cd *CapabilityDetector) needsClock(content string) bool {
	clockKeywords := []string{
		"time", "clock", "sleep", "delay", "wait",
		"timestamp", "duration", "timeout", "schedule",
	}

	for _, keyword := range clockKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// needsEnv checks if content requires environment variable access
func (cd *CapabilityDetector) needsEnv(content string) bool {
	envKeywords := []string{
		"env", "environment", "variable", "getenv", "setenv",
		"$", "secret", "config", "credential",
	}

	for _, keyword := range envKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// needsAI checks if content requires AI/LLM operations
func (cd *CapabilityDetector) needsAI(content string) bool {
	aiKeywords := []string{
		"ai", "llm", "claude", "gpt", "gemini", "openai",
		"anthropic", "model", "prompt", "generate", "completion",
		"embedding", "chat", "inference",
	}

	for _, keyword := range aiKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// needsDebug checks if content requires debugging operations
func (cd *CapabilityDetector) needsDebug(content string) bool {
	debugKeywords := []string{
		"debug", "trace", "breakpoint", "inspect", "dump",
		"profil", "benchmark", "diagnos",
	}

	for _, keyword := range debugKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// extractPaths attempts to extract file paths from content
func (cd *CapabilityDetector) extractPaths(content string) []string {
	paths := []string{}

	// Look for quoted paths
	parts := strings.Split(content, `"`)
	for i := 1; i < len(parts); i += 2 {
		if strings.Contains(parts[i], "/") || strings.Contains(parts[i], ".") {
			paths = append(paths, parts[i])
		}
	}

	// If no specific paths found, return wildcard
	if len(paths) == 0 {
		paths = []string{"*"}
	}

	return paths
}

// extractURLs attempts to extract URLs from content
func (cd *CapabilityDetector) extractURLs(content string) []string {
	urls := []string{}

	words := strings.Fields(content)
	for _, word := range words {
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			urls = append(urls, word)
		}
	}

	// If no specific URLs found, return wildcard
	if len(urls) == 0 {
		urls = []string{"*"}
	}

	return urls
}

// estimateCost estimates the cost of a task in USD based on content complexity
func (cd *CapabilityDetector) estimateCost(content string) float64 {
	words := len(strings.Fields(content))

	if words > 50 {
		return 0.50 // Complex task, estimated $0.50
	} else if words > 20 {
		return 0.20 // Medium task, estimated $0.20
	}

	return 0.10 // Simple task, estimated $0.10
}

// ClassifyImpact returns impact level as "low", "medium", or "high"
func (cd *CapabilityDetector) ClassifyImpact(caps []Capability) string {
	if len(caps) == 0 {
		return "low"
	}

	// Shell operations are always high risk
	for _, cap := range caps {
		if cap.Type == CapabilityShell {
			return "high"
		}
	}

	// AI operations are high risk (external API calls, potentially expensive)
	for _, cap := range caps {
		if cap.Type == CapabilityAI {
			return "high"
		}
	}

	// Network, FS, Env, or high-cost operations are medium risk
	for _, cap := range caps {
		switch cap.Type {
		case CapabilityNet, CapabilityBudget, CapabilityFS, CapabilityEnv:
			return "medium"
		}
	}

	// IO, Clock, Debug are low risk
	return "low"
}

// FormatImpact formats a human-readable impact description
func (cd *CapabilityDetector) FormatImpact(caps []Capability) string {
	if len(caps) == 0 {
		return "Low risk - read-only operations"
	}

	var impacts []string
	for _, cap := range caps {
		switch cap.Type {
		case CapabilityIO:
			impacts = append(impacts, "Console I/O")
		case CapabilityFS:
			if len(cap.Paths) > 0 && cap.Paths[0] != "*" {
				impacts = append(impacts, "May modify files: "+strings.Join(cap.Paths, ", "))
			} else {
				impacts = append(impacts, "May modify files in workspace")
			}
		case CapabilityNet:
			impacts = append(impacts, "May make network requests")
		case CapabilityClock:
			impacts = append(impacts, "Time/scheduling operations")
		case CapabilityEnv:
			impacts = append(impacts, "May access environment variables")
		case CapabilityAI:
			impacts = append(impacts, "HIGH RISK - External AI/LLM API calls")
		case CapabilityDebug:
			impacts = append(impacts, "Debugging operations")
		case CapabilityShell:
			impacts = append(impacts, "HIGH RISK - Can execute arbitrary shell commands")
		case CapabilityBudget:
			impacts = append(impacts, "May incur additional costs")
		}
	}

	return strings.Join(impacts, "; ")
}

// EstimateTotalCost calculates total estimated cost from capabilities
func (cd *CapabilityDetector) EstimateTotalCost(caps []Capability, baseExecutionCost float64) float64 {
	total := baseExecutionCost

	for _, cap := range caps {
		total += cap.BudgetDelta
	}

	return total
}
