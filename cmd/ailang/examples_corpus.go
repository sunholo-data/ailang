package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"

	examplescorpus "github.com/sunholo-data/ailang/examples"
)

// examplesCorpus is where `ailang examples` reads from: an on-disk directory
// laid out like examples/ (manifest.json + runnable/**), or the corpus built
// into the binary.
type examplesCorpus struct {
	fsys fs.FS
	// dir is the on-disk root; "" means the embedded corpus.
	dir string
}

// where names the corpus in messages.
func (c *examplesCorpus) where() string {
	if c.dir != "" {
		return c.dir
	}
	return "the examples corpus built into this binary"
}

// openExamplesCorpus resolves the corpus: an on-disk one (findExamplesDir)
// first, then the one embedded in the binary (#1552), so a release binary on
// a clean machine — or under a per-task HOME — still has examples.
func openExamplesCorpus() (*examplesCorpus, error) {
	if dir, ok := findExamplesDir(); ok {
		return &examplesCorpus{fsys: os.DirFS(dir), dir: dir}, nil
	}
	if _, err := fs.Stat(examplescorpus.Corpus, "manifest.json"); err != nil {
		return nil, fmt.Errorf("examples not found (and this binary carries no embedded corpus)\n\n  To fix, either:\n    1. ailang examples download\n    2. Set AILANG_EXAMPLES=/path/to/ailang/examples")
	}
	return &examplesCorpus{fsys: examplescorpus.Corpus}, nil
}

// findExamplesDir looks for an on-disk corpus: AILANG_EXAMPLES, next to the
// executable, ~/.ailang/examples, then examples/, ../examples, ../../examples
// relative to the working directory. A confined process (AILANG_AGENT_POLICY
// set: the ailang_only lane, and every child `ailang policy-tool` spawns)
// skips ~/.ailang/examples and the cwd-relative candidates — hosts give a
// task its own writable HOME and working directory, so an agent can plant a
// corpus in either, and a confined tool must not read it (#1552).
func findExamplesDir() (string, bool) {
	isDir := func(p string) (string, bool) {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			abs, err := filepath.Abs(p)
			if err != nil {
				return p, true
			}
			return abs, true
		}
		return "", false
	}

	// 1. AILANG_EXAMPLES — set by the host, never by the agent.
	if env := config.ExamplesDir(); env != "" {
		if d, ok := isDir(env); ok {
			return d, true
		}
	}

	// 2. Relative to the executable (local builds: bin/ailang → ../examples).
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil && resolved != "" {
			exe = resolved
		}
		for _, rel := range []string{"../examples", "examples"} {
			if d, ok := isDir(filepath.Join(filepath.Dir(exe), rel)); ok {
				return d, true
			}
		}
	}

	if config.AgentPolicy() != "" {
		return "", false
	}

	// 3. ~/.ailang/examples (populated by `ailang examples download`) —
	// never when confined.
	if dl, err := defaultExamplesDownloadDir(); err == nil {
		if d, ok := isDir(dl); ok {
			return d, true
		}
	}

	// 4. CWD-relative (inside the ailang repo) — never when confined.
	{
		for _, p := range []string{"examples", "../examples", "../../examples"} {
			if d, ok := isDir(p); ok {
				return d, true
			}
		}
	}
	return "", false
}

// manifest reads the corpus's manifest.json. A corpus without one is a named
// error, never a nil manifest (#1553).
func (c *examplesCorpus) manifest() (*ExampleManifest, error) {
	data, err := fs.ReadFile(c.fsys, "manifest.json")
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("examples corpus at %s has no manifest.json", c.where())
	}
	if err != nil {
		return nil, fmt.Errorf("reading manifest at %s: %w", c.where(), err)
	}
	var m ExampleManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest at %s: %w", c.where(), err)
	}
	return &m, nil
}

// loadExamplesManifest resolves the corpus and reads its manifest.
func loadExamplesManifest() (*ExampleManifest, error) {
	c, err := openExamplesCorpus()
	if err != nil {
		return nil, err
	}
	return c.manifest()
}

