package iface

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("no caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// stdModulesOnDisk lists every importable std module: std/*.ail plus the two
// subdirectory modules.
func stdModulesOnDisk(t *testing.T) []string {
	t.Helper()
	root := repoRoot(t)
	var mods []string
	for _, pattern := range []string{"std/*.ail", "std/*/*.ail"} {
		files, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			rel, _ := filepath.Rel(root, f)
			mods = append(mods, strings.TrimSuffix(filepath.ToSlash(rel), ".ail"))
		}
	}
	sort.Strings(mods)
	return mods
}

func TestEveryStdModuleHasATier(t *testing.T) {
	tiers, err := StdModuleTiers()
	if err != nil {
		t.Fatal(err)
	}
	onDisk := map[string]bool{}
	for _, m := range stdModulesOnDisk(t) {
		onDisk[m] = true
		if tiers[m] == "" {
			t.Errorf("%s has no tier in std/STABILITY.toml", m)
		}
	}
	for m := range tiers {
		if !onDisk[m] {
			t.Errorf("std/STABILITY.toml lists %s, which does not exist", m)
		}
	}
	if len(onDisk) != 51 {
		t.Errorf("%d std modules; stability.md promises 51 (17 + 31 + 3) — update both", len(onDisk))
	}
}

func TestModuleTierOutsideStdIsEmpty(t *testing.T) {
	for _, m := range []string{"app", "examples/foo", "sunholo/pkg/mod", "stdx/list"} {
		if tier, err := ModuleTier(m); err != nil || tier != "" {
			t.Errorf("ModuleTier(%q) = %q, %v; want no tier", m, tier, err)
		}
	}
	if tier, _ := ModuleTier("std/list"); tier != TierStable {
		t.Errorf("std/list tier = %q", tier)
	}
	if tier, _ := ModuleTier("std/ai/streaming"); tier != TierExperimental {
		t.Errorf("std/ai/streaming tier = %q", tier)
	}
}

