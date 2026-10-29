package pkg

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestSmokeOutput_FirstDiagnosticAndByteLimit(t *testing.T) {
	full := "INFO: starting\nWarning: harmless\nError: FIRST_DIAGNOSTIC\n  at fixture.ail:3\n" + strings.Repeat("TRACE: trailing context\n", 1200)
	got := truncateOutput(full, 8192)
	if len(got) > 8192 || !strings.HasPrefix(got, "Error: FIRST_DIAGNOSTIC") {
		t.Fatalf("lost first diagnostic or exceeded byte limit: len=%d first=%q", len(got), got[:min(len(got), 100)])
	}
}

func TestSmokeOutput_UTF8AndSuccessfulShortOutput(t *testing.T) {
	full := "Error: " + strings.Repeat("界", 4000) + string([]byte{0xff})
	got := truncateOutput(full, 8192)
	if len(got) > 8192 || !utf8.ValidString(got) || !strings.HasPrefix(got, "Error:") {
		t.Fatalf("invalid bounded error: bytes=%d valid=%v", len(got), utf8.ValidString(got))
	}
	if short := "smoke ok\n"; truncateOutput(short, 8192) != short {
		t.Fatal("short successful output changed")
	}
}

func TestSmokeFailureNotes_SelectsCauseAndPreservesByteBudget(t *testing.T) {
	cases := []struct {
		name   string
		result *SmokeResult
		err    error
		want   string
	}{
		{"setup", nil, fmt.Errorf("failed to stage ailang.lock: %s", strings.Repeat("界", 1800)), "failed to stage ailang.lock"},
		{"ansi_trace_prefix", &SmokeResult{ExitCode: 23, Output: "\x1b[32mINFO: starting\x1b[0m\r\nWarning: harmless\r\n  at early.ail:1\r\ngoroutine 1\r\n\t/file.go:23\r\n\x1b[31mError: ACTUAL_CAUSE\x1b[0m\r\n  at later.ail:2"}, nil, "Error: ACTUAL_CAUSE"},
		{"osc", &SmokeResult{ExitCode: 23, Output: "\x1b]8;;https://example.invalid\x07\x1b]8;;\x07Error: LINK_CAUSE"}, nil, "Error: LINK_CAUSE"},
		{"parser", &SmokeResult{ExitCode: 1, Output: "INFO: compiling\nPAR_UNEXPECTED_TOKEN at fixture.ail:3"}, nil, "PAR_UNEXPECTED_TOKEN"},
		{"timeout", &SmokeResult{ExitCode: -1, TimedOut: true, Output: "Error: before deadline\n" + strings.Repeat("TRACE: tail\n", 500)}, nil, "timed out after 1s"},
		{"quiet_exit", &SmokeResult{ExitCode: 23}, nil, "smoke exited with code 23"},
		{"only_noise", &SmokeResult{ExitCode: 23, Output: "INFO: starting\nWarning: harmless\n  at file.ail:2"}, nil, "smoke exited with code 23"},
		{"plain_failure", &SmokeResult{ExitCode: 2, Output: "\noperation refused by fixture\n"}, nil, "operation refused"},
		{"long_unicode", &SmokeResult{ExitCode: 1, Output: "Error: " + strings.Repeat("界", 1800) + string([]byte{0xff})}, nil, "Error:"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := SmokeFailureNotes(tt.result, tt.err, time.Second)
			if !strings.HasPrefix(got, tt.want) || len(got) > 2048 || !utf8.ValidString(got) || strings.Contains(got, "\x1b") {
				t.Fatalf("bad diagnostic bytes=%d: %q", len(got), got)
			}
		})
	}
	if got := SmokeFailureNotes(&SmokeResult{Passed: true, Output: "ok"}, nil, time.Second); got != "" {
		t.Fatalf("successful smoke got failure notes: %s", got)
	}
}

func TestSmokeFailureNotes_ChildStartError(t *testing.T) {
	_, author := smokeStageFixture(t)
	missing := filepath.Join(t.TempDir(), "absent-smoke-binary")
	result, err := RunSmokeInTempDir(author, missing, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || !strings.Contains(result.Output, "absent-smoke-binary") {
		t.Fatalf("child-start cause lost: %+v", result)
	}
	notes := SmokeFailureNotes(result, nil, time.Second)
	if !strings.Contains(notes, "absent-smoke-binary") || len(notes) > 2048 {
		t.Fatalf("child-start Notes: %s", notes)
	}
}
