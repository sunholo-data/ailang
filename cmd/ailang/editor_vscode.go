package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// VS Code stopped treating ~/.vscode/extensions/extensions.json as a cache in
// 1.74 (profiles): it is now the registry of which user extensions are
// installed. Two consequences drive everything in this file:
//
//   - Dropping a folder into ~/.vscode/extensions without a matching registry
//     entry installs nothing. VS Code never scans it.
//   - REMOVING an entry is an uninstall. It makes VS Code flag the leftover
//     folder in .obsolete and skip it in every window.
//
// So the primary install path hands a VSIX to the editor's own CLI and lets
// VS Code do the registration. The folder fallback exists only for machines
// with no editor CLI on PATH, and it WRITES a registry entry — it never
// removes one.

// vscodeCLIs is the probe order for an editor CLI that can install a VSIX.
// All are VS Code derivatives and accept --install-extension.
var vscodeCLIs = []string{"code", "code-insiders", "cursor", "windsurf", "codium"}

// vscodeAssets maps each embedded asset to its path inside the VSIX, relative
// to the archive's extension/ directory.
//
// These are ZIP entry names, so they are always forward-slash separated —
// filepath.Join would emit backslashes on Windows and produce an archive whose
// entries VS Code cannot find.
var vscodeAssets = []struct{ src, rel string }{
	{"editor_assets/vscode/package.json", "package.json"},
	{"editor_assets/vscode/language-configuration.json", "language-configuration.json"},
	{"editor_assets/vscode/extension.js", "extension.js"},
	{"editor_assets/vscode/syntaxes/ailang.tmLanguage.json", "syntaxes/ailang.tmLanguage.json"},
}

// vscodeManifest is the subset of the extension's package.json that the
// installer needs. The embedded package.json is the single source of truth for
// the extension's identity and version — nothing here is hardcoded.
type vscodeManifest struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Publisher   string `json:"publisher"`
	Engines     struct {
		VSCode string `json:"vscode"`
	} `json:"engines"`
}

// extID is the identifier VS Code keys the registry by, e.g. "sunholo.ailang".
func (m vscodeManifest) extID() string { return m.Publisher + "." + m.Name }

// folder is the on-disk directory name VS Code uses for an installed
// extension, e.g. "sunholo.ailang-0.3.0". It doubles as relativeLocation in
// the registry, so the two can never disagree.
func (m vscodeManifest) folder() string { return m.extID() + "-" + m.Version }

// readVSCodeManifest parses the embedded extension package.json.
//
// A missing name/publisher/version is fatal rather than defaulted: those three
// fields decide the extension ID and the versioned folder name, and an install
// that guessed them would land somewhere VS Code does not look.
func readVSCodeManifest() (vscodeManifest, error) {
	var m vscodeManifest
	data, err := editorAssets.ReadFile("editor_assets/vscode/package.json")
	if err != nil {
		return m, fmt.Errorf("read embedded package.json: %w", err)
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("parse embedded package.json: %w", err)
	}
	var missing []string
	if m.Name == "" {
		missing = append(missing, "name")
	}
	if m.Publisher == "" {
		missing = append(missing, "publisher")
	}
	if m.Version == "" {
		missing = append(missing, "version")
	}
	if len(missing) > 0 {
		return m, fmt.Errorf("embedded package.json is missing %s", strings.Join(missing, ", "))
	}
	return m, nil
}

// --- VSIX packaging -------------------------------------------------------

// A VSIX is an OPC zip. VS Code needs three things at the root: the content
// types part, the manifest, and the extension payload under extension/.

const vsixContentTypes = `<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="json" ContentType="application/json"/>
  <Default Extension="vsixmanifest" ContentType="text/xml"/>
  <Default Extension="js" ContentType="application/javascript"/>
  <Default Extension="xml" ContentType="text/xml"/>
</Types>
`

