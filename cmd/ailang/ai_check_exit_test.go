package main

import "testing"

// TestAICheckExitCode_Lanes is the AC3.2 guard.
//
// `ai-check` is documented to AI agents as THE convergence signal, so its process
// exit status must never disagree with the JSON it just printed. Before
// M-Z3-ADT-RECORD-SORT the verifier-errors lane was missing entirely: a file whose
// report said `verify.errors: 1` still exited 0.
//
// Red mutation: delete `|| verify.Errors > 0` from aiCheckExitCode. It compiles,
// and the "verifier errors" and "errors dominate a clean check" cases below fail.
// (Measured end-to-end: with that clause removed, cross_module_functions.ail —
// check.passed=true, verify.errors=3 — exits 0 instead of 1.)
func TestAICheckExitCode_Lanes(t *testing.T) {
	tests := []struct {
		name  string
		check aiCheckSection
		verif aiVerifySection
		want  int
	}{
		{
			name:  "clean check, nothing verified",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true},
			want:  0,
		},
		{
			name:  "verified only",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Verified: 3},
			want:  0,
		},
		{
			// A skip is "not proved", NOT "disproved" — it must stay green, else
			// every unencodable-but-correct program would fail the gate.
			name:  "skipped only",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Verified: 1, Skipped: 2},
			want:  0,
		},
		{
			name:  "counterexample",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Counterexample: 1},
			want:  1,
		},
		{
			// THE regression this sprint closed.
			name:  "verifier errors",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Errors: 1},
			want:  1,
		},
		{
			name:  "errors dominate an otherwise clean run",
			check: aiCheckSection{Passed: true},
			verif: aiVerifySection{Available: true, Verified: 5, Skipped: 1, Errors: 3},
			want:  1,
		},
		{
			name:  "check failure",
			check: aiCheckSection{Passed: false},
			verif: aiVerifySection{Available: false},
			want:  1,
		},
		{
			name:  "check failure outranks a clean verify section",
			check: aiCheckSection{Passed: false},
			verif: aiVerifySection{Available: true, Verified: 2},
			want:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aiCheckExitCode(tt.check, tt.verif); got != tt.want {
				t.Fatalf("aiCheckExitCode() = %d, want %d (check.Passed=%v verify=%+v)",
					got, tt.want, tt.check.Passed, tt.verif)
			}
		})
	}
}
