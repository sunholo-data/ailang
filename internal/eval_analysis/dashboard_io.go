package eval_analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// loadExistingDashboard reads the existing dashboard JSON file and returns its structure
// If the file doesn't exist, returns an empty dashboard with an empty history array
func loadExistingDashboard(path string) (*DashboardJSON, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &DashboardJSON{History: []HistoryEntry{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var dashboard DashboardJSON
	if err := json.Unmarshal(data, &dashboard); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return &dashboard, nil
}

// normalizeVersion ensures consistent "v" prefix for version strings.
// Both "0.9.0" and "v0.9.0" become "v0.9.0".
func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	return "v" + v
}

// mergeHistory adds a new entry to the dashboard history or updates an existing entry
// If the version already exists, it updates that entry. Otherwise, prepends the new entry.
// History is maintained in reverse chronological order (newest first)
// Version comparison is normalized: "v0.9.0" and "0.9.0" are treated as the same version.
func mergeHistory(dashboard *DashboardJSON, newEntry HistoryEntry) {
	// Normalize version on the new entry
	newEntry.Version = normalizeVersion(newEntry.Version)

	// Check for duplicate version (normalized to handle v-prefix inconsistency)
	for i, entry := range dashboard.History {
		if normalizeVersion(entry.Version) == newEntry.Version {
			// Update existing entry
			dashboard.History[i] = newEntry
			return
		}
	}

	// Prepend new entry (reverse chronological order)
	dashboard.History = append([]HistoryEntry{newEntry}, dashboard.History...)
}

// buildHistoryEntryFromMatrix creates a HistoryEntry from a PerformanceMatrix and results
func buildHistoryEntryFromMatrix(matrix *PerformanceMatrix, results []*BenchmarkResult) HistoryEntry {
	successCount := 0
	for _, r := range results {
		if r.StdoutOk {
			successCount++
		}
	}

	successRate := 0.0
	if matrix.TotalRuns > 0 {
		successRate = float64(successCount) / float64(matrix.TotalRuns)
	}

	// Build language stats
	langStats := make(map[string]interface{})
	for lang, stats := range matrix.Languages {
		if stats.TotalRuns > 0 {
			langStats[lang] = map[string]interface{}{
				"success_rate": stats.SuccessRate,
				"total_runs":   stats.TotalRuns,
			}
		}
	}

	// Build per-model stats (for trend charts)
	modelStats := make(map[string]interface{})
	for modelName, modelData := range matrix.Models {
		if len(modelData.Languages) > 0 {
			modelLangStats := make(map[string]interface{})
			for lang, langData := range modelData.Languages {
				if langData.TotalRuns > 0 {
					modelLangStats[lang] = map[string]interface{}{
						"successRate": langData.SuccessRate,
						"totalRuns":   langData.TotalRuns,
						"avgTokens":   langData.AvgTokens,
					}
				}
			}
			if len(modelLangStats) > 0 {
				modelStats[modelName] = modelLangStats
			}
		}
	}

	// Determine languages string
	languages := ""
	if len(matrix.Languages) > 0 {
		langList := make([]string, 0, len(matrix.Languages))
		for lang := range matrix.Languages {
			langList = append(langList, lang)
		}
		sort.Strings(langList)
		languages = strings.Join(langList, ",")
	}

	entry := HistoryEntry{
		Version:       matrix.Version,
		Timestamp:     matrix.Timestamp.Format(time.RFC3339),
		SuccessRate:   successRate,
		TotalRuns:     matrix.TotalRuns,
		SuccessCount:  successCount,
		Languages:     languages,
		LanguageStats: langStats,
	}

	// Only add ModelStats if we have data (keeps backward compatibility)
	if len(modelStats) > 0 {
		entry.ModelStats = modelStats
	}

	return entry
}

// writeJSONAtomic writes JSON data to a file atomically
// Uses a temp file + rename to ensure all-or-nothing writes
func writeJSONAtomic(path string, data interface{}) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	tmpPath := path + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tmpPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Validate temp file
	tmpData, err := os.ReadFile(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to read temp file: %w", err)
	}

	// Parse and validate
	if dashboard, ok := data.(*DashboardJSON); ok {
		var test DashboardJSON
		if err := json.Unmarshal(tmpData, &test); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("validation failed: %w", err)
		}

		if err := test.Validate(); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("validation failed: %w", err)
		}

		// Verify version matches
		if test.Version != dashboard.Version {
			os.Remove(tmpPath)
			return fmt.Errorf("version mismatch after marshaling: expected %s, got %s",
				dashboard.Version, test.Version)
		}
	}

	// Atomic rename (on Unix, overwrites atomically)
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename: %w", err)
	}

	return nil
}
