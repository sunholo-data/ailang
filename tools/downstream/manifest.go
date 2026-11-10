package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Tier levels. "off" leaves a corpus out of that tier; "advisory" checks and
// reports it but never fails the gate; "blocking" counts its breaks in the gate.
const (
	tierOff      = "off"
	tierAdvisory = "advisory"
	tierBlocking = "blocking"
)

// selfRepo names the ailang repository the tool runs from (the in-repo corpora).
const selfRepo = "self"

// Corpus is one entry of corpora.yaml (design doc §6.4).
type Corpus struct {
	Name  string `yaml:"name"`
	Repo  string `yaml:"repo"` // owner/name on GitHub, or "self"
	Ref   RefSet `yaml:"ref"`
	Local string `yaml:"local"` // the operator's checkout; source only in REF=worktree
	// Include/Exclude are globs relative to the corpus root (** spans directories).
	// An empty Include means every .ail file in the snapshot.
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
	Modes   []string `yaml:"modes"` // files, package
	// ExpectedFail globs name intended-failure fixtures: a candidate failure there
	// is expected_fail, not a break.
	ExpectedFail []string `yaml:"expected_fail"`
	// ExamplesManifest is an examples/manifest.json-shaped file (relative to the
	// corpus root); entries whose expected.exit_code is non-zero are expected to
	// fail with their error_pattern.
	ExamplesManifest string    `yaml:"examples_manifest"`
	PinFiles         []string  `yaml:"pin_files"` // "path:regex" lines for pins.md
	Test             *TestSpec `yaml:"test"`      // informational; never run by the gate
	Owner            string    `yaml:"owner"`
	Tier             TierSet   `yaml:"tier"`
	Notes            string    `yaml:"notes"`
}

// RefSet holds the refs for REF=default and REF=pinned.
type RefSet struct {
	Default string `yaml:"default"` // empty = the remote's default branch
	Pinned  string `yaml:"pinned"`
}

// TierSet is the corpus's membership in each tier.
type TierSet struct {
	CI      string `yaml:"ci"`
	Release string `yaml:"release"`
}

// TestSpec records how the corpus tests itself; the gate does not run it.
type TestSpec struct {
	Cmd      string `yaml:"cmd"`
	Hermetic bool   `yaml:"hermetic"`
}

// ExpectedBreak acknowledges a (corpus, code) bucket of a deliberate break (§6.5).
type ExpectedBreak struct {
	Change    string   `yaml:"change"`
	Code      string   `yaml:"code"`
	Corpus    string   `yaml:"corpus"`
	Max       int      `yaml:"max"`
	Fix       string   `yaml:"fix"`
	Republish []string `yaml:"republish"`
	Expires   string   `yaml:"expires"`
}

var corpusNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func decodeStrict(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		if err.Error() == "EOF" {
			return nil // an empty file is an empty list
		}
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// LoadCorpora reads and validates corpora.yaml. Unknown keys, duplicate names
// and missing tier values are errors: the manifest decides what blocks a release,
// so nothing in it is defaulted silently (except modes, which §6.4 defaults).
func LoadCorpora(path string) ([]Corpus, error) {
	var cs []Corpus
	if err := decodeStrict(path, &cs); err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return nil, fmt.Errorf("%s: no corpora", path)
	}
	seen := map[string]bool{}
	for i := range cs {
		c := &cs[i]
		if !corpusNameRE.MatchString(c.Name) {
			return nil, fmt.Errorf("%s: corpus %d: bad or missing name %q", path, i, c.Name)
		}
		if seen[c.Name] {
			return nil, fmt.Errorf("%s: duplicate corpus %q", path, c.Name)
		}
		seen[c.Name] = true
		if c.Repo != selfRepo && strings.Count(c.Repo, "/") != 1 {
			return nil, fmt.Errorf("%s: corpus %s: repo must be owner/name or %q, got %q", path, c.Name, selfRepo, c.Repo)
		}
		if len(c.Modes) == 0 {
			c.Modes = []string{"files", "package"}
		}
		for _, m := range c.Modes {
			if m != "files" && m != "package" {
				return nil, fmt.Errorf("%s: corpus %s: unknown mode %q", path, c.Name, m)
			}
		}
		for _, t := range []struct{ name, v string }{{"ci", c.Tier.CI}, {"release", c.Tier.Release}} {
			if t.v != tierOff && t.v != tierAdvisory && t.v != tierBlocking {
				return nil, fmt.Errorf("%s: corpus %s: tier.%s must be off|advisory|blocking, got %q", path, c.Name, t.name, t.v)
			}
		}
		for _, g := range append(append(append([]string{}, c.Include...), c.Exclude...), c.ExpectedFail...) {
			if _, err := globRegexp(g); err != nil {
				return nil, fmt.Errorf("%s: corpus %s: %w", path, c.Name, err)
			}
		}
		for _, p := range c.PinFiles {
			if _, _, err := splitPinSpec(p); err != nil {
				return nil, fmt.Errorf("%s: corpus %s: %w", path, c.Name, err)
			}
		}
	}
	return cs, nil
}

