package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	path := filepath.Join(dir, pkg.ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	depLine := fmt.Sprintf("\"%s\" = { %s }", name, strings.Join(parts, ", "))
	content, previous, replaced := upsertDependencyLine(string(data), name, depLine)

	if err := writeManifestChecked(dir, path, data, content); err != nil {
		return err
	}

	label := gitURL
	if tag != "" {
		label += "@" + tag
	} else {
		label += "@" + rev[:12]
	}
	if replaced && previous != depLine {
		fmt.Printf("%s Updated %s (%s -> git: %s)\n", green("✓"), name, previous, label)
	} else {
		fmt.Printf("%s Added %s (git: %s)\n", green("✓"), name, label)
	}
	return nil
}

// appendDependencyToFile inserts or replaces a dependency line in ailang.toml.
func appendDependencyToFile(dir, name, value string, isPath bool) error {
	path := filepath.Join(dir, pkg.ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)

	var depLine string
	if isPath {
		depLine = fmt.Sprintf("\"%s\" = { path = %q }", name, value)
	} else {
		depLine = fmt.Sprintf("\"%s\" = %q", name, value)
	}

	content, previous, replaced := upsertDependencyLine(content, name, depLine)

	if err := writeManifestChecked(dir, path, data, content); err != nil {
		return err
	}

	label := value
	if isPath {
		label = fmt.Sprintf("path: %s", value)
	}
	if replaced && previous != depLine {
		fmt.Printf("%s Updated %s (%s -> %s)\n", green("✓"), name, previous, label)
	} else {
		fmt.Printf("%s Added %s (%s)\n", green("✓"), name, label)
	}
	return nil
}

// upsertDependencyLine replaces an exact key inside [dependencies], preserving
// its position, or inserts the line immediately after the section header.
//
// This is a line scanner over TOML text, so what it CANNOT see is the thing that
// breaks it. Four shapes were found by mutation/adversarial review and are
// handled explicitly below: a header carrying a trailing comment, a header that
// is the file's last line with no newline after it, lines inside a multi-line
// string that look like headers or keys, and single-quoted (literal) keys.
// Anything still missed is bounded by writeManifestChecked, which refuses to
// leave a manifest less parseable than it found it.
func upsertDependencyLine(content, name, depLine string) (updated, previous string, replaced bool) {
	lines := strings.SplitAfter(content, "\n")
	inDependencies := false
	sectionHeader := -1
	// Open multi-line string delimiter (`"""` or `'''`), empty when outside one.
	// A line inside such a string is TOML data, not structure — treating it as a
	// header ends the scan early and silently re-inserts a key that was there.
	openString := ""
	for i, line := range lines {
		body := stripLineComment(line)
		trimmed := strings.TrimSpace(body)
		if openString != "" {
			if strings.Contains(line, openString) {
				openString = ""
			}
			continue
		}
		if d := openMultilineString(line); d != "" {
			openString = d
			// The opening line may still carry the key it belongs to; fall through
			// so a `key = """` line is matched as a key before we start skipping.
		}
		if openString == "" && strings.HasPrefix(trimmed, "[") {
			if inDependencies {
				break
			}
			inDependencies = trimmed == "[dependencies]"
			if inDependencies {
				sectionHeader = i
			}
			continue
		}
		if inDependencies && openString == "" && dependencyLineKey(body) == name {
			previous = strings.TrimSpace(strings.TrimRight(line, "\r\n"))
			// Preserve the original line ending: a CRLF manifest must not gain a
			// lone LF in the middle of it.
			ending := line[len(strings.TrimRight(line, "\r\n")):]
			lines[i] = depLine + ending
			return strings.Join(lines, ""), previous, true
		}
	}

	if sectionHeader >= 0 {
		header := lines[sectionHeader]
		// A header that is the file's last line has no newline of its own, so an
		// insert glues the key onto it: `[dependencies]"name" = "1.0"`, which no
		// longer parses. Give it one.
		if !strings.HasSuffix(header, "\n") {
			lines[sectionHeader] = header + "\n"
		}
		insertAt := sectionHeader + 1
		line := depLine + "\n"
		lines = append(lines, "")
		copy(lines[insertAt+1:], lines[insertAt:])
		lines[insertAt] = line
		return strings.Join(lines, ""), "", false
	}

	// A brand-new section is separated from the preceding one by a blank line,
	// matching every manifest written before the upsert rewrite. Without the
	// leading newline the header lands flush against the previous key — valid
	// TOML, but a visible formatting regression in a file users read.
	prefix := "\n"
	if !strings.HasSuffix(content, "\n") {
		prefix = "\n\n"
	}
	if content == "" {
		prefix = ""
	}
	return content + prefix + "[dependencies]\n" + depLine + "\n", "", false
}

// dependencyLineKey extracts a quoted, literal-quoted or bare TOML key, only
// when the line is a simple key/value entry. The exact returned token prevents
// prefix matches ("a/b" must never match "a/b_extra").
func dependencyLineKey(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return ""
	}
	// TOML admits both basic ("k") and literal ('k') quoted keys. Handling only
	// the basic form silently fails to find a literal-quoted dependency, and the
	// insert that follows duplicates it.
	for _, q := range []byte{'"', '\''} {
		if trimmed[0] != q {
			continue
		}
		end := strings.IndexByte(trimmed[1:], q)
		if end < 0 {
			return ""
		}
		end++
		if !strings.HasPrefix(strings.TrimSpace(trimmed[end+1:]), "=") {
			return ""
		}
		return trimmed[1:end]
	}
	key, rest, ok := strings.Cut(trimmed, "=")
	if !ok || strings.TrimSpace(rest) == "" {
		return ""
	}
	return strings.TrimSpace(key)
}

// stripLineComment removes a trailing `#` comment that lies outside any quoted
// string. A section header may legally carry one — `[dependencies] # deps` — and
// an exact-string comparison against the raw line does not recognise it as the
// dependencies table at all, so the writer appends a SECOND [dependencies]
// table and the manifest stops parsing.
func stripLineComment(line string) string {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return line[:i]
		}
	}
	return line
}

// openMultilineString reports the delimiter of a multi-line string this line
// opens and does not close, or "" when the line leaves no string open.
func openMultilineString(line string) string {
	for _, d := range []string{`"""`, `'''`} {
		if n := strings.Count(line, d); n%2 == 1 {
			return d
		}
	}
	return ""
}

// writeManifestChecked writes the rewritten manifest, then refuses to leave the
// file less parseable than it found it. The upsert above is a line scanner over
// TOML, so its blind spots are open-ended by construction; this bounds them.
// A manifest that was ALREADY broken is not held against the write — only a
// manifest this call broke is rolled back.
func writeManifestChecked(dir, path string, original []byte, updated string) error {
	parsedBefore := true
	if _, err := pkg.LoadManifest(dir); err != nil {
		parsedBefore = false
	}
	if err := os.WriteFile(path, []byte(updated), 0644); err != nil {
		return err
	}
	if !parsedBefore {
		return nil
	}
	if _, err := pkg.LoadManifest(dir); err != nil {
		if restoreErr := os.WriteFile(path, original, 0644); restoreErr != nil {
			return fmt.Errorf("%s became unparseable and could not be restored: %w (restore failed: %v)", path, err, restoreErr)
		}
		return fmt.Errorf("refusing to write %s: the edit would make it unparseable (%w); the file is unchanged — please add the dependency by hand", path, err)
	}
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