// vsixManifestXML renders extension.vsixmanifest for the given manifest.
func vsixManifestXML(m vscodeManifest) ([]byte, error) {
	esc := func(s string) (string, error) {
		var b bytes.Buffer
		if err := xml.EscapeText(&b, []byte(s)); err != nil {
			return "", err
		}
		return b.String(), nil
	}
	display, err := esc(m.DisplayName)
	if err != nil {
		return nil, fmt.Errorf("escape displayName: %w", err)
	}
	desc, err := esc(m.Description)
	if err != nil {
		return nil, fmt.Errorf("escape description: %w", err)
	}
	engine, err := esc(m.Engines.VSCode)
	if err != nil {
		return nil, fmt.Errorf("escape engine: %w", err)
	}
	id, err := esc(m.Name)
	if err != nil {
		return nil, fmt.Errorf("escape name: %w", err)
	}
	publisher, err := esc(m.Publisher)
	if err != nil {
		return nil, fmt.Errorf("escape publisher: %w", err)
	}
	version, err := esc(m.Version)
	if err != nil {
		return nil, fmt.Errorf("escape version: %w", err)
	}
	if engine == "" {
		engine = "*"
	}

	var b bytes.Buffer
	fmt.Fprint(&b, xml.Header)
	fmt.Fprint(&b, `<PackageManifest Version="2.0.0" xmlns="http://schemas.microsoft.com/developer/vsx-schema/2011">`+"\n")
	fmt.Fprint(&b, "  <Metadata>\n")
	fmt.Fprintf(&b, "    <Identity Language=\"en-US\" Id=%q Version=%q Publisher=%q/>\n", id, version, publisher)
	fmt.Fprintf(&b, "    <DisplayName>%s</DisplayName>\n", display)
	fmt.Fprintf(&b, "    <Description xml:space=\"preserve\">%s</Description>\n", desc)
	fmt.Fprint(&b, "  </Metadata>\n")
	fmt.Fprint(&b, "  <Installation>\n")
	fmt.Fprint(&b, "    <InstallationTarget Id=\"Microsoft.VisualStudio.Code\"/>\n")
	fmt.Fprint(&b, "  </Installation>\n")
	fmt.Fprint(&b, "  <Dependencies/>\n")
	fmt.Fprint(&b, "  <Assets>\n")
	fmt.Fprint(&b, "    <Asset Type=\"Microsoft.VisualStudio.Code.Manifest\" Path=\"extension/package.json\" Addressable=\"true\"/>\n")
	fmt.Fprint(&b, "  </Assets>\n")
	fmt.Fprintf(&b, "  <Properties>\n    <Property Id=\"Microsoft.VisualStudio.Code.Engine\" Value=%q/>\n  </Properties>\n", engine)
	fmt.Fprint(&b, "</PackageManifest>\n")
	return b.Bytes(), nil
}

// buildVSIX assembles the installable archive in memory from the embedded
// assets. Go's archive/zip is enough — no node or vsce dependency.
func buildVSIX(m vscodeManifest) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	add := func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("create vsix entry %s: %w", name, err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write vsix entry %s: %w", name, err)
		}
		return nil
	}

	if err := add("[Content_Types].xml", []byte(vsixContentTypes)); err != nil {
		return nil, err
	}
	manifest, err := vsixManifestXML(m)
	if err != nil {
		return nil, err
	}
	if err := add("extension.vsixmanifest", manifest); err != nil {
		return nil, err
	}
	for _, a := range vscodeAssets {
		data, err := editorAssets.ReadFile(a.src)
		if err != nil {
			return nil, fmt.Errorf("read embedded %s: %w", a.src, err)
		}
		if err := add("extension/"+a.rel, data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("finalize vsix: %w", err)
	}
	return buf.Bytes(), nil
}

// --- registry (extensions.json) -------------------------------------------

func vscodeExtRoot(home string) string { return filepath.Join(home, ".vscode", "extensions") }
func vscodeRegistryPath(home string) string {
	return filepath.Join(vscodeExtRoot(home), "extensions.json")
}
func vscodeObsoletePath(home string) string {
	return filepath.Join(vscodeExtRoot(home), ".obsolete")
}

// readVSCodeRegistry loads extensions.json as raw maps so that entries we do
// not own keep every field they arrived with.
func readVSCodeRegistry(path string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var entries []map[string]interface{}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return entries, nil
}

