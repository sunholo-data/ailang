package eval_analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/strutil"
)

// IssueReport represents a pattern of failures discovered in eval results
type IssueReport struct {
	Category      string   `json:"category"`       // "syntax_error", "type_error", "missing_feature", etc.
	Title         string   `json:"title"`          // Human-readable issue title
	Frequency     int      `json:"frequency"`      // How many evals hit this
	Benchmarks    []string `json:"benchmarks"`     // Which benchmarks failed
	Examples      []string `json:"examples"`       // Failed code examples
	ErrorMessages []string `json:"error_messages"` // Stderr from failures
	Impact        string   `json:"impact"`         // "critical", "high", "medium", "low"
	Lang          string   `json:"lang"`           // Language where issue occurred
	Models        []string `json:"models"`         // Models that encountered this
}

// AnalysisResult contains all issues discovered from eval results
type AnalysisResult struct {
	Issues       []IssueReport `json:"issues"`
	TotalRuns    int           `json:"total_runs"`
	FailureCount int           `json:"failure_count"`
	SuccessRate  float64       `json:"success_rate"`
	// InvalidExcluded is the number of banked rows that were NOT analysed
	// because they are not measurements (harness crash, dead subject, wrong
	// config — eval_harness.Validity). Before M-V1-SIMPLIFY-S3 M1 these were
	// counted as model failures.
	InvalidExcluded int `json:"invalid_excluded,omitempty"`
}

// Analyzer aggregates eval results and identifies patterns
type Analyzer struct {
	resultsDir   string
	minFrequency int
	categories   map[string]bool
	// invalidExcluded counts rows the loader dropped because they are not
	// measurements (validity.go). Reported on AnalysisResult so a shrunken
	// sample is stated, never silent.
	invalidExcluded int
}

// NewAnalyzer creates a new eval results analyzer
func NewAnalyzer(resultsDir string, minFrequency int, categories []string) *Analyzer {
	catMap := make(map[string]bool)
	for _, cat := range categories {
		catMap[cat] = true
	}

	return &Analyzer{
		resultsDir:   resultsDir,
		minFrequency: minFrequency,
		categories:   catMap,
	}
}

// Analyze processes all eval results and returns discovered issues
func (a *Analyzer) Analyze() (*AnalysisResult, error) {
	// Load all metrics files
	metrics, err := a.loadAllMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to load metrics: %w", err)
	}

	if len(metrics) == 0 {
		return nil, fmt.Errorf("no eval results found in %s", a.resultsDir)
	}

	// Separate failures from successes
	var failures []*eval_harness.RunMetrics
	var successes []*eval_harness.RunMetrics

	for _, m := range metrics {
		if !m.Passed() {
			failures = append(failures, m)
		} else {
			successes = append(successes, m)
		}
	}

	// Group failures by error pattern
	issues := a.extractIssues(failures)

	// Filter by frequency and category
	issues = a.filterIssues(issues)

	// Calculate impact
	for i := range issues {
		issues[i].Impact = calculateImpact(issues[i], len(failures))
	}

	// Sort by impact (critical first)
	sort.Slice(issues, func(i, j int) bool {
		return impactScore(issues[i].Impact) > impactScore(issues[j].Impact)
	})

	successRate := 0.0
	if len(metrics) > 0 {
		successRate = float64(len(successes)) / float64(len(metrics)) * 100.0
	}

	return &AnalysisResult{
		Issues:          issues,
		TotalRuns:       len(metrics),
		FailureCount:    len(failures),
		SuccessRate:     successRate,
		InvalidExcluded: a.invalidExcluded,
	}, nil
}

// loadAllMetrics loads every banked row under the results directory through
// the ONE row loader, eval_harness.LoadRows (M-V1-SIMPLIFY-S3 M1).
//
// Until then this package had its own decoder: a flat *.json glob with no
// validity filter, so every harness crash banked as api_error was analysed as
// a model failure and surfaced as an "issue" — and rows the suite wrote under
// standard/ or agent/ were never seen at all. Now invalid rows are excluded
// (the default), the tree is walked, and re-runs collapse to the newest row
// per slot, the same as every published rate.
func (a *Analyzer) loadAllMetrics() ([]*eval_harness.RunMetrics, error) {
	rows, stats, err := eval_harness.LoadRows([]string{a.resultsDir}, eval_harness.LoadOptions{})
	if err != nil {
		return nil, err
	}
	a.invalidExcluded = stats.Invalid
	metrics := make([]*eval_harness.RunMetrics, 0, len(rows))
	for i := range rows {
		metrics = append(metrics, &rows[i])
	}
	return metrics, nil
}

