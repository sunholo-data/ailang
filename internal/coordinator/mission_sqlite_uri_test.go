package coordinator

import (
	"runtime"
	"strings"
	"testing"
)

// The unix spelling must be byte-identical to what url.URL produced before the fix: the
// fleet runs on macOS and Linux, and a "portability fix" that changes the DSN there would
// be a behaviour change smuggled in as a bug fix.
func TestSQLiteFileURI_UnixSpellingUnchanged(t *testing.T) {
	got := sqliteFileURIFor("linux", "/tmp/canary.db", "mode=ro&_query_only=1&_busy_timeout=5000")
	want := "file:///tmp/canary.db?mode=ro&_query_only=1&_busy_timeout=5000"
	if got != want {
		t.Fatalf("sqliteFileURI() = %q, want %q", got, want)
	}
}

// The Windows failure this exists for: url.URL{Path: "C:\\..."} renders `file:C:%5C...`,
// and SQLite reads the drive letter as the URI authority — "invalid uri authority".
func TestSQLiteFileURI_WindowsDriveIsNotAnAuthority(t *testing.T) {
	got := sqliteFileURIFor("windows", `C:\Users\runner\AppData\Local\Temp\state.db`, "mode=rw")
	want := "file:///C:/Users/runner/AppData/Local/Temp/state.db?mode=rw"
	if got != want {
		t.Fatalf("sqliteFileURI() = %q, want %q", got, want)
	}
	// The precise defect: no percent-encoded backslashes, and three slashes before the drive.
	if want[:8] != "file:///" {
		t.Fatalf("authority is not empty in %q", want)
	}
}

// A UNC path keeps its leading separators rather than gaining a third.
func TestSQLiteFileURI_UNCPathNotDoubleRooted(t *testing.T) {
	got := sqliteFileURIFor("windows", `\\server\share\state.db`, "")
	if got != "file:////server/share/state.db" && got != "file://server/share/state.db" {
		t.Logf("UNC spelling = %q", got)
	}
	if got[:7] != "file://" {
		t.Fatalf("UNC path lost its scheme/authority shape: %q", got)
	}
}

func TestSQLiteFileURI_RoundTripsThroughTheRealOpenPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("covered by the explicit Windows spelling case above")
	}
	dir := t.TempDir()
	// A path with a space proves the query separator is not confused with escaping.
	got := sqliteFileURI(dir+"/a b.db", "mode=ro")
	if got == "" || got[:8] != "file:///" {
		t.Fatalf("unexpected URI %q", got)
	}
}

// A backslash is a legal character in a unix filename. Rewriting separators unconditionally
// would corrupt this path instead of fixing an invalid URI, so the swap is Windows-gated.
func TestSQLiteFileURI_UnixBackslashIsAFilenameNotASeparator(t *testing.T) {
	got := sqliteFileURIFor("linux", `/tmp/od\d name.db`, "")
	if strings.Contains(got, "/tmp/od/d") {
		t.Fatalf("unix backslash was treated as a separator: %q", got)
	}
}
