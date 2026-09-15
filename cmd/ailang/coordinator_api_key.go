package main

import (
	"os"
	"path/filepath"
	"regexp"
)

// discoverCoordinatorAPIKey returns the bearer token the local daemon
// requires, from COORDINATOR_API_KEY or, like discoverCoordinatorHTTPPort,
// from the rendered launchd plist — the daemon's env and the shell's are not
// the same environment, and the daemon fails closed without a key
// (M-V1-SIMPLIFY-S3 M5). Returns "" when neither declares one.
func discoverCoordinatorAPIKey() string {
	if key := os.Getenv("COORDINATOR_API_KEY"); key != "" {
		return key
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, "Library", "LaunchAgents", "dev.ailang.coordinator.plist"))
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`<key>COORDINATOR_API_KEY</key>\s*<string>([^<]+)</string>`)
	if m := re.FindSubmatch(data); len(m) == 2 {
		return string(m[1])
	}
	return ""
}
