package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

const accessNow = int64(1_800_000_000)

func newAccessStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewAccessApprovalSQLiteStore(filepath.Join(t.TempDir(), "access.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func accessContext(id, owner, worker, decision, execution string, expiresAt, claimUntil int64) string {
	raw, _ := json.Marshal(map[string]any{
		"id": id, "owner": owner, "worker": worker, "decision": decision,
		"expires_at": expiresAt, "claim_until": claimUntil,
		"execution": map[string]string{"state": execution},
	})
	return string(raw)
}

func putAccess(t *testing.T, s *SQLiteStore, id, owner, worker, contextJSON string, created int64) bool {
	t.Helper()
	ok, err := s.CreateAccessApprovalIfAbsent(context.Background(), &ApprovalRequestRecord{
		ID: id, TaskID: id, Type: DurableAccessApprovalType, Description: "test",
		ContextJSON: contextJSON, Status: "access_pending", CreatedAt: time.Unix(created, 0).UTC(),
	}, owner, worker, accessNow)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestAccessApprovalStoreRequiresPath(t *testing.T) {
	if _, err := NewAccessApprovalSQLiteStore(""); err == nil {
		t.Fatal("empty path accepted")
	}
}

func TestAccessApprovalCreateIsFirstWriteWinsAndTyped(t *testing.T) {
	s := newAccessStore(t)
	ctx := context.Background()
	pending := accessContext("", "mark", "machine:daneel", "pending", "unclaimed", accessNow+60, 0)
	if !putAccess(t, s, "a1", "mark", "machine:daneel", pending, accessNow) {
		t.Fatal("first create refused")
	}
	if putAccess(t, s, "a1", "mark", "machine:daneel", pending, accessNow) {
		t.Fatal("replay created a second row")
	}
	_, err := s.CreateAccessApprovalIfAbsent(ctx, &ApprovalRequestRecord{ID: "m1", Type: "merge"}, "mark", "machine:daneel", accessNow)
	if err == nil {
		t.Fatal("non-access type accepted")
	}
}

func TestAccessApprovalPendingCapCountsOnlyLivePending(t *testing.T) {
	s := newAccessStore(t)
	expired := accessContext("", "mark", "machine:daneel", "pending", "unclaimed", accessNow, 0)
	if !putAccess(t, s, "old", "mark", "machine:daneel", expired, accessNow-10) {
		t.Fatal("expired seed refused")
	}
	live := accessContext("", "mark", "machine:daneel", "pending", "unclaimed", accessNow+60, 0)
	for i := 0; i < 100; i++ {
		if !putAccess(t, s, fmt.Sprintf("p%03d", i), "mark", "machine:daneel", live, accessNow) {
			t.Fatalf("request %d refused below the cap", i)
		}
	}
	if putAccess(t, s, "p100", "mark", "machine:daneel", live, accessNow) {
		t.Fatal("101st live pending request accepted")
	}
	if !putAccess(t, s, "other", "mark", "machine:other", live, accessNow) {
		t.Fatal("cap leaked across workers")
	}
}

func TestAccessApprovalCompareAndSwapRequiresExactPrior(t *testing.T) {
	s := newAccessStore(t)
	ctx := context.Background()
	before := accessContext("", "mark", "machine:daneel", "pending", "unclaimed", accessNow+60, 0)
	putAccess(t, s, "a1", "mark", "machine:daneel", before, accessNow)
	after := accessContext("", "mark", "machine:daneel", "approved", "unclaimed", accessNow+60, accessNow+600)
	if ok, err := s.CompareAndSwapAccessApproval(ctx, "a1", after, before, "pending"); err != nil || ok {
		t.Fatalf("stale prior swapped: ok=%v err=%v", ok, err)
	}
	if ok, err := s.CompareAndSwapAccessApproval(ctx, "a1", before, after, "approved"); err != nil || !ok {
		t.Fatalf("exact prior refused: ok=%v err=%v", ok, err)
	}
	if ok, _ := s.CompareAndSwapAccessApproval(ctx, "a1", before, after, "approved"); ok {
		t.Fatal("second swap from the same prior succeeded")
	}
	var status string
	if err := s.db.QueryRowContext(ctx, `SELECT status FROM approval_requests WHERE id='a1'`).Scan(&status); err != nil || status != "access_approved" {
		t.Fatalf("status=%q err=%v", status, err)
	}
}

func TestAccessApprovalListScopesExpiresOrdersAndPages(t *testing.T) {
	s := newAccessStore(t)
	ctx := context.Background()
	rows := []struct {
		id, owner, worker, ctx string
		created                int64
	}{
		{"p-old", "mark", "machine:daneel", accessContext("p-old", "mark", "machine:daneel", "pending", "unclaimed", accessNow+60, 0), 100},
		{"p-new", "mark", "machine:daneel", accessContext("p-new", "mark", "machine:daneel", "pending", "unclaimed", accessNow+60, 0), 300},
		{"p-lapsed", "mark", "machine:daneel", accessContext("p-lapsed", "mark", "machine:daneel", "pending", "unclaimed", accessNow, 0), 200},
		{"a-unclaimed-lapsed", "mark", "machine:daneel", accessContext("a-unclaimed-lapsed", "mark", "machine:daneel", "approved", "unclaimed", accessNow+60, accessNow), 400},
		{"a-claimed", "mark", "machine:daneel", accessContext("a-claimed", "mark", "machine:daneel", "approved", "claimed", accessNow+60, accessNow), 500},
		{"declined", "mark", "machine:daneel", accessContext("declined", "mark", "machine:daneel", "declined", "unclaimed", accessNow+60, 0), 600},
		{"other-worker", "mark", "machine:other", accessContext("other-worker", "mark", "machine:other", "pending", "unclaimed", accessNow+60, 0), 700},
		{"other-owner", "eve", "machine:daneel", accessContext("other-owner", "eve", "machine:daneel", "pending", "unclaimed", accessNow+60, 0), 800},
	}
	for _, r := range rows {
		putAccess(t, s, r.id, r.owner, r.worker, r.ctx, r.created)
	}
	ids := func(raws []string) []string {
		out := []string{}
		for _, raw := range raws {
			for _, r := range rows {
				if r.ctx == raw {
					out = append(out, r.id)
					break
				}
			}
		}
		return out
	}
	cases := []struct {
		name, worker, status string
		limit, offset, total int
		want                 []string
	}{
		{"pending oldest first, lapsed excluded", "machine:daneel", "pending", 10, 0, 2, []string{"p-old", "p-new"}},
		{"history newest first, expiry included", "machine:daneel", "history", 10, 0, 4, []string{"declined", "a-claimed", "a-unclaimed-lapsed", "p-lapsed"}},
		{"all is newest first", "machine:daneel", "all", 10, 0, 6, []string{"declined", "a-claimed", "a-unclaimed-lapsed", "p-new", "p-lapsed", "p-old"}},
		{"page bounds the rows, not the total", "machine:daneel", "history", 2, 1, 4, []string{"a-claimed", "a-unclaimed-lapsed"}},
		{"empty worker spans workers, never owners", "", "pending", 10, 0, 3, []string{"p-old", "p-new", "other-worker"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raws, total, err := s.ListAccessApprovalContexts(ctx, "mark", c.worker, c.status, accessNow, c.limit, c.offset)
			if err != nil {
				t.Fatal(err)
			}
			got := ids(raws)
			if total != c.total || fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Fatalf("total=%d got=%v, want total=%d %v", total, got, c.total, c.want)
			}
		})
	}
}
