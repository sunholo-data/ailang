package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sandboxCheckCommand implements `ailang sandbox-check <path>`.
//
// Prints whether the given path would be ALLOW or REJECT under the current
// AILANG_FS_SANDBOX configuration, and what the resolved path would be.
// Exit 0 = ALLOW, exit 1 = REJECT (or no sandbox configured).
//
// Designed for shell-level debugging of sandbox path issues — pipe into
// scripts or run manually to diagnose silent-false from exists/isDir/isFile.
func sandboxCheckCommand(args []string) {
	sandbox := os.Getenv("AILANG_FS_SANDBOX")

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

	resolved, err := resolveSandboxPathCheck(sandbox, path)
	fmt.Printf("sandbox:  %s\n", sandbox)
	fmt.Printf("path:     %s (%s)\n", path, absKind)
	if err != nil {
		fmt.Printf("result:   REJECT — %s\n", err)
		fmt.Printf("          exists/isDir/isFile → false\n")
		fmt.Printf("          readFile/writeFile/etc → error\n")
		os.Exit(1)
	}
	fmt.Printf("result:   ALLOW → %s\n", resolved)
	os.Exit(0)
}

// resolveSandboxPathCheck is a copy of the effects.resolveSandboxPath logic
// without the effects package dependency (cmd/ cannot import internal/effects/).
func resolveSandboxPathCheck(sandbox, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return filepath.Join(sandbox, path), nil
	}
	clean := filepath.Clean(path)
	sandboxClean := filepath.Clean(sandbox)
	if clean != sandboxClean && !strings.HasPrefix(clean, sandboxClean+string(filepath.Separator)) {
		return "", fmt.Errorf("escapes sandbox %q", sandbox)
	}
	return clean, nil
}
