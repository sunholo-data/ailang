package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/pubsub"
)

func pkgPublishCommand(args []string) error {
	flagSet := flag.NewFlagSet("publish", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")
	dryRunFlag := flagSet.Bool("dry-run", false, "Create tarball but don't upload")
	allowDottedToolNames := flagSet.Bool("allow-dotted-tool-names", false,
		"Migration grace: downgrade the M-EXT-AUTHOR-DX naming gate from error to warning. Removed in v0.21.0.")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		fmt.Println("Usage: ailang publish [--dry-run] [--allow-dotted-tool-names]")
		fmt.Println()
		fmt.Println("Publish the current package to the AILANG registry.")
		fmt.Println("Reads ailang.toml, creates a tarball, and uploads to the validation service.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --dry-run                    Create tarball and show what would be published, without uploading")
		fmt.Println("  --allow-dotted-tool-names    Downgrade naming-gate errors to warnings (Bedrock-incompatible names")
		fmt.Println("                               like 'ctx.execute' rejected by default; one-cycle migration grace, removed in v0.21.0)")
		fmt.Println()
		fmt.Println("Registry: $AILANG_REGISTRY (default: https://storage.googleapis.com/ailang-registry)")
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

	fmt.Printf("Publishing %s@%s...\n", manifest.Package.Name, manifest.Package.Version)

	// Warn if ailang version constraint is missing
	if manifest.Package.AILANG == "" {
		constraint := pkg.FormatVersionConstraint(Version)
		fmt.Printf("%s No ailang version constraint in ailang.toml\n", yellow("⚠"))
		if constraint != "" {
			fmt.Printf("  Consider adding: ailang = %q\n", constraint)
		}
	}

	// Rewrite path deps to registry versions in ailang.toml before creating tarball.
	// Published packages must reference dependencies by registry version, not local paths.
	// We save and restore the original so the developer's local file isn't permanently changed.
	tomlPath := filepath.Join(cwd, pkg.ManifestFile)
	originalToml, _ := os.ReadFile(tomlPath)

	if pathDepsRewritten, err := rewritePathDepsForPublish(cwd, manifest); err != nil {
		return fmt.Errorf("failed to rewrite path deps: %w", err)
	} else if pathDepsRewritten {
		// Restore original after tarball creation (deferred)
		defer os.WriteFile(tomlPath, originalToml, 0644)
		// Reload manifest after rewrite
		manifest, err = pkg.LoadManifest(cwd)
		if err != nil {
			return fmt.Errorf("failed to reload manifest after path dep rewrite: %w", err)
		}
	}

	// M-EXT-PORTABILITY-GATE (v0.19.0): verify declared assets exist before
	// shipping a tarball whose runtime assetPath() lookups would all return Err.
	if err := pkg.VerifyDeclaredAssets(cwd, manifest); err != nil {
		return fmt.Errorf("asset validation failed: %w", err)
	}

	// M-EXT-PORTABILITY-GATE (v0.19.0): run pre-publish smoke test in a temp
	// dir so packages whose tools crash in an empty workdir are rejected at
	// publish time rather than discovered by consumers at runtime.
	if err := runPrePublishSmoke(cwd, manifest); err != nil {
		return err
	}

	// Create tarball (uses the rewritten ailang.toml with registry deps)
	tarballData, err := pkg.CreateTarball(cwd)
	if err != nil {
		return fmt.Errorf("failed to create package tarball: %w", err)
	}

	tarballHash := pkg.TarballHash(tarballData)
	contentHash, _ := pkg.ContentHash(cwd)
	interfaceHash := pkg.InterfaceHash(manifest)

	fmt.Printf("  Tarball: %d bytes (%s)\n", len(tarballData), tarballHash[:24]+"...")
	fmt.Printf("  Content hash: %s\n", contentHash[:24]+"...")
	fmt.Printf("  Interface hash: %s\n", interfaceHash[:24]+"...")
	fmt.Printf("  Exports: %v\n", manifest.Exports.Modules)
	fmt.Printf("  Effects: %v\n", manifest.Effects.Max)

	if *dryRunFlag {
		fmt.Printf("\n%s Dry run complete. Tarball ready but not uploaded.\n", yellow("⚠"))
		return nil
	}

	// Upload to validator
	validatorURL := registryValidatorURL()

	fmt.Printf("  Uploading to %s...\n", validatorURL)

	if err := uploadTarball(validatorURL+"/publish", tarballData, *allowDottedToolNames); err != nil {
		return err
	}

	fmt.Printf("%s Published %s@%s\n", green("✓"), manifest.Package.Name, manifest.Package.Version)

	// Auto-emit package coordination messages (M-PKG-MSG)
	emitPublishMessages(manifest, cwd, contentHash, interfaceHash)

	return nil
}

// runPrePublishSmoke executes the package's _smoke.ail in a temp directory
// (M-EXT-PORTABILITY-GATE, v0.19.0). On crash it blocks publish with a clear
// error; on absence it warns (or hard-fails for [extension] packages).
func runPrePublishSmoke(packageDir string, manifest *pkg.PackageManifest) error {
	smokePath := filepath.Join(packageDir, pkg.SmokeFile)
	smokePresent := false
	if info, err := os.Stat(smokePath); err == nil && !info.IsDir() {
		smokePresent = true
	}

	if !smokePresent {
		if pkg.HasExtensionBlock(manifest) {
			return fmt.Errorf("publish blocked: package declares [extension] but has no %s (required for extension packages)", pkg.SmokeFile)
		}
		fmt.Printf("%s no %s — publishing without smoke gate (recommended for v1.0+)\n", yellow("⚠"), pkg.SmokeFile)
		return nil
	}

	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate ailang binary for smoke run: %w", err)
	}

	timeout := pkg.DefaultSmokeTimeout
	if manifest != nil && manifest.Smoke.TimeoutSeconds > 0 {
		timeout = time.Duration(manifest.Smoke.TimeoutSeconds) * time.Second
	}
	fmt.Printf("  Running %s in temp workdir (timeout %s)...\n", pkg.SmokeFile, timeout)

	res, err := pkg.RunSmokeInTempDir(packageDir, bin, timeout)
	if err != nil {
		return fmt.Errorf("smoke runner failed: %w", err)
	}
	if !res.Passed {
		fmt.Printf("\n--- %s output ---\n%s\n--- end output ---\n\n", pkg.SmokeFile, res.Output)
		if res.TimedOut {
			return fmt.Errorf("publish blocked: %s timed out after %s", pkg.SmokeFile, res.Duration.Truncate(time.Millisecond))
		}
		return fmt.Errorf("publish blocked: %s failed with exit code %d (see output above)", pkg.SmokeFile, res.ExitCode)
	}
	fmt.Printf("%s %s passed (%.2fs)\n", green("✓"), pkg.SmokeFile, res.Duration.Seconds())
	return nil
}

