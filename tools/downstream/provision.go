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
//     The candidate runs its own `ailang lock` there too (provision_lock.go).
//   - Registry versions missing from the package root, and the registry
//     dependencies of each fetched package, are fetched only in the explicit
//     fetch mode (--fetch-deps / FETCH_DEPS=1), with `ailang install --no-bin
//     name@version` into the provision arm's HOME and then copied into the
//     run's package root. A version the registry does not have (HTTP 404)
//     stays unprovisioned and named; any other fetch failure is an instrument
//     failure (exit 3), never a quiet "not measured".
//
// `ailang lock` never gets the network: its HTTP(S) proxy is a closed port, so
// a resolver that reaches for the registry index fails instead of fetching.
// Every command has the per-check deadline, and provisioning as a whole has a
// budget (provisionBudget); running out is exit 3.
//
// Nothing is written outside the run dir: the operator's package root was
// copied earlier (pkgroot.go) and is never written, and the write audit covers
// the snapshot, the provisioned copy and the package root copy.

const lockFile = "ailang.lock"

// provisionBudget bounds provisioning as a whole: commands run one at a time
// under the per-check deadline, so without it a slow resolver times many
// locks at up to --timeout each.
const provisionBudget = 30 * time.Minute

// deadProxy is a closed local port: a child given it as its HTTP(S) proxy
// cannot reach the network (the CI test job uses the same poison).
const deadProxy = "http://127.0.0.1:9"

var errProvisionBudget = fmt.Errorf("provisioning exceeded its %s budget", provisionBudget)

// Provisioner runs the provision arms. Root is the run's package root copy.
type Provisioner struct {
	Arm      *Arm // the baseline binary: generates the locks both arms read
	CandArm  *Arm // the candidate binary: its own lock, compared (nil = skip)
	Root     string
	RunDir   string
	Fetch    bool
	Timeout  int
	Nice     bool
	Deadline time.Time
	Report   *ProvisionReport
	Checks   []LockCheck     // one per generated lock, scored with the entries
	fetched  map[string]bool // name@version added to Root by a fetch
}

func newProvisioner(a, cand *Arm, root, runDir string, fetch bool, timeout int, nice bool) *Provisioner {
	mode := "offline"
	if fetch {
		mode = "fetch"
	}
	rep := &ProvisionReport{Binary: a.Version, Mode: mode, Fetched: []ProvisionedPackage{}}
	if cand != nil {
		rep.Candidate = cand.Version
	}
	return &Provisioner{Arm: a, CandArm: cand, Root: root, RunDir: runDir, Fetch: fetch, Timeout: timeout, Nice: nice,
		Deadline: time.Now().Add(provisionBudget), Report: rep, fetched: map[string]bool{}}
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

// lockPlan is one package dir that needs a lock generated.
type lockPlan struct {
	dir, why   string
	unreadable bool   // a committed lock exists but does not parse
	action     int    // index of its LockAction
	failed     string // the baseline's `ailang lock` error, if any
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
	var plans []lockPlan
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
			plans = append(plans, lockPlan{dir: rel, why: "no committed " + lockFile})
		default:
			plans = append(plans, lockPlan{dir: rel, why: "committed " + lockFile + " " + lerr.Error(), unreadable: true})
			p.Report.Findings = append(p.Report.Findings, fmt.Sprintf(
				"%s: the committed %s/%s cannot be read (%s); this run regenerated it, the corpus should fix or re-lock it", c.Name, rel, lockFile, lerr))
		}
	}
	for _, pl := range plans {
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
	if len(plans) > 0 {
		tree = filepath.Join(p.RunDir, "provisioned", c.Name)
		if _, _, _, err := copyRegular(snap, tree, func(string) bool { return false }); err != nil {
			return "", fmt.Errorf("copy %s for provisioning: %w", c.Name, err)
		}
		cp.Tree, _ = filepath.Rel(p.RunDir, tree)
		for i := range plans {
			pl := &plans[i]
			dir := filepath.Join(tree, pl.dir)
			act := LockAction{Dir: pl.dir, Lock: "generated", Reason: pl.why}
			if pl.unreadable {
				act.Lock = "regenerated"
				if err := os.Remove(filepath.Join(dir, lockFile)); err != nil {
					return "", err
				}
			}
			out, err := p.run(p.Arm, dir, false, "lock")
			if err == errProvisionBudget {
				return "", err
			}
			if err != nil {
				pl.failed = lastErrorLine(out, err)
				act.Lock, act.Reason = "FAILED", pl.why+"; `ailang lock`: "+pl.failed
				cp.Unprovisioned = append(cp.Unprovisioned, pl.dir+": "+act.Reason)
			}
			pl.action = len(cp.Locks)
			cp.Locks = append(cp.Locks, act)
		}
		if err := p.candidateLocks(c, snap, tree, plans, &cp); err != nil {
			return "", err
		}
	}
	cp.Packages = p.inventory(tree, dirs)
	return tree, nil
}

