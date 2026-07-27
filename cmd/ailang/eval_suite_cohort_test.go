package main

import (
	"strings"
	"testing"
)

// TestCohortSourceRef_DefaultIsByteIdentical is the load-bearing no-regression
// test: with no --baseline, the composed source_ref must be EXACTLY the string
// eval-suite has always written ("<taskID>/<mode><condRef>"). A change here
// silently re-buckets every historical chain.
func TestCohortSourceRef_DefaultIsByteIdentical(t *testing.T) {
	tests := []struct {
		name     string
		taskID   string
		evalMode string
		condRef  string
		want     string
	}{
		{"agent no condition", "eval-123", "agent", "", "eval-123/agent"},
		{"standard no condition", "eval-123", "standard", "", "eval-123/standard"},
		{"agent with condition", "eval-123", "agent", "/full", "eval-123/agent/full"},
		{"coordinator task id", "task_abc", "agent", "/baseline", "task_abc/agent/baseline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cohortSourceRef("", tt.taskID, tt.evalMode, tt.condRef)
			if got != tt.want {
				t.Errorf("cohortSourceRef(\"\", %q, %q, %q) = %q, want %q",
					tt.taskID, tt.evalMode, tt.condRef, got, tt.want)
			}
		})
	}
}

func TestCohortSourceRef_WithBaselinePrefixesCohort(t *testing.T) {
	tests := []struct {
		baseline string
		taskID   string
		evalMode string
		condRef  string
		want     string
	}{
		{"v1.0", "eval-123", "agent", "/full", "v1.0/eval-123/agent/full"},
		{"v1.0", "eval-123", "agent", "", "v1.0/eval-123/agent"},
		{"v1.0-rc1", "eval-9", "standard", "", "v1.0-rc1/eval-9/standard"},
		{"os-rolling.2", "eval-9", "agent", "/contract", "os-rolling.2/eval-9/agent/contract"},
	}
	for _, tt := range tests {
		got := cohortSourceRef(tt.baseline, tt.taskID, tt.evalMode, tt.condRef)
		if got != tt.want {
			t.Errorf("cohortSourceRef(%q, %q, %q, %q) = %q, want %q",
				tt.baseline, tt.taskID, tt.evalMode, tt.condRef, got, tt.want)
		}
	}
}

// TestCohortSourceRef_RoundTripsWithReaderPrefix is THE load-bearing invariant:
// the string the writer banks must be matched by the LIKE prefix the reader
// (`ailang chains stats --cost-per-verified-success --baseline <id>`) derives
// from the SAME baseline id. If these two ever drift, the KPI silently returns
// empty_cohort for a cohort that was actually banked.
func TestCohortSourceRef_RoundTripsWithReaderPrefix(t *testing.T) {
	baselines := []string{"v1.0", "v1.0-rc1", "v1", "os-rolling.2", "V1.0.0", "b0"}
	taskIDs := []string{"eval-1750000000000", "task_abc123"}
	modes := []string{"agent", "standard"}
	condRefs := []string{"", "/full", "/baseline"}

	for _, b := range baselines {
		// The write side must accept every id the round-trip covers.
		if err := validateBaselineID(b); err != nil {
			t.Fatalf("validateBaselineID(%q) rejected a round-trip id: %v", b, err)
		}
		prefix := cohortSourceRefPrefix(b)
		for _, task := range taskIDs {
			for _, mode := range modes {
				for _, cond := range condRefs {
					ref := cohortSourceRef(b, task, mode, cond)
					if !strings.HasPrefix(ref, prefix) {
						t.Errorf("writer/reader DRIFT: cohortSourceRef(%q,…) = %q does not start with cohortSourceRefPrefix(%q) = %q",
							b, ref, b, prefix)
					}
				}
			}
		}
	}
}

// TestCohortSourceRef_PrefixDoesNotBleedAcrossCohorts guards the reason the
// reader appends '/': baseline "v1.0" must NOT match a "v1.05" cohort.
func TestCohortSourceRef_PrefixDoesNotBleedAcrossCohorts(t *testing.T) {
	neighbour := cohortSourceRef("v1.05", "eval-1", "agent", "")
	if strings.HasPrefix(neighbour, cohortSourceRefPrefix("v1.0")) {
		t.Errorf("cohort bleed: %q matches the v1.0 prefix %q", neighbour, cohortSourceRefPrefix("v1.0"))
	}
}

