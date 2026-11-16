package main

import (
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/mission"
	"github.com/sunholo-data/ailang/internal/mission/quorum"
)

func stubCodexSeat(t *testing.T, state string) *[]string {
	t.Helper()
	origQ, origR := codexSeatQuota, codexSeatReview
	var calls []string
	codexSeatQuota = func(time.Time) mission.CodexQuotaObservation {
		return mission.CodexQuotaObservation{State: state, Reason: "test " + state}
	}
	codexSeatReview = func(model, _, _ string) *quorum.ReviewerOutcome {
		calls = append(calls, model)
		return &quorum.ReviewerOutcome{Model: quorum.CodexReviewerPrefix + model, Present: true}
	}
	t.Cleanup(func() { codexSeatQuota, codexSeatReview = origQ, origR })
	return &calls
}

// D-FLEET-20: the OpenAI seat runs on the codex subscription lane, never the metered API.
func TestQuorumSeatRouter_CodexSeatUsesSubscription(t *testing.T) {
	calls := stubCodexSeat(t, "ok")
	o := quorumSeatRouter(quorum.CodexReviewerID, "d.md", "body", 0.30)
	if !o.Present || len(*calls) != 1 || (*calls)[0] != quorum.CodexReviewerModel {
		t.Fatalf("codex seat not routed to the subscription reviewer: outcome %+v, calls %v", o, *calls)
	}
}

// The Codex bucket gates the seat as it gates the roles' codex rungs: over, expired and
// unknown all refuse (codex fails closed), and the review is never started.
func TestQuorumSeatRouter_CodexSeatRefusedWhenBucketBlocked(t *testing.T) {
	for _, state := range []string{"over", "expired", "unknown"} {
		calls := stubCodexSeat(t, state)
		o := quorumSeatRouter(quorum.CodexReviewerID, "d.md", "body", 0.30)
		if o.Present || o.AbsentReason != quorum.ReasonQuota || len(*calls) != 0 {
			t.Fatalf("state %s: outcome %+v, calls %v; want a quota absence and no review", state, o, *calls)
		}
	}
}

// The default pool's OpenAI seat must be a codex-lane entry: the API-billed row
// (gpt6-1-sol) is what left the seat absent.
func TestDefaultQuorumPool_OpenAISeatOnCodexLane(t *testing.T) {
	var openai []string
	for _, m := range splitCSV(defaultQuorumPool) {
		if quorum.VendorOf(m) == "openai" {
			openai = append(openai, m)
		}
	}
	if len(openai) != 1 || openai[0] != quorum.CodexReviewerID {
		t.Fatalf("default pool OpenAI seats = %v, want exactly [%s]", openai, quorum.CodexReviewerID)
	}
	for _, m := range openai {
		if !strings.HasPrefix(m, quorum.CodexReviewerPrefix) {
			t.Fatalf("OpenAI seat %q would call the metered API", m)
		}
	}
}
