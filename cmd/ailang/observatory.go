package main

import (
	"flag"
	"fmt"
	"os"
)

// observatoryCommand is `ailang observatory <subcommand>`.
//
// M-V1-SIMPLIFY-S5 M3 folded it into `chains`: the canonical spelling is now
// `ailang chains observatory <subcommand>` and this one survives as an alias
// for one release (D1 — the fleet's scripts and skills keep working).
// chainsCommand routes both spellings through this same function, so they
// cannot diverge.
//
// Eight subcommands were deleted here under D7 (seed, heatmap, evolution,
// usage, tokens, outliers, metrics, hierarchy). Each had ZERO references in
// Makefile/make/tools/scripts/.github/.claude/skills/.agents/skills and a last
// substantive commit in January 2026; the evidence is in the removal commit.
// The four that survive all have live callers.
func observatoryCommand() {
	if flag.NArg() < 2 {
		printObservatoryHelp()
		// Exit 1, not 0. A group asked for no subcommand did nothing, and
		// M-V1-SIMPLIFY-S5 M1 measured five groups exiting 0 for it while
		// `chains`, `budget`, `pkg` and `daemon` exited 1. M3 normalises the
		// ones in its own file set; `models`, `workspaces` and `budget`
		// belong to other milestones.
		os.Exit(1)
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "sync-chat":
		observatorySyncChatCommand()
	case "repair-ids":
		observatoryRepairIDsCommand()
	case "backfill":
		observatoryBackfillCommand()
	case "cleanup":
		observatoryCleanupCommand()
	default:
		fmt.Fprintf(os.Stderr, "Unknown observatory subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func printObservatoryHelp() {
	fmt.Println("Usage: ailang chains observatory <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  sync-chat   Import Claude Code conversation history to database")
	fmt.Println("  backfill    Link existing spans to tasks by time correlation")
	fmt.Println("  repair-ids  Repair OTLP/JSON-corrupted trace/span ids in the CLOUD observatory (dry-run by default)")
	fmt.Println("  cleanup     Delete old/noise spans based on retention policy")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang chains observatory backfill          # Link spans to tasks")
	fmt.Println("  ailang chains observatory cleanup --dry-run # Preview what would be deleted")
	fmt.Println("  ailang chains observatory cleanup --vacuum  # Delete and reclaim disk space")
	fmt.Println("  ailang chains observatory sync-chat         # Import all Claude Code history")
	fmt.Println("  ailang chains observatory sync-chat --status # Show import status")
	fmt.Println()
	fmt.Println("`ailang observatory <subcommand>` remains an alias for one release.")
}
