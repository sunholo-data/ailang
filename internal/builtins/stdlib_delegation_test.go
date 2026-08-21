package builtins

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// stdlibSourceDir is resolved relative to this package, as
// internal/cihygiene/workflow_timeouts_test.go:46 does for workflow files.
const stdlibSourceDir = "../../std"

// listDelegationExemptions records, for every registered _list_* builtin with no
// call site in std/*.ail, WHY that is acceptable. Two categories only:
//
//	DELEGATION-CANDIDATE — a std/list export shadows this builtin with a recursive
//	                       AILANG implementation that is asymptotically worse.
//	                       Delegating each one needs its own semantic-equivalence
//	                       verification; tracked as m-stdlib-list-delegation-sweep.
//	NOT-NEEDED           — there is nothing to delegate, or the std/list form is
//	                       already as cheap as the builtin.
//
// _list_reverse is deliberately ABSENT: std/list.reverse delegates to it, and that
// absence is the fixture proving this gate is live.
var listDelegationExemptions = map[string]string{
	"_list_any":       "DELEGATION-CANDIDATE: std/list.any is recursive; see m-stdlib-list-delegation-sweep",
	"_list_contains":  "NOT-NEEDED: std/list exposes membership as member, which delegates to _list_member; both builtins use the same structural valuesEqual, so contains has no std/list counterpart to serve",
	"_list_drop":      "DELEGATION-CANDIDATE: std/list.drop is recursive; see m-stdlib-list-delegation-sweep",
	"_list_extract":   "NOT-NEEDED: std/list exposes no extract operation to delegate",
	"_list_filterE":   "DELEGATION-CANDIDATE: std/list.filterE is recursive; see m-stdlib-list-delegation-sweep",
	"_list_findIndex": "DELEGATION-CANDIDATE: std/list.findIndex recurses through findIndexHelper; see m-stdlib-list-delegation-sweep",
	"_list_flatMap":   "DELEGATION-CANDIDATE: std/list.flatMap is recursive and concatenates per element; see m-stdlib-list-delegation-sweep",
	"_list_flatMapE":  "DELEGATION-CANDIDATE: std/list.flatMapE is recursive and concatenates per element; see m-stdlib-list-delegation-sweep",
	"_list_foldlE":    "DELEGATION-CANDIDATE: std/list.foldlE is recursive; see m-stdlib-list-delegation-sweep",
	"_list_foldr":     "DELEGATION-CANDIDATE: std/list.foldr is recursive; see m-stdlib-list-delegation-sweep",
	"_list_forEachE":  "DELEGATION-CANDIDATE: std/list.forEachE is recursive; see m-stdlib-list-delegation-sweep",
	"_list_head":      "NOT-NEEDED: std/list.head is already O(1) through list pattern matching",
	"_list_last":      "NOT-NEEDED: std/list.last already composes the delegated _list_length and _list_nth",
	"_list_mapE":      "DELEGATION-CANDIDATE: std/list.mapE is recursive and appends per element; see m-stdlib-list-delegation-sweep",
	"_list_sortBy":    "DELEGATION-CANDIDATE: std/list.sortBy is recursive and builds on the recursive take/drop/mergeBy; see m-stdlib-list-delegation-sweep",
	"_list_tail":      "NOT-NEEDED: std/list.tail is already O(1) through list pattern matching",
	"_list_take":      "DELEGATION-CANDIDATE: std/list.take is recursive and appends per element; see m-stdlib-list-delegation-sweep",
	"_list_zip":       "DELEGATION-CANDIDATE: std/list.zip is recursive and appends per element; see m-stdlib-list-delegation-sweep",
}

