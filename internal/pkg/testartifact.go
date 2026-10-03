package pkg

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NamedTestBodyPrefix starts the name of the temporary copy of a test module
// that `ailang test` compiles a named-test body from. Releases up to v0.51.0
// wrote that copy into the package directory, so an interrupted run could
// leave it there (#1502); the copy now lives in a private temp dir, and this
// predicate lets tools recognise leftovers from older binaries.
const NamedTestBodyPrefix = "_namedtest_body_"

// NamedTestBodyGitignore is the .gitignore line `ailang pkg init` scaffolds so
// a leftover body copy is never committed.
const NamedTestBodyGitignore = NamedTestBodyPrefix + "*.ail"

// IsNamedTestBodyFile reports whether base (a file's base name) is a leftover
// named-test body copy: "_namedtest_body_<digits>.ail".
func IsNamedTestBodyFile(base string) bool {
	rest, ok := strings.CutPrefix(base, NamedTestBodyPrefix)
	if !ok {
		return false
	}
	digits, ok := strings.CutSuffix(rest, ".ail")
	if !ok || digits == "" {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ScaffoldGitignore makes dir's .gitignore ignore leftover named-test body
// copies, creating the file if needed. Existing lines are kept, and the entry
// is added at most once, so running it again changes nothing.
func ScaffoldGitignore(dir string) error {
	path := filepath.Join(dir, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == NamedTestBodyGitignore {
			return nil
		}
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += NamedTestBodyGitignore + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// FindNamedTestBodyFiles returns every leftover named-test body file under
// dir (hidden directories skipped, as source discovery skips them), as
// sorted slash-separated paths relative to dir. `pkg quality` and `publish`
// refuse a package that has any (PUB024).
func FindNamedTestBodyFiles(dir string) ([]string, error) {
	var found []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != dir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if IsNamedTestBodyFile(d.Name()) {
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				return relErr
			}
			found = append(found, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(found)
	return found, nil
}