// rewritePathDepsForPublish replaces path dependencies in ailang.toml with
// registry version strings. Published packages must not contain path deps
// since consumers won't have the local filesystem layout.
//
// Returns true if any deps were rewritten.
func rewritePathDepsForPublish(dir string, manifest *pkg.PackageManifest) (bool, error) {
	hasPathDeps := false
	for _, dep := range manifest.Dependencies {
		if dep.Path != "" {
			hasPathDeps = true
			break
		}
	}
	if !hasPathDeps {
		return false, nil
	}

	tomlPath := dir + "/" + pkg.ManifestFile
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return false, err
	}
	content := string(data)
	rewritten := false

	for depName, dep := range manifest.Dependencies {
		if dep.Path == "" {
			continue
		}
		// Look up the dep's version from its own ailang.toml (local path)
		depManifestPath := dep.Path
		if !filepath.IsAbs(depManifestPath) {
			depManifestPath = filepath.Join(dir, dep.Path)
		}
		depManifest, err := pkg.LoadManifest(depManifestPath)
		if err != nil {
			return false, fmt.Errorf("cannot read dependency %s at %s: %w\nPath deps must be resolvable at publish time", depName, dep.Path, err)
		}
		version := depManifest.Package.Version

		// Replace path dep with version dep.
		// Use a regex to tolerate any whitespace variant the author may have
		// used (single-space, column-aligned, extra inner-brace spaces, etc.).
		replacement := fmt.Sprintf(`"%s" = "%s"`, depName, version)
		pattern := fmt.Sprintf(`"%s"\s*=\s*\{\s*path\s*=\s*"%s"\s*\}`,
			regexp.QuoteMeta(depName), regexp.QuoteMeta(dep.Path))
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, fmt.Errorf("internal: bad rewrite pattern for %s: %w", depName, err)
		}
		if re.MatchString(content) {
			content = re.ReplaceAllLiteralString(content, replacement)
			fmt.Printf("  %s Rewrote dep %s: path %q → registry %s\n", cyan("→"), depName, dep.Path, version)
			rewritten = true
		}
	}

	if rewritten {
		if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
			return false, err
		}
	}
	return rewritten, nil
}

