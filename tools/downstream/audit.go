package main

import (
	"io/fs"
	"path/filepath"
	"sort"
	"time"
)

// AuditFinding is a path written during the run where nothing may be written.
type AuditFinding struct {
	Scope string `json:"scope"` // "snapshot <corpus>" or "local checkout <path>"
	Path  string `json:"path"`
}

// writtenSince lists entries under root (pruning .git, whose mtimes other
// processes move) modified after stamp. A non-empty result is an instrument
// failure: either a downstream checkout was touched, or the two arms did not see
// the same input.
func writtenSince(root string, stamp time.Time, limit int) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() && d.Name() == "node_modules" {
			return filepath.SkipDir
		}
		info, err := d.Info()
		if err != nil {
			return nil // raced with a concurrent delete; not ours
		}
		if info.ModTime().After(stamp) {
			rel, _ := filepath.Rel(root, p)
			out = append(out, rel)
			if len(out) >= limit {
				return filepath.SkipAll
			}
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}
