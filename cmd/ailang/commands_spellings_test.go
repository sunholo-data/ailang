package main

import (
	"bufio"
	"os"
	"sort"
	"strings"
	"testing"
)

// D1 — every spelling that worked before S5 M2 still works.
//
// This is the constraint that makes M2 more than a rename. The fleet calls
// these names from launchd drivers, make targets, skills and workflows:
// measured over tracked files at HEAD, `messages` 593 references,
// `coordinator` 332, `chains` 216, `eval-suite` 182, `trace` 66,
// `observatory` 63. Not one caller may have to change, so the groups ADD a
// route and never move one.
//
// The list is NOT hand-typed. It was read out of the pre-M2 binary:
//
//	go build -ldflags "-X .../version.Version=vSNAP -X .../version.Commit=SNAPSHOT \
//	  -X .../version.BuildTime=SNAPSHOT" -o bin/ailang-before ./cmd/ailang
//	tools/cli_surface_snapshot.sh --spellings bin/ailang-before \
//	  cmd/ailang/testdata/pre_s5_spellings.txt
//
// That matters because the spellings a person forgets are exactly the ones
// that break something: `msg`, `brain`, `microrag`, `urag`, `serve`,
// `serve-api`, `internal-dump-iface`, and the seven bare pkg verbs `add lock
// tree install search publish unpublish`. A hand-written list omits them; the
// binary cannot. (Control: the 83 top-level routes parsed out of the binary's
// help were diffed against the COMMANDS list in tools/cli_surface_snapshot.sh,
// which M1 measured independently against main.go's pre-S5 switch. Zero diff.)
const preS5SpellingsFixture = "testdata/pre_s5_spellings.txt"

func loadPreS5Spellings(t *testing.T) (topLevel []string, pkgVerbs []string) {
	t.Helper()
	f, err := os.Open(preS5SpellingsFixture)
	if err != nil {
		t.Fatalf("the D1 fixture is missing: %v\nRegenerate it with "+
			"tools/cli_surface_snapshot.sh --spellings bin/ailang-before %s", err, preS5SpellingsFixture)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if verb, ok := strings.CutPrefix(line, "pkg "); ok {
			pkgVerbs = append(pkgVerbs, verb)
			continue
		}
		topLevel = append(topLevel, line)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("reading %s: %v", preS5SpellingsFixture, err)
	}

	// An empty fixture would make every assertion below vacuously true. The
	// counts are the ones the generator produced; they change only when a
	// route is deliberately added or removed, and then this line is the place
	// that says so out loud.
	if len(topLevel) != 83 {
		t.Fatalf("fixture holds %d top-level spellings, want 83 — regenerate it "+
			"from bin/ailang-before, or state the intended change here", len(topLevel))
	}
	if len(pkgVerbs) != 10 {
		t.Fatalf("fixture holds %d pkg verbs, want 10", len(pkgVerbs))
	}
	return topLevel, pkgVerbs
}

// TestSpellings_EveryPreS5TopLevelRouteStillResolves is the D1 gate.
//
// Mutation-tested: drop `msg` from the messages row's Aliases, or delete the
// pkgLegacyCommands() call from init(), and this fails naming the exact
// spelling that no longer resolves.
func TestSpellings_EveryPreS5TopLevelRouteStillResolves(t *testing.T) {
	topLevel, _ := loadPreS5Spellings(t)
	for _, name := range topLevel {
		if _, ok := lookupCommand(name); !ok {
			t.Errorf("`ailang %s` no longer resolves — D1 says every pre-S5 spelling keeps working", name)
		}
	}
}

// TestSpellings_EveryPreS5PkgVerbStillResolves does the same one level down.
func TestSpellings_EveryPreS5PkgVerbStillResolves(t *testing.T) {
	_, pkgVerbs := loadPreS5Spellings(t)
	have := map[string]bool{}
	for _, sub := range pkgSubcommands() {
		have[sub.Name] = true
	}
	for _, verb := range pkgVerbs {
		if !have[verb] {
			t.Errorf("`ailang pkg %s` no longer resolves", verb)
		}
	}
}

