package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The measured bug: --remote was accepted and ignored, so the command answered
// about the wrong plane and said nothing.
func TestRejectUnknownCoordinatorFlags_RemoteOnALocalOnlySubcommand(t *testing.T) {
	err := rejectUnknownCoordinatorFlags("diff", []string{"task-abc", "--remote", "gcp"})
	if err == nil {
		t.Fatal("`coordinator diff --remote gcp` must be refused, not silently answered from local SQLite")
	}
	msg := err.Error()
	for _, want := range []string{"--remote", "approve", "approvals"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the refusal must name %q so the operator knows where the flag IS real: %q", want, msg)
		}
	}
}

// list IS remote-aware, so the flag must pass through untouched.
func TestRejectUnknownCoordinatorFlags_RemoteOnListIsAccepted(t *testing.T) {
	if err := rejectUnknownCoordinatorFlags("list", []string{"--remote", "gcp", "--limit", "5"}); err != nil {
		t.Errorf("list understands --remote: %v", err)
	}
}

func TestRejectUnknownCoordinatorFlags_KnownFlagsPass(t *testing.T) {
	cases := map[string][]string{
		"list":     {"--json", "--limit", "20", "--running"},
		"pending":  {"--json"},
		"diff":     {"task-abc", "--stat"},
		"logs":     {"task-abc", "--follow"},
		"worktree": {"task-abc", "--open"},
		"retry":    {"--all", "--yes"},
		"cleanup":  {"--older-than", "2h", "--yes"},
		"status":   {"--json"},
	}
	for sub, args := range cases {
		if err := rejectUnknownCoordinatorFlags(sub, args); err != nil {
			t.Errorf("%s %v: %v", sub, args, err)
		}
	}
}

// A flag's VALUE must never be mistaken for a flag. `--status pending` and
// `--older-than 2h` are fine, but a value that begins with a dash would be
// read as an unknown flag if the guard did not consume it.
func TestRejectUnknownCoordinatorFlags_ValuesAreNotParsedAsFlags(t *testing.T) {
	if err := rejectUnknownCoordinatorFlags("cleanup", []string{"--older-than", "--weird"}); err != nil {
		t.Errorf("a value following a value-taking flag is consumed, not judged: %v", err)
	}
	if err := rejectUnknownCoordinatorFlags("list", []string{"--limit=5"}); err != nil {
		t.Errorf("--flag=value form must be understood: %v", err)
	}
}

func TestRejectUnknownCoordinatorFlags_TypoNamesTheKnownFlags(t *testing.T) {
	err := rejectUnknownCoordinatorFlags("list", []string{"--jsonn"})
	if err == nil {
		t.Fatal("a typo'd flag must be refused")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Errorf("the refusal must list what IS understood: %q", err.Error())
	}
}

// flag.FlagSet subcommands do their own refusing; double-guarding would mean
// two vocabularies for one command, which is how they drift apart.
func TestRejectUnknownCoordinatorFlags_FlagSetSubcommandsAreNotGuardedHere(t *testing.T) {
	if err := rejectUnknownCoordinatorFlags("agent-check", []string{"--anything"}); err != nil {
		t.Errorf("FlagSet subcommands must pass through to their own parser: %v", err)
	}
}

// The ratchet. Every subcommand in the dispatch switch must be declared either
// hand-parsed (and so guarded here) or FlagSet-based (and so guarded there). A
// new subcommand that is neither would silently swallow unknown flags, which is
// precisely the defect this file exists to end.
func TestCoordinatorFlagGuardIsExhaustive(t *testing.T) {
	src, err := os.ReadFile("coordinator.go")
	if err != nil {
		t.Fatalf("read coordinator.go: %v", err)
	}
	body := string(src)
	start := strings.Index(body, "switch subcommand {")
	end := strings.Index(body[start:], "\n\tdefault:")
	if start < 0 || end < 0 {
		t.Fatal("could not locate the dispatch switch — this test must be updated with it")
	}
	re := regexp.MustCompile(`case ([^:]+):`)
	var missing []string
	for _, m := range re.FindAllStringSubmatch(body[start:start+end], -1) {
		for _, raw := range strings.Split(m[1], ",") {
			name := strings.Trim(strings.TrimSpace(raw), `"`)
			if name == "" {
				continue
			}
			_, hand := handParsedCoordinatorFlags[name]
			if !hand && !flagSetCoordinatorSubcommands[name] {
				missing = append(missing, name)
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("subcommand(s) %v are in the dispatch switch but declared in neither "+
			"handParsedCoordinatorFlags nor flagSetCoordinatorSubcommands.\n"+
			"An undeclared subcommand silently ignores unknown flags — add it to whichever it is.",
			missing)
	}
}
