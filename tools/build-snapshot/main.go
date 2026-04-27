// build-snapshot — populate build/snapshot/ from the current repo state for
// the M-AGENT-MCP MCP server.
//
// Layout produced:
//
//	build/snapshot/
//	  versioned/<ver>/
//	    effects.json
//	    limitations.json
//	    stdlib_summary.json
//	    stdlib/<module>.json
//	    prompts/full.json
//	    prompts/agent.json    (if a versioned agent prompt exists)
//	    prompts/devtools.json (if a versioned devtools prompt exists)
//	    examples_index.json   (if examples_report.json exists)
//	  versioned/latest -> versioned/<ver>
//	  unscoped/
//	    versions_index.json
//	    benchmarks_index.json
//	    benchmarks/by_model/<model>.json
//	    changelog/<ver>.json
//	    design_docs_index.json
//	    roadmap.json
//
// All outputs are stable JSON (sorted keys, deterministic iteration order)
// so the snapshot is byte-identical between runs (modulo built_at).
//
// SCOPE NOTE: design doc M2 specifies SQLite (benchmarks.sqlite + docs.sqlite
// FTS5). We chose JSON for v1 because the AILANG-side mcp_tools/ modules read
// from `std/fs` and have no `std/sqlite` builtin yet. SQLite is a clean v2
// optimization — either via a new builtin or by registering query-shaped tools
// as Go-side MCP handlers alongside the AILANG ones.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	repoRoot := flag.String("repo", ".", "Path to AILANG repo root")
	outDir := flag.String("out", "build/snapshot", "Snapshot output directory")
	flag.Parse()

	root, err := filepath.Abs(*repoRoot)
	must(err)
	out, err := filepath.Abs(*outDir)
	must(err)

	versionBytes, err := os.ReadFile(filepath.Join(root, "std", "VERSION"))
	must(err)
	version := strings.TrimSpace(string(versionBytes))
	version = strings.TrimPrefix(version, "v") // normalize: directory names are unprefixed
	if version == "" {
		fail("std/VERSION is empty")
	}

	fmt.Printf("→ Building snapshot for v%s into %s\n", version, out)

	versionedDir := filepath.Join(out, "versioned", version)
	unscopedDir := filepath.Join(out, "unscoped")
	must(os.MkdirAll(versionedDir, 0o755))
	must(os.MkdirAll(unscopedDir, 0o755))

	if err := buildVersioned(root, versionedDir, version); err != nil {
		fail(fmt.Sprintf("versioned build failed: %v", err))
	}
	if err := buildUnscoped(root, unscopedDir, version); err != nil {
		fail(fmt.Sprintf("unscoped build failed: %v", err))
	}

	// (Re)point the latest symlink at the version we just built.
	latestPath := filepath.Join(out, "versioned", "latest")
	_ = os.Remove(latestPath)
	must(os.Symlink(version, latestPath))

	fmt.Printf("✓ Snapshot built: %s\n", out)
}

// ---------------------------------------------------------------------------
// versioned/<ver>/
// ---------------------------------------------------------------------------

func buildVersioned(root, dir, version string) error {
	if err := writeStdlibSummary(root, dir); err != nil {
		return fmt.Errorf("stdlib_summary: %w", err)
	}
	if err := writeEffects(dir); err != nil {
		return fmt.Errorf("effects: %w", err)
	}
	if err := writeLimitations(root, dir); err != nil {
		return fmt.Errorf("limitations: %w", err)
	}
	if err := writePrompts(root, dir, version); err != nil {
		return fmt.Errorf("prompts: %w", err)
	}
	if err := writeExamplesIndex(root, dir); err != nil {
		return fmt.Errorf("examples: %w", err)
	}
	return nil
}

