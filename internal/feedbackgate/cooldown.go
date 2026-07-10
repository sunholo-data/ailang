package feedbackgate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

// CooldownStore records dispatch attempts per contact key and reports the
// counts within the trailing hour and day windows. The Firestore-backed impl
// (constructed only in the M4 cloud wiring) uses a TTL'd collection; unit
// tests use an in-memory fake. Kept narrow (single method) so the fake is
// tiny — the same pattern triage_router_test.go uses for triageStore.
type CooldownStore interface {
	// Increment records a dispatch attempt for key at now and returns the
	// number of attempts within the trailing hour and day (inclusive of the
	// just-recorded one). Implementations use a sliding window, not fixed
	// clock buckets, so a stroke-of-the-hour burst can't reset the count.
	Increment(ctx context.Context, key string, now time.Time) (hourCount, dayCount int, err error)
}

// contactRegexp extracts a best-effort contact: line the publisher folds into
// the body (internal/feedback/publisher.go formatBody). There is no dedicated
// Contact field on the wire, and no IP (consumed at the MCP edge), so this is
// the only per-contact signal available at this layer.
var contactRegexp = regexp.MustCompile(`(?mi)^contact:\s*(.+?)\s*$`)

// cooldownKey derives the per-contact key: From + category + bodyHash, plus a
// best-effort contact: line parsed out of the body. The bodyHash catches
// identical resubmits; the contact line catches a single reporter across
// bodies. IP is intentionally absent (see discrepancy #2 in the sprint plan).
func cooldownKey(in Input) string {
	sum := sha256.Sum256([]byte(in.Body))
	bodyHash := hex.EncodeToString(sum[:8])
	contact := ""
	if m := contactRegexp.FindStringSubmatch(in.Body); m != nil {
		contact = strings.TrimSpace(m[1])
	}
	return strings.Join([]string{in.From, strippedCategory(in.Category), bodyHash, contact}, "|")
}

// applyCooldown records a dispatch attempt and files the message when the
// per-contact hour or day limit is exceeded. Called only when the
// deterministic rules would dispatch (see Decide). cfg is assumed normalized
// and cfg.Cooldown non-nil.
func applyCooldown(ctx context.Context, in Input, cfg FeedbackGateConfig) (Verdict, error) {
	key := cooldownKey(in)
	hourCount, dayCount, err := cfg.Cooldown.Increment(ctx, key, time.Now())
	if err != nil {
		return Verdict{}, err
	}
	if hourCount > cfg.MaxDispatchPerHour || dayCount > cfg.MaxDispatchPerDay {
		return Verdict{Action: ActionFile, Reason: ReasonContactCooldown}, nil
	}
	return Verdict{Action: ActionDispatch, Reason: ReasonPassed, Cost: estimatedDispatchCostUSD}, nil
}