func writeVSCodeRegistry(path string, entries []map[string]interface{}) error {
	if entries == nil {
		entries = []map[string]interface{}{}
	}
	out, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("encode registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	// Atomic write so a crash mid-write cannot corrupt the registry — a
	// truncated extensions.json would deregister every extension on the box.
	tmp := path + ".ailang-tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// registryEntryID extracts an entry's extension ID, tolerating both the
// nested identifier.id form VS Code writes and a flat id.
func registryEntryID(e map[string]interface{}) string {
	if ident, ok := e["identifier"].(map[string]interface{}); ok {
		if s, ok := ident["id"].(string); ok && s != "" {
			return s
		}
	}
	if s, ok := e["id"].(string); ok {
		return s
	}
	return ""
}

// vscodeURIPath renders an absolute filesystem path the way VS Code stores it
// in location.path: a URI path, so a Windows drive letter is preceded by "/".
func vscodeURIPath(abs string) string {
	slashed := filepath.ToSlash(abs)
	if !strings.HasPrefix(slashed, "/") {
		return "/" + slashed
	}
	return slashed
}

// registerVSCodeExtension adds or replaces this extension's registry entry.
//
// This is the inverse of deregisterVSCodeExtension and the only correct thing
// to do on an install path: without an entry, the extension folder is inert.
func registerVSCodeExtension(home string, m vscodeManifest, installedAtMillis int64) error {
	path := vscodeRegistryPath(home)
	entries, err := readVSCodeRegistry(path)
	if err != nil {
		return err
	}
	want := strings.ToLower(m.extID())
	kept := make([]map[string]interface{}, 0, len(entries)+1)
	for _, e := range entries {
		if strings.EqualFold(registryEntryID(e), want) {
			continue // replaced below
		}
		kept = append(kept, e)
	}
	abs := filepath.Join(vscodeExtRoot(home), m.folder())
	kept = append(kept, map[string]interface{}{
		"identifier": map[string]interface{}{"id": m.extID()},
		"version":    m.Version,
		"location": map[string]interface{}{
			"$mid":   1,
			"path":   vscodeURIPath(abs),
			"scheme": "file",
		},
		"relativeLocation": m.folder(),
		"metadata": map[string]interface{}{
			"installedTimestamp":   installedAtMillis,
			"source":               "vsix",
			"publisherDisplayName": m.Publisher,
			"targetPlatform":       "undefined",
			"updated":              false,
			"private":              false,
			"isPreReleaseVersion":  false,
			"hasPreReleaseVersion": false,
		},
	})
	return writeVSCodeRegistry(path, kept)
}

// deregisterVSCodeExtension removes the extension's entry from
// extensions.json.
//
// Naming matters here: an earlier version of this function was called
// invalidateVSCodeExtensionCache and was called on the INSTALL path to "force a
// metadata re-scan". Since VS Code 1.74 removing an entry is an uninstall, so
// that call was deregistering the extension it had just written. Only
// uninstall may call this.
//
// Returns (true, nil) when an entry was removed, (false, nil) when none
// matched.
func deregisterVSCodeExtension(home, extID string) (bool, error) {
	path := vscodeRegistryPath(home)
	entries, err := readVSCodeRegistry(path)
	if err != nil {
		return false, err
	}
	if entries == nil {
		return false, nil
	}
	want := strings.ToLower(extID)
	kept := make([]map[string]interface{}, 0, len(entries))
	removed := false
	for _, e := range entries {
		if strings.EqualFold(registryEntryID(e), want) {
			removed = true
			continue
		}
		kept = append(kept, e)
	}
	if !removed {
		return false, nil
	}
	if err := writeVSCodeRegistry(path, kept); err != nil {
		return false, err
	}
	return true, nil
}

// clearVSCodeObsolete drops folder names from .obsolete, the list VS Code uses
// to skip directories it believes were uninstalled. A folder we are installing
// into must not be on it.
func clearVSCodeObsolete(home string, folders ...string) (bool, error) {
	path := vscodeObsoletePath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	var flags map[string]bool
	if err := json.Unmarshal(data, &flags); err != nil {
		return false, fmt.Errorf("parse %s: %w", path, err)
	}
	changed := false
	for _, f := range folders {
		if _, ok := flags[f]; ok {
			delete(flags, f)
			changed = true
		}
	}
	if !changed {
		return false, nil
	}
	out, err := json.Marshal(flags)
	if err != nil {
		return false, fmt.Errorf("encode obsolete: %w", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return false, fmt.Errorf("write %s: %w", path, err)
	}
	return true, nil
}

// --- install / uninstall / status -----------------------------------------

// lookPathFunc and cmdRunner are seams so the install logic can be exercised
// without a real editor on the box.
type lookPathFunc func(string) (string, error)
type cmdRunner func(name string, args ...string) ([]byte, error)

func execRunner(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return out, err
}

// findVSCodeCLI returns the first editor CLI found on PATH.
func findVSCodeCLI(look lookPathFunc) (name, path string, found bool) {
	for _, c := range vscodeCLIs {
		if p, err := look(c); err == nil && p != "" {
			return c, p, true
		}
	}
	return "", "", false
}

// vscodeInstallResult describes what an install actually did, so the caller
// can report it truthfully rather than assuming success.
type vscodeInstallResult struct {
	Method       string // "cli" or "folder"
	CLIName      string // editor CLI used, when Method == "cli"
	Folder       string // extension folder, when Method == "folder"
	LegacyPurged bool   // the pre-fix unversioned ~/.vscode/extensions/ailang was removed
	Warnings     []string
}

// legacyVSCodeFolder is the unversioned directory pre-fix builds installed
// into. VS Code never registered it, so it is dead weight at best and an
// ID conflict with a real install at worst.
const legacyVSCodeFolder = "ailang"

// installVSCodeExtension installs the extension and returns what it did.
//
// Order: hand a VSIX to the editor's own CLI if there is one (VS Code then
// owns registration, which is the only way to be sure it is correct), else
// write a versioned folder AND its registry entry.
func installVSCodeExtension(home string, m vscodeManifest, look lookPathFunc, run cmdRunner, installedAtMillis int64) (vscodeInstallResult, error) {
	var res vscodeInstallResult

	// The pre-fix install site. Removing it is safe precisely because VS Code
	// never registered it.
	legacy := filepath.Join(vscodeExtRoot(home), legacyVSCodeFolder)
	if _, err := os.Stat(legacy); err == nil {
		if err := os.RemoveAll(legacy); err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("could not remove stale %s: %v", legacy, err))
		} else {
			res.LegacyPurged = true
		}
	}
	if _, err := clearVSCodeObsolete(home, legacyVSCodeFolder, m.folder()); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("could not update .obsolete: %v", err))
	}

	if cliName, cliPath, ok := findVSCodeCLI(look); ok {
		vsix, err := buildVSIX(m)
		if err != nil {
			return res, fmt.Errorf("build vsix: %w", err)
		}
		tmp, err := os.CreateTemp("", m.folder()+"-*.vsix")
		if err != nil {
			return res, fmt.Errorf("create temp vsix: %w", err)
		}
		tmpName := tmp.Name()
		defer func() { _ = os.Remove(tmpName) }()
		if _, err := tmp.Write(vsix); err != nil {
			_ = tmp.Close()
			return res, fmt.Errorf("write temp vsix: %w", err)
		}
		if err := tmp.Close(); err != nil {
			return res, fmt.Errorf("close temp vsix: %w", err)
		}
		out, err := run(cliPath, "--install-extension", tmpName, "--force")
		if err == nil {
			res.Method = "cli"
			res.CLIName = cliName
			return res, nil
		}
		// Fall through to the folder install, but say why — a silent
		// downgrade is how the original defect stayed invisible.
		res.Warnings = append(res.Warnings, fmt.Sprintf("%s --install-extension failed (%v); falling back to a direct install: %s",
			cliName, err, strings.TrimSpace(string(out))))
	}

	folder := m.folder()
	extDir := filepath.Join(vscodeExtRoot(home), folder)
	if err := os.RemoveAll(extDir); err != nil {
		return res, fmt.Errorf("remove previous %s: %w", extDir, err)
	}
	for _, a := range vscodeAssets {
		data, err := editorAssets.ReadFile(a.src)
		if err != nil {
			return res, fmt.Errorf("read embedded %s: %w", a.src, err)
		}
		dst := filepath.Join(extDir, filepath.FromSlash(a.rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return res, fmt.Errorf("create %s: %w", filepath.Dir(dst), err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return res, fmt.Errorf("write %s: %w", dst, err)
		}
	}
	if err := registerVSCodeExtension(home, m, installedAtMillis); err != nil {
		return res, fmt.Errorf("register extension: %w", err)
	}
	res.Method = "folder"
	res.Folder = folder
	return res, nil
}

// vscodeStatus is what `ailang editor status` reports for VS Code.
type vscodeStatus struct {
	Installed bool
	Version   string
	Detail    string
}

// vscodeExtensionStatus reports whether VS Code will actually load the
// extension.
//
// It asks the editor CLI when there is one, and otherwise reads the registry —
// never a bare os.Stat of the folder, which is what let a dead install report
// a green checkmark.
func vscodeExtensionStatus(home string, m vscodeManifest, look lookPathFunc, run cmdRunner) vscodeStatus {
	if cliName, cliPath, ok := findVSCodeCLI(look); ok {
		out, err := run(cliPath, "--list-extensions", "--show-versions")
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(line)
				id, version, _ := strings.Cut(line, "@")
				if strings.EqualFold(id, m.extID()) {
					return vscodeStatus{Installed: true, Version: version, Detail: "registered with " + cliName}
				}
			}
			return vscodeStatus{Detail: "not installed (" + cliName + " --list-extensions)"}
		}
		// CLI present but unusable — fall through to the registry rather than
		// reporting a state we did not establish.
	}

	entries, err := readVSCodeRegistry(vscodeRegistryPath(home))
	if err != nil {
		return vscodeStatus{Detail: fmt.Sprintf("cannot read extensions.json: %v", err)}
	}
	for _, e := range entries {
		if !strings.EqualFold(registryEntryID(e), m.extID()) {
			continue
		}
		version, _ := e["version"].(string)
		rel, _ := e["relativeLocation"].(string)
		if rel == "" {
			return vscodeStatus{Detail: "registry entry has no relativeLocation"}
		}
		dir := filepath.Join(vscodeExtRoot(home), rel)
		if _, err := os.Stat(dir); err != nil {
			return vscodeStatus{Detail: "registered as " + rel + " but that folder is missing"}
		}
		if obsolete, err := vscodeFolderIsObsolete(home, rel); err == nil && obsolete {
			return vscodeStatus{Detail: rel + " is flagged obsolete — reinstall"}
		}
		return vscodeStatus{Installed: true, Version: version, Detail: dir}
	}

	// A folder with no registry entry is precisely the broken state this
	// command used to produce, so name it rather than reporting "not
	// installed".
	for _, cand := range []string{legacyVSCodeFolder, m.folder()} {
		if _, err := os.Stat(filepath.Join(vscodeExtRoot(home), cand)); err == nil {
			return vscodeStatus{Detail: cand + " exists but is not registered in extensions.json — run: ailang editor install vscode"}
		}
	}
	return vscodeStatus{Detail: "not installed"}
}

