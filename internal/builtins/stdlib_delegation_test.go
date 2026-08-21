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
// call site in std/*.ail, WHY that is acceptable. Runtime and codegen registries
// describe different execution paths: a codegen helper is not callable by the
// interpreter. Categories are therefore machine-checked against AllNames(), the
// runtime registry, rather than trusted as prose.
//
// In particular, 11 entries previously described as "delegation candidates" are
// blocked on a MISSING INTERPRETER IMPLEMENTATION, not on their std/list forms
// being recursive. Their codegen helpers run only in generated Go programs.
//
// _list_reverse is deliberately ABSENT: std/list.reverse delegates to it, and that
// absence is the fixture proving this gate is live.
type listDelegationCategory uint8

const (
	DelegableNow listDelegationCategory = iota
	NoRuntimeImpl
	NotNeeded
)

type listDelegationExemption struct {
	category      listDelegationCategory
	runtimeBacked bool
	reason        string
}

var listDelegationExemptions = map[string]listDelegationExemption{
	"_list_any":       {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.any can delegate"},
	"_list_contains":  {NotNeeded, true, "std/list exposes membership as member, which delegates to _list_member; contains has no std/list counterpart to serve"},
	"_list_extract":   {NotNeeded, true, "std/list exposes no extract operation to delegate"},
	"_list_filterE":   {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.filterE can delegate"},
	"_list_findIndex": {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.findIndex can delegate"},
	"_list_flatMap":   {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.flatMap can delegate"},
	"_list_flatMapE":  {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.flatMapE can delegate"},
	"_list_foldlE":    {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.foldlE can delegate"},
	"_list_foldr":     {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.foldr can delegate"},
	"_list_forEachE":  {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.forEachE can delegate"},
	"_list_head":      {NotNeeded, true, "std/list.head is already O(1) through list pattern matching"},
	"_list_last":      {NotNeeded, false, "codegen-only helper is unnecessary in the interpreter because std/list.last composes _list_length and _list_nth"},
	"_list_mapE":      {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.mapE can delegate"},
	"_list_sortBy":    {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.sortBy can delegate"},
	"_list_tail":      {NotNeeded, false, "codegen-only helper is unnecessary in the interpreter because std/list.tail is O(1) pattern matching"},
	"_list_take":      {DelegableNow, true, "runtime-backed, but delegation is deferred because the builtin writes a materialization note to stderr"},
	"_list_zip":       {NoRuntimeImpl, false, "codegen-only: the interpreter has no implementation to which std/list.zip can delegate"},
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
	runtimeNames := map[string]struct{}{}
	runtimeListCount := 0
	for _, name := range AllNames() {
		if strings.HasPrefix(name, "_list_") {
			runtimeNames[name] = struct{}{}
			runtimeListCount++
		}
	}
	if runtimeListCount < 15 {
		t.Fatalf("instrument failure: found only %d runtime _list_* builtins; want at least 15", runtimeListCount)
	}

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

	categorized := make(map[string]struct{}, len(listDelegationExemptions))
	for name, exemption := range listDelegationExemptions {
		if exemption.reason == "" {
			t.Errorf("exemption %s has an empty reason", name)
		}
		_, hasRuntimeImpl := runtimeNames[name]
		switch exemption.category {
		case NoRuntimeImpl:
			categorized[name] = struct{}{}
			if hasRuntimeImpl {
				t.Errorf("%s is categorized NoRuntimeImpl but is present in the runtime registry", name)
			}
		case DelegableNow:
			categorized[name] = struct{}{}
			if !exemption.runtimeBacked {
				t.Errorf("%s is DelegableNow but does not claim a runtime implementation", name)
			}
			if !hasRuntimeImpl {
				t.Errorf("%s is categorized DelegableNow but is absent from the runtime registry", name)
			}
		case NotNeeded:
			categorized[name] = struct{}{}
			if exemption.runtimeBacked && !hasRuntimeImpl {
				t.Errorf("%s claims a runtime implementation but is absent from the runtime registry", name)
			}
		default:
			t.Errorf("%s has unknown delegation category %d", name, exemption.category)
		}
	}
	if len(categorized) != len(listDelegationExemptions) {
		t.Fatalf("instrument failure: %d categorized names != %d exempted names", len(categorized), len(listDelegationExemptions))
	}
	for name := range listDelegationExemptions {
		if _, ok := categorized[name]; !ok {
			t.Errorf("exempted name %s is not categorized", name)
		}
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
		if _, exempted := listDelegationExemptions[name]; counts[name] == 0 && !exempted {
			unexplained = append(unexplained, name)
		}
	}
	sort.Strings(unexplained)
	for _, name := range unexplained {
		t.Errorf("%s has zero exact call sites in std/*.ail and no delegation exemption "+
			"(delegate it from std/, or add a categorized reason)", name)
	}

	t.Logf("scan summary: %d registered _list_* builtins, %d exact std/ call sites, %d exemptions",
		len(names), total, len(listDelegationExemptions))
}