func uploadTarball(url string, tarballData []byte, allowDottedToolNames bool) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("package", "package.tar.gz")
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(tarballData); err != nil {
		return fmt.Errorf("failed to write tarball to form: %w", err)
	}
	if err := writer.Close(); err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// API key auth (from AILANG_REGISTRY_API_KEY env var)
	if apiKey := os.Getenv("AILANG_REGISTRY_API_KEY"); apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	// M-EXT-AUTHOR-DX M3 (v0.20.1): downgrade naming-gate errors to warnings
	// during the v0.20.x migration window for packages that still advertise
	// Bedrock-incompatible names (e.g. dotted aliases like "ctx.execute").
	if allowDottedToolNames {
		req.Header.Set("X-Allow-Dotted-Tool-Names", "true")
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		return fmt.Errorf("version already published (immutable). Bump the version in ailang.toml")
	case http.StatusBadRequest:
		return fmt.Errorf("validation failed:\n%s", string(body))
	case http.StatusForbidden:
		return fmt.Errorf("not authorized to publish to this namespace")
	default:
		return fmt.Errorf("publish failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
}

// emitPublishMessages auto-emits package coordination messages after a successful publish.
// This is best-effort — failures are logged but don't block the publish.
func emitPublishMessages(manifest *pkg.PackageManifest, cwd, contentHash, interfaceHash string) {
	store, err := openPkgMsgStore()
	if err != nil {
		return // No messaging store available
	}
	defer store.Close()

	pkgName := manifest.Package.Name
	newInfo := messaging.PackageVersionInfo{
		Name:          pkgName,
		Version:       manifest.Package.Version,
		InterfaceHash: interfaceHash,
		ContentHash:   contentHash,
		Effects:       manifest.Effects.Max,
		Exports:       manifest.Exports.Modules,
	}

	// Try to load previous version from lockfile
	var oldInfo messaging.PackageVersionInfo
	lf, err := pkg.LoadLockFile(cwd)
	if err == nil {
		for _, lp := range lf.Packages {
			if lp.Name == pkgName {
				oldInfo = messaging.PackageVersionInfo{
					Name:          lp.Name,
					Version:       lp.Version,
					InterfaceHash: lp.InterfaceHash,
					ContentHash:   lp.ContentHash,
					Effects:       lp.Effects,
					Exports:       lp.Exports,
				}
				break
			}
		}
	}

	recipients := []string{messaging.FormatPackageInbox(pkgName)}

	// Emit upgrade-available
	if msgID, err := messaging.EmitUpgradeAvailable(store, oldInfo, newInfo, recipients); err == nil && msgID != "" {
		fmt.Printf("%s Emitted upgrade-available (ID: %s)\n", green("✓"), msgID)
	}

	// Emit interface-change-notice if interface hash changed
	if msgID, err := messaging.EmitInterfaceChangeNotice(store, oldInfo, newInfo, recipients); err == nil && msgID != "" {
		fmt.Printf("%s Emitted interface-change-notice (ID: %s)\n", green("✓"), msgID)
	}

	// Emit effect-widening-warning if effects expanded
	if msgID, err := messaging.EmitEffectWideningWarning(store, pkgName, oldInfo, newInfo, recipients); err == nil && msgID != "" {
		fmt.Printf("%s Emitted effect-widening-warning (ID: %s)\n", yellow("⚠"), msgID)
	}

	// Supersede older messages
	if count, err := store.SupersedeOlderMessages(pkgName, manifest.Package.Version); err == nil && count > 0 {
		fmt.Printf("%s Superseded %d older message(s)\n", cyan("→"), count)
	}

	// Notify dependent packages (M-PKG-AUTONOMOUS-UPDATES)
	emitDependentNotifications(store, manifest, contentHash, interfaceHash)
}

// emitDependentNotifications queries the registry index for packages that depend on
// the just-published package and sends them upgrade-available messages.
// Best-effort — failures are logged but don't block the publish.
func emitDependentNotifications(store *messaging.Store, manifest *pkg.PackageManifest, contentHash, interfaceHash string) {
	client := pkg.NewRegistryClient()
	index, err := client.FetchIndex()
	if err != nil {
		// Registry unavailable — skip dependent notifications silently
		return
	}

	pkgName := manifest.Package.Name
	newVersion := manifest.Package.Version

	// Find all packages that list us as a dependency
	dependents := index.FindDependents(pkgName)

	if len(dependents) == 0 {
		return
	}

	// Determine change class from interface hash
	changeClass := "patch"
	// Check if interface hash changed (use the old info from lockfile if available)
	// Interface hash change → minor, same → patch
	lf, lfErr := pkg.LoadLockFile(".")
	if lfErr == nil {
		for _, lp := range lf.Packages {
			if lp.Name == pkgName && lp.InterfaceHash != "" && lp.InterfaceHash != interfaceHash {
				changeClass = "minor"
				break
			}
		}
	}

	// M-PKG-AUTONOMOUS-CASCADE-SAFE M2: Set up the cascade-topic publisher.
	// We dual-write: the legacy inbox path (for backward compat — agents
	// that still poll inboxes will see it but won't act per the M1 template
	// guard), AND the new ailang-cascade Pub/Sub topic with source=cascade
	// attribute (which triggers the agent to actually do the bump).
	//
	// Both writes are best-effort and logged. A failure on either side
	// doesn't abort the publish — the publish has already succeeded; this
	// is dependent notification.
	cascadePublisher := newCascadePublisher()
	defer func() {
		if cascadePublisher != nil {
			cascadePublisher.Stop()
		}
	}()

	rootRef := pkgName + "@" + newVersion

	// Look up previous version + its interface hash from the registry index.
	// Required by the messaging schema validator (upgrade-available demands
	// from_version + from_interface_hash). We pull the second-most-recent
	// version from the index entry and fetch its metadata.json for the hash.
	oldInfo := messaging.PackageVersionInfo{Name: pkgName}
	for _, e := range index.Packages {
		if e.Name != pkgName {
			continue
		}
		// Versions list contains the just-published version too. Find the
		// most recent version that isn't the new one.
		for i := len(e.Versions) - 1; i >= 0; i-- {
			if e.Versions[i] == newVersion {
				continue
			}
			oldInfo.Version = e.Versions[i]
			break
		}
		break
	}
	if oldInfo.Version != "" {
		if oldMeta, mErr := client.FetchMetadata(pkgName, oldInfo.Version); mErr == nil && oldMeta != nil {
			oldInfo.InterfaceHash = oldMeta.InterfHash
			oldInfo.ContentHash = oldMeta.ContentHash
			// M-PKG-CASCADE-DETERMINISTIC-FIRST: also pull effects + exports
			// so the cascade envelope can carry the old/new ceiling deltas
			// for the cloud coordinator's deterministic dispatcher.
			oldInfo.Effects = oldMeta.Manifest.EffectsMax
			oldInfo.Exports = oldMeta.Manifest.Exports
		}
	}

	// Populate the new dep's Effects + Exports from the manifest we just published.
	depEffects := manifest.Effects.Max
	depExports := manifest.Exports.Modules

	for _, depName := range dependents {
		recipients := []string{messaging.FormatPackageInbox(depName)}
		depInfo := messaging.PackageVersionInfo{
			Name:          pkgName,
			Version:       newVersion,
			InterfaceHash: interfaceHash,
			ContentHash:   contentHash,
			Effects:       depEffects,
			Exports:       depExports,
		}
		msgID, err := messaging.EmitUpgradeAvailable(store, oldInfo, depInfo, recipients)
		if err == nil && msgID != "" {
			fmt.Printf("%s Notified dependent %s (ID: %s, class: %s, legacy inbox)\n", cyan("→"), depName, msgID, changeClass)
		} else if err != nil {
			fmt.Printf("%s Legacy inbox notify failed for %s: %v\n", yellow("⚠"), depName, err)
		}

		// Cascade-topic publish (M-PKG-AUTONOMOUS-CASCADE-SAFE M2).
		// Decoupled from the legacy inbox path: this is the authoritative,
		// IAM-restricted path that agents act on. If legacy inbox failed,
		// fabricate a stable correlation ID so the cascade still fires.
		cascadeMsgID := msgID
		if cascadeMsgID == "" {
			cascadeMsgID = fmt.Sprintf("cascade-%s-%s", strings.ReplaceAll(rootRef, "/", "_"), strings.ReplaceAll(depName, "/", "_"))
		}
		if cascadePublisher != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			attrs := pubsub.MessageAttributes{
				Inbox:       messaging.FormatPackageInbox(depName),
				FromAgent:   "coordinator",
				Category:    changeClass,
				MessageType: "upgrade-available",
			}
			// M-PKG-CASCADE-DETERMINISTIC-FIRST: pass the full envelope so the
			// cloud coordinator can decide deterministic-bump vs AI-escalation
			// without fetching from a separate store. Map our local change-class
			// labels to the schema's A/B/C taxonomy.
			schemaClass := mapChangeClassToSchema(changeClass, oldInfo, depInfo)
			env := &pubsub.CascadeEnvelopeFields{
				RootPackage:       rootRef,
				ChangeClass:       schemaClass,
				FromVersion:       oldInfo.Version,
				ToVersion:         depInfo.Version,
				FromInterfaceHash: oldInfo.InterfaceHash,
				ToInterfaceHash:   depInfo.InterfaceHash,
				FromContentHash:   oldInfo.ContentHash,
				ToContentHash:     depInfo.ContentHash,
				EffectsWidened:    effectsWidened(oldInfo.Effects, depInfo.Effects),
				PrevEffectCeiling: oldInfo.Effects,
				NewEffectCeiling:  depInfo.Effects,
			}
			if cascadeErr := cascadePublisher.PublishCascadeWithEnvelope(ctx, cascadeMsgID, attrs, env); cascadeErr != nil {
				fmt.Printf("%s Cascade publish failed for %s: %v\n", yellow("⚠"), depName, cascadeErr)
			} else {
				fmt.Printf("%s Cascade-topic notification published for %s (root=%s, class=%s)\n", cyan("→"), depName, rootRef, schemaClass)
			}
			cancel()
		}
	}
}

