package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// --- helpers ---------------------------------------------------------------

// noCLI is a lookPathFunc for a machine with no editor CLI on PATH.
func noCLI(string) (string, error) { return "", fmt.Errorf("not found") }

// fakeCLI reports the named binary as present at a fixed path.
func fakeCLI(want string) lookPathFunc {
	return func(name string) (string, error) {
		if name == want {
			return "/usr/local/bin/" + want, nil
		}
		return "", fmt.Errorf("not found")
	}
}

// recordingRunner captures every invocation and returns a canned result.
type recordingRunner struct {
	calls [][]string
	out   []byte
	err   error
	// onCall runs before the canned result is returned, so a test can inspect
	// artifacts (e.g. the .vsix) while they still exist.
	onCall func(t *testing.T, name string, args []string)
	t      *testing.T
}

func (r *recordingRunner) run(name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if r.onCall != nil {
		r.onCall(r.t, name, args)
	}
	return r.out, r.err
}

// mustRunnerNeverCalled fails if anything shells out.
func mustRunnerNeverCalled(t *testing.T) cmdRunner {
	t.Helper()
	return func(name string, args ...string) ([]byte, error) {
		t.Fatalf("unexpected exec: %s %v", name, args)
		return nil, nil
	}
}

func testManifest(t *testing.T) vscodeManifest {
	t.Helper()
	m, err := readVSCodeManifest()
	if err != nil {
		t.Fatalf("readVSCodeManifest: %v", err)
	}
	return m
}

func extRoot(t *testing.T, home string) string {
	t.Helper()
	root := filepath.Join(home, ".vscode", "extensions")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", root, err)
	}
	return root
}

func writeRegistryFixture(t *testing.T, home string, entries []map[string]interface{}) {
	t.Helper()
	data, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(extRoot(t, home), "extensions.json"), data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func readRegistry(t *testing.T, home string) []map[string]interface{} {
	t.Helper()
	entries, err := readVSCodeRegistry(vscodeRegistryPath(home))
	if err != nil {
		t.Fatalf("readVSCodeRegistry: %v", err)
	}
	return entries
}

func findEntry(entries []map[string]interface{}, id string) map[string]interface{} {
	for _, e := range entries {
		if strings.EqualFold(registryEntryID(e), id) {
			return e
		}
	}
	return nil
}

// --- manifest --------------------------------------------------------------

func TestReadVSCodeManifestIdentity(t *testing.T) {
	m := testManifest(t)
	if m.Name == "" || m.Publisher == "" || m.Version == "" {
		t.Fatalf("embedded manifest is missing identity fields: %+v", m)
	}
	if got, want := m.extID(), m.Publisher+"."+m.Name; got != want {
		t.Errorf("extID = %q, want %q", got, want)
	}
	if got, want := m.folder(), m.extID()+"-"+m.Version; got != want {
		t.Errorf("folder = %q, want %q", got, want)
	}
	// The folder name is what VS Code stores as relativeLocation. An
	// unversioned folder is exactly the state that produced #694.
	if !strings.Contains(m.folder(), "-") || strings.HasSuffix(m.folder(), "-") {
		t.Errorf("folder %q is not versioned", m.folder())
	}
}

// --- VSIX packaging --------------------------------------------------------

func TestBuildVSIXIsAWellFormedArchive(t *testing.T) {
	m := testManifest(t)
	data, err := buildVSIX(m)
	if err != nil {
		t.Fatalf("buildVSIX: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("vsix is not a readable zip: %v", err)
	}

	names := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)

	want := []string{
		"[Content_Types].xml",
		"extension.vsixmanifest",
		"extension/extension.js",
		"extension/language-configuration.json",
		"extension/package.json",
		"extension/syntaxes/ailang.tmLanguage.json",
	}
	sort.Strings(want)
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("vsix entries = %v, want %v", names, want)
	}
	// Anti-vacuity: a zip whose entries all happened to be empty would satisfy
	// the name check above.
	if len(names) == 0 {
		t.Fatal("instrument failure: vsix has no entries")
	}
	for _, f := range zr.File {
		if strings.ContainsRune(f.Name, '\\') {
			t.Errorf("zip entry %q uses a backslash; entry names must be forward-slash separated on every GOOS", f.Name)
		}
	}
}

