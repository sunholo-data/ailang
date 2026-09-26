package mission

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// DefaultPaths must honour AILANG_STATE_DIR — the mission loop's pid files,
// quota ledger and ollama observations used to be pinned beneath $HOME even
// when the driver had been pointed elsewhere (M-V1-SIMPLIFY-S2 M4).
func TestDefaultPathsHonoursStateDir(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	state := filepath.Join(t.TempDir(), "elsewhere")
	t.Setenv("AILANG_STATE_DIR", state)

	p := DefaultPaths()
	if p.StateDir != state {
		t.Fatalf("StateDir = %q, want %q", p.StateDir, state)
	}
	if got := LedgerPath(p); got != filepath.Join(state, "quota-ledger.json") {
		t.Fatalf("LedgerPath = %q, want it under AILANG_STATE_DIR", got)
	}

	// A fixture that sets only Home gets the default tree beneath it.
	fixture := Paths{Home: home}
	if got := fixture.State("x"); got != filepath.Join(home, ".ailang", "state", "x") {
		t.Fatalf("fixture State = %q", got)
	}
}