// registryClosure is every registry name@version a package dir's manifest
// reaches through its path dependencies inside the snapshot (a path that leaves
// the snapshot is not followed), for fetch mode.
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
// fetching in fetch mode — and then each fetched package's own registry
// dependencies. A 404 leaves it unprovisioned (named); any other fetch failure
// is returned (exit 3).
func (p *Provisioner) provideAll(want map[string]string, cp *CorpusProvision) error {
	queue := make([]string, 0, len(want))
	for k := range want {
		queue = append(queue, k)
	}
	sort.Strings(queue)
	for len(queue) > 0 {
		k := queue[0]
		queue = queue[1:]
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
			continue
		}
		deps, _ := readDeps(filepath.Join(p.Root, filepath.FromSlash(name), version))
		for _, d := range deps {
			dk := d.Name + "@" + d.Version
			if d.Path == "" && d.Git == "" && d.Version != "" && want[dk] == "" {
				want[dk] = k
				queue = append(queue, dk)
			}
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
	out, err := p.run(p.Arm, cwd, true, "install", "--no-bin", name+"@"+version)
	if m := registryURLRE.FindStringSubmatch(out); m != nil && p.Report.Registry == "" {
		p.Report.Registry = m[1]
	}
	if err != nil {
		// Only the registry's own "no such version" is a fact about the
		// dependency; a 5xx, a timeout or no network says nothing about it.
		if strings.Contains(out, "registry returned HTTP 404 for metadata of "+name+"@"+version) {
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

// run executes a provision arm under the per-check deadline (bounded by the
// provisioning budget) and kills its process group when it expires. Without
// network it reads the run's package root and its proxy is a closed port; with
// network (install only) it has no package root and writes its own cache.
func (p *Provisioner) run(a *Arm, dir string, network bool, args ...string) (string, error) {
	timeout := time.Duration(p.Timeout) * time.Second
	left := time.Until(p.Deadline)
	if left <= 0 {
		return "", errProvisionBudget
	}
	if left < timeout {
		timeout = left
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	name := a.Bin
	if p.Nice {
		args = append([]string{"-n", "15", a.Bin}, args...)
		name = "nice"
	}
	cmd := exec.CommandContext(ctx, name, args...)
	ownGroup(cmd)
	cmd.WaitDelay = pipeGrace
	cmd.Dir = dir
	cmd.Env = a.Env()
	if !network {
		cmd.Env = append(cmd.Env, "AILANG_PACKAGE_ROOT="+p.Root, "HTTP_PROXY="+deadProxy, "HTTPS_PROXY="+deadProxy,
			"http_proxy="+deadProxy, "https_proxy="+deadProxy, "NO_PROXY=", "no_proxy=")
	}
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		if time.Until(p.Deadline) <= 0 {
			return buf.String(), errProvisionBudget
		}
		err = fmt.Errorf("timed out after %s", timeout.Round(time.Second))
	}
	return buf.String(), err
}

// lastErrorLine is the most useful line of a failed command's output.
func lastErrorLine(out string, err error) string {
	if err != nil && strings.HasPrefix(err.Error(), "timed out") {
		return err.Error()
	}
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
