package cloudrun

import (
	"testing"

	runpb "cloud.google.com/go/run/apiv2/runpb"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The dispatcher must hand the job the names the job reads (config.EnvPRLabels,
// config.EnvMergeStarts) — the seam where a renamed variable would silently
// leave every PR unlabelled.
// MU: rename either variable in prMetaEnv and this fails.
func TestPRMetaEnvUsesTheJobsVariableNames(t *testing.T) {
	env := prMetaEnv(coordinator.DispatchParams{
		PRLabels:    []string{"needs-sprint-approval"},
		MergeStarts: []string{"sprint-executor"},
	})
	got := map[string]string{}
	for _, e := range env {
		got[e.Name] = e.Values.(*runpb.EnvVar_Value).Value
	}
	if got[config.EnvPRLabels] != "needs-sprint-approval" {
		t.Errorf("%s = %q", config.EnvPRLabels, got[config.EnvPRLabels])
	}
	if got[config.EnvMergeStarts] != "sprint-executor" {
		t.Errorf("%s = %q", config.EnvMergeStarts, got[config.EnvMergeStarts])
	}
	if len(prMetaEnv(coordinator.DispatchParams{})) != 0 {
		t.Error("empty params produced env vars")
	}
}
