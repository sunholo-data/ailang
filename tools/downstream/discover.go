package main

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Entry is one `ailang check` invocation: a file, or a package directory.
type Entry struct {
	Corpus string `json:"corpus"`
	Kind   string `json:"kind"` // file | package
	Rel    string `json:"rel"`  // path relative to the corpus root
	Cwd    string `json:"-"`    // absolute working directory for the check
	Arg    string `json:"-"`    // argument passed to check
	Rule   string `json:"rule"` // how Cwd was chosen
}

func (e Entry) key() string { return e.Corpus + "\x00" + e.Kind + "\x00" + e.Rel }

// prunedDir names directories never walked: VCS data, compile caches, agent
// state, dependency trees and worktree copies (design doc §3 notes).
func prunedDir(name string) bool {
	switch name {
	case ".git", ".ailang", ".claude", "node_modules", ".motoko", ".godot":
		return true
	}
	return strings.HasPrefix(name, ".wt-")
}

var moduleLineRE = regexp.MustCompile(`^\s*module\s+([A-Za-z0-9_/\-]+)`)

// moduleOf returns the module path declared in the first lines of a file.
func moduleOf(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for n := 0; sc.Scan() && n < 64; n++ {
		if m := moduleLineRE.FindStringSubmatch(sc.Text()); m != nil {
			return m[1]
		}
	}
	return ""
}

// fileEntry applies the §6.4 root rule: when the file path ends with its module
// path, cwd is the prefix and the argument is <module>.ail; otherwise cwd is the
// nearest ancestor (within the corpus) holding ailang.toml; otherwise the file's dir.
func fileEntry(corpus, root, rel string) Entry {
	abs := filepath.Join(root, filepath.FromSlash(rel))
	e := Entry{Corpus: corpus, Kind: "file", Rel: rel}
	noext := strings.TrimSuffix(rel, ".ail")
	if mod := moduleOf(abs); mod != "" && (noext == mod || strings.HasSuffix(noext, "/"+mod)) {
		prefix := strings.TrimSuffix(strings.TrimSuffix(noext, mod), "/")
		e.Cwd, e.Arg, e.Rule = filepath.Join(root, filepath.FromSlash(prefix)), mod+".ail", "module-suffix"
		return e
	}
	for d := filepath.Dir(abs); strings.HasPrefix(d, root); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "ailang.toml")); err == nil {
			r, _ := filepath.Rel(d, abs)
			e.Cwd, e.Arg, e.Rule = d, filepath.ToSlash(r), "package-root"
			return e
		}
		if d == root {
			break
		}
	}
	e.Cwd, e.Arg, e.Rule = filepath.Dir(abs), filepath.Base(abs), "dirname"
	return e
}

// Discover lists the entries of a materialised corpus, sorted for determinism.
func Discover(c Corpus, root string) ([]Entry, error) {
	inc, exc := newGlobSet(c.Include), newGlobSet(c.Exclude)
	wantFiles, wantPkgs := false, false
	for _, m := range c.Modes {
		wantFiles = wantFiles || m == "files"
		wantPkgs = wantPkgs || m == "package"
	}
	var out []Entry
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && prunedDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if exc.match(rel) {
			return nil
		}
		switch {
		case wantFiles && strings.HasSuffix(rel, ".ail") && (len(inc) == 0 || inc.match(rel)):
			out = append(out, fileEntry(c.Name, root, rel))
		case wantPkgs && d.Name() == "ailang.toml":
			dir := filepath.Dir(p)
			relDir := filepath.ToSlash(filepath.Dir(rel))
			out = append(out, Entry{Corpus: c.Name, Kind: "package", Rel: relDir, Cwd: dir, Arg: ".", Rule: "package"})
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].key() < out[j].key() })
	return out, err
}
