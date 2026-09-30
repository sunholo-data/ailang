package firestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// TestTaskDirectiveRoundTrip_Live writes a directive over Cloud Run's 32 KB env
// cap and reads it back through real Firestore — the seam the in-memory fakes
// on both sides cannot see.
//
//	AILANG_FIRESTORE_LIVE_TEST_PROJECT=ailang-multivac-dev go test ./internal/storage/firestore/ -run TaskDirective -count=1
func TestTaskDirectiveRoundTrip_Live(t *testing.T) {
	project := os.Getenv("AILANG_FIRESTORE_LIVE_TEST_PROJECT") //nolint:forbidigo // opt-in live test knob, never read by the product
	if project == "" {
		t.Skip("set AILANG_FIRESTORE_LIVE_TEST_PROJECT to run against a real Firestore project")
	}
	ctx := context.Background()
	client, err := NewClientForProject(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	s := NewTaskDirectiveStore(client)
	id := fmt.Sprintf("task-livetest-%d", time.Now().UnixNano())
	defer func() { _, _ = client.Doc(collTaskDirectives, id).Delete(ctx) }()

	if _, err := s.GetDirective(ctx, id); err == nil {
		t.Fatal("a missing directive must be an error")
	}
	big := strings.Repeat("directive ✓ ", 4*1024) // ~57 KB, multi-byte runes
	if err := s.PutDirective(ctx, id, big); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetDirective(ctx, id)
	if err != nil || got != big {
		t.Fatalf("round trip: %d bytes, err %v; want %d bytes", len(got), err, len(big))
	}
	err = s.PutDirective(ctx, id, strings.Repeat("z", coordinator.MaxDirectiveBytes+1))
	if !errors.Is(err, coordinator.ErrDispatchPermanent) {
		t.Fatalf("oversized put: err = %v, want ErrDispatchPermanent", err)
	}
}
