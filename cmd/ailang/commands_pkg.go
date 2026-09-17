package main

import (
	"fmt"
	"io"
	"os"
	"sort"
)

// pkgSubcommands is the `ailang pkg` group. It is a Command table for the same
// reason the top level is: the hand-written usage block it replaces listed nine
// verbs while the switch accepted ten — `pkg cascade` had been undocumented
// since it landed (M-PKG-AUTONOMOUS-CASCADE-SAFE M4).
func pkgSubcommands() []Command {
	return []Command{
		{
			Name:    "quality",
			Summary: "Quality report: compile, contracts, identity, effects, tests ([--json] [<dir>])",
			Run:     pkgQualityCommand,
		},
		{
			Name:    "info",
			Summary: "Show detailed package information (<vendor/name>)",
			Run:     pkgInfoCommand,
		},
		{
			Name:    "versions",
			Summary: "List all versions with hashes (<vendor/name>)",
			Run:     pkgVersionsCommand,
		},
		{
			Name:    "stats",
			Summary: "Show ecosystem-wide statistics",
			Run:     pkgStatsCommand,
		},
		{
			Name:    "provenance",
			Summary: "Show the provenance chain for a version (<pkg>@<ver>)",
			Run:     pkgProvenanceCommand,
		},
		{
			Name:    "history",
			Summary: "Show the version history timeline (<pkg>@<ver>)",
			Run:     pkgHistoryCommand,
		},
		{
			Name:    "notify-upgrade",
			Summary: "Emit an upgrade-available message, manual fallback (<pkg>@<ver>)",
			Run:     pkgNotifyUpgradeCommand,
		},
		{
			Name:    "affected-by",
			Summary: "List workspaces depending on a package (<pkg>)",
			Run:     pkgAffectedByCommand,
		},
		{
			Name:    "cascade",
			Summary: "Drive the autonomous upgrade cascade",
			Run:     pkgCascadeCommand,
		},
		{
			Name:    "key",
			Summary: "Manage scoped publish keys, superuser only (create|list|revoke)",
			Run:     pkgKeyCommand,
		},
	}
}

// pkgHelpFallback are the pkg verbs whose own parsing rejects --help (both
// demand a <pkg>@<version> and reject "--help" as a malformed one). Measured
// against bin/ailang-before, same as helpFallbackCommands.
var pkgHelpFallback = map[string]bool{
	"provenance": true,
	"history":    true,
}

// pkgGroupCommand dispatches `ailang pkg <verb>`. A bare `ailang pkg` prints
// the group help and exits 1, exactly as before; `ailang pkg --help` prints the
// same help and exits 0, where it used to fail with "unknown pkg command".
func pkgGroupCommand(args []string) error {
	if wantsHelp(args) {
		printPkgHelp(os.Stdout)
		return nil
	}
	if len(args) == 0 {
		printPkgHelp(os.Stdout)
		os.Exit(1)
	}
	subs := pkgSubcommands()
	for i := range subs {
		sub := &subs[i]
		if sub.Name != args[0] {
			continue
		}
		rest := args[1:]
		if wantsHelp(rest) && pkgHelpFallback[sub.Name] {
			fmt.Printf("%s - %s\n\n", bold("ailang pkg "+sub.Name), sub.Summary)
			fmt.Println("Usage:")
			fmt.Printf("  ailang pkg %s [arguments]\n", sub.Name)
			return nil
		}
		if err := sub.Run(rest); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		return nil
	}
	fmt.Fprintf(os.Stderr, "%s: unknown pkg command '%s'\n", red("Error"), args[0])
	os.Exit(1)
	return nil
}

func printPkgHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: ailang pkg <command>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	subs := pkgSubcommands()
	sort.Slice(subs, func(i, j int) bool { return subs[i].Name < subs[j].Name })
	for _, sub := range subs {
		fmt.Fprintf(w, "  %s%s%s\n", cyan(sub.Name), helpPad(sub.Name, 18), sub.Summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "To publish a new version and fire the full cascade bus:")
	fmt.Fprintln(w, "  ailang publish              (preferred — wraps notify-upgrade + cascade-topic)")
}
