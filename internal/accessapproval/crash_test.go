package accessapproval

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The child exits from inside the operation, after the runtime persisted start
// but before it can persist a receipt. No production failpoint is exposed.
func TestProcessDeathAfterStartIsUnknown(t *testing.T) {
	if dir := os.Getenv("AILANG_APPROVAL_TEST_CRASH_DIR"); dir != "" {
		s, err := Open(dir, "daneel-local")
		if err != nil {
			t.Fatal(err)
		}
		r := approved(t, s)
		in := binding(r)
		r = record(t, s, "claim", in)
		in.ClaimID = r.Execution.ClaimID
		s.runDummy = func(*Request) map[string]any { os.Exit(0); return nil }
		_, _ = s.Handle(context.Background(), "execute", in)
		t.Fatal("crash hook did not exit")
	}
	dir := filepath.Join(t.TempDir(), "runtime")
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessDeathAfterStartIsUnknown$")
	cmd.Env = append(os.Environ(), "AILANG_APPROVAL_TEST_CRASH_DIR="+dir, "AILANG_STORAGE_COORDINATOR=local")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("crash child: %v %s", err, out)
	}
	s, err := Open(dir, "daneel-local")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	out, err := s.Handle(context.Background(), "list", Input{Owner: "mark"})
	if err != nil {
		t.Fatal(err)
	}
	r := out.(*List).Requests[0]
	if r.Execution.State != "started" {
		t.Fatalf("start not durable: %+v", r)
	}
	s.now = func() int64 { return r.Execution.ClaimUntil + 1 }
	calls := 0
	s.runDummy = func(*Request) map[string]any { calls++; return nil }
	in := binding(r)
	in.ClaimID = r.Execution.ClaimID
	got := record(t, s, "execute", in)
	if got.Execution.State != "unknown" || calls != 0 {
		raw, _ := json.Marshal(got)
		t.Fatalf("crashed operation retried: %s calls=%d", raw, calls)
	}
}
