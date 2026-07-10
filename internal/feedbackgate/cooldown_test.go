package feedbackgate

import (
	"context"
	"testing"
	"time"
)

// fakeCooldownStore is an in-memory CooldownStore. It records the timestamps
// of every Increment for each key and reports sliding-window counts, so tests
// exercise the real window math without a live Firestore. Mirrors the
// triage_router_test.go fakeTriageStore pattern.
type fakeCooldownStore struct {
	seen map[string][]time.Time
}

func newFakeCooldownStore() *fakeCooldownStore {
	return &fakeCooldownStore{seen: map[string][]time.Time{}}
}

func (f *fakeCooldownStore) Increment(_ context.Context, key string, now time.Time) (int, int, error) {
	f.seen[key] = append(f.seen[key], now)
	var hour, day int
	for _, t := range f.seen[key] {
		if now.Sub(t) < time.Hour {
			hour++
		}
		if now.Sub(t) < 24*time.Hour {
			day++
		}
	}
	return hour, day, nil
}

func TestCooldownKeyStableAndContactAware(t *testing.T) {
	a := Input{From: "mcp-public", Category: "auto:bug", Body: "hello\ncontact: alice@example.com\n"}
	b := Input{From: "mcp-public", Category: "auto:bug", Body: "hello\ncontact: alice@example.com\n"}
	if cooldownKey(a) != cooldownKey(b) {
		t.Fatal("identical inputs must yield identical keys")
	}
	// Different contact line -> different key (contact folded into body).
	c := Input{From: "mcp-public", Category: "auto:bug", Body: "hello\ncontact: bob@example.com\n"}
	if cooldownKey(a) == cooldownKey(c) {
		t.Fatal("different contact must yield different key")
	}
	// Different category -> different key (auto: stripped, so bug vs feature).
	d := Input{From: "mcp-public", Category: "auto:feature", Body: "hello\ncontact: alice@example.com\n"}
	if cooldownKey(a) == cooldownKey(d) {
		t.Fatal("different category must yield different key")
	}
}

// TestCooldownHourBoundary: 3 dispatches/hour allowed, 4th files.
func TestCooldownHourBoundary(t *testing.T) {
	store := newFakeCooldownStore()
	in := Input{From: "mcp-public", Category: "auto:bug", Body: "same body", Inbox: "pkg:a/b"}
	cfg := FeedbackGateConfig{}.normalized()
	cfg.Cooldown = store

	for i := 1; i <= 3; i++ {
		v, err := applyCooldown(context.Background(), in, cfg)
		if err != nil {
			t.Fatalf("attempt %d error: %v", i, err)
		}
		if v.Action != ActionDispatch {
			t.Fatalf("attempt %d: Action = %q, want dispatch", i, v.Action)
		}
	}
	v, _ := applyCooldown(context.Background(), in, cfg)
	if v.Action != ActionFile || v.Reason != ReasonContactCooldown {
		t.Fatalf("4th attempt: got %q/%q, want file/contact_cooldown", v.Action, v.Reason)
	}
}

// TestCooldownDayBoundary: with a large hourly limit, the 10/day cap files the
// 11th. Uses a store seeded with timestamps spread across the day so the hour
// window never trips first.
func TestCooldownDayBoundary(t *testing.T) {
	store := newFakeCooldownStore()
	cfg := FeedbackGateConfig{MaxDispatchPerHour: 1000}.normalized()
	cfg.Cooldown = store
	in := Input{From: "mcp-public", Category: "auto:bug", Body: "daybody", Inbox: "pkg:a/b"}
	key := cooldownKey(in)

	now := time.Now()
	// Seed 10 attempts spread every ~2h over the trailing day (all within 24h,
	// none within the same hour as `now`).
	for i := 0; i < 10; i++ {
		store.seen[key] = append(store.seen[key], now.Add(-time.Duration(2*(i+1))*time.Hour))
	}
	// The 11th (now) makes dayCount=11 > 10 -> file.
	hour, day, _ := store.Increment(context.Background(), key, now)
	if day <= cfg.MaxDispatchPerDay {
		t.Fatalf("expected dayCount > %d, got %d (hour=%d)", cfg.MaxDispatchPerDay, day, hour)
	}
	v, _ := applyCooldown(context.Background(), in, cfg)
	if v.Action != ActionFile || v.Reason != ReasonContactCooldown {
		t.Fatalf("over-day: got %q/%q, want file/contact_cooldown", v.Action, v.Reason)
	}
}

// TestCooldownWindowExpiryReallows: old attempts outside the hour window don't
// count, so a contact whose burst aged out can dispatch again.
func TestCooldownWindowExpiryReallows(t *testing.T) {
	store := newFakeCooldownStore()
	cfg := FeedbackGateConfig{}.normalized()
	cfg.Cooldown = store
	in := Input{From: "mcp-public", Category: "auto:bug", Body: "expirybody", Inbox: "pkg:a/b"}
	key := cooldownKey(in)

	now := time.Now()
	// 3 attempts that are all > 1h old (aged out of the hour window).
	for i := 0; i < 3; i++ {
		store.seen[key] = append(store.seen[key], now.Add(-90*time.Minute))
	}
	// A fresh attempt: hourCount should be 1 (only the new one), so dispatch.
	v, err := applyCooldown(context.Background(), in, cfg)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if v.Action != ActionDispatch {
		t.Fatalf("after window expiry: Action = %q, want dispatch", v.Action)
	}
}

// TestDecideComposesCooldownAfterRules: a message that fails a deterministic
// rule never touches the cooldown store (cooldown only consulted on dispatch).
func TestDecideComposesCooldownAfterRules(t *testing.T) {
	store := newFakeCooldownStore()
	cfg := FeedbackGateConfig{}.normalized()
	cfg.Cooldown = store

	// Non-auto category -> filed by rules; cooldown must NOT be incremented.
	in := baseInput()
	in.Category = "bug" // no auto: prefix
	v, err := Decide(context.Background(), in, cfg)
	if err != nil {
		t.Fatalf("Decide error: %v", err)
	}
	if v.Action != ActionFile || v.Reason != ReasonNotAuthorized {
		t.Fatalf("got %q/%q, want file/not_authorized", v.Action, v.Reason)
	}
	if len(store.seen) != 0 {
		t.Fatalf("cooldown store was consulted for a rule-filed message: %v", store.seen)
	}

	// A clean message DOES consult cooldown and dispatches.
	v2, _ := Decide(context.Background(), baseInput(), cfg)
	if v2.Action != ActionDispatch {
		t.Fatalf("clean message: Action = %q, want dispatch", v2.Action)
	}
	if len(store.seen) != 1 {
		t.Fatalf("expected exactly 1 cooldown key recorded, got %d", len(store.seen))
	}
}