func writeStdlibSummary(root, dir string) error {
	stdDir := filepath.Join(root, "std")
	entries, err := os.ReadDir(stdDir)
	if err != nil {
		return err
	}
	type mod struct {
		Name          string `json:"name"`
		Summary       string `json:"summary"`
		FunctionCount int    `json:"function_count"`
	}
	var mods []mod
	exportRe := regexp.MustCompile(`(?m)^\s*export\s+(?:pure\s+)?func\s+`)
	commentRe := regexp.MustCompile(`(?m)^--\s*([^\n]+)`)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ail") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(stdDir, e.Name()))
		if err != nil {
			continue
		}
		name := "std/" + strings.TrimSuffix(e.Name(), ".ail")
		summary := ""
		// Pick the first `--` comment that isn't just the filename label
		// (`-- std/ai.ail`) — those are bookkeeping, not summaries.
		for _, m := range commentRe.FindAllStringSubmatch(string(body), -1) {
			line := strings.TrimSpace(m[1])
			if line == "" {
				continue
			}
			if strings.EqualFold(line, e.Name()) || strings.EqualFold(line, name+".ail") {
				continue
			}
			summary = line
			break
		}
		count := len(exportRe.FindAllStringIndex(string(body), -1))
		mods = append(mods, mod{Name: name, Summary: summary, FunctionCount: count})
	}
	sort.Slice(mods, func(i, j int) bool { return mods[i].Name < mods[j].Name })
	return writeJSON(filepath.Join(dir, "stdlib_summary.json"), map[string]any{
		"modules": mods,
	})
}

func writeEffects(dir string) error {
	// Hardcoded for v1; effects rarely change. A future builder could parse
	// internal/effects/registry.go for the authoritative list.
	effects := []map[string]any{
		{"effect": "IO", "capabilities": []string{"IO"}, "since_version": "0.0.1"},
		{"effect": "FS", "capabilities": []string{"FS"}, "since_version": "0.0.1"},
		{"effect": "Net", "capabilities": []string{"Net"}, "since_version": "0.0.1"},
		{"effect": "AI", "capabilities": []string{"AI"}, "since_version": "0.4.0"},
		{"effect": "Env", "capabilities": []string{"Env"}, "since_version": "0.4.0"},
		{"effect": "Clock", "capabilities": []string{"Clock"}, "since_version": "0.4.0"},
		{"effect": "Process", "capabilities": []string{"Process"}, "since_version": "0.5.0"},
		{"effect": "Stream", "capabilities": []string{"Net"}, "since_version": "0.7.0"},
	}
	return writeJSON(filepath.Join(dir, "effects.json"), map[string]any{"effects": effects})
}