// registeredListBuiltins reads the LIVE registries rather than parsing source.
//
// An earlier form of this gate AST-parsed internal/builtins/*.go for `Name:` string
// literals. That was measurably incomplete: a builtin registered with
// `Name: someConstant` (an *ast.Ident, not a *ast.BasicLit) was invisible, so it
// needed neither a call site nor an exemption and the gate passed. The two live
// registries are complete by construction — nothing can be a builtin without
// registering — but NEITHER IS COMPLETE ALONE: measured on this tree, specRegistry
// (AllNames) holds 18 _list_* names and the metadata Registry (GetBuiltinNames)
// holds 26, disjointly enough that only their UNION reaches all 31.
func registeredListBuiltins() []string {
	seen := map[string]struct{}{}
	for _, src := range [][]string{AllNames(), GetBuiltinNames()} {
		for _, name := range src {
			if strings.HasPrefix(name, "_list_") {
				seen[name] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ailLineComment strips `-- …` to end of line. The call-site scan must not count a
// builtin named inside a comment: with a raw whole-file regex, reverting a
// delegation while leaving a comment that mentions `_list_reverse(` kept this gate
// green over a genuinely reverted function.
var ailLineComment = regexp.MustCompile(`--[^\n]*`)

// stdlibListCallSites counts calls of each name in std/*.ail. Matching is on
// `name` followed by optional whitespace and `(` — substring matching is wrong
// here, because `_list_take` is a prefix of `_list_takeMap` and `_list_takeFlatMap`.
func stdlibListCallSites(t *testing.T, names []string) map[string]int {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(stdlibSourceDir, "*.ail"))
	if err != nil {
		t.Fatalf("instrument failure: enumerate stdlib sources: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("instrument failure: no std/*.ail files found")
	}

	patterns := make(map[string]*regexp.Regexp, len(names))
	for _, name := range names {
		patterns[name] = regexp.MustCompile(regexp.QuoteMeta(name) + `\s*\(`)
	}

	counts := make(map[string]int, len(names))
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		code := ailLineComment.ReplaceAll(body, nil)
		for name, pattern := range patterns {
			counts[name] += len(pattern.FindAll(code, -1))
		}
	}
	return counts
}

func TestEveryListBuiltinIsDelegatedOrExplained(t *testing.T) {
	names := registeredListBuiltins()

	// Anti-vacuity floor: an enumeration that finds (almost) nothing must fail
	// loudly, never pass. "no builtins found" and "every builtin delegates"
	// otherwise share an exit code.
	if len(names) < 25 {
		t.Fatalf("instrument failure: found only %d registered _list_* builtins; want at least 25", len(names))
	}

	counts := stdlibListCallSites(t, names)
	total := 0
	for _, count := range counts {
		total += count
	}
	if total == 0 {
		t.Fatal("instrument failure: std/ call-site scan found zero _list_* calls")
	}

	// A known-positive control: _list_map is delegated by std/list.map, so a scan
	// that cannot see it is broken rather than informative.
	if counts["_list_map"] == 0 {
		t.Fatal("instrument failure: known-positive control _list_map has zero call sites")
	}

	for name := range listDelegationExemptions {
		if counts[name] > 0 {
			t.Errorf("stale exemption %s: it now has %d call site(s) in std/*.ail — delete the exemption", name, counts[name])
		}
	}
	registered := map[string]struct{}{}
	for _, name := range names {
		registered[name] = struct{}{}
	}
	for name := range listDelegationExemptions {
		if _, ok := registered[name]; !ok {
			t.Errorf("stale exemption %s: name is not a registered _list_* builtin", name)
		}
	}

	var unexplained []string
	for _, name := range names {
		if counts[name] == 0 && listDelegationExemptions[name] == "" {
			unexplained = append(unexplained, name)
		}
	}
	sort.Strings(unexplained)
	for _, name := range unexplained {
		t.Errorf("%s has zero exact call sites in std/*.ail and no delegation exemption "+
			"(delegate it from std/, or add a DELEGATION-CANDIDATE / NOT-NEEDED reason)", name)
	}

	t.Logf("scan summary: %d registered _list_* builtins, %d exact std/ call sites, %d exemptions",
		len(names), total, len(listDelegationExemptions))
}