// exampleSearchResult is one hit of searchExamples.
type exampleSearchResult struct {
	path        string // file name under runnable/
	score       float64
	matchSource string // "tag", "description", "content"
}

// searchExamples scores every runnable/**.ail by tag, description and
// content against query, best first, at most limit. It also returns the
// manifest entries by file name for rendering.
func searchExamples(c *examplesCorpus, query string, limit int) ([]exampleSearchResult, map[string]ExampleEntry, error) {
	m, err := c.manifest()
	if err != nil {
		return nil, nil, err
	}
	meta := make(map[string]ExampleEntry, len(m.Examples))
	for _, ex := range m.Examples {
		meta[ex.Path] = ex
	}

	query = strings.ToLower(query)
	queryWords := strings.Fields(query)
	var results []exampleSearchResult
	walkErr := fs.WalkDir(c.fsys, "runnable", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".ail") {
			return nil
		}
		filename := path.Base(p)
		ex, hasMeta := meta[filename]
		var score float64
		var source string
		if hasMeta {
			score, source = scoreExampleMeta(ex, query, queryWords)
		}
		if content, err := fs.ReadFile(c.fsys, p); err == nil {
			if s := scoreExampleContent(strings.ToLower(string(content)), query, queryWords); s > 0 {
				score = maxFloat(score, s)
				if source == "" {
					source = "content"
				}
			}
		}
		if score > 0 {
			results = append(results, exampleSearchResult{path: filename, score: score, matchSource: source})
		}
		return nil
	})
	if walkErr != nil {
		return nil, nil, fmt.Errorf("searching %s: %w", c.where(), walkErr)
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].score > results[j].score })
	if limit >= 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, meta, nil
}

// scoreExampleMeta scores tags (highest) and the description.
func scoreExampleMeta(ex ExampleEntry, query string, queryWords []string) (float64, string) {
	var score float64
	var source string
	for _, tag := range ex.Tags {
		t := strings.ToLower(tag)
		if strings.Contains(t, query) {
			score, source = 1.0, "tag"
			break
		}
		for _, qw := range queryWords {
			if strings.Contains(t, qw) {
				score, source = maxFloat(score, 0.9), "tag"
			}
		}
	}
	if ex.Description == "" {
		return score, source
	}
	desc := strings.ToLower(ex.Description)
	if strings.Contains(desc, query) {
		return maxFloat(score, 0.95), "description"
	}
	for _, qw := range queryWords {
		if strings.Contains(desc, qw) {
			score = maxFloat(score, 0.7)
			if source == "" {
				source = "description"
			}
		}
	}
	return score, source
}

// scoreExampleContent scores the file text: the whole query, else the share
// of query words present.
func scoreExampleContent(content, query string, queryWords []string) float64 {
	if strings.Contains(content, query) {
		return 0.8
	}
	if len(queryWords) == 0 {
		return 0
	}
	n := 0
	for _, qw := range queryWords {
		if strings.Contains(content, qw) {
			n++
		}
	}
	return float64(n) / float64(len(queryWords)) * 0.6
}

// exampleFile is one example read for `examples show`.
type exampleFile struct {
	name    string // e.g. adt_option.ail
	rel     string // slash path inside the corpus
	content []byte
}

// fsReadExample finds name (".ail" optional) under runnable/, then at the
// corpus root. fs.FS paths cannot climb out of the corpus.
func fsReadExample(c *examplesCorpus, name string) (exampleFile, error) {
	if !strings.HasSuffix(name, ".ail") {
		name += ".ail"
	}
	for _, rel := range []string{"runnable/" + name, name} {
		if !fs.ValidPath(rel) {
			continue
		}
		if b, err := fs.ReadFile(c.fsys, rel); err == nil {
			return exampleFile{name: name, rel: rel, content: b}, nil
		}
	}
	return exampleFile{name: name}, fmt.Errorf("example not found in %s: %s", c.where(), name)
}