// LoadExpected reads expected-breaks.yaml and checks every entry names a known corpus.
func LoadExpected(path string, corpora []Corpus) ([]ExpectedBreak, error) {
	var es []ExpectedBreak
	if err := decodeStrict(path, &es); err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, c := range corpora {
		known[c.Name] = true
	}
	for i, e := range es {
		switch {
		case e.Code == "":
			return nil, fmt.Errorf("%s: entry %d: code is required", path, i)
		case !known[e.Corpus]:
			return nil, fmt.Errorf("%s: entry %d: unknown corpus %q", path, i, e.Corpus)
		case e.Max <= 0:
			return nil, fmt.Errorf("%s: entry %d (%s/%s): max must be > 0", path, i, e.Corpus, e.Code)
		case e.Change == "":
			return nil, fmt.Errorf("%s: entry %d (%s/%s): change is required", path, i, e.Corpus, e.Code)
		}
	}
	return es, nil
}

// tierOf returns the corpus's level in the named tier.
func (c Corpus) tierOf(tier string) string {
	if tier == "ci" {
		return c.Tier.CI
	}
	return c.Tier.Release
}

func splitPinSpec(spec string) (string, *regexp.Regexp, error) {
	i := strings.Index(spec, ":")
	if i <= 0 {
		return "", nil, fmt.Errorf("pin_files entry %q: want path:regex", spec)
	}
	re, err := regexp.Compile(spec[i+1:])
	if err != nil {
		return "", nil, fmt.Errorf("pin_files entry %q: %w", spec, err)
	}
	return spec[:i], re, nil
}

// globRegexp compiles a slash-separated glob: ** spans directories, * and ? stay
// within one path segment.
func globRegexp(glob string) (*regexp.Regexp, error) {
	if glob == "" || strings.HasPrefix(glob, "/") {
		return nil, fmt.Errorf("glob %q: must be non-empty and relative", glob)
	}
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(glob); i++ {
		ch := glob[i]
		switch {
		case ch == '*' && i+1 < len(glob) && glob[i+1] == '*':
			i++
			if i+1 < len(glob) && glob[i+1] == '/' {
				i++
				b.WriteString("(?:.*/)?")
			} else {
				b.WriteString(".*")
			}
		case ch == '*':
			b.WriteString("[^/]*")
		case ch == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(ch)))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

// globSet matches a relative path against any of several globs.
type globSet []*regexp.Regexp

func newGlobSet(globs []string) globSet {
	var gs globSet
	for _, g := range globs {
		re, err := globRegexp(g)
		if err != nil {
			panic(err) // validated in LoadCorpora
		}
		gs = append(gs, re)
	}
	return gs
}

func (gs globSet) match(rel string) bool {
	for _, re := range gs {
		if re.MatchString(rel) {
			return true
		}
	}
	return false
}
