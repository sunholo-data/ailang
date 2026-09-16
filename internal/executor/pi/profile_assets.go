package pi

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// The execution pair an `ailang_only` profile depends on, embedded so the
// profile CARRIES its extensions: `ailang_run` (ailang-exec.ts) and
// `ailang_check` (ailang-lsp-lite.ts). Source of truth is .pi/extensions/;
// `make pi-assets` syncs, `make verify-pi-assets` gates drift.
//
// Why carried rather than discovered (M-AGENT-AILANG-ONLY-EXECUTION M5,
// measured 2026-09-16): the rig's global ~/.pi/agent/extensions deliberately
// lacks the suite (it would collide with the repo's own copies inside ailang
// worktrees), and an eval workspace is an untrusted temp dir — so a pi run
// there had read/edit/write and NO way to execute, and the model went round
// in circles for 12 minutes. Passing `--no-extensions -e a -e b` makes the
// lane identical on the rig and in the images, where the global copy exists
// and would otherwise register the same tools twice.
//
//go:embed profile_assets/*.ts
var profileAssets embed.FS

// ProfileExtensionFiles are the extensions a tool list needs, by canonical
// tool name. A list that names none of these gets no -e flags.
var profileExtensionFiles = map[string]string{
	"AilangRun":   "ailang-exec.ts",
	"AilangCheck": "ailang-lsp-lite.ts",
}

var (
	materializeOnce sync.Once
	materializeDir  string
	materializeErr  error
)

// materializeProfileAssets writes the embedded pair to a per-process temp dir
// (NOT ~/.pi/agent/extensions, which pi discovers on its own) and returns it.
func materializeProfileAssets() (string, error) {
	materializeOnce.Do(func() {
		dir, err := os.MkdirTemp("", "ailang-pi-profile-")
		if err != nil {
			materializeErr = err
			return
		}
		for _, name := range profileExtensionFiles {
			b, err := profileAssets.ReadFile("profile_assets/" + name)
			if err != nil {
				materializeErr = fmt.Errorf("embedded %s: %w", name, err)
				return
			}
			if err := os.WriteFile(filepath.Join(dir, name), b, 0o444); err != nil {
				materializeErr = err
				return
			}
		}
		materializeDir = dir
	})
	return materializeDir, materializeErr
}

// extensionArgs returns `--no-extensions -e <file>…` for the extensions the
// canonical tool list needs, or nothing when it needs none. Discovery is
// disabled alongside so the SAME extension cannot also load from a global or
// project dir and register its tools twice (pi refuses to start on that).
func extensionArgs(allowed []string) ([]string, error) {
	var files []string
	seen := map[string]bool{}
	for _, t := range allowed {
		if f, ok := profileExtensionFiles[t]; ok && !seen[f] {
			seen[f] = true
			files = append(files, f)
		}
	}
	if len(files) == 0 {
		return nil, nil
	}
	dir, err := materializeProfileAssets()
	if err != nil {
		return nil, fmt.Errorf("pi: cannot materialize profile extensions: %w", err)
	}
	args := []string{"--no-extensions"}
	for _, f := range files {
		args = append(args, "-e", filepath.Join(dir, f))
	}
	return args, nil
}
