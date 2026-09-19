package feedbackgate

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

// The shadow program travels INSIDE the binary: the coordinator images copy
// /usr/local/bin/ailang, not the source tree (docker/Dockerfile.agent-base:68),
// so a repo-relative path would leave the stage off in exactly the place it
// matters (the prod coordinator, where the classifier runs). The manifest and
// lock ride along so the subprocess resolves the pinned sunholo/decisions
// version from the registry, not "latest".
//
//go:embed shadow/feedback_shadow.ail shadow/ailang.toml shadow/ailang.lock
var shadowProgramFS embed.FS

var shadowProgramFiles = []string{"feedback_shadow.ail", "ailang.toml", "ailang.lock"}

// MaterializeShadowProgram writes the embedded program into dir (created if
// needed) and returns dir. Idempotent: existing files are overwritten with the
// embedded bytes, so a binary upgrade never runs a stale program.
func MaterializeShadowProgram(dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("shadow program dir: %w", err)
	}
	for _, name := range shadowProgramFiles {
		data, err := shadowProgramFS.ReadFile("shadow/" + name)
		if err != nil {
			return "", fmt.Errorf("embedded %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return "", fmt.Errorf("write %s: %w", name, err)
		}
	}
	return dir, nil
}

// DefaultShadowProgramDir is where the daemon materialises the program when it
// is not running from a source checkout.
func DefaultShadowProgramDir() string {
	return filepath.Join(os.TempDir(), "ailang-feedbackgate-shadow")
}
