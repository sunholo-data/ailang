package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestApplyValidityBackstop is the producer side of the measurement contract.
//
// The framework shipped with MarkInvalid called from exactly two places, so the
// dominant local-rig failure — a motoko crash banked as api_error — was recorded
// as a VALID model failure. That fed A/B arms, ELO difficulty, published pass
// rates, and --skip-existing (which only retries INVALID rows, and so never
// retried anything).
func TestApplyValidityBackstop(t *testing.T) {
	tests := []struct {
		name        string
		metrics     RunMetrics
		wantValid   bool
		wantReason  string
		wantDetail  string
		detailExact bool
	}{
		{
			// The case that motivated all of this.
			name:       "api_error is not a measurement",
			metrics:    RunMetrics{ErrorCategory: ErrorCategoryAPI, Stderr: "motoko terminated with finish_reason=tool_calls and no run_summary"},
			wantValid:  false,
			wantReason: ReasonHarnessError,
			wantDetail: "motoko terminated with finish_reason=tool_calls and no run_summary",
		},
		{
			// A model that genuinely got the wrong answer IS a measurement, and
			// must keep counting — otherwise every hard benchmark silently
			// vanishes from the denominator.
			name:      "logic_error is a real model failure",
			metrics:   RunMetrics{ErrorCategory: ErrorCategoryLogic},
			wantValid: true,
		},
		{
			name:      "compile_error is a real model failure",
			metrics:   RunMetrics{ErrorCategory: ErrorCategoryCompile},
			wantValid: true,
		},
		{
			// Typed failure categories exist precisely so they are NOT swept
			// into the catch-all; they say something about the model.
			name:      "step_exhausted is a real model failure",
			metrics:   RunMetrics{ErrorCategory: ErrorCategoryStepExhausted},
			wantValid: true,
		},
		{
			name:      "refused is a real model behaviour",
			metrics:   RunMetrics{ErrorCategory: ErrorCategoryRefused},
			wantValid: true,
		},
		{
			name:      "a passing run is untouched",
			metrics:   RunMetrics{ErrorCategory: ErrorCategoryNone, StdoutOk: true},
			wantValid: true,
		},
		{
			// An earlier stage that actively ruled wins — config_assert and
			// fmt_treatment must not be overwritten by the backstop.
			name: "an existing invalid verdict is not overwritten",
			metrics: RunMetrics{
				ErrorCategory: ErrorCategoryAPI,
				Validity:      MarkInvalid(ReasonConfigMismatch),
			},
			wantValid:  false,
			wantReason: ReasonConfigMismatch,
		},
		{
			// A stage may also have explicitly certified the row as valid.
			name: "an explicit VALID verdict is not overwritten",
			metrics: RunMetrics{
				ErrorCategory: ErrorCategoryAPI,
				Validity:      MarkValid(),
			},
			wantValid: true,
		},
		{
			name:       "empty stderr still yields a usable detail",
			metrics:    RunMetrics{ErrorCategory: ErrorCategoryAPI},
			wantValid:  false,
			wantReason: ReasonHarnessError,
			wantDetail: "no stderr captured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.metrics
			m.applyValidityBackstop()

			if got := m.IsValid(); got != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", got, tt.wantValid)
			}
			if tt.wantReason != "" && m.InvalidReason() != tt.wantReason {
				t.Errorf("InvalidReason() = %q, want %q", m.InvalidReason(), tt.wantReason)
			}
			if tt.wantDetail != "" {
				if m.Validity == nil || m.Validity.Detail != tt.wantDetail {
					got := "<nil validity>"
					if m.Validity != nil {
						got = m.Validity.Detail
					}
					t.Errorf("Detail = %q, want %q", got, tt.wantDetail)
				}
			}
		})
	}
}

// TestValidityDetailFromStderr: the detail is what an operator reads instead of
// re-deriving the failure from the transcript, so it must pick the line that
// actually says what broke — the LAST one, not the telemetry noise at the top.
func TestValidityDetailFromStderr(t *testing.T) {
	stderr := `point http://localhost:1957 unreachable — telemetry disabled
Trace: standard
Error: execution failed: path "/tmp/x/benchmark/solution.ail" escapes sandbox "/tmp/x/ws"
`
	got := validityDetailFromStderr(stderr)
	if !strings.Contains(got, "escapes sandbox") {
		t.Errorf("detail should be the final, informative line; got %q", got)
	}
	if strings.Contains(got, "telemetry disabled") {
		t.Errorf("detail picked the leading noise line: %q", got)
	}
}

func TestValidityDetailFromStderr_Truncates(t *testing.T) {
	got := validityDetailFromStderr(strings.Repeat("x", 5000))
	if len(got) > 320 {
		t.Errorf("detail not truncated: %d chars", len(got))
	}
}

// TestLogAppliesBackstop proves the backstop is wired into the ACTUAL write
// path. Testing applyValidityBackstop alone would have passed happily while the
// framework stayed disconnected — which is exactly the failure being fixed.
func TestLogAppliesBackstop(t *testing.T) {
	dir := t.TempDir()
	logger := NewMetricsLogger(dir)

	m := &RunMetrics{
		ID:            "bytecode_vm_trace",
		Lang:          "ailang",
		Model:         "motoko-local-qwen3-6-fmt",
		EvalMode:      EvalModeAgent,
		ErrorCategory: ErrorCategoryAPI,
		Stderr:        "Error: motoko terminated without emitting run_summary",
		Timestamp:     time.Now(),
	}
	if err := logger.Log(m); err != nil {
		t.Fatalf("Log: %v", err)
	}

	matches, _ := filepath.Glob(filepath.Join(dir, "agent", "*.json"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 banked row, got %d", len(matches))
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var banked RunMetrics
	if err := json.Unmarshal(data, &banked); err != nil {
		t.Fatal(err)
	}
	if banked.IsValid() {
		t.Error("an api_error row was banked as VALID — the backstop is not wired into Log")
	}
	if banked.InvalidReason() != ReasonHarnessError {
		t.Errorf("reason = %q, want %q", banked.InvalidReason(), ReasonHarnessError)
	}
	if banked.Validity.Detail == "" {
		t.Error("banked row carries no detail; an operator cannot see which failure this was")
	}
}
