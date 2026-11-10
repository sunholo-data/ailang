package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// PackageRootCopy describes the run's private copy of the operator's package root.
//
// Both arms read the copy, never the operator's directory: that directory is
// shared with every other ailang process on the machine (a scheduled job
// compiling a registry package writes its compile cache there), so auditing it
// blamed the gate for other processes' writes, and reading it live let them
// change an arm's input mid-run. The copy leaves out `.ailang` dirs (compile
// caches, never input), so each arm also starts without anyone's compiled
// artifacts. Provisioning (provision.go) may add fetched registry versions to
// the copy before the arms start; a write into it after that is the gate's own,
// and an instrument failure.
type PackageRootCopy struct {
	Source string `json:"source"`
	Path   string `json:"-"`
	Files  int    `json:"files"`
	Bytes  int64  `json:"bytes"`
	Links  int    `json:"links_skipped,omitempty"`
}

func copyPackageRoot(src, dest string) (PackageRootCopy, error) {
	pc := PackageRootCopy{Source: src, Path: dest}
	var err error
	pc.Files, pc.Bytes, pc.Links, err = copyRegular(src, dest, func(name string) bool { return name == ".ailang" })
	if err != nil {
		return pc, fmt.Errorf("copy package root %s: %w", src, err)
	}
	return pc, nil
}

// copyRegular copies the directories and regular files under src into dest
// (which must not hold them yet), skipping directories skipDir names and
// counting, not following, every other kind of entry.
func copyRegular(src, dest string, skipDir func(name string) bool) (files int, size int64, links int, err error) {
	if err = os.MkdirAll(dest, 0o755); err != nil {
		return
	}
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		switch {
		case d.IsDir() && p != src && skipDir(d.Name()):
			return filepath.SkipDir
		case d.IsDir():
			return os.MkdirAll(filepath.Join(dest, rel), 0o755)
		case !d.Type().IsRegular():
			links++
			return nil
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		info, err := in.Stat()
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dest, rel), in); err != nil {
			return err
		}
		files++
		size += info.Size()
		return nil
	})
	return
}