// TestSpellings_TheOnesAHumanForgets names them explicitly.
//
// The fixture already covers these, so this test is redundant by construction
// — deliberately. It is the readable record of WHICH routes the sprint plan
// called out, so a future reader who breaks one sees the name rather than an
// index into a generated file. If the fixture is ever regenerated from a
// binary that has already lost one of these, this test still fails.
func TestSpellings_TheOnesAHumanForgets(t *testing.T) {
	for _, name := range []string{
		"msg", "brain", "microrag", "urag", "serve", "server", "serve-api",
		"internal-dump-iface",
		"add", "lock", "tree", "install", "search", "publish", "unpublish",
		"ai-check", "pkg-docs",
	} {
		if _, ok := lookupCommand(name); !ok {
			t.Errorf("`ailang %s` no longer resolves", name)
		}
	}
}

// TestSpellings_GroupedCommandsKeepTheirTopLevelRoute is the property the
// fixture test measures, stated as a rule rather than a list: filing a command
// in a group must never take its own name away. Any command added to dev, ops
// or eval in future is covered without touching the fixture.
func TestSpellings_GroupedCommandsKeepTheirTopLevelRoute(t *testing.T) {
	for i := range allCommands {
		c := &allCommands[i]
		if c.Group == "" {
			continue
		}
		got, ok := lookupCommand(c.Name)
		if !ok {
			t.Errorf("%s is filed in group %q and lost its top-level route", c.Name, c.Group)
			continue
		}
		if got.Name != c.Name {
			t.Errorf("top-level %q resolves to %q", c.Name, got.Name)
		}
	}
}

// TestSpellings_EveryGroupMemberIsReachableThroughItsGroup is the other half:
// the group route exists for every member, under its derived subcommand name.
func TestSpellings_EveryGroupMemberIsReachableThroughItsGroup(t *testing.T) {
	for _, group := range dispatchGroups {
		members := groupMembers(group)
		if len(members) == 0 {
			t.Errorf("group %q has no members — the group help would be empty", group)
		}
		seen := map[string]string{}
		for _, c := range members {
			sub := groupSubName(group, c.Name)
			if prev, dup := seen[sub]; dup {
				t.Errorf("group %q: %q and %q both answer to `ailang %s %s`", group, prev, c.Name, group, sub)
			}
			seen[sub] = c.Name
			if got := lookupGroupMember(group, sub); got == nil || got.Name != c.Name {
				t.Errorf("`ailang %s %s` does not resolve to %q", group, sub, c.Name)
			}
		}
	}
}

// TestSpellings_PkgLegacyRowsShareTheVerbsDefinition guards the one place two
// routes could drift apart: `ailang add` and `ailang pkg add`. They are
// generated from the SAME row, so the test asserts the mapping is complete and
// consistent in both directions rather than comparing two hand-written lists.
func TestSpellings_PkgLegacyRowsShareTheVerbsDefinition(t *testing.T) {
	verbs := map[string]bool{}
	for _, sub := range pkgSubcommands() {
		verbs[sub.Name] = true
	}
	for verb, top := range pkgLegacyTopLevel {
		if !verbs[verb] {
			t.Errorf("pkgLegacyTopLevel names pkg verb %q, which does not exist", verb)
		}
		c, ok := lookupCommand(top)
		if !ok {
			t.Errorf("legacy top-level route %q (for `pkg %s`) does not resolve", top, verb)
			continue
		}
		if !c.Hidden {
			t.Errorf("%q is a legacy duplicate of `pkg %s` and must be hidden from help", top, verb)
		}
	}
}

// TestSpellings_NoRouteIsLostToADuplicate — commandIndex panics on a duplicate
// route at startup, which is the real guard; this states the invariant so the
// reason is written down where the table is read.
func TestSpellings_NoRouteIsLostToADuplicate(t *testing.T) {
	var routes []string
	for i := range allCommands {
		c := &allCommands[i]
		routes = append(routes, c.Name)
		routes = append(routes, c.Aliases...)
	}
	sort.Strings(routes)
	for i := 1; i < len(routes); i++ {
		if routes[i] == routes[i-1] {
			t.Errorf("route %q is declared twice", routes[i])
		}
	}
}