// TestValidateBaselineID pins the charset. This is the BF-2 fix: the baseline id
// flows into an UNESCAPED SQL LIKE pattern (internal/observatory/store_chains_eval.go
// builds `c.source_ref LIKE ?` with no ESCAPE clause), where SQLite treats '_' as a
// single-character wildcard and '%' as any-sequence. So "v1_0" would ALSO match
// "v1x0/…" and silently widen the cohort. Pinning the charset on the write side —
// and reusing the SAME validator on the read side — makes a frozen id always a
// literal prefix.
func TestValidateBaselineID(t *testing.T) {
	valid := []string{"v1.0", "v1.0-rc1", "os-rolling.2", "v1", "V1", "a", "0", "v1.0.0-rc.2"}
	for _, id := range valid {
		if err := validateBaselineID(id); err != nil {
			t.Errorf("validateBaselineID(%q) = %v, want nil", id, err)
		}
	}

	invalid := []struct {
		id     string
		reason string
	}{
		{"", "empty"},
		{"v1_0", "SQL LIKE single-char wildcard"},
		{"50%", "SQL LIKE any-sequence wildcard"},
		{"/v1.0", "leading separator"},
		{"v1.0/x", "embedded separator"},
		{"v1.0/", "trailing separator"},
		{".v1", "leading dot"},
		{"-v1", "leading dash"},
		{"v1 0", "space"},
		{"v1\t0", "tab"},
		{"v1.0\n", "newline"},
		{"v1'0", "quote"},
		{"v1;0", "semicolon"},
		{"v1*0", "glob"},
		{"café", "non-ASCII"},
	}
	for _, tc := range invalid {
		if err := validateBaselineID(tc.id); err == nil {
			t.Errorf("validateBaselineID(%q) = nil, want error (%s)", tc.id, tc.reason)
		}
	}
}

// TestValidateBaselineID_ErrorNamesTheCharset — a rejected id must tell the
// operator what IS allowed, because this error fires before an expensive
// metered run.
func TestValidateBaselineID_ErrorNamesTheCharset(t *testing.T) {
	err := validateBaselineID("v1_0")
	if err == nil {
		t.Fatal("expected an error for v1_0")
	}
	msg := err.Error()
	for _, want := range []string{"baseline", baselineIDPattern} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q does not mention %q", msg, want)
		}
	}
}

// TestValidateCohortFreeze covers the pre-flight gate. The --verify coupling is
// a HARD error, not a warning: the failure mode it prevents is an expensive
// metered run that provably cannot yield the KPI it was frozen for.
func TestValidateCohortFreeze(t *testing.T) {
	tests := []struct {
		name        string
		baselineSet bool
		baselineID  string
		verify      bool
		wantErr     bool
		errContains string
	}{
		{name: "flag absent is always ok (default path)", baselineSet: false, baselineID: "", verify: false},
		{name: "flag absent ignores a stale value", baselineSet: false, baselineID: "v1_0", verify: false},
		{name: "valid id with verify", baselineSet: true, baselineID: "v1.0", verify: true},
		{
			name:        "explicit empty id is an error, not a silent no-freeze",
			baselineSet: true, baselineID: "", verify: true,
			wantErr: true, errContains: "must not be empty",
		},
		{
			name:        "wildcard id rejected before any spend",
			baselineSet: true, baselineID: "v1_0", verify: true,
			wantErr: true, errContains: baselineIDPattern,
		},
		{
			name:        "baseline without verify names --verify",
			baselineSet: true, baselineID: "v1.0", verify: false,
			wantErr: true, errContains: "--verify",
		},
		{
			name:        "id validation precedes the verify coupling",
			baselineSet: true, baselineID: "50%", verify: false,
			wantErr: true, errContains: baselineIDPattern,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCohortFreeze(tt.baselineSet, tt.baselineID, tt.verify)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateCohortFreeze(%v, %q, %v) = nil, want error",
						tt.baselineSet, tt.baselineID, tt.verify)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("validateCohortFreeze(%v, %q, %v) = %v, want nil",
					tt.baselineSet, tt.baselineID, tt.verify, err)
			}
		})
	}
}

