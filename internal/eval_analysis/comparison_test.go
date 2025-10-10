package eval_analysis

import (
	"testing"
	"time"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		baseline []*BenchmarkResult
		new      []*BenchmarkResult
		wantErr  bool
		checkFn  func(t *testing.T, report *ComparisonReport)
	}{
		{
			name:     "empty baseline",
			baseline: []*BenchmarkResult{},
			new:      []*BenchmarkResult{{ID: "test", Lang: "ailang", Model: "claude"}},
			wantErr:  true,
		},
		{
			name:     "empty new",
			baseline: []*BenchmarkResult{{ID: "test", Lang: "ailang", Model: "claude"}},
			new:      []*BenchmarkResult{},
			wantErr:  true,
		},
		{
			name: "fixed benchmark",
			baseline: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: false, Timestamp: time.Now()},
			},
			new: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
			},
			wantErr: false,
			checkFn: func(t *testing.T, report *ComparisonReport) {
				if len(report.Fixed) != 1 {
					t.Errorf("Expected 1 fixed, got %d", len(report.Fixed))
				}
				if len(report.Broken) != 0 {
					t.Errorf("Expected 0 broken, got %d", len(report.Broken))
				}
				if report.ImprovementPercent() <= 0 {
					t.Errorf("Expected positive improvement, got %v", report.ImprovementPercent())
				}
			},
		},
		{
			name: "broken benchmark",
			baseline: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
			},
			new: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: false, ErrorCategory: "compile_error", Timestamp: time.Now()},
			},
			wantErr: false,
			checkFn: func(t *testing.T, report *ComparisonReport) {
				if len(report.Broken) != 1 {
					t.Errorf("Expected 1 broken, got %d", len(report.Broken))
				}
				if len(report.Fixed) != 0 {
					t.Errorf("Expected 0 fixed, got %d", len(report.Fixed))
				}
				if report.ImprovementPercent() >= 0 {
					t.Errorf("Expected negative improvement (regression), got %v", report.ImprovementPercent())
				}
			},
		},
		{
			name: "still passing",
			baseline: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
			},
			new: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
			},
			wantErr: false,
			checkFn: func(t *testing.T, report *ComparisonReport) {
				if len(report.StillPassing) != 1 {
					t.Errorf("Expected 1 still passing, got %d", len(report.StillPassing))
				}
				if report.ImprovementPercent() != 0 {
					t.Errorf("Expected 0 improvement, got %v", report.ImprovementPercent())
				}
			},
		},
		{
			name: "still failing",
			baseline: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: false, ErrorCategory: "compile_error", Timestamp: time.Now()},
			},
			new: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: false, ErrorCategory: "compile_error", Timestamp: time.Now()},
			},
			wantErr: false,
			checkFn: func(t *testing.T, report *ComparisonReport) {
				if len(report.StillFailing) != 1 {
					t.Errorf("Expected 1 still failing, got %d", len(report.StillFailing))
				}
			},
		},
		{
			name: "new benchmark added",
			baseline: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
			},
			new: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
				{ID: "test2", Lang: "ailang", Model: "claude", StdoutOk: false, Timestamp: time.Now()},
			},
			wantErr: false,
			checkFn: func(t *testing.T, report *ComparisonReport) {
				if len(report.NewBenchmarks) != 1 {
					t.Errorf("Expected 1 new benchmark, got %d", len(report.NewBenchmarks))
				}
			},
		},
		{
			name: "benchmark removed",
			baseline: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
				{ID: "test2", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
			},
			new: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
			},
			wantErr: false,
			checkFn: func(t *testing.T, report *ComparisonReport) {
				if len(report.Removed) != 1 {
					t.Errorf("Expected 1 removed benchmark, got %d", len(report.Removed))
				}
			},
		},
		{
			name: "mixed results - multiple models",
			baseline: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: false, Timestamp: time.Now()},
				{ID: "test1", Lang: "ailang", Model: "gpt5", StdoutOk: true, Timestamp: time.Now()},
				{ID: "test2", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
			},
			new: []*BenchmarkResult{
				{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()}, // Fixed!
				{ID: "test1", Lang: "ailang", Model: "gpt5", StdoutOk: false, Timestamp: time.Now()},  // Broken!
				{ID: "test2", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()}, // Still passing
			},
			wantErr: false,
			checkFn: func(t *testing.T, report *ComparisonReport) {
				if len(report.Fixed) != 1 {
					t.Errorf("Expected 1 fixed, got %d", len(report.Fixed))
				}
				if len(report.Broken) != 1 {
					t.Errorf("Expected 1 broken, got %d", len(report.Broken))
				}
				if len(report.StillPassing) != 1 {
					t.Errorf("Expected 1 still passing, got %d", len(report.StillPassing))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report, err := Compare(tt.baseline, tt.new, "baseline", "new")
			if (err != nil) != tt.wantErr {
				t.Errorf("Compare() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if tt.checkFn != nil {
				tt.checkFn(t, report)
			}
		})
	}
}

