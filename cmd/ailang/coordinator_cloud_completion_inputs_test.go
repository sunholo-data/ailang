package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

func TestTaskInputsFailedCompletionRetainsProvenance(t *testing.T) {
	evidence := gitEvidence{Inputs: []taskInputProvenance{{Repo: "org/data", Ref: "main", Commit: strings.Repeat("a", 40), Dest: "images/", Digests: map[string]string{"poster.png": strings.Repeat("b", 64)}}}}
	for _, result := range []*executor.Result{nil, {Transcript: "execution failed"}} {
		got := buildCloudCompletion("task", "site", "failed", "push failed", "branch", result, evidence, "")
		if got.Status != "failed" || got.ErrorMsg != "push failed" || !strings.Contains(got.Summary, "org/data") || !strings.Contains(got.Summary, "poster.png") {
			t.Fatalf("failed completion discarded inputs: %+v", got)
		}
	}
	got := buildCloudCompletion("task", "site", "completed", "", "branch", nil, gitEvidence{}, "")
	if got.Summary != "" {
		t.Fatal("zero-input completion changed")
	}
}