// mapChangeClassToSchema converts the publish-flow change class label
// ("patch"/"minor"/"major") to the M-PKG-MSG schema taxonomy ("A"/"B"/"C")
// the cloud coordinator's deterministic dispatcher uses.
//
// Mapping:
//
//	A → content change only (interface hash unchanged)
//	B → interface changed AND new module exports added (purely additive at module level)
//	C → interface hash changed but module exports unchanged or shrunk
//	    (a function may have been removed/changed; without function-level diff,
//	    treat conservatively as breaking so the AI is invoked to verify)
//
// The conservative C-on-any-interface-change rule mirrors the original
// classifyChange in internal/messaging/pkg_events.go, with B reserved for
// the rare module-add case the wrapper can prove safe.
func mapChangeClassToSchema(localClass string, oldInfo, newInfo messaging.PackageVersionInfo) string {
	if oldInfo.InterfaceHash == "" || oldInfo.InterfaceHash == newInfo.InterfaceHash {
		return "A" // Content-only change
	}
	// Effects widening is always breaking (consumers may not declare new effects).
	if effectsWidened(oldInfo.Effects, newInfo.Effects) {
		return "C"
	}
	// Module-level export removal is definitely breaking.
	if exportsRemoved(oldInfo.Exports, newInfo.Exports) {
		return "C"
	}
	// New modules added with no removals → additive at module level.
	// Note: this is permissive — a new module could still hide a function-level
	// removal in an unchanged module. For now we trust the +module case.
	if len(newInfo.Exports) > len(oldInfo.Exports) {
		return "B"
	}
	// Interface hash changed but module exports list is unchanged. We cannot
	// distinguish "added function" from "removed function" without a finer
	// interface diff. Conservative: assume breaking → AI escalates and either
	// confirms additive (PR opens with no consumer changes) or repairs.
	return "C"
}

