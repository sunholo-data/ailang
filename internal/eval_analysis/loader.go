package eval_analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/strutil"
)

// LoadResults loads all benchmark results from a directory
// Returns results sorted by timestamp (newest first)
// Recursively searches all subdirectories for .json files
func LoadResults(dir string) ([]*BenchmarkResult, error) {
	return LoadResultsFromDirs(dir)
}

// LoadResultsIncludingInvalid is LoadResults WITHOUT the validity filter.
//
// Use only when investigating the harness itself. Invalid rows are quarantined,
// never deleted (they are evidence of the bug that produced them), so this is
// how you reach them. Everything that reports a rate must use LoadResults.
func LoadResultsIncludingInvalid(dir string) ([]*BenchmarkResult, error) {
	return LoadResultsFromDirsIncludingInvalid(dir)
}

// LoadResultsFromDirsIncludingInvalid is LoadResultsFromDirs without the
// validity filter. See LoadResultsIncludingInvalid.
func LoadResultsFromDirsIncludingInvalid(dirs ...string) ([]*BenchmarkResult, error) {
	return loadResultsFromDirs(dirs, eval_harness.LoadOptions{IncludeInvalid: true})
}

// LoadResultsFromDirs loads and merges all benchmark results from one or more
// directories. This is the durable primitive behind `eval-report --merge`:
// it walks every directory, concatenates the results, and applies a single
// cross-directory dedup pass so overlapping result files (same
// model/benchmark/lang/seed/mode/trial) collapse to the newest one regardless
// of which directory they came from. The merge is order-independent because
// the dedup key is content-derived and selection is by timestamp, not by
// argument order.
//
// At least one directory must be provided. A directory is required to exist,
// but a directory that simply contains no JSON files is tolerated as long as
// the combined set across all directories is non-empty (so callers can merge a
// populated baseline with an empty/absent rotation without failing).
// Rows that are not MEASUREMENTS (dead subject, harness error, wrong config)
// are excluded by default — that default is the point of the measurement
// contract (see eval_harness.LoadOptions.IncludeInvalid). Use
// LoadResultsFromDirsIncludingInvalid to opt back in.
func LoadResultsFromDirs(dirs ...string) ([]*BenchmarkResult, error) {
	return loadResultsFromDirs(dirs, eval_harness.LoadOptions{})
}

// loadResultsFromDirs is the analysis-side adapter over the ONE row loader,
// eval_harness.LoadRows (M-V1-SIMPLIFY-S3 M1). The walk, decode, required-field
// check, newest-first order, per-slot dedup and validity filter all live there;
// this only wraps each row in BenchmarkResult, applies the read-side refusal
// annotation, and keeps this package's error contract: an empty result set is
// an error here, and files that failed to decode are fatal only when NOTHING
// loaded.
func loadResultsFromDirs(dirs []string, opts eval_harness.LoadOptions) ([]*BenchmarkResult, error) {
	rows, stats, err := eval_harness.LoadRows(dirs, opts)
	if err != nil {
		return nil, err
	}
	if stats.Files == 0 {
		return nil, fmt.Errorf("no JSON files found in %v", dirs)
	}
	if len(stats.ParseErrors) > 0 && stats.Loaded == 0 && stats.Invalid == 0 && stats.Duplicates == 0 {
		return nil, fmt.Errorf("failed to load any results: %v", stats.ParseErrors)
	}
	results := make([]*BenchmarkResult, 0, len(rows))
	for i := range rows {
		results = append(results, annotate(rows[i]))
	}
	return results, nil
}

// annotate wraps a banked row with the read-side annotations the harness never
// writes. RefusalDetected is derived here so historical baselines — written
// before the field existed — inherit it.
func annotate(m eval_harness.RunMetrics) *BenchmarkResult {
	r := &BenchmarkResult{RunMetrics: m}
	r.RefusalDetected = DetectRefusal(r.Code, r.Stderr, r.Stdout)
	return r
}

// LoadArmForPairing loads ONE A/B arm WITHOUT the re-run dedup.
//
// This is LoadResultsFromDirs today — kept as a named entry point because
// paired analysis has a requirement the general loader must never quietly lose:
// every TRIAL must survive. Until 2026-07-29 the dedup key omitted Trial, so an
// 84-row arm loaded as 42 AND the surviving trial could differ between arms,
// silently producing delta=9.5/unpaired=46 instead of the correct
// delta=13.1/unpaired=0. Trial is now part of the key; this name documents the
// dependency so it is not re-broken.
func LoadArmForPairing(dir string) ([]*BenchmarkResult, error) {
	return LoadResultsFromDirs(dir)
}

// LoadResult loads a single benchmark result file, including an invalid one
// (a single-row read is an inspection, not a rate).
func LoadResult(path string) (*BenchmarkResult, error) {
	rows, stats, err := eval_harness.LoadRowFiles([]string{path}, eval_harness.LoadOptions{IncludeInvalid: true})
	if err != nil {
		return nil, err
	}
	if len(stats.ParseErrors) > 0 {
		return nil, fmt.Errorf("%s", stats.ParseErrors[0])
	}
	if len(rows) != 1 {
		return nil, fmt.Errorf("expected one row in %s, got %d", path, len(rows))
	}
	return annotate(rows[0]), nil
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

	// If baseline.json was missing or carried a zero timestamp (true for
	// every pre-M-EVAL-SUITE-PREP baseline that only has result JSONs),
	// fall back to the newest result's timestamp. Without this the history
	// entry gets serialised as "0001-01-01T00:00:00Z" and PerModelTrend's
	// `date.getFullYear() > 2000` filter silently drops the version from
	// the line chart — regression noticed when the recursive hasJSONFiles
	// fix started picking up these older baselines.
	if baseline.Timestamp.IsZero() && len(results) > 0 {
		baseline.Timestamp = results[0].Timestamp
	}

	// Recalculate stats from loaded results (in case metadata is stale)
	baseline.TotalBenchmarks = len(results)
	baseline.SuccessCount = 0
	baseline.FailCount = 0

	for _, r := range results {
		if r.Passed() {
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
			hasMetadata := strutil.FileExists(filepath.Join(dir, "baseline.json"))
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
		if filter.SuccessOnly && !r.Passed() {
			continue
		}
		if filter.FailuresOnly && r.Passed() {
			continue
		}

		filtered = append(filtered, r)
	}

	return filtered
}

// Helper functions

func hasJSONFiles(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if !d.IsDir() && filepath.Ext(path) == ".json" && filepath.Base(path) != "baseline.json" {
			found = true
		}
		return nil
	})
	return found
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
