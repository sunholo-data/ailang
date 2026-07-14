package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestVerify_UnencodableCalleeSkipsNotErrors is the end-to-end acceptance test for
// M-SMT-CALLEE-SORT-GATE: `ailang verify` on a contracted function whose callee
// returns Option[float] must SKIP it with an UNENCODABLE_TYPE reason — NOT crash Z3
// with a hard error. The example doubles as the regression fixture.
func TestVerify_UnencodableCalleeSkipsNotErrors(t *testing.T) {
	bin := buildAilang(t)
	stdout, stderr, _ := runAilangBin(t, bin, "verify", "--json",
		"examples/runnable/contracts/unencodable_callee_skip.ail")

	// Must NOT be a Z3 hard error.
	combined := stdout + stderr
	for _, bad := range []string{"unknown sort", "unknown constant", "Invalid function definition"} {
		if strings.Contains(combined, bad) {
			t.Fatalf("verify leaked a Z3 error %q; output:\n%s", bad, combined)
		}
	}

	var payload struct {
		Results []struct {
			Function   string `json:"function"`
			Status     string `json:"status"`
			Rejections []struct {
				Code string `json:"code"`
			} `json:"rejections"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("verify --json did not emit valid JSON: %v\nstdout:\n%s", err, stdout)
	}

	var found bool
	for _, r := range payload.Results {
		if r.Function != "gradeNumeric" {
			continue
		}
		found = true
		if r.Status != "skipped" {
			t.Fatalf("gradeNumeric status = %q, want \"skipped\"", r.Status)
		}
		hasUnencodable := false
		for _, rej := range r.Rejections {
			if rej.Code == "UNENCODABLE_TYPE" {
				hasUnencodable = true
			}
		}
		if !hasUnencodable {
			t.Fatalf("gradeNumeric skip missing UNENCODABLE_TYPE rejection: %+v", r.Rejections)
		}
	}
	if !found {
		t.Fatalf("gradeNumeric not present in verify results:\n%s", stdout)
	}
}

// TestVerify_CrossFunctionIntChainStillVerifies guards against the gate falsely
// rejecting callees with encodable (primitive / monomorphic-enum) signatures.
func TestVerify_CrossFunctionIntChainStillVerifies(t *testing.T) {
	bin := buildAilang(t)
	stdout, _, _ := runAilangBin(t, bin, "verify", "--json",
		"examples/runnable/contracts/cross_function.ail")

	var payload struct {
		Results []struct {
			Function string `json:"function"`
			Status   string `json:"status"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("verify --json invalid: %v\n%s", err, stdout)
	}
	if len(payload.Results) == 0 {
		t.Fatalf("no results for cross_function.ail:\n%s", stdout)
	}
	for _, r := range payload.Results {
		if r.Status != "verified" {
			t.Fatalf("cross_function %q status = %q, want \"verified\" (gate false-positive?)", r.Function, r.Status)
		}
	}
}
