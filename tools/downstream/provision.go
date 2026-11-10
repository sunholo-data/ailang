package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Dependency provisioning (design doc §6.1 step 3, milestone M5).
//
// Each corpus gets the dependencies its own manifests and lock files declare,
// through the project's own commands run by the BASELINE binary in its own
// scratch HOME (the "provision" arm), so both arms read one provisioned input:
//
//   - A committed ailang.lock that parses is honoured as committed: its
//     registry packages must be in the run's package root.
//   - A package dir with dependencies and no lock, or with a lock that does not
//     parse, gets `ailang lock` — in a per-run COPY of the snapshot
//     (<run>/provisioned/<corpus>), never in the snapshot itself. Path
//     dependencies resolve inside that copy, i.e. from the snapshot's content.
//   - Registry versions missing from the package root are fetched only in the
//     explicit fetch mode (--fetch-deps / FETCH_DEPS=1), with `ailang install
//     --no-bin name@version` into the provision arm's HOME and then copied into
//     the run's package root. A version the registry does not have (HTTP 404)
//     stays unprovisioned and named; any other fetch failure is an instrument
//     failure (exit 3), never a quiet "not measured".
//
// Nothing is written outside the run dir: the operator's package root was
// copied earlier (pkgroot.go) and is never written, and the write audit covers
// the snapshot, the provisioned copy and the package root copy.

const lockFile = "ailang.lock"

// ProvisionReport is report.json's "provisioning" object.
type ProvisionReport struct {
	Binary   string               `json:"binary"` // the provision arm's version line
	Mode     string               `json:"mode"`   // offline | fetch
	Registry string               `json:"registry,omitempty"`
	Fetched  []ProvisionedPackage `json:"fetched"`
	Corpora  []CorpusProvision    `json:"corpora"`
}

// CorpusProvision is what one corpus needed and what it got.
type CorpusProvision struct {
	Corpus string `json:"corpus"`
	// Tree is the provisioned copy, relative to the run dir; empty when the
	// entries read the snapshot as it is (nothing had to be locked).
	Tree          string               `json:"tree,omitempty"`
	Locks         []LockAction         `json:"locks"`
	Packages      []ProvisionedPackage `json:"packages"`
	Unprovisioned []string             `json:"unprovisioned,omitempty"`
}

// LockAction is one package dir's lock: committed, generated (none committed),
// regenerated (the committed one does not parse) or FAILED.
type LockAction struct {
	Dir    string `json:"dir"`
	Lock   string `json:"lock"`
	Reason string `json:"reason,omitempty"`
}

// ProvisionedPackage is one dependency an arm reads, with where it came from.
type ProvisionedPackage struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Kind        string `json:"kind"`   // registry | git | path
	Source      string `json:"source"` // package root copy | registry fetch | MISSING | path <rel>
	ContentHash string `json:"content_hash,omitempty"`
}

// Provisioner runs the provision arm. Root is the run's package root copy.
type Provisioner struct {
	Arm     *Arm
	Root    string
	RunDir  string
	Fetch   bool
	Timeout int
	Nice    bool
	Report  *ProvisionReport
	fetched map[string]bool // name@version added to Root by a fetch
}

func newProvisioner(a *Arm, root, runDir string, fetch bool, timeout int, nice bool) *Provisioner {
	mode := "offline"
	if fetch {
		mode = "fetch"
	}
	return &Provisioner{Arm: a, Root: root, RunDir: runDir, Fetch: fetch, Timeout: timeout, Nice: nice,
		Report: &ProvisionReport{Binary: a.Version, Mode: mode, Fetched: []ProvisionedPackage{}}, fetched: map[string]bool{}}
}

// lockJSON is the part of ailang.lock the gate reads.
type lockJSON struct {
	Schema   string `json:"schema"`
	Packages []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Source      string `json:"source"`
		Path        string `json:"path"`
		ContentHash string `json:"content_hash"`
	} `json:"packages"`
}

func readLock(dir string) (*lockJSON, error) {
	data, err := os.ReadFile(filepath.Join(dir, lockFile))
	if err != nil {
		return nil, err
	}
	var l lockJSON
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, fmt.Errorf("does not parse: %v", err)
	}
	if l.Schema != "ailang.lock/v1" {
		return nil, fmt.Errorf("schema %q is not ailang.lock/v1", l.Schema)
	}
	return &l, nil
}

