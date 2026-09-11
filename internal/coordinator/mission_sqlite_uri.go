package coordinator

// SQLite `file:` URIs are not `url.URL{Path: <os path>}`.
//
// Measured on the Windows CI runner, 2026-09-08: four mission tests failed with
//
//	invalid uri authority: C:%5CUsers%5CRUNNER~1%5CAppData%5CLocal%5CTemp%5C...
//
// `url.URL{Scheme: "file", Path: "C:\\Users\\..."}.String()` renders as
// `file:C:%5CUsers%5C...` — no authority slashes, backslashes percent-encoded — and SQLite
// parses the leading `C:` as the URI authority. The canonical spelling is `file:///C:/Users/...`.
//
// Unix paths already begin with `/`, so this is a no-op there: `/tmp/x` renders as
// `file:///tmp/x` before and after. That is deliberate — the fix must not change the string
// on the platform the fleet actually runs, only stop producing an invalid one elsewhere.

import (
	"net/url"
	"runtime"
	"strings"
)

// sqliteFileURI builds a SQLite `file:` URI for an absolute OS path.
//
// The caller is responsible for having checked filepath.IsAbs; this only handles spelling.
func sqliteFileURI(path, rawQuery string) string {
	return sqliteFileURIFor(runtime.GOOS, path, rawQuery)
}

// sqliteFileURIFor takes the OS explicitly so the Windows spelling is verifiable from any
// host. filepath.ToSlash cannot serve here: it switches on the RUNTIME separator, so it is
// a no-op on macOS and the Windows behaviour would be untestable on the machine that
// actually runs this fleet — the exact shape of an assertion that passes and proves nothing.
//
// The separator swap is gated on Windows rather than applied unconditionally because a
// backslash is a legal character in a unix filename, and rewriting it there would corrupt
// a valid path instead of fixing an invalid URI.
func sqliteFileURIFor(goos, path, rawQuery string) string {
	p := path
	if goos == "windows" {
		p = strings.ReplaceAll(p, `\`, "/")
	}
	// A Windows path is absolute but does not start with a separator ("C:/x"), and the URI
	// grammar needs one so the drive letter lands in the path and not the authority.
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	uri := url.URL{Scheme: "file", Path: p, RawQuery: rawQuery}
	return uri.String()
}
