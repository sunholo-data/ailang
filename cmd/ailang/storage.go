package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/statedir"
	"github.com/sunholo-data/ailang/internal/storage"
)

// `ailang storage` is the storage plane's REPORT. It used to also carry the
// one-time local-SQLite → GCP-Firestore migration (`migrate`, `verify`), which
// M-V1-SIMPLIFY-S5 M4 removed under D7 along with internal/storage/migrate.
//
// Evidence, re-derived 2026-09-17: `git log -S storageMigrate` and
// `git log -S NewMigrator` each return exactly ONE commit, 56ac6827c
// (2026-02-18, "firestore"). The package's only other touch is de4aa7ba8, the
// module-path rename. Seven months, one commit, no tests, and zero references
// in Makefile, make/, tools/, scripts/, .github/ or .claude/skills/ — its only
// mentions anywhere were two v0.8.1/v0.9.0 design docs recording the migration
// that has already happened.
//
// `status` stays, and is the live half: CLAUDE.md's session-start check tells
// every machine to confirm `messaging gcp (AILANG_STORAGE_MESSAGING)` with it,
// and M-V1-SIMPLIFY-S3 M3 (def7fd900) rewrote its output to print one line per
// store. Moving a store between planes is what AILANG_STORAGE and its
// per-store overrides do now; nothing copies records any more.
func storageCommand(args []string) error {
	// No subcommand is a usage error, not a success. See bareGroupExitsOne in
	// commands_groups_test.go: `chains`, `pkg`, `daemon`, `dev` and `ops`
	// already exited 1 for this shape; storage, models and workspaces exited 0.
	if len(args) == 0 {
		printStorageHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "status":
		return storageStatus()
	case "--help", "-h", "help":
		printStorageHelp()
		return nil
	default:
		return fmt.Errorf("unknown storage subcommand: %s", args[0])
	}
}

// storageStatus prints ONE line per store: the resolved mode, the variable
// (or default) it came from, and the path or project it means. Before
// M-V1-SIMPLIFY-S3 M3 this printed AILANG_STORAGE alone while four other
// selectors could move a store elsewhere — the status could say "local"
// while `ailang messages list` read Firestore.
func storageStatus() error {
	return writeStorageStatus(os.Stdout)
}

func writeStorageStatus(w io.Writer) error {
	sel, err := config.StoragePlane()
	if err != nil {
		return err
	}
	fmt.Fprintln(w, bold("Storage plane"))
	fmt.Fprintf(w, "  plane  %-7s (%s)\n", sel.Plane, sel.PlaneSource)
	fmt.Fprintln(w)

	// Resolve the project once, with its source, for the stores that need it.
	// A plane that needs one and has none says so per store and still exits
	// 0: status is a report, not a gate.
	var project, projectSrc string
	if sel.AnyGCP() || sel.Shared() {
		p, src, perr := config.CloudProjectSource(context.Background())
		if perr != nil {
			project, projectSrc = "", perr.Error()
		} else {
			project, projectSrc = p, string(src)
		}
	}
	for _, st := range []config.StoreSelection{sel.Messaging, sel.Coordinator, sel.Observatory} {
		var where string
		switch st.Mode {
		case config.StoreGCP:
			if project == "" {
				where = "project (unresolved: " + projectSrc + ")"
			} else {
				where = fmt.Sprintf("project %s (%s)", project, projectSrc)
			}
		default:
			if p := storage.LocalPath(st.Store); p != "" {
				where = p
			} else if d, derr := statedir.Dir(); derr != nil {
				where = "(unresolved: " + derr.Error() + ")"
			} else {
				where = d
			}
		}
		fmt.Fprintf(w, "  %-12s %-6s (%s)  %s\n", st.Store, st.Mode, st.Source, where)
	}
	if sel.Plane == config.PlaneHybrid {
		fmt.Fprintln(w)
		if project != "" {
			fmt.Fprintf(w, "  shared plane: project %s (%s) — Pub/Sub publisher and secret approver on; every store stays in SQLite\n", project, projectSrc)
		} else {
			fmt.Fprintf(w, "  shared plane: project unresolved (%s)\n", projectSrc)
		}
	}
	if mode, src, merr := config.CoordinatorMode(); merr != nil {
		fmt.Fprintf(w, "\n  %s %v\n", yellow("!"), merr)
	} else if src != config.SourceDefault {
		fmt.Fprintf(w, "\n  coordinator mode  %s  (%s)\n", mode, src)
	}
	return nil
}

func printStorageHelp() {
	fmt.Println("Usage: ailang storage <command>")
	fmt.Println("")
	fmt.Println("Report the storage plane: which backend each store resolved to,")
	fmt.Println("and the variable or default that decided it.")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  status     Show the resolved plane, one line per store")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  ailang storage status")
	fmt.Println("")
	fmt.Println("Which backend a store uses is set by AILANG_STORAGE and the")
	fmt.Println("per-store overrides AILANG_STORAGE_MESSAGING / _COORDINATOR /")
	fmt.Println("_OBSERVATORY. See `ailang docs search storage plane`.")
}
