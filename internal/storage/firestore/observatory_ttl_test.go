package firestore

import (
	"testing"
	"time"

	obs "github.com/sunholo-data/ailang/internal/observatory"
)

// Every collection the Terraform TTL policies cover must carry expire_at as a
// time.Time, or the policy is a no-op and the collection grows forever — the
// state obs_chains, obs_chain_stages, obs_metrics and obs_sessions were in
// until 2026-09-21 (dev: 830 chains and 2,291 metrics older than 30 days).
func TestExpireAtWrittenForEveryTTLCollection(t *testing.T) {
	created := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	started := created.Add(time.Minute)
	ttl := 90 * 24 * time.Hour

	cases := map[string]map[string]interface{}{
		"obs_chains":       chainToMap(&obs.ExecutionChain{ID: "c1", CreatedAt: created}, ttl),
		"obs_chain_stages": stageToMap(&obs.ChainStage{ID: "s1", ChainID: "c1", StartedAt: &started}, ttl),
		"obs_metrics":      obsMetricToMap(&obs.Metric{Name: "m", CreatedAt: created}, 7*24*time.Hour),
		"obs_spans":        spanToMap(&obs.Span{ID: "sp", CreatedAt: created}, 7*24*time.Hour),
	}
	want := map[string]time.Time{
		"obs_chains":       created.Add(ttl),
		"obs_chain_stages": started.Add(ttl),
		"obs_metrics":      created.Add(7 * 24 * time.Hour),
		"obs_spans":        created.Add(7 * 24 * time.Hour),
	}
	for coll, doc := range cases {
		got, ok := doc["expire_at"].(time.Time)
		if !ok {
			t.Errorf("%s: expire_at is %T, want time.Time", coll, doc["expire_at"])
			continue
		}
		if !got.Equal(want[coll]) {
			t.Errorf("%s: expire_at=%v want %v", coll, got, want[coll])
		}
	}
}

func TestExpireAt_UndatedDocumentCountsFromNow(t *testing.T) {
	before := time.Now()
	got := expireAt(time.Time{}, time.Hour)
	if got.Before(before.Add(time.Hour)) || got.After(time.Now().Add(time.Hour+time.Second)) {
		t.Fatalf("undated document expires at %v, want ~now+1h", got)
	}
	// A stage that has not started yet still gets a deadline, and a span
	// with no created_at must not expire in year 0001 (zero time + ttl).
	for name, m := range map[string]map[string]interface{}{
		"stage": stageToMap(&obs.ChainStage{ID: "s"}, time.Hour),
		"span":  spanToMap(&obs.Span{ID: "sp"}, time.Hour),
	} {
		e, ok := m["expire_at"].(time.Time)
		if !ok || e.Before(before.Add(time.Hour)) {
			t.Fatalf("undated %s has expire_at %v (%T)", name, m["expire_at"], m["expire_at"])
		}
	}
}