func TestBuildVSIXPayloadIsByteIdenticalToEmbeddedAssets(t *testing.T) {
	m := testManifest(t)
	data, err := buildVSIX(m)
	if err != nil {
		t.Fatalf("buildVSIX: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open vsix: %v", err)
	}
	checked := 0
	for _, a := range vscodeAssets {
		want, err := editorAssets.ReadFile(a.src)
		if err != nil {
			t.Fatalf("read embedded %s: %v", a.src, err)
		}
		got := zipEntry(t, zr, "extension/"+a.rel)
		if !bytes.Equal(got, want) {
			t.Errorf("vsix entry extension/%s differs from embedded %s", a.rel, a.src)
		}
		checked++
	}
	if checked != len(vscodeAssets) || checked == 0 {
		t.Fatalf("instrument failure: checked %d assets, expected %d", checked, len(vscodeAssets))
	}
}

func TestVSIXManifestIdentityMatchesPackageJSON(t *testing.T) {
	m := testManifest(t)
	raw, err := vsixManifestXML(m)
	if err != nil {
		t.Fatalf("vsixManifestXML: %v", err)
	}
	var parsed struct {
		Metadata struct {
			Identity struct {
				ID        string `xml:"Id,attr"`
				Version   string `xml:"Version,attr"`
				Publisher string `xml:"Publisher,attr"`
			} `xml:"Identity"`
		} `xml:"Metadata"`
		Installation struct {
			Target struct {
				ID string `xml:"Id,attr"`
			} `xml:"InstallationTarget"`
		} `xml:"Installation"`
	}
	if err := xml.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("manifest is not well-formed XML: %v\n%s", err, raw)
	}
	if parsed.Metadata.Identity.ID != m.Name {
		t.Errorf("Identity/@Id = %q, want %q", parsed.Metadata.Identity.ID, m.Name)
	}
	if parsed.Metadata.Identity.Version != m.Version {
		t.Errorf("Identity/@Version = %q, want %q", parsed.Metadata.Identity.Version, m.Version)
	}
	if parsed.Metadata.Identity.Publisher != m.Publisher {
		t.Errorf("Identity/@Publisher = %q, want %q", parsed.Metadata.Identity.Publisher, m.Publisher)
	}
	if parsed.Installation.Target.ID != "Microsoft.VisualStudio.Code" {
		t.Errorf("InstallationTarget = %q, want Microsoft.VisualStudio.Code", parsed.Installation.Target.ID)
	}
}

func zipEntry(t *testing.T, zr *zip.Reader, name string) []byte {
	t.Helper()
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		defer func() { _ = rc.Close() }()
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		return buf.Bytes()
	}
	t.Fatalf("vsix has no entry %q", name)
	return nil
}

// --- the #694 regression guard ---------------------------------------------

// TestInstallNeverDeregistersAnExistingEntry is the pin for #694.
//
// The pre-fix installer called invalidateVSCodeExtensionCache, which STRIPPED
// the sunholo.ailang entry from extensions.json. Since VS Code 1.74 that is an
// uninstall: VS Code flags the folder in .obsolete and never loads it. An
// install that ends with the extension unregistered is the whole defect.
func TestInstallNeverDeregistersAnExistingEntry(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)
	root := extRoot(t, home)

	other := map[string]interface{}{
		"identifier":       map[string]interface{}{"id": "ms-vscode.makefile-tools"},
		"version":          "0.12.17",
		"relativeLocation": "ms-vscode.makefile-tools-0.12.17",
	}
	mine := map[string]interface{}{
		"identifier":       map[string]interface{}{"id": m.extID()},
		"version":          m.Version,
		"relativeLocation": m.folder(),
	}
	writeRegistryFixture(t, home, []map[string]interface{}{other, mine})

	// Anti-vacuity floor: if the fixture did not actually contain our entry,
	// the post-condition below would pass for the wrong reason.
	if findEntry(readRegistry(t, home), m.extID()) == nil {
		t.Fatal("instrument failure: fixture does not contain the extension entry")
	}

	if _, err := installVSCodeExtension(home, m, noCLI, mustRunnerNeverCalled(t), 1234); err != nil {
		t.Fatalf("installVSCodeExtension: %v", err)
	}

	entries := readRegistry(t, home)
	got := findEntry(entries, m.extID())
	if got == nil {
		t.Fatalf("install DEREGISTERED %s — extensions.json now holds %d entries and none is ours (this is #694)", m.extID(), len(entries))
	}
	if rel, _ := got["relativeLocation"].(string); rel != m.folder() {
		t.Errorf("relativeLocation = %q, want %q", rel, m.folder())
	}
	if findEntry(entries, "ms-vscode.makefile-tools") == nil {
		t.Error("install removed an unrelated extension's entry")
	}
	if _, err := os.Stat(filepath.Join(root, m.folder())); err != nil {
		t.Errorf("versioned extension folder was not created: %v", err)
	}
}