func TestBuildResultMap(t *testing.T) {
	now := time.Now()
	older := now.Add(-1 * time.Hour)

	results := []*BenchmarkResult{
		{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: older},
		{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: false, Timestamp: now}, // Newer, should win
	}

	m := buildResultMap(results)

	key := "test1|ailang|claude"
	result := m[key]

	if result.Timestamp != now {
		t.Error("Expected newer result to be kept")
	}

	if result.StdoutOk != false {
		t.Error("Expected newer result's status")
	}
}

func TestFindRegressions(t *testing.T) {
	baseline := []*BenchmarkResult{
		{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
		{ID: "test2", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
	}

	new := []*BenchmarkResult{
		{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: false, ErrorCategory: "compile_error", Timestamp: time.Now()}, // Regression!
		{ID: "test2", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()},
	}

	regressions, err := FindRegressions(baseline, new)
	if err != nil {
		t.Fatalf("FindRegressions() error = %v", err)
	}

	if len(regressions) != 1 {
		t.Errorf("Expected 1 regression, got %d", len(regressions))
	}

	if regressions[0].ID != "test1" {
		t.Errorf("Expected test1 to be regressed, got %s", regressions[0].ID)
	}
}

func TestFindImprovements(t *testing.T) {
	baseline := []*BenchmarkResult{
		{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: false, Timestamp: time.Now()},
		{ID: "test2", Lang: "ailang", Model: "claude", StdoutOk: false, Timestamp: time.Now()},
	}

	new := []*BenchmarkResult{
		{ID: "test1", Lang: "ailang", Model: "claude", StdoutOk: true, Timestamp: time.Now()}, // Fixed!
		{ID: "test2", Lang: "ailang", Model: "claude", StdoutOk: false, Timestamp: time.Now()},
	}

	improvements, err := FindImprovements(baseline, new)
	if err != nil {
		t.Fatalf("FindImprovements() error = %v", err)
	}

	if len(improvements) != 1 {
		t.Errorf("Expected 1 improvement, got %d", len(improvements))
	}

	if improvements[0].ID != "test1" {
		t.Errorf("Expected test1 to be fixed, got %s", improvements[0].ID)
	}
}

func TestReportHelpers(t *testing.T) {
	report := &ComparisonReport{
		Fixed:               []*BenchmarkChange{{ID: "test1"}},
		Broken:              []*BenchmarkChange{{ID: "test2"}},
		BaselineSuccessRate: 0.5,
		NewSuccessRate:      0.7,
		SuccessRateDelta:    0.2,
	}

	if !report.HasImprovements() {
		t.Error("Expected HasImprovements to be true")
	}

	if !report.HasRegressions() {
		t.Error("Expected HasRegressions to be true")
	}

	if report.NetChange() != 0 {
		t.Errorf("NetChange = %d, want 0", report.NetChange())
	}

	if report.ImprovementPercent() != 20.0 {
		t.Errorf("ImprovementPercent = %v, want 20.0", report.ImprovementPercent())
	}

	summary := report.Summary()
	if summary == "" {
		t.Error("Expected non-empty summary")
	}
}
