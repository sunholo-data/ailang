// Package test provides structured test reporting for AI consumption.
package test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"runtime"
	"sort"
	"time"

	"github.com/sunholo/ailang/internal/schema"
)

// Case represents a single test case result
type Case struct {
	SID        string   `json:"sid"`
	Suite      string   `json:"suite"`
	Name       string   `json:"name"`
	Status     string   `json:"status"` // passed|failed|errored|skipped
	TimeMs     int64    `json:"time_ms"`
	TraceSlice []string `json:"trace_slice,omitempty"` // SIDs for failure navigation
	Error      any      `json:"error,omitempty"`       // errors.Encoded
}

// Counts provides test result statistics
type Counts struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Errored int `json:"errored"`
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
}

// Report represents a complete test run report
type Report struct {
	Schema        string   `json:"schema"`
	RunID         string   `json:"run_id"`
	Seed          *int     `json:"seed,omitempty"`
	EnvLockDigest string   `json:"env_lock_digest,omitempty"`
	DurationMs    int64    `json:"duration_ms"`
	Counts        Counts   `json:"counts"`
	Cases         []Case   `json:"cases"`
	Platform      Platform `json:"platform"`
}

// Platform captures environment information for reproducibility
type Platform struct {
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Timestamp string `json:"timestamp"`
}

// NewReport creates a new test report
func NewReport() *Report {
	return &Report{
		Schema: schema.TestV1,
		RunID:  generateRunID(),
		Cases:  []Case{},
		Platform: Platform{
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// AddCase adds a test case to the report
func (r *Report) AddCase(c Case) {
	r.Cases = append(r.Cases, c)

	// Update counts
	r.Counts.Total++
	switch c.Status {
	case "passed":
		r.Counts.Passed++
	case "failed":
		r.Counts.Failed++
	case "errored":
		r.Counts.Errored++
	case "skipped":
		r.Counts.Skipped++
	}
}

// Finalize sorts cases and sets final timing
func (r *Report) Finalize(startTime time.Time) {
	r.DurationMs = time.Since(startTime).Milliseconds()

	// Sort cases by (suite, name) for deterministic output
	sort.Slice(r.Cases, func(i, j int) bool {
		if r.Cases[i].Suite != r.Cases[j].Suite {
			return r.Cases[i].Suite < r.Cases[j].Suite
		}
		return r.Cases[i].Name < r.Cases[j].Name
	})
}

// SetSeed sets the random seed if deterministic testing was used
func (r *Report) SetSeed(seed int) {
	r.Seed = &seed
}

// SetEnvLockDigest sets the environment lock file digest
func (r *Report) SetEnvLockDigest(digest string) {
	r.EnvLockDigest = digest
}

// ToJSON converts the report to deterministic JSON
func (r *Report) ToJSON() ([]byte, error) {
	// Ensure we always have valid counts even with 0 tests
	if r.Cases == nil {
		r.Cases = []Case{}
	}

	data, err := schema.MarshalDeterministic(r)
	if err != nil {
		return nil, err
	}
	return schema.FormatJSON(data)
}

// generateRunID creates a unique run identifier
func generateRunID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // Ignore error as crypto/rand.Read rarely fails
	return hex.EncodeToString(b)
}

// GenerateTestSID generates a stable SID for a test case
func GenerateTestSID(suite, name string) string {
	// Create a unique identifier from suite and name using SHA256
	combined := suite + "::" + name
	hash := sha256.Sum256([]byte(combined))
	// Use first 8 bytes of hash for SID (16 hex chars)
	return "T#" + hex.EncodeToString(hash[:8])
}

// TestRunner provides methods for running and reporting tests
type TestRunner struct {
	report    *Report
	startTime time.Time
}

// NewRunner creates a new test runner
func NewRunner() *TestRunner {
	return &TestRunner{
		report:    NewReport(),
		startTime: time.Now(),
	}
}

// RunTest executes a test and records the result
func (tr *TestRunner) RunTest(suite, name string, testFunc func() error) {
	startTime := time.Now()
	sid := GenerateTestSID(suite, name)

	var status string
	var testErr any

	err := testFunc()
	if err != nil {
		status = "failed"
		testErr = err.Error() // Could be errors.Encoded
	} else {
		status = "passed"
	}

	timeMs := time.Since(startTime).Milliseconds()

	tr.report.AddCase(Case{
		SID:    sid,
		Suite:  suite,
		Name:   name,
		Status: status,
		TimeMs: timeMs,
		Error:  testErr,
	})
}

// Skip marks a test as skipped
func (tr *TestRunner) Skip(suite, name string, reason string) {
	sid := GenerateTestSID(suite, name)
	tr.report.AddCase(Case{
		SID:    sid,
		Suite:  suite,
		Name:   name,
		Status: "skipped",
		TimeMs: 0,
		Error:  reason,
	})
}

// GetReport finalizes and returns the test report
func (tr *TestRunner) GetReport() *Report {
	tr.report.Finalize(tr.startTime)
	return tr.report
}

// EmptyReport returns a valid empty report (for 0 tests)
func EmptyReport() *Report {
	r := NewReport()
	r.Finalize(time.Now())
	return r
}