// TestValidateCohortFreeze_VerifyMessageExplainsWhy — the operator is being
// stopped before a spend, so the message must say WHY, not just "requires".
func TestValidateCohortFreeze_VerifyMessageExplainsWhy(t *testing.T) {
	err := validateCohortFreeze(true, "v1.0", false)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"--verify", "zero_denominator", "verify_verified"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q does not mention %q", err.Error(), want)
		}
	}
}

// TestEvalSuiteBaselineRequiresVerify_CLI is the end-to-end assertion that the
// gate is actually WIRED into `ailang eval-suite` (not merely implemented) and
// that it fires with a non-zero exit code BEFORE any benchmark work — no
// network, no models, no spend. Guards against the M4a-1 gate being defined but
// never called, which is exactly the class of bug BF-1 was.
func TestEvalSuiteBaselineRequiresVerify_CLI(t *testing.T) {
	bin := buildAilang(t)

	t.Run("baseline without verify exits non-zero naming --verify", func(t *testing.T) {
		_, stderr, code := runAilangBin(t, bin, "eval-suite", "--baseline", "v1.0", "--dry-run")
		if code == 0 {
			t.Fatalf("expected non-zero exit, got 0\nstderr: %s", stderr)
		}
		if !strings.Contains(stderr, "--verify") {
			t.Errorf("stderr does not name --verify:\n%s", stderr)
		}
	})

	t.Run("invalid baseline id exits non-zero naming the charset", func(t *testing.T) {
		_, stderr, code := runAilangBin(t, bin, "eval-suite", "--baseline", "v1_0", "--verify", "--dry-run")
		if code == 0 {
			t.Fatalf("expected non-zero exit, got 0\nstderr: %s", stderr)
		}
		if !strings.Contains(stderr, baselineIDPattern) {
			t.Errorf("stderr does not name the charset %q:\n%s", baselineIDPattern, stderr)
		}
	})

	t.Run("explicit empty baseline exits non-zero", func(t *testing.T) {
		_, stderr, code := runAilangBin(t, bin, "eval-suite", "--baseline", "", "--verify", "--dry-run")
		if code == 0 {
			t.Fatalf("expected non-zero exit for --baseline \"\", got 0\nstderr: %s", stderr)
		}
	})

	t.Run("no baseline flag is unaffected", func(t *testing.T) {
		stdout, stderr, code := runAilangBin(t, bin, "eval-suite", "--dry-run", "--benchmarks", "fizzbuzz")
		if code != 0 {
			t.Fatalf("default --dry-run path regressed: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, "Dry-run:") {
			t.Errorf("expected dry-run output, got:\n%s", stdout)
		}
	})
}

// TestEvalModeName / TestConditionRef pin the two source_ref sub-expressions
// extracted in M4a-0, so a refactor cannot quietly change the banked ref.
func TestEvalModeName(t *testing.T) {
	if got := evalModeName(true); got != "agent" {
		t.Errorf("evalModeName(true) = %q, want agent", got)
	}
	if got := evalModeName(false); got != "standard" {
		t.Errorf("evalModeName(false) = %q, want standard", got)
	}
}

func TestConditionRef(t *testing.T) {
	tests := []struct {
		conds []string
		want  string
	}{
		{nil, ""},
		{[]string{""}, ""},                 // legacy no-condition run
		{[]string{"full"}, "/full"},        // single condition
		{[]string{"baseline", "full"}, ""}, // multi-condition fans out under one chain
		{[]string{"", "full"}, ""},         // mixed → no single condition
	}
	for _, tt := range tests {
		if got := conditionRef(tt.conds); got != tt.want {
			t.Errorf("conditionRef(%v) = %q, want %q", tt.conds, got, tt.want)
		}
	}
}