func writeLimitations(root, dir string) error {
	path := filepath.Join(root, "docs", "LIMITATIONS.md")
	body, err := os.ReadFile(path)
	if err != nil {
		// Limitations file is optional; emit an empty list rather than failing.
		return writeJSON(filepath.Join(dir, "limitations.json"), map[string]any{"limitations": []any{}})
	}
	// Very lightweight extraction: each `## ` heading becomes a limitation entry
	// with the rest of its section as rationale_md. Good enough for v1.
	lines := strings.Split(string(body), "\n")
	type lim struct {
		Title       string `json:"title"`
		Category    string `json:"category"`
		Status      string `json:"status"`
		RationaleMD string `json:"rationale_md"`
	}
	var lims []lim
	var cur *lim
	var buf strings.Builder
	flush := func() {
		if cur != nil {
			cur.RationaleMD = strings.TrimSpace(buf.String())
			lims = append(lims, *cur)
		}
		buf.Reset()
	}
	h2Re := regexp.MustCompile(`^##\s+(.+)$`)
	for _, line := range lines {
		if m := h2Re.FindStringSubmatch(line); m != nil {
			flush()
			cur = &lim{Title: strings.TrimSpace(m[1]), Category: "general", Status: "by-design"}
			continue
		}
		if cur != nil {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	flush()
	return writeJSON(filepath.Join(dir, "limitations.json"), map[string]any{"limitations": lims})
}

func writePrompts(root, dir, version string) error {
	promptsDir := filepath.Join(dir, "prompts")
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		return err
	}

	// "full" prompt: pick the highest-version v*.md in cmd/ailang/prompts/.
	full, fullVer := pickLatestPrompt(filepath.Join(root, "cmd", "ailang", "prompts"))
	if full != "" {
		body, err := os.ReadFile(full)
		if err == nil {
			if err := writeJSON(filepath.Join(promptsDir, "full.json"), map[string]any{
				"markdown":       string(body),
				"prompt_version": fullVer,
				"served_for":     version,
				"size_bytes":     len(body),
			}); err != nil {
				return err
			}
		}
	}

	// "agent" + "devtools" — same pattern over their respective subdirs.
	for _, kind := range []string{"agent", "devtools"} {
		path, ver := pickLatestPrompt(filepath.Join(root, "cmd", "ailang", "prompts", kind))
		if path == "" {
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		_ = writeJSON(filepath.Join(promptsDir, kind+".json"), map[string]any{
			"markdown":       string(body),
			"prompt_version": ver,
			"served_for":     version,
			"size_bytes":     len(body),
		})
	}
	return nil
}

// pickLatestPrompt returns the path to the highest-version vX.Y.Z(-suffix).md
// in dir, plus the version label. Empty path when none exists.
func pickLatestPrompt(dir string) (string, string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ""
	}
	verRe := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+).*\.md$`)
	type cand struct {
		name string
		key  []int // [major, minor, patch]
	}
	var cands []cand
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := verRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		k := make([]int, 3)
		for i := 0; i < 3; i++ {
			fmt.Sscanf(m[i+1], "%d", &k[i])
		}
		cands = append(cands, cand{e.Name(), k})
	}
	if len(cands) == 0 {
		return "", ""
	}
	sort.Slice(cands, func(i, j int) bool {
		for k := 0; k < 3; k++ {
			if cands[i].key[k] != cands[j].key[k] {
				return cands[i].key[k] > cands[j].key[k]
			}
		}
		return cands[i].name > cands[j].name
	})
	return filepath.Join(dir, cands[0].name), strings.TrimSuffix(cands[0].name, ".md")
}

func writeExamplesIndex(root, dir string) error {
	report := filepath.Join(root, "examples", "examples_report.json")
	body, err := os.ReadFile(report)
	if err != nil {
		return writeJSON(filepath.Join(dir, "examples_index.json"), map[string]any{"examples": []any{}})
	}
	// Pass through the existing report under a wrapping key so the consumer
	// can detect schema later. For M2 v1 we don't reshape the data.
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return writeJSON(filepath.Join(dir, "examples_index.json"), map[string]any{"examples": []any{}})
	}
	return writeJSON(filepath.Join(dir, "examples_index.json"), map[string]any{"report": raw})
}

// ---------------------------------------------------------------------------
// unscoped/
// ---------------------------------------------------------------------------

func buildUnscoped(root, dir, version string) error {
	if err := writeVersionsIndex(root, dir, version); err != nil {
		return fmt.Errorf("versions_index: %w", err)
	}
	if err := writeBenchmarksIndex(root, dir); err != nil {
		return fmt.Errorf("benchmarks_index: %w", err)
	}
	if err := writeDesignDocsIndex(root, dir); err != nil {
		return fmt.Errorf("design_docs_index: %w", err)
	}
	if err := writeChangelog(root, dir); err != nil {
		return fmt.Errorf("changelog: %w", err)
	}
	return nil
}

func writeVersionsIndex(root, dir, current string) error {
	available := []string{current}
	// Also expose any vX.Y.Z prompt-file versions we have around — useful as a
	// hint to clients about what historical content might exist.
	if entries, err := os.ReadDir(filepath.Join(root, "cmd", "ailang", "prompts")); err == nil {
		seen := map[string]bool{current: true}
		verRe := regexp.MustCompile(`^v(\d+\.\d+\.\d+).*\.md$`)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if m := verRe.FindStringSubmatch(e.Name()); m != nil {
				if !seen[m[1]] {
					available = append(available, m[1])
					seen[m[1]] = true
				}
			}
		}
	}
	sort.Slice(available, func(i, j int) bool { return semverLess(available[i], available[j]) })
	return writeJSON(filepath.Join(dir, "versions_index.json"), map[string]any{
		"latest":    current,
		"available": available,
	})
}

// semverLess compares two version strings (e.g. "0.11.4" vs "0.2.0") by their
// numeric segments rather than lexicographically. Non-numeric segments compare
// as strings; missing trailing segments are treated as 0.
func semverLess(a, b string) bool {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		var sx, sy string
		if i < len(pa) {
			fmt.Sscanf(pa[i], "%d%s", &x, &sx)
		}
		if i < len(pb) {
			fmt.Sscanf(pb[i], "%d%s", &y, &sy)
		}
		if x != y {
			return x < y
		}
		if sx != sy {
			return sx < sy
		}
	}
	return false
}

func writeBenchmarksIndex(root, dir string) error {
	baselinesDir := filepath.Join(root, "eval_results", "baselines")
	versions, err := os.ReadDir(baselinesDir)
	if err != nil {
		return writeJSON(filepath.Join(dir, "benchmarks_index.json"), map[string]any{"baselines": []any{}})
	}

	type runRow struct {
		ID            string  `json:"id"`
		Lang          string  `json:"lang"`
		Model         string  `json:"model"`
		AILangVersion string  `json:"ailang_version"`
		CompileOK     bool    `json:"compile_ok"`
		RuntimeOK     bool    `json:"runtime_ok"`
		StdoutOK      bool    `json:"stdout_ok"`
		CostUSD       float64 `json:"cost_usd"`
		InputTokens   int     `json:"input_tokens"`
		OutputTokens  int     `json:"output_tokens"`
		ErrorCategory string  `json:"error_category"`
		DurationMS    int     `json:"duration_ms"`
		Timestamp     string  `json:"timestamp"`
	}
	type baselineSummary struct {
		Version   string   `json:"ailang_version"`
		Models    []string `json:"models"`
		TotalRuns int      `json:"total_runs"`
		Passing   int      `json:"passing_runs"`
		PassRate  float64  `json:"pass_rate"`
	}

	var baselines []baselineSummary
	byModel := map[string][]runRow{}

	for _, ver := range versions {
		if !ver.IsDir() {
			continue
		}
		stdDir := filepath.Join(baselinesDir, ver.Name(), "standard")
		runs, err := os.ReadDir(stdDir)
		if err != nil {
			continue
		}
		s := baselineSummary{Version: ver.Name()}
		modelSet := map[string]bool{}
		for _, runEnt := range runs {
			if runEnt.IsDir() || !strings.HasSuffix(runEnt.Name(), ".json") {
				continue
			}
			body, err := os.ReadFile(filepath.Join(stdDir, runEnt.Name()))
			if err != nil {
				continue
			}
			var r runRow
			if err := json.Unmarshal(body, &r); err != nil {
				continue
			}
			r.AILangVersion = ver.Name()
			s.TotalRuns++
			if r.CompileOK && r.RuntimeOK && r.StdoutOK {
				s.Passing++
			}
			modelSet[r.Model] = true
			byModel[r.Model] = append(byModel[r.Model], r)
		}
		if s.TotalRuns == 0 {
			continue
		}
		for m := range modelSet {
			s.Models = append(s.Models, m)
		}
		sort.Strings(s.Models)
		s.PassRate = float64(s.Passing) / float64(s.TotalRuns)
		baselines = append(baselines, s)
	}
	sort.Slice(baselines, func(i, j int) bool { return baselines[i].Version < baselines[j].Version })

	if err := writeJSON(filepath.Join(dir, "benchmarks_index.json"), map[string]any{
		"baselines": baselines,
	}); err != nil {
		return err
	}

	// Per-model rollup.
	byModelDir := filepath.Join(dir, "benchmarks", "by_model")
	if err := os.MkdirAll(byModelDir, 0o755); err != nil {
		return err
	}
	for model, runs := range byModel {
		passes := []runRow{}
		fails := map[string]int{}
		var totalCost float64
		for _, r := range runs {
			totalCost += r.CostUSD
			if r.CompileOK && r.RuntimeOK && r.StdoutOK {
				passes = append(passes, r)
			} else {
				fails[r.ErrorCategory]++
			}
		}
		safeName := strings.NewReplacer("/", "_", " ", "_").Replace(model)
		_ = writeJSON(filepath.Join(byModelDir, safeName+".json"), map[string]any{
			"model":             model,
			"total_runs":        len(runs),
			"passes":            len(passes),
			"total_cost_usd":    totalCost,
			"fails_by_category": fails,
		})
	}

	return nil
}

func writeDesignDocsIndex(root, dir string) error {
	type entry struct {
		Slug    string `json:"slug"`
		Title   string `json:"title"`
		State   string `json:"state"`
		Path    string `json:"path"`
		Version string `json:"version"`
	}
	var docs []entry
	titleRe := regexp.MustCompile(`(?m)^#\s+(.+)$`)

	for _, state := range []string{"planned", "implemented", "rejected"} {
		base := filepath.Join(root, "design_docs", state)
		_ = filepath.Walk(base, func(p string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(p, ".md") {
				return nil
			}
			body, _ := os.ReadFile(p)
			title := strings.TrimSuffix(filepath.Base(p), ".md")
			if m := titleRe.FindStringSubmatch(string(body)); len(m) > 1 {
				title = strings.TrimSpace(m[1])
			}
			rel, _ := filepath.Rel(filepath.Join(root, "design_docs"), p)
			parts := strings.Split(filepath.Dir(rel), string(filepath.Separator))
			version := ""
			if len(parts) > 1 {
				version = parts[1]
			}
			docs = append(docs, entry{
				Slug:    strings.TrimSuffix(filepath.Base(p), ".md"),
				Title:   title,
				State:   state,
				Path:    "design_docs/" + rel,
				Version: version,
			})
			return nil
		})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	return writeJSON(filepath.Join(dir, "design_docs_index.json"), map[string]any{"docs": docs})
}

func writeChangelog(root, dir string) error {
	changelogDir := filepath.Join(dir, "changelog")
	if err := os.MkdirAll(changelogDir, 0o755); err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(root, "changelogs", "*.md"))
	if err != nil {
		return err
	}
	// Match `## [v0.14.2] - 2026-04-27` style headings. The version capture must
	// look like a real version number (digits.digits[.digits][...]) — without
	// this anchor we false-match `## Usage`, `## Notes` etc. as version sections.
	verHeading := regexp.MustCompile(`(?m)^##\s+\[?v?(\d+\.\d+(?:\.\d+)?[\w.\-]*)\]?\s+-?\s*(.*)$`)
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		text := string(body)
		matches := verHeading.FindAllStringSubmatchIndex(text, -1)
		for i, m := range matches {
			start := m[0]
			end := len(text)
			if i+1 < len(matches) {
				end = matches[i+1][0]
			}
			version := text[m[2]:m[3]]
			section := text[start:end]
			_ = writeJSON(filepath.Join(changelogDir, version+".json"), map[string]any{
				"version":  version,
				"markdown": section,
				"source":   filepath.Base(f),
			})
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func writeJSON(path string, payload any) error {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

func must(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(msg string) {
	fmt.Fprintf(os.Stderr, "build-snapshot: %s\n", msg)
	os.Exit(1)
}