func TestInstallRegistersWithTheSchemaVSCodeReads(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)

	if _, err := installVSCodeExtension(home, m, noCLI, mustRunnerNeverCalled(t), 1755000000000); err != nil {
		t.Fatalf("installVSCodeExtension: %v", err)
	}

	entry := findEntry(readRegistry(t, home), m.extID())
	if entry == nil {
		t.Fatal("install wrote no extensions.json entry, so VS Code would never load the folder")
	}

	// Fields taken from a real ~/.vscode/extensions/extensions.json.
	ident, ok := entry["identifier"].(map[string]interface{})
	if !ok || ident["id"] != m.extID() {
		t.Errorf("identifier = %#v, want id %q", entry["identifier"], m.extID())
	}
	if entry["version"] != m.Version {
		t.Errorf("version = %#v, want %q", entry["version"], m.Version)
	}
	rel, _ := entry["relativeLocation"].(string)
	if rel != m.folder() {
		t.Errorf("relativeLocation = %q, want %q", rel, m.folder())
	}
	loc, ok := entry["location"].(map[string]interface{})
	if !ok {
		t.Fatalf("location = %#v, want an object", entry["location"])
	}
	if loc["scheme"] != "file" {
		t.Errorf("location.scheme = %#v, want \"file\"", loc["scheme"])
	}
	path, _ := loc["path"].(string)
	if !strings.HasPrefix(path, "/") {
		t.Errorf("location.path = %q, want a URI path beginning with /", path)
	}
	if !strings.HasSuffix(path, "/"+m.folder()) {
		t.Errorf("location.path = %q, want it to end in /%s", path, m.folder())
	}
	if _, ok := entry["metadata"].(map[string]interface{}); !ok {
		t.Errorf("metadata = %#v, want an object", entry["metadata"])
	}

	// relativeLocation must name a directory that exists, or VS Code skips it.
	if _, err := os.Stat(filepath.Join(extRoot(t, home), rel)); err != nil {
		t.Errorf("relativeLocation %q does not exist on disk: %v", rel, err)
	}
	for _, a := range vscodeAssets {
		p := filepath.Join(extRoot(t, home), rel, filepath.FromSlash(a.rel))
		if _, err := os.Stat(p); err != nil {
			t.Errorf("asset %s missing from the install: %v", a.rel, err)
		}
	}
}

func TestInstallPurgesLegacyFolderAndClearsObsolete(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)
	root := extRoot(t, home)

	legacy := filepath.Join(root, legacyVSCodeFolder)
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}
	obsolete := map[string]bool{
		legacyVSCodeFolder:             true,
		m.folder():                     true,
		"someone-else.extension-1.0.0": true,
	}
	data, _ := json.Marshal(obsolete)
	if err := os.WriteFile(filepath.Join(root, ".obsolete"), data, 0o644); err != nil {
		t.Fatalf("write .obsolete: %v", err)
	}

	res, err := installVSCodeExtension(home, m, noCLI, mustRunnerNeverCalled(t), 1)
	if err != nil {
		t.Fatalf("installVSCodeExtension: %v", err)
	}
	if !res.LegacyPurged {
		t.Error("LegacyPurged = false, want true")
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Errorf("legacy folder still present: %v", err)
	}

	stale, err := vscodeFolderIsObsolete(home, m.folder())
	if err != nil {
		t.Fatalf("vscodeFolderIsObsolete: %v", err)
	}
	if stale {
		t.Error("the folder we just installed into is still flagged obsolete; VS Code would skip it")
	}
	// Someone else's obsolete flag is not ours to clear.
	if other, err := vscodeFolderIsObsolete(home, "someone-else.extension-1.0.0"); err != nil || !other {
		t.Errorf("unrelated .obsolete entry was cleared (err=%v, present=%v)", err, other)
	}
}

// --- editor CLI path -------------------------------------------------------