// extractIssues groups failures by error pattern and creates issue reports
func (a *Analyzer) extractIssues(failures []*eval_harness.RunMetrics) []IssueReport {
	// Group by error category and language
	type key struct {
		category string
		lang     string
	}

	groups := make(map[key][]eval_harness.RunMetrics)

	for _, m := range failures {
		k := key{
			category: m.ErrorCategory,
			lang:     m.Lang,
		}
		groups[k] = append(groups[k], *m)
	}

	// Convert groups to issue reports
	var issues []IssueReport

	for k, group := range groups {
		// Extract unique benchmarks, models, error messages
		benchmarks := make(map[string]bool)
		models := make(map[string]bool)
		var errorMsgs []string
		var examples []string

		for _, m := range group {
			benchmarks[m.ID] = true
			models[m.Model] = true

			if m.Stderr != "" && !contains(errorMsgs, m.Stderr) {
				errorMsgs = append(errorMsgs, strutil.Truncate(m.Stderr, 500+3)) // N+3: the old local truncate kept N chars THEN appended "..."
			}

			if m.Code != "" && len(examples) < 3 {
				examples = append(examples, strutil.Truncate(m.Code, 1000+3))
			}
		}

		// Convert maps to sorted slices
		benchmarkList := strutil.SortedKeys(benchmarks)
		modelList := strutil.SortedKeys(models)

		// Generate title from category and context
		title := generateTitle(k.category, k.lang, benchmarkList)

		issues = append(issues, IssueReport{
			Category:      k.category,
			Title:         title,
			Frequency:     len(group),
			Benchmarks:    benchmarkList,
			Examples:      examples,
			ErrorMessages: errorMsgs,
			Lang:          k.lang,
			Models:        modelList,
		})
	}

	return issues
}

// filterIssues applies frequency and category filters
func (a *Analyzer) filterIssues(issues []IssueReport) []IssueReport {
	var filtered []IssueReport

	for _, issue := range issues {
		// Filter by frequency
		if issue.Frequency < a.minFrequency {
			continue
		}

		// Filter by category (if specified)
		if len(a.categories) > 0 && !a.categories[issue.Category] {
			continue
		}

		filtered = append(filtered, issue)
	}

	return filtered
}

// calculateImpact determines the severity of an issue
func calculateImpact(issue IssueReport, totalFailures int) string {
	// Impact based on frequency and category
	percentage := float64(issue.Frequency) / float64(totalFailures) * 100.0

	// Critical: affects > 50% of failures or is a compile error
	if percentage > 50.0 || issue.Category == eval_harness.ErrorCategoryCompile {
		return "critical"
	}

	// High: affects > 25% of failures or is a runtime error
	if percentage > 25.0 || issue.Category == eval_harness.ErrorCategoryRuntime {
		return "high"
	}

	// Medium: affects > 10% of failures
	if percentage > 10.0 {
		return "medium"
	}

	return "low"
}

// impactScore converts impact string to numeric score for sorting
func impactScore(impact string) int {
	switch impact {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// generateTitle creates a human-readable title for an issue
func generateTitle(category, lang string, benchmarks []string) string {
	langPrefix := strings.ToUpper(lang)

	switch category {
	case eval_harness.ErrorCategoryCompile:
		return fmt.Sprintf("%s: Compilation Failures", langPrefix)
	case eval_harness.ErrorCategoryRuntime:
		return fmt.Sprintf("%s: Runtime Errors", langPrefix)
	case eval_harness.ErrorCategoryLogic:
		return fmt.Sprintf("%s: Logic Errors in %s", langPrefix, strings.Join(benchmarks, ", "))
	default:
		return fmt.Sprintf("%s: %s", langPrefix, category)
	}
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
