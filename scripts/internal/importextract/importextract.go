// Package importextract provides the single, parser-backed implementation for
// discovering the std/* modules an AILANG example imports. Both the one-time
// manifest `modules` backfill and the validate_manifest drift assertion call
// ExtractModules so the CI authority can never disagree with the language:
// aliases, selective imports, comments, and duplicates are handled by the same
// grammar the compiler uses (internal/parser -> ast.File.Imports), never by
// line/regex scanning.
package importextract

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// ResolvePath maps a manifest entry path to an on-disk file under examplesDir.
// The canonical manifest carries a mix of "runnable/foo.ail" and bare "foo.ail"
// entries (legacy drift); the latter live under examplesDir/runnable/. Both are
// tried, in that order. Returns ("", false) if neither exists.
func ResolvePath(examplesDir, entryPath string) (string, bool) {
	direct := filepath.Join(examplesDir, entryPath)
	if _, err := os.Stat(direct); err == nil {
		return direct, true
	}
	underRunnable := filepath.Join(examplesDir, "runnable", entryPath)
	if _, err := os.Stat(underRunnable); err == nil {
		return underRunnable, true
	}
	return "", false
}

// ExtractModulesFromSource parses AILANG source and returns the sorted, de-duplicated
// set of std/* module paths it imports (e.g. ["std/io", "std/list"]).
// Only std/* imports are reported (the docs --examples lookup is std-module keyed).
func ExtractModulesFromSource(source, filename string) ([]string, error) {
	l := lexer.New(source, filename)
	p := parser.New(l)
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		// A file that fails to parse is already red via the verify gate; the
		// extractor never guesses at malformed input.
		return nil, fmt.Errorf("parse error in %s: %s", filename, errs[0])
	}
	if file == nil {
		return nil, fmt.Errorf("nil AST for %s", filename)
	}

	seen := make(map[string]struct{})
	for _, imp := range file.Imports {
		if imp == nil {
			continue
		}
		if strings.HasPrefix(imp.Path, "std/") {
			seen[imp.Path] = struct{}{}
		}
	}

	mods := make([]string, 0, len(seen))
	for m := range seen {
		mods = append(mods, m)
	}
	sort.Strings(mods)
	return mods, nil
}

// ExtractModules reads a file and returns its imported std/* modules.
func ExtractModules(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ExtractModulesFromSource(string(data), path)
}

// Equal reports whether two module slices are equal as sets (order-independent).
// Both are expected to already be sorted+deduped by ExtractModules, but we
// normalize defensively so a manifest entry stored in any order still matches.
func Equal(a, b []string) bool {
	na := normalize(a)
	nb := normalize(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}
	return true
}

func normalize(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}
