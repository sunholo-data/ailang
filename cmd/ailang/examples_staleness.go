package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	examplescorpus "github.com/sunholo-data/ailang/examples"
)

// A downloaded corpus (~/.ailang/examples, from `ailang examples download`)
// is a snapshot of one release. It used to win over the corpus embedded in
// the binary unconditionally, so a user with an old download never saw an
// example added after it: the CLI listed 238 while the binary carried 239.
//
// The manifest's generated_at is not a usable version (it has read
// 2026-01-29T21:31:54Z for months), so staleness is measured by content: a
// download that lacks any example the binary carries is older than the
// binary, and the embedded corpus is used instead. A download that is a
// superset — newer than the binary — still wins. AILANG_EXAMPLES is never
// subject to this: a host set it on purpose.

// examplesLaggingEmbedded returns how many of the embedded corpus's examples
// the corpus at dir lacks. 0 means dir is at least as new as the binary; a
// binary without an embedded corpus never reports lag. An unreadable or
// unparsable manifest lags by the whole embedded corpus.
func examplesLaggingEmbedded(dir string) int {
	embedded, err := readManifestPaths(examplescorpus.Corpus)
	if err != nil || len(embedded) == 0 {
		return 0
	}
	onDisk, err := readManifestPaths(os.DirFS(dir))
	if err != nil {
		return len(embedded)
	}
	missing := 0
	for p := range embedded {
		if !onDisk[p] {
			missing++
		}
	}
	return missing
}

// readManifestPaths returns the manifest's example paths, normalised to the
// path under runnable/ — the manifest spells them both "runnable/x.ail" and
// "x.ail".
func readManifestPaths(fsys fs.FS) (map[string]bool, error) {
	data, err := fs.ReadFile(fsys, "manifest.json")
	if err != nil {
		return nil, err
	}
	var m ExampleManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	paths := make(map[string]bool, len(m.Examples))
	for _, ex := range m.Examples {
		paths[strings.TrimPrefix(filepath.ToSlash(ex.Path), "runnable/")] = true
	}
	return paths, nil
}

var staleDownloadNote sync.Once

// noteStaleDownload prints, once per process, why a downloaded corpus was skipped.
func noteStaleDownload(dir string, missing int) {
	staleDownloadNote.Do(func() {
		fmt.Fprintf(os.Stderr, "note: ignoring %s — it lacks %d example(s) built into this binary (an older `ailang examples download`); re-run the download or remove the directory\n", dir, missing)
	})
}
