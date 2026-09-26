package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Group dispatch (M-V1-SIMPLIFY-S5 M2).
//
// A group is a second way to REACH a command, never a replacement for the
// first. `ailang messages list` and `ailang ops messages list` run the same
// code on the same argv; the group exists so `ailang --help` can fit on one
// screen (89 top-level entries -> 17) without any of the 593 fleet references
// to `ailang messages` having to change. That is D1, and it is the whole
// reason this milestone is not a rename.

// runAsTopLevel invokes c as if the user had typed `ailang <c.Name> <args...>`.
//
// It rewrites os.Args, and that is not an accident or a shortcut. Measured at
// HEAD: 44 call sites across cmd/ailang parse os.Args[2:] or os.Args[3:]
// DIRECTLY rather than the tail the dispatcher hands them — messages.go:81
// reads os.Args[2] as its subcommand, trace.go:92 parses os.Args[3:], and so
// on. Handing those functions a tail changes nothing about what they read. So
// a group route has to present the same argv the top-level spelling presents,
// or `ailang ops messages list` reads its subcommand as "messages" and every
// folded command is subtly wrong in a way no help text would reveal.
//
// Rewriting here, in ONE place, is what makes the group routes byte-identical
// to the top-level ones by construction instead of by 44 separate edits — and
// those 44 files belong to M3/M4/M5, not to this milestone.
func runAsTopLevel(c *Command, args []string) error {
	return runAsName(c.Name, c.Run, args)
}

// runAsName is runAsTopLevel for a route whose canonical spelling is not its
// own row's Name: `ailang eval run` is `ailang eval`, so it rewrites to "eval".
//
// It rewrites TWO argument sources, because the binary has two and different
// commands read different ones. os.Args covers the 44 sites that slice it
// directly; flag.CommandLine covers the ones that read the global flag.Args()
// — `trace`, `chains`, `dashboard`, `axioms`, `editor`, `budget`, `daemon` and
// the rest of the noArgs family, which take no tail at all and find their
// subcommand at flag.Arg(1).
//
// Measured, before the second rewrite existed: `ailang ops trace status` read
// its subcommand as "trace" and exited 1 where `ailang trace status` exited 0,
// and `ailang dev axioms --help` printed the whole axiom scorecard instead of
// the usage block. Both exited without complaint — a group route that reads
// the wrong argument does not look broken, it looks like a different command.
// That is why the equivalence is measured by byte-diffing the two spellings
// rather than by reading the code.
//
// Re-parsing flag.CommandLine does not disturb the global flags: the rewritten
// argv holds none, and flag.Parse only assigns the flags it actually sees, so
// values set by the real command line survive.
func runAsName(name string, run func([]string) error, args []string) error {
	savedArgs := os.Args
	savedInvoked := invokedAs
	savedFlagArgs := flag.Args()

	os.Args = append([]string{savedArgs[0], name}, args...)
	invokedAs = name
	_ = flag.CommandLine.Parse(os.Args[1:])

	defer func() {
		os.Args = savedArgs
		invokedAs = savedInvoked
		_ = flag.CommandLine.Parse(savedFlagArgs)
	}()
	return run(args)
}

