package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/mission"
)

// missionQuotaCodexReset spends one Codex reset credit.
//
// Operator ruling (Mark, attended 2026-09-24): surface reset credits everywhere quota runs
// dry, but "leave that as a choice for attended sessions". So this is refused whenever the
// process is inside a mission iteration — the driver exports MISSION_CONTROL_ACTIVE=1 for
// the whole slot and every role it spawns inherits it, and pinned roles carry MISSION_ROLE.
// That is a guard against the loop doing it by accident, not a security boundary: an agent
// that unsets both variables gets through, exactly like an operator would.
func missionQuotaCodexReset(codexHome, creditID string, yes bool, now time.Time) error {
	if config.MissionControlActive() || config.MissionRole() != "" {
		return fmt.Errorf("--codex-reset is an attended decision and is refused inside a mission iteration (MISSION_CONTROL_ACTIVE/MISSION_ROLE set); the notice names the credit, a human spends it")
	}
	if !yes {
		return fmt.Errorf("--codex-reset spends a one-shot Codex reset credit; re-run with --yes to confirm")
	}
	// One key per credit per day: an accidental double run the same day cannot spend two.
	key := fmt.Sprintf("ailang-mission-quota-%s-%d", creditID, now.Unix()/86400)
	outcome, err := mission.ConsumeCodexResetCredit(context.Background(), codexHome, creditID, key)
	if err != nil {
		return err
	}
	switch outcome {
	case "reset":
		fmt.Println("codex reset credit spent: the account's eligible rate-limit windows were reset")
	case "nothingToReset":
		fmt.Println("codex reset credit NOT spent: no current window is eligible for a reset")
	case "noCredit":
		fmt.Println("codex reset credit NOT spent: the account has no reset credits available")
	case "alreadyRedeemed":
		fmt.Println("codex reset already done today with this key: nothing further spent")
	default:
		fmt.Printf("codex reset: provider outcome %q\n", outcome)
	}
	return nil
}
