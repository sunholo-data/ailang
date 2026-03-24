package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/pkg"
)

func pkgPublishCommand(args []string) error {
	flagSet := flag.NewFlagSet("publish", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")
	dryRunFlag := flagSet.Bool("dry-run", false, "Create tarball but don't upload")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		fmt.Println("Usage: ailang publish [--dry-run]")
		fmt.Println()
		fmt.Println("Publish the current package to the AILANG registry.")
		fmt.Println("Reads ailang.toml, creates a tarball, and uploads to the validation service.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --dry-run    Create tarball and show what would be published, without uploading")
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
	validatorURL := os.Getenv("AILANG_REGISTRY_VALIDATOR")
	if validatorURL == "" {
		return fmt.Errorf("AILANG_REGISTRY_VALIDATOR not set\nSet the Cloud Run validator URL: export AILANG_REGISTRY_VALIDATOR=https://ailang-registry-validator-XXXX.run.app")
	}

	fmt.Printf("  Uploading to %s...\n", validatorURL)

	if err := uploadTarball(validatorURL+"/publish", tarballData); err != nil {
		return err
	}

	fmt.Printf("%s Published %s@%s\n", green("✓"), manifest.Package.Name, manifest.Package.Version)

	// Auto-emit package coordination messages (M-PKG-MSG)
	emitPublishMessages(manifest, cwd, contentHash, interfaceHash)

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

		// Replace path dep with version dep
		old := fmt.Sprintf(`"%s" = { path = "%s" }`, depName, dep.Path)
		replacement := fmt.Sprintf(`"%s" = "%s"`, depName, version)
		if strings.Contains(content, old) {
			content = strings.Replace(content, old, replacement, 1)
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

func uploadTarball(url string, tarballData []byte) error {
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
}