// groupMembers returns the rows filed in group, ordered by name.
func groupMembers(group string) []*Command {
	var out []*Command
	for i := range allCommands {
		if allCommands[i].Group == group {
			out = append(out, &allCommands[i])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// groupSubName is the name a command answers to INSIDE its group. For every
// group but eval that is just its name (`ailang ops messages`). The eval
// commands carry the group in their top-level spelling — `eval-suite`,
// `eval-report` — so inside the group the prefix comes off and `ailang eval
// suite` is the subcommand, exactly as the Phase 3 item-2 list specifies.
// Deriving it beats a second hand-maintained column, which could drift.
func groupSubName(group, name string) string {
	if p := group + "-"; strings.HasPrefix(name, p) {
		return strings.TrimPrefix(name, p)
	}
	return name
}

// lookupGroupMember resolves a subcommand name within a group.
func lookupGroupMember(group, sub string) *Command {
	for _, c := range groupMembers(group) {
		if groupSubName(group, c.Name) == sub {
			return c
		}
	}
	return nil
}

// isGroupName reports whether name is one of the dispatching groups.
func isGroupName(name string) bool {
	for _, g := range dispatchGroups {
		if g == name {
			return true
		}
	}
	return false
}

// groupSummaries describe each group in `ailang --help`'s footer and at the top
// of its own help.
var groupSummaries = map[string]string{
	groupDev:  "Language-development and diagnostic commands",
	groupOps:  "Fleet operations: messages, coordinator, mission, chains, models",
	groupEval: "Benchmarks and eval analysis",
}

// groupEntryCommands generates the `dev` and `ops` entry points. `eval` is NOT
// generated here: it is a real, visible, pre-existing command whose bare form
// runs a benchmark, so evalCommands() owns its row (see evalGroupCommand).
//
// Both are Hidden — they are drawers, not commands, and the top-level help
// points at them in a footer line instead of spending two of its rows.
func groupEntryCommands() []Command {
	var out []Command
	for _, g := range []string{groupDev, groupOps} {
		group := g
		out = append(out, Command{
			Name:    group,
			Hidden:  true,
			Summary: groupSummaries[group],
			Run:     func(args []string) error { return runGroup(group, args) },
		})
	}
	return out
}

// runGroup dispatches `ailang <group> <subcommand> [args...]`.
//
// A bare `ailang dev` prints the group's help and exits 1, matching the two
// groups that already existed (`ailang pkg` and `ailang daemon`); `--help`
// prints the same block and exits 0.
func runGroup(group string, args []string) error {
	if wantsHelp(args) {
		printGroupHelp(os.Stdout, group)
		return nil
	}
	if len(args) == 0 {
		printGroupHelp(os.Stdout, group)
		os.Exit(1)
	}
	c := lookupGroupMember(group, args[0])
	if c == nil {
		fmt.Fprintf(os.Stderr, "%s: unknown %s command '%s'\n", red("Error"), group, args[0])
		fmt.Fprintf(os.Stderr, "Run '%s' for the list.\n", "ailang "+group+" --help")
		os.Exit(1)
	}
	if wantsHelp(args[1:]) && helpFallbackCommands[c.Name] {
		printCommandHelp(os.Stdout, c)
		return nil
	}
	return runAsTopLevel(c, args[1:])
}

// groupExtraRows are subcommands a group answers to that have no row of their
// own in allCommands, so the generated list would otherwise omit them.
//
// There is exactly one: `ailang eval run` is the bare `ailang eval` benchmark
// runner, which cannot also be a member row of the group it heads.
var groupExtraRows = map[string][][2]string{
	groupEval: {{"run", "Run one benchmark (the bare `ailang eval` form)"}},
}

// printGroupHelp renders one group's command list, generated from the table.
func printGroupHelp(w io.Writer, group string) {
	fmt.Fprintf(w, "%s - %s\n\n", bold("ailang "+group), groupSummaries[group])
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  ailang %s <command> [arguments]\n", group)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")

	type row struct{ name, summary string }
	var rows []row
	for _, c := range groupMembers(group) {
		if c.Hidden {
			continue
		}
		rows = append(rows, row{groupSubName(group, c.Name), c.Summary})
	}
	for _, extra := range groupExtraRows[group] {
		rows = append(rows, row{extra[0], extra[1]})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	for _, r := range rows {
		// Width 30, the same column the top-level list uses, so the two help
		// screens line up — and so `generate-extension-registry` (27) does not
		// collapse the whole column to a single space.
		fmt.Fprintf(w, "  %s%s%s\n", cyan(r.name), helpPad(r.name, 30), r.summary)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Every command above also keeps its own top-level spelling — `%s` is `%s`.\n",
		"ailang "+group+" "+exampleSub(group), "ailang "+exampleTopLevel(group))
	if group == groupEval {
		fmt.Fprintf(w, "For the benchmark runner's own flags: %s\n", cyan("ailang eval run --help"))
	}
}

// exampleSub / exampleTopLevel pick the group's first visible member so the
// closing line of its help names a real pair rather than a made-up one.
func exampleSub(group string) string {
	for _, c := range groupMembers(group) {
		if !c.Hidden {
			return groupSubName(group, c.Name)
		}
	}
	return "<command>"
}

func exampleTopLevel(group string) string {
	for _, c := range groupMembers(group) {
		if !c.Hidden {
			return c.Name
		}
	}
	return "<command>"
}

// helpCommand is `ailang help [command [subcommand]]`, the visible top-level
// entry the Phase 3 item-2 list calls for. `ailang help X` is `ailang X
// --help`, routed through the same fallback table the dispatcher uses, so the
// two spellings cannot answer differently.
func helpCommand(args []string) error {
	if len(args) == 0 || wantsHelp(args) {
		printHelp()
		return nil
	}
	c, ok := lookupCommand(args[0])
	if !ok {
		unknownCommand(args[0])
		return nil // unreachable: unknownCommand exits
	}
	if len(args) == 1 && helpFallbackCommands[c.Name] {
		printCommandHelp(os.Stdout, c)
		return nil
	}
	rest := append(append([]string{}, args[1:]...), "--help")
	return runAsTopLevel(c, rest)
}

// pkgLegacyTopLevel maps a `pkg` verb to the bare top-level spelling that
// reaches it today: `ailang add`, `ailang publish`, `ailang pkg-docs` and the
// rest. S5 M2 files these verbs under `pkg` where they belong, and D1 keeps
// every bare spelling working — so each becomes a HIDDEN top-level row whose
// Run is the SAME function pkgSubcommands() holds. One definition, two routes,
// no drift. commands_spellings_test.go checks both ends against the fixture.
var pkgLegacyTopLevel = map[string]string{
	"add":       "add",
	"lock":      "lock",
	"tree":      "tree",
	"install":   "install",
	"bin":       "bin", // M-PKG-BIN-ENTRYPOINTS: new with the verb, not legacy — same one-definition rule
	"search":    "search",
	"publish":   "publish",
	"unpublish": "unpublish",
	"docs":      "pkg-docs",
}

// pkgLegacyCommands generates those hidden rows. The order is the sorted verb
// order, not map order, so allCommands is deterministic.
func pkgLegacyCommands() []Command {
	verbs := make([]string, 0, len(pkgLegacyTopLevel))
	for v := range pkgLegacyTopLevel {
		verbs = append(verbs, v)
	}
	sort.Strings(verbs)

	subs := pkgSubcommands()
	byName := make(map[string]*Command, len(subs))
	for i := range subs {
		byName[subs[i].Name] = &subs[i]
	}

	var out []Command
	for _, verb := range verbs {
		sub, ok := byName[verb]
		if !ok {
			// A key with no verb means the two tables disagree. Panic at
			// startup rather than silently dropping a route the fleet calls.
			panic("ailang: pkgLegacyTopLevel names " + verb + ", which is not a pkg verb")
		}
		out = append(out, Command{
			Name:    pkgLegacyTopLevel[verb],
			Group:   groupPkg,
			Hidden:  true,
			Summary: sub.Summary,
			Run:     sub.Run,
		})
	}
	return out
}
