package prompt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/repo"
)

// Kind selects one prompt series. Every series is a versions.json manifest plus
// the files it names, all under prompts/: the syntax prompt sits at prompts/
// itself, the others in a subdirectory named after the kind. One loader serves
// all of them (M-V1-SIMPLIFY-S3 M4) — internal/agentprompt and
// internal/devtoolsprompt used to be byte-for-byte copies of this file
// differing only in the literal; they were deleted in S4 M3B.
type Kind string

const (
	// Syntax is the language teaching prompt: prompts/versions.json (`ailang prompt`).
	Syntax Kind = ""
	// Agent is the minimal agent coding guide: prompts/agent/versions.json (`ailang agent-prompt`).
	Agent Kind = "agent"
	// DevTools is the toolchain/CLI reference: prompts/devtools/versions.json (`ailang devtools-prompt`).
	DevTools Kind = "devtools"
)

// manifestPath is the slash-separated manifest path inside prompts/ — the
// same string addresses the embedded FS and, joined onto a root, the disk.
func (k Kind) manifestPath() string {
	if k == Syntax {
		return "prompts/versions.json"
	}
	return path.Join("prompts", string(k), "versions.json")
}

// label prefixes messages so "agent versions.json" reads as before; the
// syntax series has no prefix.
func (k Kind) label() string {
	if k == Syntax {
		return ""
	}
	return string(k) + " "
}

// LoadPrompt loads a prompt of this kind by version string ("" or "latest" =
// the active version) from the embedded FS, falling back to disk.
func (k Kind) LoadPrompt(version string) (string, error) {
	return NewLoader(k).LoadPrompt(version)
}

// LoadPromptWithVersion is LoadPrompt plus the resolved version ID.
func (k Kind) LoadPromptWithVersion(version string) (string, string, error) {
	return NewLoader(k).LoadPromptWithVersion(version)
}

// GetActiveVersion returns the active version of this kind's manifest.
func (k Kind) GetActiveVersion() (string, error) {
	return NewLoader(k).GetActiveVersion()
}

// ListVersions returns every version ID in this kind's manifest.
func (k Kind) ListVersions() ([]string, error) {
	return NewLoader(k).ListVersions()
}

// GetVersionMetadata returns the manifest entry for a version ("" or "latest"
// = active).
func (k Kind) GetVersionMetadata(version string) (*VersionMetadata, error) {
	return NewLoader(k).GetVersionMetadata(version)
}

// embeddedPrompts is set by SetEmbeddedFS (called from main). One FS carries
// every kind — prompts/, prompts/agent/, prompts/devtools/.
var embeddedPrompts fs.FS

// SetEmbeddedFS sets the embedded filesystem for prompts.
// This should be called from main() with the embedded FS.
func SetEmbeddedFS(efs fs.FS) {
	embeddedPrompts = efs
}

// FrozenMarker records that a prompt version's bytes are immutable because the
// version has been used in at least one banked eval baseline (decision
// D-41c). ABSENT (nil) means never-banked, i.e. mutable.
type FrozenMarker struct {
	At              string `json:"at"`
	Reason          string `json:"reason"`
	EvidenceCount   int    `json:"evidence_count"`
	EvidenceExample string `json:"evidence_example"`
}

// VersionMetadata represents metadata for a prompt version
type VersionMetadata struct {
	File        string        `json:"file"`
	Hash        string        `json:"hash"`
	Description string        `json:"description"`
	Created     string        `json:"created"`
	Tags        []string      `json:"tags"`
	Notes       string        `json:"notes"`
	Frozen      *FrozenMarker `json:"frozen,omitempty"`
}

// VersionsManifest represents the versions.json file structure
type VersionsManifest struct {
	SchemaVersion string                     `json:"schema_version"`
	Versions      map[string]VersionMetadata `json:"versions"`
	Active        string                     `json:"active"`
	Notes         []string                   `json:"notes"`
}