func TestInstallShellsOutToTheEditorCLIWithAValidVSIX(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)

	rr := &recordingRunner{t: t, onCall: func(t *testing.T, name string, args []string) {
		// Rule: validate the artifact the product actually produced, while it
		// still exists — not a reconstruction of it.
		var vsixPath string
		for i, a := range args {
			if a == "--install-extension" && i+1 < len(args) {
				vsixPath = args[i+1]
			}
		}
		if vsixPath == "" {
			t.Fatalf("no --install-extension argument in %v", args)
		}
		data, err := os.ReadFile(vsixPath)
		if err != nil {
			t.Fatalf("the vsix handed to the editor does not exist: %v", err)
		}
		if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
			t.Fatalf("the vsix handed to the editor is not a readable zip: %v", err)
		}
	}}

	res, err := installVSCodeExtension(home, m, fakeCLI("code"), rr.run, 1)
	if err != nil {
		t.Fatalf("installVSCodeExtension: %v", err)
	}
	if res.Method != "cli" || res.CLIName != "code" {
		t.Fatalf("Method/CLIName = %q/%q, want cli/code", res.Method, res.CLIName)
	}
	if len(rr.calls) != 1 {
		t.Fatalf("exec calls = %d, want 1: %v", len(rr.calls), rr.calls)
	}
	call := rr.calls[0]
	if call[0] != "/usr/local/bin/code" {
		t.Errorf("invoked %q, want the resolved CLI path", call[0])
	}
	joined := strings.Join(call, " ")
	if !strings.Contains(joined, "--install-extension") || !strings.Contains(joined, "--force") {
		t.Errorf("call = %v, want --install-extension … --force", call)
	}
	if !strings.HasSuffix(call[2], ".vsix") {
		t.Errorf("argument %q does not look like a .vsix", call[2])
	}
	// VS Code owns the install now, so we must not also drop a folder — two
	// installs sharing one extension ID is what the report warns about.
	if _, err := os.Stat(filepath.Join(extRoot(t, home), m.folder())); !os.IsNotExist(err) {
		t.Errorf("a folder install also happened alongside the CLI install: %v", err)
	}
}

func TestInstallFallsBackLoudlyWhenTheEditorCLIFails(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)

	rr := &recordingRunner{t: t, out: []byte("Unable to install extension"), err: fmt.Errorf("exit status 1")}
	res, err := installVSCodeExtension(home, m, fakeCLI("cursor"), rr.run, 1)
	if err != nil {
		t.Fatalf("installVSCodeExtension: %v", err)
	}
	if res.Method != "folder" {
		t.Fatalf("Method = %q, want folder", res.Method)
	}
	if len(res.Warnings) == 0 {
		t.Fatal("a silent downgrade: the CLI install failed and nothing said so")
	}
	if !strings.Contains(strings.Join(res.Warnings, " "), "cursor") {
		t.Errorf("warnings = %v, want the failing CLI named", res.Warnings)
	}
	if findEntry(readRegistry(t, home), m.extID()) == nil {
		t.Error("fallback install left the extension unregistered")
	}
}

// --- status ----------------------------------------------------------------

// TestStatusDoesNotTrustABareFolder pins the second half of #694: `ailang
// editor status` reported ✓ from an os.Stat, so a deregistered install looked
// healthy.
func TestStatusDoesNotTrustABareFolder(t *testing.T) {
	m := testManifest(t)
	for _, folder := range []string{legacyVSCodeFolder, m.folder()} {
		t.Run(folder, func(t *testing.T) {
			home := t.TempDir()
			if err := os.MkdirAll(filepath.Join(extRoot(t, home), folder), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			st := vscodeExtensionStatus(home, m, noCLI, mustRunnerNeverCalled(t))
			if st.Installed {
				t.Fatalf("status reports installed for an unregistered folder %q (this is #694)", folder)
			}
			if !strings.Contains(st.Detail, "not registered") {
				t.Errorf("Detail = %q, want it to say the folder is not registered", st.Detail)
			}
		})
	}
}

func TestStatusReportsRegisteredInstall(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)
	if _, err := installVSCodeExtension(home, m, noCLI, mustRunnerNeverCalled(t), 1); err != nil {
		t.Fatalf("install: %v", err)
	}
	st := vscodeExtensionStatus(home, m, noCLI, mustRunnerNeverCalled(t))
	if !st.Installed {
		t.Fatalf("status = not installed after a successful install: %q", st.Detail)
	}
	if st.Version != m.Version {
		t.Errorf("Version = %q, want %q", st.Version, m.Version)
	}
}

