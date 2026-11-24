package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// gapCorpus is one corpus to measure: a directory on this machine, read only.
type gapCorpus struct {
	Name    string   `yaml:"name"`
	Repo    string   `yaml:"repo"`
	Local   string   `yaml:"local"`
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`

	Root string `yaml:"-"` // resolved directory
}

// loadCorpora reads the downstream gate's inventory (tools/downstream/corpora.yaml),
// so the two instruments measure the same population. `self` is this checkout;
// every other corpus is its `local:` checkout, used as it is (never fetched,
// pulled or written). A corpus whose checkout is absent is returned with Root "".
func loadCorpora(path, repoRoot string) ([]gapCorpus, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cs []gapCorpus
	if err := yaml.Unmarshal(raw, &cs); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for i := range cs {
		c := &cs[i]
		switch {
		case c.Repo == "self":
			c.Root = repoRoot
		case c.Local != "":
			c.Root = expandHome(c.Local)
			if st, err := os.Stat(c.Root); err != nil || !st.IsDir() {
				c.Root = ""
			}
		}
		for _, g := range append(append([]string{}, c.Include...), c.Exclude...) {
			if _, err := globPrefix(g); err != nil {
				return nil, fmt.Errorf("corpus %s: %w", c.Name, err)
			}
		}
	}
	return cs, nil
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// globPrefix accepts the glob shapes corpora.yaml uses — "dir/**" and
// "dir/**/*.ail" — and returns the directory prefix ("dir/"). Anything else is
// an error rather than a silently different match.
func globPrefix(g string) (string, error) {
	for _, suffix := range []string{"/**/*.ail", "/**"} {
		if dir, ok := strings.CutSuffix(g, suffix); ok && dir != "" && !strings.ContainsAny(dir, "*?[") {
			return dir + "/", nil
		}
	}
	return "", fmt.Errorf("glob %q: only \"dir/**\" and \"dir/**/*.ail\" are supported", g)
}

func matchesAny(rel string, globs []string) bool {
	for _, g := range globs {
		if p, _ := globPrefix(g); strings.HasPrefix(rel, p) {
			return true
		}
	}
	return false
}

var contractLine = regexp.MustCompile(`^\s*(ensures|requires)\b`)

// contractFiles lists, sorted and relative to c.Root, the .ail files that
// declare a contract. Build caches, dependency trees, tests and nested git
// worktrees are not the corpus's own sources.
func contractFiles(c gapCorpus) ([]string, error) {
	var out []string
	err := filepath.WalkDir(c.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entry: not part of what we can measure
		}
		rel, _ := filepath.Rel(c.Root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if p != c.Root {
				switch d.Name() {
				case ".git", ".ailang", "node_modules", ".claude", ".agents":
					return filepath.SkipDir
				}
				if _, err := os.Stat(filepath.Join(p, ".git")); err == nil {
					return filepath.SkipDir // a nested checkout or worktree
				}
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".ail") || strings.HasSuffix(rel, "_test.ail") {
			return nil
		}
		if len(c.Include) > 0 && !matchesAny(rel, c.Include) {
			return nil
		}
		if matchesAny(rel, c.Exclude) {
			return nil
		}
		if hasContract(p) {
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func hasContract(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		if contractLine.MatchString(sc.Text()) {
			return true
		}
	}
	return false
}

func fileHash(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

// gitHead returns the checkout's HEAD and whether its tree has uncommitted
// changes ("" when it is not a git checkout).
func gitHead(dir string) (sha string, dirty bool) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return "", false
	}
	st, _ := exec.Command("git", "-C", dir, "status", "--porcelain", "--untracked-files=no").Output()
	return strings.TrimSpace(string(out)), len(strings.TrimSpace(string(st))) > 0
}
