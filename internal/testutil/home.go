package testutil

import "testing"

// SetHomeDir points os.UserHomeDir() at dir on every platform in the CI
// matrix, for the duration of the test.
//
// os.UserHomeDir reads a different variable per GOOS: USERPROFILE on Windows,
// $home on plan9, and HOME elsewhere. Setting only HOME silently leaves Windows
// tests pointed at the runner's real profile, so set all three even when dir is
// empty. The empty value intentionally makes os.UserHomeDir fail everywhere.
func SetHomeDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("home", dir)
}