// manifestDep is one [dependencies] line: a registry version or a path.
type manifestDep struct{ Name, Version, Path, Git string }

func readDeps(dir string) ([]manifestDep, error) {
	var m struct {
		Dependencies map[string]interface{} `toml:"dependencies"`
	}
	if _, err := toml.DecodeFile(filepath.Join(dir, "ailang.toml"), &m); err != nil {
		return nil, err
	}
	var out []manifestDep
	for name, v := range m.Dependencies {
		d := manifestDep{Name: name}
		switch t := v.(type) {
		case string:
			d.Version = t
		case map[string]interface{}:
			d.Version, _ = t["version"].(string)
			d.Path, _ = t["path"].(string)
			d.Git, _ = t["git"].(string)
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// packageDirs lists the corpus's package dirs (an ailang.toml), relative to
// root, pruned and excluded exactly as entry discovery is.
func packageDirs(c Corpus, root string) ([]string, error) {
	exc := newGlobSet(c.Exclude)
	var out []string
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
		rel, _ := filepath.Rel(root, p)
		if d.Name() == "ailang.toml" && d.Type().IsRegular() && !exc.match(filepath.ToSlash(rel)) {
			out = append(out, filepath.Dir(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func (p *Provisioner) inRoot(name, version string) bool {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || version == "" {
		return false
	}
	fi, err := os.Stat(filepath.Join(p.Root, parts[0], parts[1], version))
	return err == nil && fi.IsDir()
}

// Corpus provisions one materialised corpus and returns the tree its entries
// must be discovered in. An error is an instrument failure (exit 3).
func (p *Provisioner) Corpus(c Corpus, snap string) (string, error) {
	cp := CorpusProvision{Corpus: c.Name, Locks: []LockAction{}, Packages: []ProvisionedPackage{}}
	defer func() { p.Report.Corpora = append(p.Report.Corpora, cp) }()
	dirs, err := packageDirs(c, snap)
	if err != nil {
		return "", err
	}
	type plan struct{ dir, why string }
	var relock []plan
	want := map[string]string{} // name@version -> who pins it
	for _, rel := range dirs {
		abs := filepath.Join(snap, rel)
		deps, derr := readDeps(abs)
		if derr != nil {
			cp.Locks = append(cp.Locks, LockAction{Dir: rel, Lock: "none", Reason: "ailang.toml does not parse; the check reports it: " + firstLine(derr.Error())})
			continue
		}
		lk, lerr := readLock(abs)
		switch {
		case lerr == nil:
			cp.Locks = append(cp.Locks, LockAction{Dir: rel, Lock: "committed"})
			for _, e := range lk.Packages {
				switch {
				case e.Source == "registry":
					want[e.Name+"@"+e.Version] = rel + "/" + lockFile
				case e.Source == "git" && !p.inRoot(e.Name, e.Version):
					// The gate reads git packages from the package root only (as
					// AILANG_PACKAGE_ROOT does); it never clones.
					cp.Unprovisioned = append(cp.Unprovisioned, fmt.Sprintf("%s@%s (git, pinned by %s/%s): not in the package root; the gate does not clone git dependencies", e.Name, e.Version, rel, lockFile))
				}
			}
		case errors.Is(lerr, fs.ErrNotExist) && len(deps) == 0:
			// nothing to resolve: the check needs no lock
		case errors.Is(lerr, fs.ErrNotExist):
			relock = append(relock, plan{rel, "no committed " + lockFile})
		default:
			relock = append(relock, plan{rel, "committed " + lockFile + " " + lerr.Error()})
		}
	}
	for _, pl := range relock {
		for _, d := range p.registryClosure(snap, pl.dir, map[string]bool{}) {
			if _, ok := want[d]; !ok {
				want[d] = pl.dir + "/ailang.toml"
			}
		}
	}
	if err := p.provideAll(want, &cp); err != nil {
		return "", err
	}
	tree := snap
	if len(relock) > 0 {
		tree = filepath.Join(p.RunDir, "provisioned", c.Name)
		if _, _, _, err := copyRegular(snap, tree, func(string) bool { return false }); err != nil {
			return "", fmt.Errorf("copy %s for provisioning: %w", c.Name, err)
		}
		cp.Tree, _ = filepath.Rel(p.RunDir, tree)
		for _, pl := range relock {
			dir := filepath.Join(tree, pl.dir)
			if !strings.HasPrefix(pl.why, "no committed") {
				if err := os.Remove(filepath.Join(dir, lockFile)); err != nil {
					return "", err
				}
			}
			act := LockAction{Dir: pl.dir, Lock: "generated", Reason: pl.why}
			if !strings.HasPrefix(pl.why, "no committed") {
				act.Lock = "regenerated"
			}
			if out, err := p.run(dir, true, "lock"); err != nil {
				act.Lock, act.Reason = "FAILED", pl.why+"; `ailang lock`: "+lastErrorLine(out, err)
				cp.Unprovisioned = append(cp.Unprovisioned, pl.dir+": "+act.Reason)
			}
			cp.Locks = append(cp.Locks, act)
		}
	}
	cp.Packages = p.inventory(tree, dirs)
	return tree, nil
}

// registryClosure is every registry name@version a package dir's manifest
// reaches through its path dependencies (inside the snapshot), for fetch mode.
func (p *Provisioner) registryClosure(snap, rel string, seen map[string]bool) []string {
	if seen[rel] {
		return nil
	}
	seen[rel] = true
	deps, err := readDeps(filepath.Join(snap, rel))
	if err != nil {
		return nil
	}
	var out []string
	for _, d := range deps {
		switch {
		case d.Path != "":
			sub := filepath.Clean(filepath.Join(rel, d.Path))
			if filepath.IsLocal(sub) {
				out = append(out, p.registryClosure(snap, sub, seen)...)
			}
		case d.Git == "" && d.Version != "":
			out = append(out, d.Name+"@"+d.Version)
		}
	}
	return out
}

// provideAll makes sure every wanted name@version is in the package root,
// fetching in fetch mode. A 404 leaves it unprovisioned (named); any other fetch
// failure is returned (exit 3).
func (p *Provisioner) provideAll(want map[string]string, cp *CorpusProvision) error {
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		i := strings.LastIndexByte(k, '@')
		name, version := k[:i], k[i+1:]
		if p.inRoot(name, version) {
			continue
		}
		if !p.Fetch {
			cp.Unprovisioned = append(cp.Unprovisioned, fmt.Sprintf("%s (pinned by %s): not in the package root; FETCH_DEPS=1 fetches it from the registry", k, want[k]))
			continue
		}
		missing, err := p.fetch(name, version)
		if err != nil {
			return fmt.Errorf("provisioning %s (pinned by %s in %s): %w", k, want[k], cp.Corpus, err)
		}
		if missing != "" {
			cp.Unprovisioned = append(cp.Unprovisioned, fmt.Sprintf("%s (pinned by %s): %s", k, want[k], missing))
		}
	}
	return nil
}

var registryURLRE = regexp.MustCompile(`from (\S+?)\.\.\.`)

// fetch runs `ailang install --no-bin name@version` in the provision arm's HOME
// (no package root: install writes its own cache), then copies every package
// version it fetched into the run's package root. It returns a non-empty reason
// when the registry does not have that version.
func (p *Provisioner) fetch(name, version string) (string, error) {
	cwd := filepath.Join(p.Arm.Dir, "tmp", "install")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		return "", err
	}
	out, err := p.run(cwd, false, "install", "--no-bin", name+"@"+version)
	if m := registryURLRE.FindStringSubmatch(out); m != nil && p.Report.Registry == "" {
		p.Report.Registry = m[1]
	}
	if err != nil {
		if strings.Contains(out, "HTTP 404") {
			return "the registry has no such version (HTTP 404); it cannot be provisioned here", nil
		}
		return "", fmt.Errorf("`ailang install --no-bin %s@%s` failed: %s", name, version, lastErrorLine(out, err))
	}
	cache := filepath.Join(p.Arm.Dir, "home", ".ailang", "cache", "registry")
	vendors, _ := os.ReadDir(cache)
	for _, v := range vendors {
		pkgs, _ := os.ReadDir(filepath.Join(cache, v.Name()))
		for _, pk := range pkgs {
			vers, _ := os.ReadDir(filepath.Join(cache, v.Name(), pk.Name()))
			for _, ver := range vers {
				n := v.Name() + "/" + pk.Name()
				if p.inRoot(n, ver.Name()) {
					continue
				}
				dst := filepath.Join(p.Root, v.Name(), pk.Name(), ver.Name())
				if _, _, _, err := copyRegular(filepath.Join(cache, v.Name(), pk.Name(), ver.Name()), dst, func(d string) bool { return d == ".ailang" }); err != nil {
					return "", err
				}
				p.fetched[n+"@"+ver.Name()] = true
				p.Report.Fetched = append(p.Report.Fetched, ProvisionedPackage{Name: n, Version: ver.Name(), Kind: "registry", Source: "registry fetch"})
			}
		}
	}
	if !p.inRoot(name, version) {
		return "", fmt.Errorf("`ailang install` reported success but %s@%s is not in its cache", name, version)
	}
	return "", nil
}

// run executes the provision arm under a deadline that kills its process group.
func (p *Provisioner) run(dir string, withRoot bool, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.Timeout)*time.Second)
	defer cancel()
	name := p.Arm.Bin
	if p.Nice {
		args = append([]string{"-n", "15", p.Arm.Bin}, args...)
		name = "nice"
	}
	cmd := exec.CommandContext(ctx, name, args...)
	ownGroup(cmd)
	cmd.WaitDelay = pipeGrace
	cmd.Dir = dir
	cmd.Env = p.Arm.Env()
	if withRoot {
		cmd.Env = append(cmd.Env, "AILANG_PACKAGE_ROOT="+p.Root)
	}
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("timed out after %d s", p.Timeout)
	}
	return buf.String(), err
}

// lastErrorLine is the most useful line of a failed command's output.
func lastErrorLine(out string, err error) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(lines[i]); strings.HasPrefix(l, "Error") || strings.Contains(l, "failed") {
			return firstLine(strings.ReplaceAll(l, "\t", " "))
		}
	}
	if s := strings.TrimSpace(out); s != "" {
		return firstLine(lines[len(lines)-1])
	}
	return err.Error()
}

