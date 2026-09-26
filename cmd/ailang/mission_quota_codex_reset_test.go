package main

import (
	"strings"
	"testing"
	"time"
)

// Spending a Codex reset credit is an attended decision (Mark, 2026-09-24). These pin the
// two refusals that stand between the loop and the credit; neither may reach the provider.
func TestCodexResetRefusedInsideMissionIteration(t *testing.T) {
	for _, env := range [][2]string{{"MISSION_CONTROL_ACTIVE", "1"}, {"MISSION_ROLE", "executor"}} {
		t.Run(env[0], func(t *testing.T) {
			t.Setenv("MISSION_CONTROL_ACTIVE", "")
			t.Setenv("MISSION_ROLE", "")
			t.Setenv(env[0], env[1])
			err := missionQuotaCodexReset(t.TempDir(), "", true, time.Now())
			if err == nil || !strings.Contains(err.Error(), "attended decision") {
				t.Fatalf("err = %v, want the attended-only refusal", err)
			}
		})
	}
}

func TestCodexResetRequiresYes(t *testing.T) {
	t.Setenv("MISSION_CONTROL_ACTIVE", "")
	t.Setenv("MISSION_ROLE", "")
	err := missionQuotaCodexReset(t.TempDir(), "", false, time.Now())
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("err = %v, want a --yes confirmation refusal", err)
	}
}
