package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func pkgAddCommand(args []string) error {
	flagSet := flag.NewFlagSet("add", flag.ExitOnError)
	pathFlag := flagSet.Bool("path", false, "Add as path dependency")
	gitFlag := flagSet.String("git", "", "Add as git dependency (repo URL)")
	tagFlag := flagSet.String("tag", "", "Git tag to pin to")
	revFlag := flagSet.String("rev", "", "Git commit hash to pin to")
	subdirFlag := flagSet.String("subdir", "", "Subdirectory within git repo")
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag || (flagSet.NArg() < 1 && *gitFlag == "") {
		fmt.Println("Usage: ailang add <dependency> [flags]")
		fmt.Println()
		fmt.Println("Add a dependency to ailang.toml.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --path              Add as a path dependency (local directory)")
		fmt.Println("  --git URL           Add as a git dependency")
		fmt.Println("  --tag TAG           Git tag to pin to (with --git)")
		fmt.Println("  --rev HASH          Git commit hash to pin to (with --git)")
		fmt.Println("  --subdir DIR        Subdirectory within git repo (with --git)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang add --path ../shared/json")
		fmt.Println("  ailang add sunholo/json@0.3.1")
		fmt.Println("  ailang add --git https://github.com/sunholo-data/ailang-packages --subdir packages/auth --tag auth-v0.1.0")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	manifest, err := pkg.LoadManifest(cwd)
	if err != nil {
		return fmt.Errorf("no ailang.toml found in current directory: %w\nRun 'ailang init package' first", err)
	}

	if *gitFlag != "" {
		return addGitDep(cwd, manifest, *gitFlag, *tagFlag, *revFlag, *subdirFlag)
	}

	arg := flagSet.Arg(0)

	if *pathFlag {
		return addPathDep(cwd, manifest, arg)
	}
	return addVersionDep(cwd, manifest, arg)
}

func addPathDep(cwd string, manifest *pkg.PackageManifest, depPath string) error {
	// Load the dependency's manifest to get its name
	depManifest, err := pkg.LoadManifest(depPath)
	if err != nil {
		// Try relative to cwd
		absPath := depPath
		if !strings.HasPrefix(depPath, "/") {
			absPath = cwd + "/" + depPath
		}
		depManifest, err = pkg.LoadManifest(absPath)
		if err != nil {
			return fmt.Errorf("no ailang.toml found at %s: %w", depPath, err)
		}
	}

	name := depManifest.Package.Name

	// Check for duplicate
	if _, exists := manifest.Dependencies[name]; exists {
		return fmt.Errorf("dependency %s already exists in ailang.toml", name)
	}

	// Re-read the manifest file to append the dependency
	return appendDependencyToFile(cwd, name, depPath, true)
}

func addVersionDep(cwd string, manifest *pkg.PackageManifest, spec string) error {
	// Parse name@version
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("version dependency must be name@version format, got %q\nExample: ailang add sunholo/json@0.3.1", spec)
	}
	name, version := parts[0], parts[1]

	// Check for duplicate
	if _, exists := manifest.Dependencies[name]; exists {
		return fmt.Errorf("dependency %s already exists in ailang.toml", name)
	}

	return appendDependencyToFile(cwd, name, version, false)
}

