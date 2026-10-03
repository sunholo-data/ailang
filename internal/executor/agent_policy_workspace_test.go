package executor

import (
	"testing"

	"github.com/BurntSushi/toml"
)

// A Windows workspace path substituted raw into fs_sandbox = "${WORKSPACE}"
// made an invalid TOML escape ("C:\Users" → \U), so every run policy failed to
// load on Windows (TestMaterializeRunPolicy, windows CI, 2026-10-01). This
// reproduces it on any OS.
func TestSubstituteWorkspace_TOMLSafe(t *testing.T) {
	for _, ws := range []string{
		`C:\Users\RUNNER~1\AppData\Local\Temp\TestX\001`,
		`/tmp/ws with "quotes"`,
		`/plain/unix/path`,
	} {
		src := substituteWorkspace("fs_sandbox = \"${WORKSPACE}\"\n", ws)
		var got struct {
			FSSandbox string `toml:"fs_sandbox"`
		}
		if _, err := toml.Decode(src, &got); err != nil {
			t.Fatalf("workspace %q: %v\n%s", ws, err, src)
		}
		if got.FSSandbox != ws {
			t.Errorf("fs_sandbox = %q, want %q", got.FSSandbox, ws)
		}
	}
}