func TestStatusFlagsAnObsoleteInstall(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)
	if _, err := installVSCodeExtension(home, m, noCLI, mustRunnerNeverCalled(t), 1); err != nil {
		t.Fatalf("install: %v", err)
	}
	// Simulate VS Code having deregistered it out from under us.
	data, _ := json.Marshal(map[string]bool{m.folder(): true})
	if err := os.WriteFile(vscodeObsoletePath(home), data, 0o644); err != nil {
		t.Fatalf("write .obsolete: %v", err)
	}
	st := vscodeExtensionStatus(home, m, noCLI, mustRunnerNeverCalled(t))
	if st.Installed {
		t.Fatal("status reports installed for a folder VS Code has flagged obsolete")
	}
	if !strings.Contains(st.Detail, "obsolete") {
		t.Errorf("Detail = %q, want it to mention obsolete", st.Detail)
	}
}

func TestStatusAsksTheEditorCLIWhenPresent(t *testing.T) {
	m := testManifest(t)
	tests := []struct {
		name      string
		listing   string
		installed bool
		version   string
	}{
		{"present", "ms-vscode.makefile-tools@0.12.17\n" + m.extID() + "@" + m.Version + "\n", true, m.Version},
		{"absent", "ms-vscode.makefile-tools@0.12.17\n", false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			extRoot(t, home)
			rr := &recordingRunner{t: t, out: []byte(tc.listing)}
			st := vscodeExtensionStatus(home, m, fakeCLI("code"), rr.run)
			if st.Installed != tc.installed {
				t.Fatalf("Installed = %v, want %v (detail %q)", st.Installed, tc.installed, st.Detail)
			}
			if st.Version != tc.version {
				t.Errorf("Version = %q, want %q", st.Version, tc.version)
			}
			if len(rr.calls) != 1 || !strings.Contains(strings.Join(rr.calls[0], " "), "--list-extensions") {
				t.Errorf("calls = %v, want one --list-extensions", rr.calls)
			}
		})
	}
}

// --- uninstall -------------------------------------------------------------

func TestUninstallRemovesRegistryEntryAndEveryVersionedFolder(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)
	root := extRoot(t, home)

	if _, err := installVSCodeExtension(home, m, noCLI, mustRunnerNeverCalled(t), 1); err != nil {
		t.Fatalf("install: %v", err)
	}
	// A folder from some other version, plus the pre-fix unversioned one.
	stray := m.extID() + "-0.9.9"
	for _, d := range []string{stray, legacyVSCodeFolder} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	keep := "ms-vscode.makefile-tools-0.12.17"
	if err := os.MkdirAll(filepath.Join(root, keep), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", keep, err)
	}

	res, err := uninstallVSCodeExtension(home, m, noCLI, mustRunnerNeverCalled(t))
	if err != nil {
		t.Fatalf("uninstallVSCodeExtension: %v", err)
	}
	if !res.Deregistered {
		t.Error("Deregistered = false, want true")
	}
	if findEntry(readRegistry(t, home), m.extID()) != nil {
		t.Error("uninstall left the extensions.json entry behind")
	}
	for _, d := range []string{m.folder(), stray, legacyVSCodeFolder} {
		if _, err := os.Stat(filepath.Join(root, d)); !os.IsNotExist(err) {
			t.Errorf("uninstall left %s on disk: %v", d, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, keep)); err != nil {
		t.Errorf("uninstall removed an unrelated extension %s: %v", keep, err)
	}
}

func TestUninstallUsesTheEditorCLIWhenPresent(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)
	extRoot(t, home)

	rr := &recordingRunner{t: t}
	res, err := uninstallVSCodeExtension(home, m, fakeCLI("code"), rr.run)
	if err != nil {
		t.Fatalf("uninstallVSCodeExtension: %v", err)
	}
	if res.CLIName != "code" {
		t.Errorf("CLIName = %q, want code", res.CLIName)
	}
	if len(rr.calls) != 1 {
		t.Fatalf("calls = %v, want one", rr.calls)
	}
	joined := strings.Join(rr.calls[0], " ")
	if !strings.Contains(joined, "--uninstall-extension") || !strings.Contains(joined, m.extID()) {
		t.Errorf("call = %v, want --uninstall-extension %s", rr.calls[0], m.extID())
	}
}

// --- registry primitives ---------------------------------------------------

