package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/mission"
	"github.com/sunholo-data/ailang/internal/mission/quorum"
)

// The OpenAI quorum seat runs on the ChatGPT subscription through `codex exec`
// (D-FLEET-20 A, Mark, attended 2026-11-16), the lane the mission roles use, at $0
// metered. The models.yml row gpt6-1-sol bills the OpenAI API, whose org has no
// credits, so that seat was ABSENT on every quorum that seated it.

// codexSeatQuota reads the Codex bucket the seat would spend. A variable so tests
// never start a codex process.
var codexSeatQuota = func(now time.Time) mission.CodexQuotaObservation {
	home := config.CodexHome()
	if home == "" {
		h, _ := os.UserHomeDir()
		home = filepath.Join(h, ".codex")
	}
	return mission.ObserveCodexQuota(home, now)
}

// codexSeatReview runs the review. A variable so tests never start a codex process.
var codexSeatReview = quorum.RunCodexSubscriptionReviewer

// quorumSeatRouter sends a codex-lane seat ("codex:<model>") to the subscription and
// every other seat to quorumSeatRunner.
//
// The Codex bucket gates the seat exactly as it gates the roles' codex rungs
// (_mc_probe_codex in tools/launchd/lib/lane-probe.sh): Blocked() refuses on over,
// expired AND unknown. Codex fails closed on an unreadable quota by the driver's
// policy; the Anthropic "unreadable is not exhausted" rule (D-FLEET-28) is scoped
// to Anthropic there and is not extended here.
func quorumSeatRouter(model, docPath, docBody string, maxCostUSD float64) *quorum.ReviewerOutcome {
	if !strings.HasPrefix(model, quorum.CodexReviewerPrefix) {
		return quorumSeatRunner(model, docPath, docBody, maxCostUSD)
	}
	if q := codexSeatQuota(time.Now()); q.Blocked() {
		return &quorum.ReviewerOutcome{
			Model:        model,
			AbsentReason: quorum.ReasonQuota,
			Err:          "Codex bucket blocked by the mission ration: " + q.Reason,
		}
	}
	return codexSeatReview(strings.TrimPrefix(model, quorum.CodexReviewerPrefix), docPath, docBody)
}