func TestParseStabilityManifestRejectsBadInput(t *testing.T) {
	for name, text := range map[string]string{
		"unknown tier": "[modules]\n\"std/x\" = \"beta\"\n",
		"unknown key":  "[modules]\n\"std/x\" = \"stable\"\n[exceptions]\nfoo = \"bar\"\n",
		"malformed":    "[modules\n",
	} {
		if _, err := parseStabilityManifest(text); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// stabilityDocTables reads stability.md's three stdlib module tables:
// heading tier -> modules listed.
func stabilityDocTables(t *testing.T) (map[string][]string, map[string]int, string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "docs", "reference", "stability.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	heading := regexp.MustCompile(`(?m)^### (Stable|Experimental|Internal) stdlib \((\d+)\)$`)
	row := regexp.MustCompile("(?m)^\\| `(std/[a-z_/]+)` \\|")
	out := map[string][]string{}
	counts := map[string]int{}
	locs := heading.FindAllStringSubmatchIndex(doc, -1)
	if len(locs) != 3 {
		t.Fatalf("found %d stdlib tier headings, want 3", len(locs))
	}
	for i, loc := range locs {
		tier := strings.ToLower(doc[loc[2]:loc[3]])
		n := 0
		for _, c := range doc[loc[4]:loc[5]] {
			n = n*10 + int(c-'0')
		}
		counts[tier] = n
		end := len(doc)
		if next := strings.Index(doc[loc[1]:], "\n### "); next >= 0 {
			end = loc[1] + next
		}
		if i+1 < len(locs) && locs[i+1][0] < end {
			end = locs[i+1][0]
		}
		for _, m := range row.FindAllStringSubmatch(doc[loc[1]:end], -1) {
			out[tier] = append(out[tier], m[1])
		}
	}
	return out, counts, doc
}

// stability.md's module tables and std/STABILITY.toml agree, module by module.
func TestStabilityDocMatchesManifest(t *testing.T) {
	tiers, err := StdModuleTiers()
	if err != nil {
		t.Fatal(err)
	}
	tables, counts, _ := stabilityDocTables(t)
	listed := map[string]string{}
	for tier, mods := range tables {
		if len(mods) != counts[tier] {
			t.Errorf("stability.md: the %s heading says %d modules, the table lists %d", tier, counts[tier], len(mods))
		}
		for _, m := range mods {
			if prev, dup := listed[m]; dup {
				t.Errorf("stability.md lists %s twice (%s and %s)", m, prev, tier)
			}
			listed[m] = tier
			if tiers[m] != tier {
				t.Errorf("%s: stability.md says %s, std/STABILITY.toml says %q", m, tier, tiers[m])
			}
		}
	}
	for m, tier := range tiers {
		if listed[m] == "" {
			t.Errorf("%s (%s in std/STABILITY.toml) is missing from stability.md", m, tier)
		}
	}
}

// experimentalInSource parses std/*.ail and returns module -> @experimental
// function names, for modules whose default tier is not already experimental.
func experimentalInSource(t *testing.T) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, m := range stdModulesOnDisk(t) {
		src, err := os.ReadFile(filepath.Join(repoRoot(t), m+".ail"))
		if err != nil {
			t.Fatal(err)
		}
		p := parser.New(lexer.New(string(src), m+".ail"))
		prog := p.Parse()
		if errs := p.Errors(); len(errs) > 0 {
			t.Fatalf("%s: %v", m, errs[0])
		}
		modTier, _ := ModuleTier(m)
		for _, fd := range prog.File.Funcs {
			if fd.IsExport && FuncTier(modTier, fd) == TierExperimental && modTier != TierExperimental {
				out[m] = append(out[m], fd.Name)
			}
		}
		sort.Strings(out[m])
	}
	return out
}

// The "Experimental functions in Stable modules" table lists exactly the
// @experimental functions in the source, and so (TestDocsTiersMatchGoldens in
// cmd/ailang) exactly the goldens' experimental entries.
func TestStabilityDocExperimentalFunctionsMatchSource(t *testing.T) {
	_, _, doc := stabilityDocTables(t)
	start := strings.Index(doc, "### Experimental functions in Stable modules (")
	if start < 0 {
		t.Fatal("stability.md has no 'Experimental functions in Stable modules' section")
	}
	section := doc[start:]
	if end := strings.Index(section[4:], "\n### "); end >= 0 {
		section = section[:end+4]
	}
	row := regexp.MustCompile("(?m)^\\| `(std/[a-z_/]+)` \\| ([^|]+) \\|")
	fn := regexp.MustCompile("`([A-Za-z0-9_]+)`")
	doc2 := map[string][]string{}
	total := 0
	for _, m := range row.FindAllStringSubmatch(section, -1) {
		for _, f := range fn.FindAllStringSubmatch(m[2], -1) {
			doc2[m[1]] = append(doc2[m[1]], f[1])
			total++
		}
		sort.Strings(doc2[m[1]])
	}
	src := experimentalInSource(t)
	srcTotal := 0
	for m, fs := range src {
		srcTotal += len(fs)
		if strings.Join(fs, ",") != strings.Join(doc2[m], ",") {
			t.Errorf("%s: source @experimental %v, stability.md %v", m, fs, doc2[m])
		}
	}
	for m := range doc2 {
		if len(src[m]) == 0 {
			t.Errorf("stability.md lists experimental functions in %s; the source has none", m)
		}
	}
	if !strings.Contains(section, "("+itoa(srcTotal)+")") || total != srcTotal || srcTotal != 25 {
		t.Errorf("experimental function count: heading/table/source/design = ?/%d/%d/25", total, srcTotal)
	}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

// The deprecated-names table names every rename and its replacement.
func TestStabilityDocListsEveryDeprecation(t *testing.T) {
	_, _, doc := stabilityDocTables(t)
	start := strings.Index(doc, "### Deprecated stdlib names")
	if start < 0 {
		t.Fatal("stability.md has no 'Deprecated stdlib names' section")
	}
	section := doc[start:]
	if end := strings.Index(section[4:], "\n### "); end >= 0 {
		section = section[:end+4]
	}
	for _, r := range importhint.StdlibRenames {
		found := false
		for _, line := range strings.Split(section, "\n") {
			if strings.HasPrefix(line, "| `"+r.Module+"` |") && strings.Contains(line, "`"+r.Old) && strings.Contains(line, "`"+r.New) {
				found = true
			}
		}
		if !found {
			t.Errorf("stability.md's deprecated table has no row for %s.%s -> %s", r.Module, r.Old, r.New)
		}
	}
}

// Marking a function @experimental or @deprecated, or re-tiering its module,
// leaves interface_hash unchanged (§6.1 item 4).
func TestHashProjectionIgnoresTierAndDeprecation(t *testing.T) {
	plain := &InterfaceJSON{
		Module: "lib", Schema: "ailang.iface/v1",
		Types: []TypeJSON{{Name: "T", Ctors: []string{"A"}}},
		Funcs: []FuncJSON{{Name: "f", Type: "(int)->int", Effects: []string{}}},
	}
	marked := &InterfaceJSON{
		Module: "lib", Schema: "ailang.iface/v1",
		Types: []TypeJSON{{Name: "T", Ctors: []string{"A"}, Tier: TierExperimental}},
		Funcs: []FuncJSON{{Name: "f", Type: "(int)->int", Effects: []string{}, Tier: TierStable, ReplacedBy: "g", DeprecationNote: "note"}},
	}
	a, err := HashProjection(plain)
	if err != nil {
		t.Fatal(err)
	}
	b, err := HashProjection(marked)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Errorf("marking changed the hash projection:\n%s\n%s", a, b)
	}
	// Control: a real change still moves it.
	marked.Funcs[0].Type = "(int)->string"
	c, _ := HashProjection(marked)
	if string(c) == string(a) {
		t.Error("control: a type change must move the projection")
	}
}