func vscodeFolderIsObsolete(home, folder string) (bool, error) {
	data, err := os.ReadFile(vscodeObsoletePath(home))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	var flags map[string]bool
	if err := json.Unmarshal(data, &flags); err != nil {
		return false, err
	}
	return flags[folder], nil
}

// vscodeUninstallResult records what uninstall removed.
type vscodeUninstallResult struct {
	CLIName      string
	Deregistered bool
	Folders      []string
	Warnings     []string
}

// uninstallVSCodeExtension removes the extension by every route it could have
// been installed by: the editor CLI, the registry entry, and any on-disk
// folder — including versions other than the one this binary ships.
func uninstallVSCodeExtension(home string, m vscodeManifest, look lookPathFunc, run cmdRunner) (vscodeUninstallResult, error) {
	var res vscodeUninstallResult

	if cliName, cliPath, ok := findVSCodeCLI(look); ok {
		if out, err := run(cliPath, "--uninstall-extension", m.extID()); err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("%s --uninstall-extension: %v: %s", cliName, err, strings.TrimSpace(string(out))))
		} else {
			res.CLIName = cliName
		}
	}

	// Removing the registry entry is correct HERE and only here.
	removed, err := deregisterVSCodeExtension(home, m.extID())
	if err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("could not update extensions.json: %v", err))
	}
	res.Deregistered = removed

	root := vscodeExtRoot(home)
	prefix := m.extID() + "-"
	var doomed []string
	if entries, err := os.ReadDir(root); err == nil {
		for _, e := range entries {
			name := e.Name()
			if name == legacyVSCodeFolder || strings.HasPrefix(name, prefix) {
				doomed = append(doomed, name)
			}
		}
	}
	sort.Strings(doomed)
	for _, name := range doomed {
		if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
			res.Warnings = append(res.Warnings, fmt.Sprintf("could not remove %s: %v", name, err))
			continue
		}
		res.Folders = append(res.Folders, name)
	}
	if _, err := clearVSCodeObsolete(home, doomed...); err != nil {
		res.Warnings = append(res.Warnings, fmt.Sprintf("could not update .obsolete: %v", err))
	}
	return res, nil
}

