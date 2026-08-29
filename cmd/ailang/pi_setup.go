// pi_setup — `ailang pi install|uninstall|status` (M-DX-PI-HARNESS Distribution v2)
//
// The binary is the distribution channel for the pi extension suite: embed
// cmd/ailang/pi_assets (synced from .pi/extensions by `make pi-assets`) and
// materialize into ~/.pi/agent/extensions/ — pi's global dir, so the suite
// applies to every repo on the machine.
//
// Managed-file contract: .ailang-managed.json in the extensions dir records
// which files ailang installed (path → sha256 + binary version). Install
// NEVER clobbers files it does not own: a user-modified managed file or an
// unmanaged file with the same name is preserved and this binary's version is
// offered under .ailang-suggested/. Uninstall removes only managed files.
//
// Mirrors editor_vscode.go's install quality bar (version-managed, conflict
// handling) for the pi surface.
package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
)

//go:embed all:pi_assets
var piAssets embed.FS

const (
	piManagedFileName = ".ailang-managed.json"
	piSuggestedDir    = ".ailang-suggested"
)

type piManagedFile struct {
	SHA256  string `json:"sha256"`
	Version string `json:"binary_version"`
	Size    int64  `json:"size"`
}

type piManagedManifest struct {
	ManagedBy string                   `json:"managed_by"`
	Version   string                   `json:"ailang_version"`
	Files     map[string]piManagedFile `json:"files"`
}

// ── pure helpers (unit-tested) ───────────────────────────────────────────────

// decidePiInstall returns the action for one embedded file:
//   - "install"   absent on disk → write + record
//   - "adopt"     present, content identical to embedded, previously unmanaged → record only
//   - "current"   present, managed, content identical, same binary version → nothing
//   - "update"    managed content is unchanged since install and this binary ships a newer asset → replace safely
//   - "conflict-user-modified"  managed file whose content changed on disk → preserve, suggest
//   - "conflict-unmanaged"      present, unmanaged, different content → preserve, suggest
func decidePiInstall(
	name string,
	embedded []byte,
	diskHash string, // "" = absent on disk
	managed *piManagedFile,
	binaryVersion string,
) (action string, suggested bool) {
	embeddedHash := sha256Hex(embedded)
	if diskHash == "" {
		return "install", false
	}
	if diskHash == embeddedHash {
		if managed != nil {
			if managed.Version == binaryVersion {
				return "current", false
			}
			return "update", false
		}
		return "adopt", false
	}
	if managed != nil {
		if diskHash == managed.SHA256 {
			return "update", false
		}
		return "conflict-user-modified", true
	}
	return "conflict-unmanaged", true
}

// ── embedded assets + manifest IO ────────────────────────────────────────────

func piEmbeddedFiles() (map[string][]byte, []string, error) {
	out := map[string][]byte{}
	var names []string
	err := fs.WalkDir(piAssets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, "pi_assets/")
		if rel == "" {
			return nil
		}
		data, rerr := fs.ReadFile(piAssets, path)
		if rerr != nil {
			return rerr
		}
		out[rel] = data
		names = append(names, rel)
		return nil
	})
	sort.Strings(names)
	return out, names, err
}

func piExtensionsDir(home string) string { return filepath.Join(home, ".pi", "agent", "extensions") }
func piManifestPath(home string) string {
	return filepath.Join(piExtensionsDir(home), ".ailang-managed.json")
}
func piSuggestedPath(home, name string) string {
	return filepath.Join(piExtensionsDir(home), piSuggestedDir, name)
}