// Loader reads one prompt series. The zero-option loader reads the embedded FS
// first and the project's prompts/ directory second (so a checkout can edit a
// prompt without rebuilding); options pin a manifest file, a root directory,
// or turn on hash verification.
type Loader struct {
	kind         Kind
	useEmbedded  bool   // false → disk only (WithManifestFile: an explicit file is a request for THAT file)
	root         string // disk root for prompt files; "" → findProjectRoot()
	manifestFile string // explicit disk manifest; "" → root/<kind manifest>
	verify       bool
}

// Option configures a Loader.
type Option func(*Loader)

// WithManifestFile reads the manifest from an explicit disk path instead of
// the embedded FS. Prompt files resolve against WithRoot (or the project root).
func WithManifestFile(p string) Option {
	return func(l *Loader) {
		l.manifestFile = p
		l.useEmbedded = false
	}
}

// WithRoot sets the disk directory that manifest `file` entries (e.g.
// "prompts/v0.16.6.md") are relative to. Default: the project root found by
// walking up from the working directory.
func WithRoot(dir string) Option {
	return func(l *Loader) { l.root = dir }
}

// WithVerify checks every loaded prompt against its manifest hash: a frozen
// version must match exactly (and must carry an enforceable sha256); a mutable
// version must match unless its hash is the "PLACEHOLDER" development escape
// hatch. This is the guard the eval harness runs so a banked baseline can never
// silently measure edited bytes (D-41c).
func WithVerify() Option {
	return func(l *Loader) { l.verify = true }
}

// NewLoader builds a Loader for one kind.
func NewLoader(kind Kind, opts ...Option) *Loader {
	l := &Loader{kind: kind, useEmbedded: true}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// Manifest loads and parses the series' versions.json.
func (l *Loader) Manifest() (*VersionsManifest, error) {
	label := l.kind.label()
	data, err := l.readFile(l.kind.manifestPath(), l.manifestDiskPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read %sversions.json (tried embedded and disk): %w", label, err)
	}
	var manifest VersionsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse %sversions.json: %w", label, err)
	}
	return &manifest, nil
}

// LoadPrompt loads a prompt by version string. "" or "latest" selects the
// active version. Returns the prompt content.
func (l *Loader) LoadPrompt(version string) (string, error) {
	content, _, err := l.LoadPromptWithVersion(version)
	return content, err
}

// LoadPromptWithVersion loads a prompt and returns the resolved version ID —
// the active version's ID when "" or "latest" was asked for.
func (l *Loader) LoadPromptWithVersion(version string) (string, string, error) {
	manifest, err := l.Manifest()
	if err != nil {
		return "", "", fmt.Errorf("failed to load %sversions manifest: %w", l.kind.label(), err)
	}
	targetVersion, metadata, err := l.resolve(manifest, version)
	if err != nil {
		return "", "", err
	}

	content, err := l.readFile(metadata.File, filepath.Join(l.rootDir(), filepath.FromSlash(metadata.File)))
	if err != nil {
		return "", "", fmt.Errorf("failed to read %sprompt file %s (tried embedded and disk): %w", l.kind.label(), metadata.File, err)
	}
	if l.verify {
		if err := verifyHash(targetVersion, metadata, content); err != nil {
			return "", "", err
		}
	}
	return string(content), targetVersion, nil
}

// GetActiveVersion returns the active version ID. A manifest whose active
// field is the "latest" sentinel resolves to its newest production version.
func (l *Loader) GetActiveVersion() (string, error) {
	manifest, err := l.Manifest()
	if err != nil {
		return "", err
	}
	return activeVersion(manifest), nil
}

// ListVersions returns every version ID in the manifest (map order).
func (l *Loader) ListVersions() ([]string, error) {
	manifest, err := l.Manifest()
	if err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(manifest.Versions))
	for version := range manifest.Versions {
		versions = append(versions, version)
	}
	return versions, nil
}