func addGitDep(cwd string, manifest *pkg.PackageManifest, gitURL, tag, rev, subdir string) error {
	if tag == "" && rev == "" {
		return fmt.Errorf("git dependency requires --tag or --rev\nExample: ailang add --git https://github.com/sunholo-data/ailang-packages --subdir packages/auth --tag auth-v0.1.0")
	}

	// Resolve to find package name
	cache, err := pkg.NewGitCache()
	if err != nil {
		return fmt.Errorf("failed to init git cache: %w", err)
	}

	fmt.Printf("Fetching %s...\n", gitURL)
	localPath, resolvedRev, err := cache.Resolve(gitURL, tag, rev, subdir)
	if err != nil {
		return fmt.Errorf("failed to resolve git dependency: %w", err)
	}

	depManifest, err := pkg.LoadManifest(localPath)
	if err != nil {
		return fmt.Errorf("no ailang.toml found at %s (subdir: %s): %w", gitURL, subdir, err)
	}

	name := depManifest.Package.Name

	if _, exists := manifest.Dependencies[name]; exists {
		return fmt.Errorf("dependency %s already exists in ailang.toml", name)
	}

	// Build the TOML line
	parts := []string{fmt.Sprintf("git = %q", gitURL)}
	if subdir != "" {
		parts = append(parts, fmt.Sprintf("subdir = %q", subdir))
	}
	if rev != "" {
		parts = append(parts, fmt.Sprintf("rev = %q", resolvedRev))
	} else {
		parts = append(parts, fmt.Sprintf("tag = %q", tag))
	}

	return appendGitDependencyToFile(cwd, name, parts, gitURL, tag, resolvedRev)
}

