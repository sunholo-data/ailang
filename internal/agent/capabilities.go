package agent

import (
	"strings"

	"github.com/sunholo/ailang/internal/messaging"
)

// CapabilityDetector analyzes directives and determines required capabilities
type CapabilityDetector struct{}

// NewCapabilityDetector creates a new capability detector
func NewCapabilityDetector() *CapabilityDetector {
	return &CapabilityDetector{}
}

// DetectCapabilities analyzes a directive and returns required effect deltas
// Returns nil if no special capabilities are needed (safe execution)
func (cd *CapabilityDetector) DetectCapabilities(directive string) []*messaging.EffectDelta {
	var deltas []*messaging.EffectDelta
	lowerDirective := strings.ToLower(directive)

	// Detect file system operations
	if cd.needsFileSystem(lowerDirective) {
		deltas = append(deltas, &messaging.EffectDelta{
			CapType:     "FS",
			Paths:       cd.extractPaths(directive),
			BudgetDelta: 0.0, // No cost for FS operations
		})
	}

	// Detect network operations
	if cd.needsNetwork(lowerDirective) {
		deltas = append(deltas, &messaging.EffectDelta{
			CapType:     "Net",
			Paths:       cd.extractURLs(directive),
			BudgetDelta: 0.0,
		})
	}

	// Detect shell/bash operations (high risk)
	if cd.needsShell(lowerDirective) {
		deltas = append(deltas, &messaging.EffectDelta{
			CapType:     "Shell",
			Paths:       []string{"*"}, // Shell has access to everything
			BudgetDelta: 0.0,
		})
	}

	// Detect high-cost operations (expensive API calls)
	if cd.needsHighCost(lowerDirective) {
		deltas = append(deltas, &messaging.EffectDelta{
			CapType:     "Budget",
			Paths:       []string{},
			BudgetDelta: cd.estimateCost(directive),
		})
	}

	return deltas
}

// needsFileSystem checks if directive requires file system access
func (cd *CapabilityDetector) needsFileSystem(directive string) bool {
	fsKeywords := []string{
		"file", "write", "create", "delete", "modify", "edit",
		"save", "read", "directory", "folder", "path",
		"refactor", "install", // Refactoring and installing typically modify files
	}

	for _, keyword := range fsKeywords {
		if strings.Contains(directive, keyword) {
			return true
		}
	}
	return false
}

// needsNetwork checks if directive requires network access
func (cd *CapabilityDetector) needsNetwork(directive string) bool {
	netKeywords := []string{
		"http", "https", "api", "fetch", "download",
		"upload", "request", "curl", "webhook",
	}

	for _, keyword := range netKeywords {
		if strings.Contains(directive, keyword) {
			return true
		}
	}
	return false
}

// needsShell checks if directive requires shell/bash execution
func (cd *CapabilityDetector) needsShell(directive string) bool {
	shellKeywords := []string{
		"bash", "shell", "command", "execute", "run script",
		"install", "npm", "git", "docker", "make",
	}

	for _, keyword := range shellKeywords {
		if strings.Contains(directive, keyword) {
			return true
		}
	}
	return false
}

// needsHighCost checks if directive might be expensive
func (cd *CapabilityDetector) needsHighCost(directive string) bool {
	// For now, all directives have some cost, but we only flag high-cost ones
	// High-cost: complex analysis, large codebases, extensive refactoring
	highCostKeywords := []string{
		"refactor", "analyze entire", "comprehensive", "all files",
		"migrate", "rewrite", "complete overhaul",
	}

	for _, keyword := range highCostKeywords {
		if strings.Contains(directive, keyword) {
			return true
		}
	}
	return false
}

// extractPaths attempts to extract file paths from directive
func (cd *CapabilityDetector) extractPaths(directive string) []string {
	// Simple heuristic: look for common path patterns
	paths := []string{}

	// Look for quoted paths
	parts := strings.Split(directive, `"`)
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

// extractURLs attempts to extract URLs from directive
func (cd *CapabilityDetector) extractURLs(directive string) []string {
	urls := []string{}

	// Simple heuristic: look for http/https URLs
	words := strings.Fields(directive)
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

// estimateCost estimates the cost of a directive in USD
func (cd *CapabilityDetector) estimateCost(directive string) float64 {
	// Simple heuristic based on directive complexity
	words := len(strings.Fields(directive))

	if words > 50 {
		return 0.50 // Complex directive, estimated $0.50
	} else if words > 20 {
		return 0.20 // Medium directive, estimated $0.20
	}

	return 0.10 // Simple directive, estimated $0.10
}

// FormatProposal formats a human-readable approval proposal
func (cd *CapabilityDetector) FormatProposal(deltas []*messaging.EffectDelta) string {
	if len(deltas) == 0 {
		return "Execute directive (no special capabilities required)"
	}

	var parts []string
	for _, delta := range deltas {
		switch delta.CapType {
		case "FS":
			parts = append(parts, "File system access")
		case "Net":
			parts = append(parts, "Network access")
		case "Shell":
			parts = append(parts, "Shell/bash execution")
		case "Budget":
			parts = append(parts, "High-cost operation")
		default:
			parts = append(parts, delta.CapType)
		}
	}

	return "Execute directive requiring: " + strings.Join(parts, ", ")
}

// ClassifyImpact returns impact level as "low", "medium", or "high" for database storage
func (cd *CapabilityDetector) ClassifyImpact(deltas []*messaging.EffectDelta) string {
	if len(deltas) == 0 {
		return "low"
	}

	// Shell operations are always high risk
	for _, delta := range deltas {
		if delta.CapType == "Shell" {
			return "high"
		}
	}

	// Network or high-cost operations are medium risk
	for _, delta := range deltas {
		if delta.CapType == "Net" || delta.CapType == "Budget" {
			return "medium"
		}
	}

	// File system only is low-medium risk
	return "medium"
}

// FormatImpact formats a human-readable impact description
func (cd *CapabilityDetector) FormatImpact(deltas []*messaging.EffectDelta) string {
	if len(deltas) == 0 {
		return "Low risk - read-only operations"
	}

	var impacts []string
	for _, delta := range deltas {
		switch delta.CapType {
		case "FS":
			if len(delta.Paths) > 0 && delta.Paths[0] != "*" {
				impacts = append(impacts, "May modify files: "+strings.Join(delta.Paths, ", "))
			} else {
				impacts = append(impacts, "May modify files in workspace")
			}
		case "Net":
			impacts = append(impacts, "May make network requests")
		case "Shell":
			impacts = append(impacts, "⚠️ HIGH RISK - Can execute arbitrary shell commands")
		case "Budget":
			impacts = append(impacts, "May incur additional costs")
		}
	}

	return strings.Join(impacts, "; ")
}

// CalculateTotalCost calculates total estimated cost from deltas
func (cd *CapabilityDetector) CalculateTotalCost(deltas []*messaging.EffectDelta, baseExecutionCost float64) float64 {
	total := baseExecutionCost // Start with base Claude Code execution cost

	for _, delta := range deltas {
		total += delta.BudgetDelta
	}

	return total
}
