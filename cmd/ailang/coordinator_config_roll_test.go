package main

import "testing"

func TestNoRollRequested_StripsTheFlagAndLeavesTheRest(t *testing.T) {
	// The flag is removed before the existing positional/flag parsing runs, so a
	// leftover "--no-roll" must never reach it as a file path or a value.
	tests := []struct {
		name     string
		in       []string
		wantArgs []string
		wantSkip bool
	}{
		{"absent", []string{"cfg.yaml", "--if-generation", "42"}, []string{"cfg.yaml", "--if-generation", "42"}, false},
		{"trailing", []string{"cfg.yaml", "--if-generation", "42", "--no-roll"}, []string{"cfg.yaml", "--if-generation", "42"}, true},
		{"leading", []string{"--no-roll", "cfg.yaml"}, []string{"cfg.yaml"}, true},
		// Between a flag and its value is the case that would corrupt parsing if
		// the flag were merely detected rather than removed.
		{"between a flag and its value", []string{"cfg.yaml", "--if-generation", "--no-roll", "42"}, []string{"cfg.yaml", "--if-generation", "42"}, true},
		{"case-insensitive", []string{"cfg.yaml", "--NO-ROLL"}, []string{"cfg.yaml"}, true},
		{"empty", nil, []string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, skip := noRollRequested(tt.in)
			if skip != tt.wantSkip {
				t.Errorf("skip = %v, want %v", skip, tt.wantSkip)
			}
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", got, tt.wantArgs)
			}
			for i := range got {
				if got[i] != tt.wantArgs[i] {
					t.Errorf("args = %v, want %v", got, tt.wantArgs)
					break
				}
			}
		})
	}
}

func TestCoordinatorService_DefaultsToTheServiceThatMountsTheConfig(t *testing.T) {
	// Verified 2026-09-10 against every Cloud Run service in the region:
	// ailang-coordinator is the only one mounting the config bucket. Rolling the
	// wrong service would report success and change nothing, which is the same
	// silent no-op the roll exists to eliminate.
	for _, v := range []string{"AILANG_CLOUD_PROJECT", "AILANG_CLOUD_REGION", "AILANG_COORDINATOR_SERVICE"} {
		t.Setenv(v, "")
	}
	project, region, service := coordinatorService()
	if project != "ailang-multivac" || region != "europe-west1" || service != "ailang-coordinator" {
		t.Errorf("defaults = %s/%s/%s, want ailang-multivac/europe-west1/ailang-coordinator", project, region, service)
	}
}

func TestCoordinatorService_IsOverridable(t *testing.T) {
	t.Setenv("AILANG_CLOUD_PROJECT", "other-project")
	t.Setenv("AILANG_CLOUD_REGION", "us-central1")
	t.Setenv("AILANG_COORDINATOR_SERVICE", "other-coordinator")
	project, region, service := coordinatorService()
	if project != "other-project" || region != "us-central1" || service != "other-coordinator" {
		t.Errorf("overrides ignored: got %s/%s/%s", project, region, service)
	}
}

func TestReportConfigRoll_NoRollReportsNotLive(t *testing.T) {
	// --no-roll is a deliberate staging choice, but it still leaves the plane on
	// the old config — so it must report false and drive a non-zero exit. A
	// staged config that reads as deployed is the whole failure being fixed.
	if reportConfigRoll(t.Context(), true) {
		t.Error("--no-roll must report NOT live")
	}
}