func appendGitDependencyToFile(dir, name string, parts []string, gitURL, tag, rev string) error {
	path := dir + "/" + pkg.ManifestFile
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	depLine := fmt.Sprintf("\"%s\" = { %s }\n", name, strings.Join(parts, ", "))

	if strings.Contains(content, "[dependencies]") {
		idx := strings.Index(content, "[dependencies]")
		lineEnd := strings.Index(content[idx:], "\n")
		insertAt := idx + lineEnd + 1
		content = content[:insertAt] + depLine + content[insertAt:]
	} else {
		content += "\n[dependencies]\n" + depLine
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	label := gitURL
	if tag != "" {
		label += "@" + tag
	} else {
		label += "@" + rev[:12]
	}
	fmt.Printf("%s Added %s (git: %s)\n", green("✓"), name, label)
	return nil
}

// appendDependencyToFile appends a dependency line to ailang.toml.
func appendDependencyToFile(dir, name, value string, isPath bool) error {
	path := dir + "/" + pkg.ManifestFile
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)

	var depLine string
	if isPath {
		depLine = fmt.Sprintf("\"%s\" = { path = %q }\n", name, value)
	} else {
		depLine = fmt.Sprintf("\"%s\" = %q\n", name, value)
	}

	// Find [dependencies] section and append
	if strings.Contains(content, "[dependencies]") {
		// Insert after [dependencies] line
		idx := strings.Index(content, "[dependencies]")
		lineEnd := strings.Index(content[idx:], "\n")
		insertAt := idx + lineEnd + 1
		content = content[:insertAt] + depLine + content[insertAt:]
	} else {
		// Add new [dependencies] section
		content += "\n[dependencies]\n" + depLine
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	label := value
	if isPath {
		label = fmt.Sprintf("path: %s", value)
	}
	fmt.Printf("%s Added %s (%s)\n", green("✓"), name, label)
	return nil
}

func pkgLockCommand(args []string) error {
	flagSet := flag.NewFlagSet("lock", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		fmt.Println("Usage: ailang lock")
		fmt.Println()
		fmt.Println("Resolve dependencies and generate ailang.lock.")
		fmt.Println("Reads ailang.toml and writes a deterministic lock file")
		fmt.Println("with content hashes for all resolved packages.")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	fmt.Printf("Resolving dependencies in %s\n", cwd)

	manifest, err := pkg.LoadManifest(cwd)
	if err != nil {
		return fmt.Errorf("no ailang.toml found in %s: %w\nRun 'ailang init package' first", cwd, err)
	}

	if len(manifest.Dependencies) == 0 {
		fmt.Printf("%s No dependencies to resolve\n", yellow("⚠"))
		// Still create an empty lock file for consistency
		lf := pkg.NewLockFile(nil, fmt.Sprintf("ailang lock %s", Version))
		lf.AILANGVersion = Version
		if err := lf.Save(cwd); err != nil {
			return fmt.Errorf("failed to write lock file: %w", err)
		}
		fmt.Printf("%s Generated %s (0 packages)\n", green("✓"), pkg.LockFileName)
		return nil
	}

	// Load previous lockfile before overwriting, for ratchet detection.
	prevLF, _ := pkg.LoadLockFile(cwd)

	resolved, err := pkg.ResolveDependencies(manifest, cwd)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Convert to LockedPackages
	locked := make([]pkg.LockedPackage, len(resolved))
	for i, r := range resolved {
		locked[i] = pkg.LockedPackage(r)
	}

	lf := pkg.NewLockFile(locked, fmt.Sprintf("ailang lock %s", Version))
	lf.AILANGVersion = Version
	if err := lf.Save(cwd); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	// Warn when a path dep's version changed without a corresponding
	// upgrade-available message — this is the "silent ratchet" failure mode.
	if prevLF != nil {
		warnSilentRatchet(resolved, prevLF)
	}

	fmt.Printf("%s Generated %s (%d packages)\n", green("✓"), pkg.LockFileName, len(resolved))
	for _, r := range resolved {
		source := r.Source
		if r.Path != "" {
			source = "path: " + r.Path
		}
		fmt.Printf("  %s %s@%s (%s)\n", cyan("→"), r.Name, r.Version, source)
	}

	return nil
}

func pkgTreeCommand(args []string) error {
	flagSet := flag.NewFlagSet("tree", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		fmt.Println("Usage: ailang tree")
		fmt.Println()
		fmt.Println("Display the dependency tree for the current package.")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	manifest, err := pkg.LoadManifest(cwd)
	if err != nil {
		return fmt.Errorf("no ailang.toml found: %w\nRun 'ailang init package' first", err)
	}

	tree, err := pkg.BuildDependencyTree(manifest, cwd)
	if err != nil {
		return fmt.Errorf("failed to build dependency tree: %w", err)
	}

	fmt.Print(tree)
	return nil
}

// warnSilentRatchet emits a stderr warning for each path dep whose version
// changed since the previous lockfile without a corresponding upgrade-available
// message in the messaging inbox.  This catches the "silent ratchet" where a
// developer bumps [package].version without running `ailang publish`.
//
// Warnings are best-effort: if the messaging store is unavailable the check is
// skipped silently.  The lock file is always written regardless.
func warnSilentRatchet(resolved []pkg.ResolvedPackage, prevLF *pkg.LockFile) {
	// Build a version map from the previous lockfile.
	prevVersions := make(map[string]string, len(prevLF.Packages))
	for _, lp := range prevLF.Packages {
		prevVersions[lp.Name] = lp.Version
	}

	// Open the messaging store read-only; skip silently if unavailable.
	dbPath := messaging.GetDefaultDatabasePath()
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		return
	}
	defer store.Close()

	for _, r := range resolved {
		if r.Source != "path" {
			continue // only path deps can silently ratchet
		}
		prev, ok := prevVersions[r.Name]
		if !ok || prev == r.Version {
			continue // new dep or version unchanged
		}

		// Version changed — check for an upgrade-available message.
		msgs, listErr := store.ListInboxMessages(messaging.InboxListOptions{
			Inbox:       messaging.FormatPackageInbox(r.Name),
			IncludeRead: true,
			Limit:       50,
		})
		if listErr != nil {
			continue
		}

		found := false
		for _, msg := range msgs {
			env, extractErr := messaging.ExtractPackageEnvelope(&msg)
			if extractErr != nil || env == nil {
				continue
			}
			if env.Kind == messaging.PkgMsgUpgradeAvailable && env.Package.ToVersion == r.Version {
				found = true
				break
			}
		}

		if !found {
			fmt.Fprintf(os.Stderr, "%s %s resolved to %s (was %s in lockfile)\n",
				yellow("⚠"), r.Name, r.Version, prev)
			fmt.Fprintf(os.Stderr, "   No upgrade-available message found. Run 'ailang publish' in the\n")
			fmt.Fprintf(os.Stderr, "   package directory to announce this change and trigger cascade tests.\n")
		}
	}
}
