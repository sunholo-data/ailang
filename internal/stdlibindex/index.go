// Package stdlibindex maps a bare symbol name to the std modules that export it, so an
// "undefined variable" error can suggest the missing import (M-AGENT-ERGONOMICS). The dominant
// agent slip on multi-file AILANG tasks is using a stdlib function (length/join/repeat/...)
// without importing it; a bare "undefined variable: length" costs a fix cycle, "add
// `import std/list (length)`" costs zero.
//
// The index is built once, lazily, by scanning the resolved stdlib directory for
// `export func <name>` declarations. It is a leaf package (CLI + LSP can both use it).
package stdlibindex

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	once sync.Once
	idx  map[string][]string // symbol -> sorted []module (e.g. "std/list")
)

var (
	moduleRe = regexp.MustCompile(`^module\s+(std/\w+)`)
	exportRe = regexp.MustCompile(`^export\s+(?:pure\s+)?func\s+(\w+)`)
)

// stdlibDir resolves the stdlib directory the same way the runtime does: AILANG_STDLIB_PATH
// (first existing entry of a path-list) else ./std.
func stdlibDir() string {
	if p := strings.TrimSpace(os.Getenv("AILANG_STDLIB_PATH")); p != "" {
		for _, e := range strings.Split(p, string(os.PathListSeparator)) {
			if e == "" {
				continue
			}
			if st, err := os.Stat(e); err == nil && st.IsDir() {
				return e
			}
		}
	}
	return "std"
}

func build() {
	idx = map[string][]string{}
	entries, err := os.ReadDir(stdlibDir())
	if err != nil {
		return // no stdlib dir resolvable; Modules() just returns empty (no suggestion)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ail") {
			continue
		}
		f, err := os.Open(filepath.Join(stdlibDir(), e.Name()))
		if err != nil {
			continue
		}
		var module string
		sc := bufio.NewScanner(f)
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
		_ = f.Close()
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