// GetVersionMetadata returns the manifest entry for a version ("" or "latest"
// = active).
func (l *Loader) GetVersionMetadata(version string) (*VersionMetadata, error) {
	manifest, err := l.Manifest()
	if err != nil {
		return nil, err
	}
	_, metadata, err := l.resolve(manifest, version)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

// resolve maps a requested version onto a manifest entry.
func (l *Loader) resolve(manifest *VersionsManifest, version string) (string, *VersionMetadata, error) {
	targetVersion := version
	if targetVersion == "" || targetVersion == "latest" {
		targetVersion = activeVersion(manifest)
		if targetVersion == "" {
			return "", nil, fmt.Errorf("no active prompt version specified in %sversions.json", l.kind.label())
		}
	}
	metadata, ok := manifest.Versions[targetVersion]
	if !ok {
		return "", nil, l.unknownVersionError(manifest, targetVersion)
	}
	return targetVersion, &metadata, nil
}

// activeVersion returns the manifest's active version, resolving the "latest"
// sentinel to the most recently created version tagged "production".
func activeVersion(manifest *VersionsManifest) string {
	if manifest.Active != "latest" {
		return manifest.Active
	}
	var latest, latestDate string
	for id, v := range manifest.Versions {
		production := false
		for _, tag := range v.Tags {
			if tag == "production" {
				production = true
				break
			}
		}
		if !production {
			continue
		}
		// Created is YYYY-MM-DD, so string order is date order.
		if v.Created > latestDate {
			latestDate = v.Created
			latest = id
		}
	}
	return latest
}

// readFile tries the embedded FS first (works anywhere, bundled in the
// binary), then disk (development: edit without rebuild).
func (l *Loader) readFile(fsPath, diskPath string) ([]byte, error) {
	if l.useEmbedded && embeddedPrompts != nil {
		if content, err := fs.ReadFile(embeddedPrompts, fsPath); err == nil {
			return content, nil
		}
	}
	return os.ReadFile(diskPath) // #nosec G304 -- manifest-named prompt file under the project root
}

func (l *Loader) manifestDiskPath() string {
	if l.manifestFile != "" {
		return l.manifestFile
	}
	return filepath.Join(l.rootDir(), filepath.FromSlash(l.kind.manifestPath()))
}

func (l *Loader) rootDir() string {
	if l.root != "" {
		return l.root
	}
	return findProjectRoot()
}

// findProjectRoot finds the project root: the nearest ancestor of the working
// directory holding go.mod, .git or a prompts/ directory, else the working
// directory itself.
func findProjectRoot() string {
	root, _ := repo.FindRoot("go.mod", ".git", "prompts")
	return root
}

// SHA256Hex is the manifest hash function: lowercase hex sha256 of the bytes.
func SHA256Hex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// IsHexSHA256 reports whether s is exactly 64 lowercase hex characters.
func IsHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// verifyHash enforces the manifest hash (see WithVerify).
func verifyHash(versionID string, v *VersionMetadata, content []byte) error {
	if v.Frozen != nil {
		if !IsHexSHA256(v.Hash) {
			return fmt.Errorf("prompt version %q is FROZEN but its recorded hash %q is not a 64-hex sha256: refuse to load — a frozen version with an unenforceable hash is a freeze with no invariant (D-41c, Q6). Restore the recorded hash from git history, or bump to a new version.", versionID, v.Hash)
		}
		if actual := SHA256Hex(content); actual != v.Hash {
			return fmt.Errorf("prompt version %q is FROZEN: it is cited by %d banked baseline files under eval_results/baselines/ (e.g. %s; frozen %s, reason: %s — decision D-41c). Its bytes are immutable: editing %s in place would silently change what those baselines measured.\nTo change the teaching prompt, create a NEW version instead:\n  .claude/skills/prompt-manager/scripts/create_prompt_version.sh <new-id> %s \"<why>\"\n(expected sha256 %s, got %s)", versionID, v.Frozen.EvidenceCount, v.Frozen.EvidenceExample, v.Frozen.At, v.Frozen.Reason, v.File, versionID, v.Hash, actual)
		}
		return nil
	}
	// never-banked: the PLACEHOLDER development escape hatch remains available.
	if v.Hash == "PLACEHOLDER" {
		return nil
	}
	if actual := SHA256Hex(content); actual != v.Hash {
		expectedPreview, actualPreview := v.Hash, actual
		if len(expectedPreview) > 16 {
			expectedPreview = expectedPreview[:16] + "..."
		}
		if len(actualPreview) > 16 {
			actualPreview = actualPreview[:16] + "..."
		}
		return fmt.Errorf("hash mismatch for %q: expected %s, got %s. This version is not yet banked, so in-place editing is allowed (D-41c) — regenerate the change-detector hash:\n  shasum -a 256 %s   # then update the \"hash\" field in prompts/versions.json", versionID, expectedPreview, actualPreview, v.File)
	}
	return nil
}

// --- package-level API = the syntax series (existing callers) ---

// LoadPrompt loads the syntax prompt by version string ("" or "latest" = active).
func LoadPrompt(version string) (string, error) { return Syntax.LoadPrompt(version) }

// LoadPromptWithVersion loads the syntax prompt and returns the resolved version ID.
func LoadPromptWithVersion(version string) (string, string, error) {
	return Syntax.LoadPromptWithVersion(version)
}

// GetActiveVersion returns the active syntax prompt version.
func GetActiveVersion() (string, error) { return Syntax.GetActiveVersion() }

// ListVersions returns all available syntax prompt versions.
func ListVersions() ([]string, error) { return Syntax.ListVersions() }

// GetVersionMetadata returns metadata for a syntax prompt version.
func GetVersionMetadata(version string) (*VersionMetadata, error) {
	return Syntax.GetVersionMetadata(version)
}

// unknownVersionError explains an unresolvable prompt version.
//
// Teaching prompts carry their own version series, which is INDEPENDENT of the
// binary's release version: v0.16.6 is the current prompt on a v0.34.0 binary,
// and there has never been a prompt numbered v0.34.0. The bare
// "not found in versions.json" gave no hint of that, so `--version 0.34.0` read
// as eighteen minor versions of drift rather than as a category error. The
// other series keep the short form — their versions are not what anyone
// confuses with a release.
func (l *Loader) unknownVersionError(manifest *VersionsManifest, targetVersion string) error {
	if l.kind != Syntax {
		return fmt.Errorf("version %q not found in %sversions.json", targetVersion, l.kind.label())
	}
	msg := fmt.Sprintf("%q is not a known prompt version", targetVersion)

	if looksLikeBinaryVersion(manifest, targetVersion) {
		msg += "\n\n  Prompt versions are their own series and do not track the binary's" +
			"\n  release version — there is no prompt numbered for each AILANG release."
	}
	if manifest.Active != "" {
		msg += fmt.Sprintf("\n\n  Active prompt version: %s  (omit --version, or pass --version latest)", manifest.Active)
	}
	msg += "\n  List every available version: ailang prompt --list"
	return fmt.Errorf("%s", msg)
}

// looksLikeBinaryVersion reports whether the requested version is plausibly the
// caller's binary version rather than a typo'd prompt version: it parses as a
// version and sorts above every prompt version on record.
func looksLikeBinaryVersion(manifest *VersionsManifest, target string) bool {
	tMajor, tMinor, ok := parseMajorMinor(target)
	if !ok {
		return false
	}
	for v := range manifest.Versions {
		vMajor, vMinor, ok := parseMajorMinor(v)
		if !ok {
			continue
		}
		if vMajor > tMajor || (vMajor == tMajor && vMinor >= tMinor) {
			return false
		}
	}
	return true
}

// parseMajorMinor extracts major/minor from a "vX.Y.Z" or "X.Y.Z" string.
func parseMajorMinor(v string) (int, int, bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return major, minor, true
}