// --- command entry points -------------------------------------------------

func installVSCode() {
	fmt.Printf("%s Installing AILANG VS Code extension...\n", cyan("→"))

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot determine home directory: %v\n", red("Error"), err)
		os.Exit(1)
	}
	m, err := readVSCodeManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	res, err := installVSCodeExtension(home, m, exec.LookPath, execRunner, time.Now().UnixMilli())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if res.LegacyPurged {
		fmt.Printf("  %s removed the old unregistered install at ~/.vscode/extensions/%s\n", green("✓"), legacyVSCodeFolder)
	}
	for _, w := range res.Warnings {
		fmt.Printf("  %s %s\n", yellow("⚠"), w)
	}

	switch res.Method {
	case "cli":
		fmt.Printf("%s Installed %s@%s via `%s --install-extension`\n", green("✓"), m.extID(), m.Version, res.CLIName)
	default:
		fmt.Printf("%s Installed %s@%s to ~/.vscode/extensions/%s and registered it in extensions.json\n",
			green("✓"), m.extID(), m.Version, res.Folder)
		fmt.Printf("  %s no editor CLI (%s) on PATH — installed directly.\n", cyan("ℹ"), strings.Join(vscodeCLIs, ", "))
		fmt.Println("    Enabling the `code` command (Cmd+Shift+P → \"Shell Command: Install 'code' command in PATH\")")
		fmt.Println("    lets future installs go through VS Code itself.")
	}

	fmt.Println()
	fmt.Println("What you get:")
	fmt.Println("  • Syntax highlighting + bracket matching for .ail files")
	fmt.Println("  • Language Server (diagnostics on save, hover types,")
	fmt.Println("    go-to-definition, find-references, document symbols)")
	fmt.Println("    — spawns `ailang lsp --stdio` automatically")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Fully quit VS Code (Cmd+Q on Mac, File→Exit elsewhere) and reopen")
	fmt.Println("     (a window reload is NOT enough on first install/upgrade)")
	fmt.Println("  2. Open any .ail file — diagnostics appear inline on save")
	fmt.Println("  3. Confirm with: ailang editor status")
	fmt.Println()
	fmt.Println("Troubleshooting:")
	fmt.Println("  • If no diagnostics: check `which ailang` works (binary on PATH)")
	fmt.Println("  • Errors surface in View → Output → 'AILANG Language Server'")
}

func uninstallVSCode() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot determine home directory: %v\n", red("Error"), err)
		os.Exit(1)
	}
	m, err := readVSCodeManifest()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	res, err := uninstallVSCodeExtension(home, m, exec.LookPath, execRunner)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	for _, w := range res.Warnings {
		fmt.Printf("  %s %s\n", yellow("⚠"), w)
	}
	if !res.Deregistered && len(res.Folders) == 0 && res.CLIName == "" {
		fmt.Println("VS Code extension not installed")
		return
	}
	if res.CLIName != "" {
		fmt.Printf("  %s uninstalled via `%s --uninstall-extension`\n", green("✓"), res.CLIName)
	}
	if res.Deregistered {
		fmt.Printf("  %s removed the extensions.json entry\n", green("✓"))
	}
	for _, f := range res.Folders {
		fmt.Printf("  %s removed ~/.vscode/extensions/%s\n", green("✓"), f)
	}
	fmt.Printf("%s VS Code extension uninstalled\n", green("✓"))
	fmt.Println("  Fully quit VS Code (Cmd+Q) and reopen to complete the removal.")
}