// render writes report.md's provisioning section (the full package list is in
// report.json).
func (pr *ProvisionReport) render(p func(string, ...interface{})) {
	if pr == nil {
		return
	}
	p("\n## Dependency provisioning\n\nMode `%s` · provisioned by %s (the baseline binary, own scratch HOME)", pr.Mode, cell(pr.Binary))
	if pr.Registry != "" {
		p(" · registry %s", pr.Registry)
	}
	p("\n\n| corpus | tree | locks committed / generated / regenerated / FAILED | packages: package root / fetched / path / MISSING |\n|---|---|---|---|\n")
	for _, c := range pr.Corpora {
		locks := map[string]int{}
		for _, l := range c.Locks {
			locks[l.Lock]++
		}
		src := map[string]int{}
		for _, pk := range c.Packages {
			src[strings.Fields(pk.Source)[0]]++
		}
		tree := "snapshot"
		if c.Tree != "" {
			tree = "`" + filepath.ToSlash(c.Tree) + "`"
		}
		p("| %s | %s | %d / %d / %d / **%d** | %d / %d / %d / **%d** |\n", c.Corpus, tree,
			locks["committed"], locks["generated"], locks["regenerated"], locks["FAILED"],
			src["package"], src["registry"], src["path"], src["MISSING"])
	}
	p("\n")
	for _, f := range pr.Fetched {
		p("- fetched %s@%s from the registry\n", f.Name, f.Version)
	}
	for _, c := range pr.Corpora {
		for _, u := range c.Unprovisioned {
			p("- **UNPROVISIONED** %s: %s\n", c.Corpus, cell(u))
		}
	}
}

// inventory lists every package the corpus's locks name, once, with its source.
func (p *Provisioner) inventory(tree string, dirs []string) []ProvisionedPackage {
	seen := map[string]bool{}
	out := []ProvisionedPackage{}
	for _, rel := range dirs {
		lk, err := readLock(filepath.Join(tree, rel))
		if err != nil {
			continue
		}
		for _, e := range lk.Packages {
			pp := ProvisionedPackage{Name: e.Name, Version: e.Version, Kind: e.Source, ContentHash: e.ContentHash}
			switch {
			case e.Source == "path":
				pp.Source = "path " + filepath.ToSlash(filepath.Join(rel, e.Path))
			case p.fetched[e.Name+"@"+e.Version]:
				pp.Source = "registry fetch"
			case p.inRoot(e.Name, e.Version):
				pp.Source = "package root copy"
			default:
				pp.Source = "MISSING"
			}
			key := pp.Name + "@" + pp.Version + " " + pp.Source
			if !seen[key] {
				seen[key] = true
				out = append(out, pp)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Version < out[j].Version
	})
	return out
}
