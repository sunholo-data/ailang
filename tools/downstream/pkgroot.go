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
// artifacts. A write into the copy is the gate's own, and an instrument failure.
type PackageRootCopy struct {
	Source string `json:"source"`
	Path   string `json:"-"`
	Files  int    `json:"files"`
	Bytes  int64  `json:"bytes"`
	Links  int    `json:"links_skipped,omitempty"`
}

func copyPackageRoot(src, dest string) (PackageRootCopy, error) {
	pc := PackageRootCopy{Source: src, Path: dest}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return pc, err
	}
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		switch {
		case d.IsDir() && d.Name() == ".ailang":
			return filepath.SkipDir
		case d.IsDir():
			return os.MkdirAll(filepath.Join(dest, rel), 0o755)
		case !d.Type().IsRegular():
			pc.Links++
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
		pc.Files++
		pc.Bytes += info.Size()
		return nil
	})
	if err != nil {
		return pc, fmt.Errorf("copy package root %s: %w", src, err)
	}
	return pc, nil
}
