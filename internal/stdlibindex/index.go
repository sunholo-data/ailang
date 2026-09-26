// Package stdlibindex maps a bare symbol name to the std modules that export it, so an
// "undefined variable" error can suggest the missing import (M-AGENT-ERGONOMICS). The dominant
// agent slip on multi-file AILANG tasks is using a stdlib function (length/join/repeat/...)
// without importing it; a bare "undefined variable: length" costs a fix cycle, "add
// `import std/list (length)`" costs zero.
//
// The index is built once, lazily, by scanning the process stdlib root for
// `export func <name>` declarations. It is a leaf package (CLI + LSP can both use it).
package stdlibindex

import (
	"bufio"
	"bytes"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/stdlibroot"
)

var (
	once sync.Once
	idx  map[string][]string // symbol -> sorted []module (e.g. "std/list")
)

var (
	moduleRe = regexp.MustCompile(`^module\s+(std/\w+)`)
	exportRe = regexp.MustCompile(`^export\s+(?:pure\s+)?func\s+(\w+)`)
)

// build scans the process stdlib root (internal/stdlibroot — the same root the
// loader reads, ending at the copy built into the binary) for export
// declarations. Before M-STDLIB-ROOT-RESOLUTION it read AILANG_STDLIB_PATH else
// the literal "std", so outside a repo the index was silently EMPTY and every
// "undefined variable: length" lost its import hint.
func build() {
	idx = map[string][]string{}
	root, err := stdlibroot.Resolve("")
	if err != nil {
		return // a bad explicit override; the loader reports it loudly
	}
	entries, err := fs.ReadDir(root.FS, ".")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ail") {
			continue
		}
		data, err := fs.ReadFile(root.FS, e.Name())
		if err != nil {
			continue
		}
		indexFile(data)
	}
}

func indexFile(data []byte) {
	var module string
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if module == "" {
			if m := moduleRe.FindStringSubmatch(line); m != nil {
				module = m[1]
			}
			continue
		}
		if m := exportRe.FindStringSubmatch(line); m != nil {
			idx[m[1]] = appendUnique(idx[m[1]], module)
		}
	}
}

func appendUnique(xs []string, s string) []string {
	for _, x := range xs {
		if x == s {
			return xs
		}
	}
	return append(xs, s)
}

// Modules returns the std modules that export name, sorted; empty if none (or stdlib unresolvable).
func Modules(name string) []string {
	once.Do(build)
	mods := append([]string(nil), idx[name]...)
	sort.Strings(mods)
	return mods
}

// AllModules returns every std module path (e.g. "std/clock"), sorted; empty if
// the stdlib is unresolvable. Built from the same lazily-scanned index as
// Modules/SymbolsOf. Used by import-hint "did you mean" to catch a mistyped
// stdlib MODULE name and to print the available-module list (M-DX-AI-DISCOVERY M3).
func AllModules() []string {
	once.Do(build)
	seen := map[string]bool{}
	for _, mods := range idx {
		for _, m := range mods {
			seen[m] = true
		}
	}
	out := make([]string, 0, len(seen))
	for m := range seen {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

// SymbolsOf returns the symbols exported by the given std module (e.g. "std/io"),
// sorted; empty if none. The inverse of Modules — used by import-hint "did you
// mean" to suggest a close-named export (e.g. flushStdout -> flush).
func SymbolsOf(module string) []string {
	once.Do(build)
	var out []string
	for sym, mods := range idx {
		for _, m := range mods {
			if m == module {
				out = append(out, sym)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}
