package eval_analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// LoadResults loads all benchmark results from a directory
// Returns results sorted by timestamp (newest first)
func LoadResults(dir string) ([]*BenchmarkResult, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory not found: %s", dir)
	}

	// Find all JSON files
	pattern := filepath.Join(dir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no JSON files found in %s", dir)
	}

	var results []*BenchmarkResult
	var errors []string

	for _, path := range matches {
		// Skip baseline.json metadata file
		if filepath.Base(path) == "baseline.json" {
			continue
		}

		result, err := LoadResult(path)
		if err != nil {
			// Collect errors but don't fail completely
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			continue
		}

		results = append(results, result)
	}

	// Report errors if some files failed to load
	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("failed to load any results: %v", errors)
	}

	// Sort by timestamp (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	return results, nil
}

// LoadResult loads a single benchmark result from a JSON file
func LoadResult(path string) (*BenchmarkResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var result BenchmarkResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate required fields
	if result.ID == "" {
		return nil, fmt.Errorf("missing required field: id")
	}
	if result.Lang == "" {
		return nil, fmt.Errorf("missing required field: lang")
	}
	if result.Model == "" {
		return nil, fmt.Errorf("missing required field: model")
	}

	return &result, nil
}

// LoadBaseline loads a baseline from a directory
// Expects baseline.json metadata + result JSON files
func LoadBaseline(dir string) (*Baseline, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("baseline directory not found: %s", dir)
	}

	// Load metadata
	metadataPath := filepath.Join(dir, "baseline.json")
	var baseline Baseline

	if _, err := os.Stat(metadataPath); err == nil {
		// Metadata exists, load it
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read baseline metadata: %w", err)
		}

		if err := json.Unmarshal(data, &baseline); err != nil {
			return nil, fmt.Errorf("failed to parse baseline metadata: %w", err)
		}
	} else {
		// No metadata, create minimal baseline
		baseline.Version = filepath.Base(dir)
	}

	// Load results
	results, err := LoadResults(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to load baseline results: %w", err)
	}

	baseline.Results = results

	// Recalculate stats from loaded results (in case metadata is stale)
	baseline.TotalBenchmarks = len(results)
	baseline.SuccessCount = 0
	baseline.FailCount = 0

	for _, r := range results {
		if r.StdoutOk {
			baseline.SuccessCount++
		} else {
			baseline.FailCount++
		}
	}

	return &baseline, nil
}

// LoadBaselineByVersion loads a baseline by version name
// Looks in eval_results/baselines/<version>
func LoadBaselineByVersion(version string) (*Baseline, error) {
	dir := filepath.Join("eval_results", "baselines", version)
	return LoadBaseline(dir)
}

// ListBaselines returns a list of available baseline versions
func ListBaselines() ([]string, error) {
	baselinesDir := filepath.Join("eval_results", "baselines")

	// Check if baselines directory exists
	if _, err := os.Stat(baselinesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("no baselines found: %s does not exist", baselinesDir)
	}

	entries, err := os.ReadDir(baselinesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read baselines directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Check if it contains baseline.json or any result files
			dir := filepath.Join(baselinesDir, entry.Name())
			hasMetadata := fileExists(filepath.Join(dir, "baseline.json"))
			hasResults := hasJSONFiles(dir)

			if hasMetadata || hasResults {
				versions = append(versions, entry.Name())
			}
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no valid baselines found in %s", baselinesDir)
	}

	// Sort by name (reverse to get newest first, assuming version strings)
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))

	return versions, nil
}

// GetLatestBaseline returns the most recent baseline version
func GetLatestBaseline() (*Baseline, error) {
	versions, err := ListBaselines()
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no baselines available")
	}

	return LoadBaselineByVersion(versions[0])
}

// FilterResults returns results matching the given criteria
type ResultFilter struct {
	Model        string
	Lang         string
	Benchmark    string
	SuccessOnly  bool
	FailuresOnly bool
}

// Filter applies the filter to results
func Filter(results []*BenchmarkResult, filter ResultFilter) []*BenchmarkResult {
	var filtered []*BenchmarkResult

	for _, r := range results {
		// Apply filters
		if filter.Model != "" && r.Model != filter.Model {
			continue
		}
		if filter.Lang != "" && r.Lang != filter.Lang {
			continue
		}
		if filter.Benchmark != "" && r.ID != filter.Benchmark {
			continue
		}
		if filter.SuccessOnly && !r.StdoutOk {
			continue
		}
		if filter.FailuresOnly && r.StdoutOk {
			continue
		}

		filtered = append(filtered, r)
	}

	return filtered
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hasJSONFiles(dir string) bool {
	pattern := filepath.Join(dir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	return len(matches) > 0
}

// LoadLatestResultsPerModel aggregates results from multiple baselines,
// keeping the latest result for each model.
// Returns results and a map of model -> baseline version used
func LoadLatestResultsPerModel() ([]*BenchmarkResult, map[string]string, error) {
	// Get all baselines
	versions, err := ListBaselines()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list baselines: %w", err)
	}

	// Track latest result per (benchmark_id, lang, model) tuple
	type resultKey struct {
		ID    string
		Lang  string
		Model string
	}

	latestResults := make(map[resultKey]*BenchmarkResult)
	modelBaselines := make(map[string]string) // model -> baseline version

	// Process baselines from newest to oldest
	for _, version := range versions {
		baseline, err := LoadBaselineByVersion(version)
		if err != nil {
			// Skip baselines that fail to load
			continue
		}

		for _, result := range baseline.Results {
			key := resultKey{
				ID:    result.ID,
				Lang:  result.Lang,
				Model: result.Model,
			}

			// Only update if we don't have a result for this key yet
			// (since we're processing newest first)
			if _, exists := latestResults[key]; !exists {
				latestResults[key] = result

				// Track which baseline this model came from
				if _, tracked := modelBaselines[result.Model]; !tracked {
					modelBaselines[result.Model] = version
				}
			}
		}
	}

	// Convert map to slice
	var results []*BenchmarkResult
	for _, result := range latestResults {
		results = append(results, result)
	}

	// Sort by timestamp (newest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	return results, modelBaselines, nil
}
