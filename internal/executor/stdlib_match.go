package executor

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ailangstd "github.com/sunholo-data/ailang/std"
)

// embeddedStdlibPatterns mirrors the //go:embed line in std/embed.go: the
// files a binary carries as its own stdlib. A module on disk under one of these
// patterns that the binary does not carry is a module it cannot have been
// built against.
var embeddedStdlibPatterns = []string{"*.ail", "ai/*.ail", "stream/*.ail"}

// stdlibMatchesBinary reports whether dir holds exactly the stdlib built into
// this binary: every embedded module present with identical bytes, and no
// extra module under the embedded patterns. On a mismatch it names the first
// module that differs.
//
// Why the executor needs this: eval-suite runs from the repo root, so
// childStdlibPath used to export <cwd>/std — the SHARED checkout's live std/ —
// to every agent. Other sessions edit and release from that checkout, so an
// agent's runtime compiled against whatever std/ held that minute. When std/
// used a builtin the installed binary did not have, every run that imported it
// died before step 0 with nothing on stderr, in bursts that ended only when the
// checkout or the binary changed (motoko A/B, 2026-09-27: 7-11 runs per pass,
// both trees; the stderr log read "stdlib version mismatch: expected v0.45.0,
// found v0.46.0").
func stdlibMatchesBinary(dir string) (bool, string) {
	embedded := map[string][]byte{}
	err := fs.WalkDir(ailangstd.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".ail") {
			return err
		}
		b, rerr := ailangstd.FS.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		embedded[p] = b
		return nil
	})
	if err != nil {
		return false, fmt.Sprintf("cannot read the binary's embedded stdlib: %v", err)
	}
	if len(embedded) == 0 {
		return false, "the binary embeds no stdlib"
	}
	for p, want := range embedded {
		got, rerr := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if rerr != nil {
			return false, fmt.Sprintf("%s is missing", p)
		}
		if !bytes.Equal(got, want) {
			return false, fmt.Sprintf("%s differs", p)
		}
	}
	for _, pat := range embeddedStdlibPatterns {
		matches, _ := filepath.Glob(filepath.Join(dir, pat))
		for _, m := range matches {
			rel, rerr := filepath.Rel(dir, m)
			if rerr != nil {
				continue
			}
			if _, ok := embedded[filepath.ToSlash(rel)]; !ok {
				return false, fmt.Sprintf("%s is not in the binary", filepath.ToSlash(rel))
			}
		}
	}
	return true, ""
}

var stdlibMismatchWarned sync.Map

// warnStdlibMismatchOnce prints, once per directory, why a stdlib root was not
// exported. A silent fallback would hide that agents are running the binary's
// stdlib rather than the checkout's.
func warnStdlibMismatchOnce(dir, why string) {
	if _, loaded := stdlibMismatchWarned.LoadOrStore(dir, true); loaded {
		return
	}
	fmt.Fprintf(os.Stderr, "[executor] NOT exporting AILANG_STDLIB_PATH=%s: it does not match the stdlib built into this binary (%s). Agents use the binary's own stdlib. Reinstall ailang from this tree if you meant to test these std/ edits.\n", dir, why)
}