// effectsWidened mirrors messaging.effectsWidened (which is unexported).
// Returns true when newEffects contains any effect not present in oldEffects.
func effectsWidened(oldEffects, newEffects []string) bool {
	if len(oldEffects) == 0 && len(newEffects) > 0 {
		return true
	}
	old := make(map[string]bool, len(oldEffects))
	for _, e := range oldEffects {
		old[e] = true
	}
	for _, e := range newEffects {
		if !old[e] {
			return true
		}
	}
	return false
}

// exportsRemoved returns true when any export from oldExports is missing
// from newExports (a removal — definitely breaking).
func exportsRemoved(oldExports, newExports []string) bool {
	newSet := make(map[string]bool, len(newExports))
	for _, e := range newExports {
		newSet[e] = true
	}
	for _, e := range oldExports {
		if !newSet[e] {
			return true
		}
	}
	return false
}

// newCascadePublisher constructs a Pub/Sub publisher targeting the cascade
// topic. Returns nil (with no error logged) when running outside cloud mode
// or when AILANG_CLOUD_PROJECT is unset — the legacy inbox path still fires
// in those cases. This keeps `ailang publish` working unchanged for local
// laptop use.
func newCascadePublisher() *pubsub.Publisher {
	projectID := os.Getenv("AILANG_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}
	if projectID == "" {
		return nil
	}
	prefix := os.Getenv("AILANG_TOPIC_PREFIX")
	if prefix == "" {
		prefix = pubsub.DefaultTopicPrefix
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := pubsub.NewClient(ctx, projectID, prefix)
	if err != nil {
		fmt.Printf("%s Cascade publisher init failed: %v (continuing without cascade topic)\n", yellow("⚠"), err)
		return nil
	}
	return pubsub.NewPublisher(client)
}
