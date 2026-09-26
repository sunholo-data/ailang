package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/fileguard"
)

// sandboxCheckCommand implements `ailang sandbox-check <path>`.
//
// Prints whether the given path would be ALLOW or REJECT under the current
// AILANG_FS_SANDBOX configuration, and what the resolved path would be.
// Exit 0 = ALLOW, exit 1 = REJECT (or no sandbox configured).
//
// It asks the SAME root handle the runtime uses (internal/fileguard over
// os.Root), so its verdict is the runtime's — including `..` traversal and
// symlinks that leave the root, which the previous lexical copy of the
// resolver reported as ALLOW (M-EXECUTOR-POLICY-HARDENING M5).
//
// Designed for shell-level debugging of sandbox path issues — pipe into
// scripts or run manually to diagnose silent-false from exists/isDir/isFile.
func sandboxCheckCommand(args []string) {
	sandbox := config.FSSandbox()

	if sandbox == "" {
		fmt.Println("sandbox:  (not configured — AILANG_FS_SANDBOX is unset)")
		fmt.Println("result:   N/A — no sandbox active, all paths are unrestricted")
		os.Exit(0)
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ailang sandbox-check <path>\n")
		fmt.Fprintf(os.Stderr, "       AILANG_FS_SANDBOX must be set to test path resolution.\n")
		os.Exit(1)
	}

	path := args[0]
	absKind := "relative"
	if filepath.IsAbs(path) {
		absKind = "absolute"
	}

	verdict, err := sandboxVerdict(sandbox, path)
	fmt.Printf("sandbox:  %s\n", sandbox)
	fmt.Printf("path:     %s (%s)\n", path, absKind)
	if err != nil {
		fmt.Printf("result:   REJECT — %s\n", err)
		fmt.Printf("          exists/isDir/isFile → false\n")
		fmt.Printf("          readFile/writeFile/etc → error\n")
		os.Exit(1)
	}
	fmt.Printf("result:   ALLOW → %s\n", verdict)
	os.Exit(0)
}

// sandboxVerdict opens the root the runtime would open and asks it. The
// returned string is the in-root name the handle resolves. A missing file is
// reported as ALLOW-if-created (the name is inside the root); only an escape
// is a REJECT.
func sandboxVerdict(sandbox, path string) (string, error) {
	root, err := fileguard.Open(sandbox)
	if err != nil {
		return "", err
	}
	defer root.Close()
	rel, err := root.Rel(path)
	if err != nil {
		return "", err
	}
	// Stat through the handle: a symlink or `..` that leaves the root is an
	// escape; a plain not-found is not (the name is still inside).
	if _, err := root.Stat(rel); err != nil {
		if fileguard.IsEscape(err) {
			return "", err
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		// Not found: the PARENT must still be inside the root for a create to land there.
		if dir := filepath.Dir(rel); dir != "." {
			if _, perr := root.Stat(dir); perr != nil && fileguard.IsEscape(perr) {
				return "", perr
			}
		}
		return filepath.Join(sandbox, rel) + " (does not exist yet)", nil
	}
	return filepath.Join(sandbox, rel), nil
}
