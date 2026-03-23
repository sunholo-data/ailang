package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/pkg"
)

func pkgNotifyUpgradeCommand(args []string) error {
	fs := flag.NewFlagSet("pkg notify-upgrade", flag.ExitOnError)
	summary := fs.String("summary", "", "Change summary")
	action := fs.String("action", "", "Recommended action")
	dryRun := fs.Bool("dry-run", false, "Print message without sending")
	help := fs.Bool("help", false, "Show help")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *help || fs.NArg() < 1 {
		fmt.Println("Usage: ailang pkg notify-upgrade <package>@<version> [flags]")
		fmt.Println()
		fmt.Println("Emit an upgrade-available message for a new package version.")
		fmt.Println("Compares current lockfile state with the published version to")
		fmt.Println("determine interface hash changes and change class.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --summary TEXT     Change summary")
		fmt.Println("  --action TEXT      Recommended action for consumers")
		fmt.Println("  --dry-run          Print message JSON without sending")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  ailang pkg notify-upgrade sunholo/auth@0.2.0 --summary 'Tightened bearer validation'")
		return nil
	}

	// Parse package@version
	spec := fs.Arg(0)
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("expected <package>@<version> format, got %q", spec)
	}
	pkgName, toVersion := parts[0], parts[1]

	// Load current lockfile to find previous version info
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var fromVersion, fromInterfaceHash, fromContentHash string
	lf, err := pkg.LoadLockFile(cwd)
	if err == nil {
		for _, lp := range lf.Packages {
			if lp.Name == pkgName {
				fromVersion = lp.Version
				fromInterfaceHash = lp.InterfaceHash
				fromContentHash = lp.ContentHash
				break
			}
		}
	}

	// Load manifest to get current interface hash
	manifest, err := pkg.LoadManifest(cwd)
	if err != nil {
		return fmt.Errorf("no ailang.toml found: %w", err)
	}

	// Determine change class based on interface hash change
	changeClass := "A" // default: internal only
	toInterfaceHash := ""
	toContentHash := ""
	if manifest.Package.Name == pkgName {
		// We're the package being upgraded — compute current hashes
		resolved, resolveErr := pkg.ResolveDependencies(manifest, cwd)
		if resolveErr == nil {
			for _, r := range resolved {
				if r.Name == pkgName {
					toInterfaceHash = r.InterfaceHash
					toContentHash = r.ContentHash
					break
				}
			}
		}
		if fromInterfaceHash != "" && toInterfaceHash != "" && fromInterfaceHash != toInterfaceHash {
			changeClass = "C" // contract change
		} else if fromContentHash != "" && toContentHash != "" && fromContentHash != toContentHash {
			changeClass = "B" // content changed, interface same
		}
	}

	// Find affected workspaces by scanning for lockfiles that depend on this package
	to := findAffectedWorkspaces(pkgName)
	if len(to) == 0 {
		to = []string{messaging.FormatPackageInbox(pkgName)}
	}

	env := &messaging.PackageMessageEnvelope{
		Schema:    messaging.PackageMessageSchema,
		Kind:      messaging.PkgMsgUpgradeAvailable,
		From:      messaging.FormatPackageInbox(pkgName),
		To:        to,
		Timestamp: time.Now().UTC(),
		Package: messaging.PackageRef{
			Name:              pkgName,
			FromVersion:       fromVersion,
			ToVersion:         toVersion,
			FromInterfaceHash: fromInterfaceHash,
			ToInterfaceHash:   toInterfaceHash,
			FromContentHash:   fromContentHash,
			ToContentHash:     toContentHash,
			ChangeClass:       changeClass,
		},
		Summary:           *summary,
		RecommendedAction: *action,
		Status:            "open",
	}

	if *dryRun {
		data, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Validate before sending
	if err := messaging.ValidatePackageMessage(env); err != nil {
		return fmt.Errorf("invalid package message: %w", err)
	}

	// Send to each recipient
	store, err := openPkgMsgStore()
	if err != nil {
		return err
	}
	defer store.Close()

	for _, recipient := range env.To {
		envCopy := *env
		envCopy.To = []string{recipient}
		msg, err := envCopy.ToInboxMessage()
		if err != nil {
			return fmt.Errorf("failed to create inbox message: %w", err)
		}
		if err := store.InsertInboxMessage(msg); err != nil {
			return fmt.Errorf("failed to send to %s: %w", recipient, err)
		}
		fmt.Printf("%s Sent upgrade-available to %s (ID: %s)\n", green("✓"), recipient, msg.MessageID)
	}

	return nil
}

func pkgAffectedByCommand(args []string) error {
	fs := flag.NewFlagSet("pkg affected-by", flag.ExitOnError)
	help := fs.Bool("help", false, "Show help")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *help || fs.NArg() < 1 {
		fmt.Println("Usage: ailang pkg affected-by <package>[@<version>]")
		fmt.Println()
		fmt.Println("List workspaces that depend on a package.")
		fmt.Println("Scans lockfiles in known workspace directories.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --json    Output as JSON array")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  ailang pkg affected-by sunholo/auth")
		return nil
	}

	spec := fs.Arg(0)
	pkgName := strings.SplitN(spec, "@", 2)[0]

	affected := findAffectedWorkspaces(pkgName)

	if *jsonOut {
		data, _ := json.MarshalIndent(affected, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(affected) == 0 {
		fmt.Printf("No workspaces found depending on %s\n", pkgName)
		fmt.Println("(Only scans lockfiles in the current workspace)")
		return nil
	}

	fmt.Printf("Workspaces depending on %s:\n", bold(pkgName))
	for _, ws := range affected {
		addr := messaging.ParseInboxAddress(ws)
		fmt.Printf("  %s %s\n", cyan("→"), addr.Name)
	}

	return nil
}

// findAffectedWorkspaces finds workspaces that depend on a package
// by scanning the current workspace's lockfile.
func findAffectedWorkspaces(pkgName string) []string {
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	// Check if current workspace depends on this package
	lf, err := pkg.LoadLockFile(cwd)
	if err != nil {
		return nil
	}

	for _, lp := range lf.Packages {
		if lp.Name == pkgName {
			// Current workspace depends on this package
			manifest, mErr := pkg.LoadManifest(cwd)
			if mErr != nil {
				return []string{messaging.FormatWorkspaceInbox("current")}
			}
			return []string{messaging.FormatWorkspaceInbox(manifest.Package.Name)}
		}
	}

	return nil
}

// openPkgMsgStore opens the messaging store for package commands.
func openPkgMsgStore() (*messaging.Store, error) {
	dbPath := messaging.GetDefaultDatabasePath()
	return messaging.OpenStore(dbPath)
}