func TestDeregisterVSCodeExtension(t *testing.T) {
	t.Run("removes only the matching entry", func(t *testing.T) {
		home := t.TempDir()
		writeRegistryFixture(t, home, []map[string]interface{}{
			{"identifier": map[string]interface{}{"id": "sunholo.ailang"}},
			{"identifier": map[string]interface{}{"id": "other.ext"}},
		})
		removed, err := deregisterVSCodeExtension(home, "sunholo.ailang")
		if err != nil || !removed {
			t.Fatalf("removed=%v err=%v, want true/nil", removed, err)
		}
		entries := readRegistry(t, home)
		if len(entries) != 1 || registryEntryID(entries[0]) != "other.ext" {
			t.Fatalf("entries = %#v, want just other.ext", entries)
		}
	})

	t.Run("case-insensitive match", func(t *testing.T) {
		home := t.TempDir()
		writeRegistryFixture(t, home, []map[string]interface{}{
			{"identifier": map[string]interface{}{"id": "Sunholo.AILang"}},
		})
		removed, err := deregisterVSCodeExtension(home, "sunholo.ailang")
		if err != nil || !removed {
			t.Fatalf("removed=%v err=%v, want true/nil", removed, err)
		}
	})

	t.Run("no entry is not an error", func(t *testing.T) {
		home := t.TempDir()
		writeRegistryFixture(t, home, []map[string]interface{}{
			{"identifier": map[string]interface{}{"id": "other.ext"}},
		})
		removed, err := deregisterVSCodeExtension(home, "sunholo.ailang")
		if err != nil || removed {
			t.Fatalf("removed=%v err=%v, want false/nil", removed, err)
		}
	})

	t.Run("missing registry is not an error", func(t *testing.T) {
		home := t.TempDir()
		removed, err := deregisterVSCodeExtension(home, "sunholo.ailang")
		if err != nil || removed {
			t.Fatalf("removed=%v err=%v, want false/nil", removed, err)
		}
	})

	t.Run("malformed registry is refused, not silently rewritten", func(t *testing.T) {
		home := t.TempDir()
		if err := os.WriteFile(filepath.Join(extRoot(t, home), "extensions.json"), []byte("{not json"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := deregisterVSCodeExtension(home, "sunholo.ailang"); err == nil {
			t.Fatal("want an error for a malformed extensions.json, got nil")
		}
	})
}

func TestRegisterReplacesRatherThanDuplicates(t *testing.T) {
	home := t.TempDir()
	m := testManifest(t)
	writeRegistryFixture(t, home, []map[string]interface{}{
		{"identifier": map[string]interface{}{"id": m.extID()}, "version": "0.0.1", "relativeLocation": m.extID() + "-0.0.1"},
	})
	if err := registerVSCodeExtension(home, m, 42); err != nil {
		t.Fatalf("registerVSCodeExtension: %v", err)
	}
	entries := readRegistry(t, home)
	count := 0
	for _, e := range entries {
		if strings.EqualFold(registryEntryID(e), m.extID()) {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("entry count for %s = %d, want exactly 1", m.extID(), count)
	}
	if rel, _ := findEntry(entries, m.extID())["relativeLocation"].(string); rel != m.folder() {
		t.Errorf("relativeLocation = %q, want the new %q", rel, m.folder())
	}
}

func TestVSCodeURIPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/Users/x/.vscode/extensions/sunholo.ailang-0.3.0", "/Users/x/.vscode/extensions/sunholo.ailang-0.3.0"},
		{`C:\Users\x\.vscode\extensions\sunholo.ailang-0.3.0`, "/C:/Users/x/.vscode/extensions/sunholo.ailang-0.3.0"},
	}
	for _, tc := range tests {
		// filepath.ToSlash only rewrites separators on Windows, so the second
		// case asserts the leading-slash rule that applies on every GOOS.
		got := vscodeURIPath(tc.in)
		if !strings.HasPrefix(got, "/") {
			t.Errorf("vscodeURIPath(%q) = %q, want a leading /", tc.in, got)
		}
	}
}

func TestFindVSCodeCLIProbeOrder(t *testing.T) {
	// The first candidate present wins, in the declared order.
	look := func(name string) (string, error) {
		if name == "cursor" || name == "windsurf" {
			return "/opt/" + name, nil
		}
		return "", fmt.Errorf("not found")
	}
	name, path, ok := findVSCodeCLI(look)
	if !ok || name != "cursor" || path != "/opt/cursor" {
		t.Fatalf("findVSCodeCLI = %q/%q/%v, want cursor//opt/cursor/true", name, path, ok)
	}
	if _, _, ok := findVSCodeCLI(noCLI); ok {
		t.Error("findVSCodeCLI reported a CLI on a machine with none")
	}
	// Anti-vacuity: the probe list must be non-empty, or "not found" is
	// guaranteed regardless of the machine.
	if len(vscodeCLIs) == 0 {
		t.Fatal("instrument failure: vscodeCLIs is empty")
	}
}