func readPiManaged(home string) map[string]piManagedFile {
	data, err := os.ReadFile(piManifestPath(home))
	if err != nil {
		return map[string]piManagedFile{}
	}
	var m struct {
		Files map[string]piManagedFile `json:"files"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return map[string]piManagedFile{}
	}
	if m.Files == nil {
		return map[string]piManagedFile{}
	}
	return m.Files
}

func writePiManaged(home string, files map[string]piManagedFile) error {
	manifest := piManagedManifest{
		ManagedBy: "ailang pi install — files listed here are updated/removed by `ailang pi install|uninstall`. Files NOT listed here are never touched.",
		Version:   Version,
		Files:     files,
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(piManifestPath(home), data, 0o644)
}

// ── commands ─────────────────────────────────────────────────────────────────

func installPiExtensions(home string, stdout, stderr io.Writer) error {
	extDir := piExtensionsDir(home)
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		return fmt.Errorf("cannot create %s: %w", extDir, err)
	}

	embedded, names, err := piEmbeddedFiles()
	if err != nil {
		return fmt.Errorf("embedded assets: %w", err)
	}
	managed := readPiManaged(home)

	var installed, updated, adopted, unchanged, conflicts int
	for _, name := range names {
		data := embedded[name]
		target := filepath.Join(extDir, name)
		diskHash, size, diskErr := diskHash(target)
		_ = size
		_ = diskErr // absent-on-disk is signalled by diskHash == ""

		var mf *piManagedFile
		if existing, ok := managed[name]; ok {
			mfCopy := existing
			mf = &mfCopy
		}

		action, suggested := decidePiInstall(name, data, diskHash, mf, Version)
		switch action {
		case "install":
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return fmt.Errorf("write %s: %w", target, err)
			}
			managed[name] = piManagedFile{SHA256: sha256Hex(data), Version: Version, Size: int64(len(data))}
			installed++
		case "adopt":
			managed[name] = piManagedFile{SHA256: sha256Hex(data), Version: Version, Size: int64(len(data))}
			adopted++
		case "current":
			unchanged++
		case "update":
			// The file still matches either the previous managed hash or this
			// binary's embedded asset, so replacing it cannot clobber user work.
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return fmt.Errorf("update %s: %w", target, err)
			}
			managed[name] = piManagedFile{SHA256: sha256Hex(data), Version: Version, Size: int64(len(data))}
			updated++
		case "conflict-user-modified", "conflict-unmanaged":
			if err := os.MkdirAll(filepath.Dir(piSuggestedPath(home, name)), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(piSuggestedPath(home, name), data, 0o644); err != nil {
				return fmt.Errorf("write suggestion %s: %w", piSuggestedPath(home, name), err)
			}
			conflicts++
			fmt.Fprintf(stderr, "%s %s — preserved; this binary's version written to %s\n",
				color.New(color.FgYellow).Sprintf("⚠"), name, piSuggestedPath(home, name))
		default:
			if suggested {
				fmt.Fprintf(stderr, "%s %s\n", color.New(color.FgYellow).Sprint("⚠"), action)
			}
		}
	}

	if err := writePiManaged(home, managed); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	fmt.Fprintf(stdout, "%s pi extensions in %s\n", green("✓"), extDir)
	fmt.Fprintf(stdout, "  installed: %d, updated: %d, adopted: %d, current: %d, conflicts preserved: %d\n",
		installed, updated, adopted, unchanged, conflicts)
	if installed+updated > 0 {
		fmt.Fprintf(stdout, "  %s restart pi sessions to load the refreshed extension set\n", cyan("ℹ"))
	}
	return nil
}

func uninstallPiExtensions(home string, stdout io.Writer) error {
	managed := readPiManaged(home)
	if len(managed) == 0 {
		fmt.Fprintln(stdout, "nothing to uninstall (no managed pi extension files)")
		return nil
	}
	var removed, preserved int
	for name, mf := range managed {
		target := filepath.Join(piExtensionsDir(home), name)
		diskHash, _, diskErr := diskHash(target)
		if diskErr == nil && diskHash != mf.SHA256 {
			// Modified after ailang installed it — do not delete user work.
			preserved++
			fmt.Fprintf(stdout, "  %s kept %s (modified since install — remove manually if intended)\n", yellow("⚠"), name)
			continue
		}
		if err := os.Remove(target); err == nil {
			removed++
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", target, err)
		}
	}
	if err := os.Remove(piManifestPath(home)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove manifest: %w", err)
	}
	fmt.Fprintf(stdout, "%s removed %d managed pi extension file(s); %d modified file(s) kept. Unmanaged files were left untouched.\n",
		green("✓"), removed, preserved)
	return nil
}

func statusPiExtensions(home string, stdout io.Writer) error {
	managed := readPiManaged(home)
	embedded, names, err := piEmbeddedFiles()
	if err != nil {
		return fmt.Errorf("embedded assets: %w", err)
	}

	fmt.Fprintf(stdout, "ailang pi extensions (binary %s)\n", Version)
	for _, name := range names {
		want := sha256Hex(embedded[name])
		target := filepath.Join(piExtensionsDir(home), name)
		diskHash, _, diskErr := diskHash(target)
		_ = diskHash
		_ = diskErr
		mf, wasManaged := managed[name]
		switch {
		case diskErr != nil:
			fmt.Fprintf(stdout, "  %s MISSING    %s\n", color.New(color.FgRed).Sprint("✗"), name)
		case !wasManaged:
			fmt.Fprintf(stdout, "  %s UNMANAGED  %s (present, not installed by ailang)\n", color.New(color.FgYellow).Sprint("⚠"), name)
		case diskHash != want:
			fmt.Fprintf(stdout, "  %s DRIFT      %s (modified locally, or installed by an older binary)\n", color.New(color.FgYellow).Sprint("⚠"), name)
		case mf.Version != Version:
			fmt.Fprintf(stdout, "  %s OLDER      %s (content current, installed by binary %s)\n", color.New(color.FgCyan).Sprint("ℹ"), name, mf.Version)
		default:
			fmt.Fprintf(stdout, "  %s FRESH      %s\n", green("✓"), name)
		}
	}

	// Foreign (user-owned) files in the managed dir — listed, never touched.
	entries, readErr := os.ReadDir(piExtensionsDir(home))
	if readErr != nil && !os.IsNotExist(readErr) {
		return fmt.Errorf("read extension directory: %w", readErr)
	}
	var foreign []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if _, managedByUs := managed[e.Name()]; !managedByUs {
			if _, isEmbedded := embedded[e.Name()]; !isEmbedded {
				foreign = append(foreign, e.Name())
			}
		}
	}
	if len(foreign) > 0 {
		sort.Strings(foreign)
		fmt.Fprintf(stdout, "  %s user files present (untouched): %s\n", cyan("ℹ"), strings.Join(foreign, ", "))
	}
	return nil
}

func piInstallCommand() {
	home, err := os.UserHomeDir()
	if err == nil {
		err = installPiExtensions(home, os.Stdout, os.Stderr)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func piUninstallCommand() {
	home, err := os.UserHomeDir()
	if err == nil {
		err = uninstallPiExtensions(home, os.Stdout)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func piStatusCommand() {
	home, err := os.UserHomeDir()
	if err == nil {
		err = statusPiExtensions(home, os.Stdout)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// ── disk helpers ─────────────────────────────────────────────────────────────

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func diskHash(path string) (hash string, size int64, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), int64(len(data)), nil
}

// piCommand dispatches `ailang pi install|uninstall|status`.
func piCommand() {
	if len(os.Args) < 3 {
		piUsage()
		os.Exit(1)
	}
	switch os.Args[2] {
	case "install":
		piInstallCommand()
	case "uninstall":
		piUninstallCommand()
	case "status":
		piStatusCommand()
	default:
		fmt.Fprintf(os.Stderr, "unknown 'ailang pi' subcommand %q\n", os.Args[2])
		piUsage()
		os.Exit(1)
	}
}

func piUsage() {
	fmt.Println("Usage: ailang pi <install|uninstall|status>")
	fmt.Println()
	fmt.Println("Manage the pi extension suite (session gate + dev-harness tools).")
	fmt.Println("Installs into ~/.pi/agent/extensions/ (global — every repo on this machine).")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  install    Materialize the embedded extensions (idempotent, version-stamped)")
	fmt.Println("  uninstall  Remove ailang-managed files (user files untouched)")
	fmt.Println("  status     Per-file freshness vs this binary")
}
