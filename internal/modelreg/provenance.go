package modelreg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M4 (decision D1(a), ratified by Mark 2026-08-27).
//
// THE DEFECT THIS FIXES. InitModelsConfig used to try the EMBEDDED registry
// first and fall back to disk only "for development". A registry published to
// the config bucket was therefore ignored by every installed binary — which is
// why adopting a new model needed a rebuild and a release. D1(a) inverts the
// order and makes embed the FLOOR rather than the ceiling.
//
// The accepted cost, recorded at ratification: pricing rides this same file, so
// a bad publish can now reach observatory cost accounting without a rebuild.
// Validation on publish is the mitigation; the floor and this provenance line
// are the containment.

const (
	// ModelsPathEnv names one registry file explicitly. It wins over everything
	// — it is the operator saying "use this one", and a developer with a local
	// file must be able to beat a published registry.
	ModelsPathEnv = "AILANG_MODELS_PATH"

	// PublishedDirEnv is the directory a published registry is mounted into
	// (the gcsfuse volume on Cloud Run — see M2). Contains models.yml.
	PublishedDirEnv = "AILANG_MODELS_PUBLISHED_DIR"

	// DefaultPublishedDir is where cloud jobs mount the registry when
	// PublishedDirEnv is unset.
	DefaultPublishedDir = "/registry"
)

// SourceKind says which of the three levels supplied the loaded registry.
type SourceKind string

const (
	SourceExplicitPath SourceKind = "explicit-path"
	SourcePublished    SourceKind = "published"
	SourceEmbedded     SourceKind = "embedded"
)

// Source is the provenance of the registry this process is using: the answer to
// "which registry am I running?", which before M4 was only inferable.
type Source struct {
	Kind SourceKind
	// Path is the file read, empty for the embedded registry.
	Path string
	// Version is a content digest. models.yml carries no version field, so the
	// honest identifier is what the bytes hash to — it distinguishes two
	// registries without claiming a semantic version nobody maintains.
	Version string
	// Degraded, when non-empty, says a HIGHER-precedence source was present and
	// rejected. Never let that happen silently: a rejected published registry
	// means this process is running different model assignments than intended.
	Degraded string
}

func (s Source) String() string {
	line := fmt.Sprintf("models registry: source=%s version=%s", s.Kind, s.Version)
	if s.Path != "" {
		line += " path=" + s.Path
	}
	if s.Degraded != "" {
		line += " DEGRADED=" + s.Degraded
	}
	return line
}

// LoadedSource is the provenance of the currently loaded registry, set by
// InitModelsConfig. Zero value means nothing has been loaded yet.
var LoadedSource Source

var (
	provenanceMu     sync.Mutex
	lastLoggedSource string
	// ProvenanceWriter is where the startup line goes. A variable so tests can
	// capture it; stderr in production.
	ProvenanceWriter io.Writer = os.Stderr
)

// logProvenance emits the startup line, once per DISTINCT source.
//
// It lives here rather than at the six InitModelsConfig call sites on purpose:
// a per-call-site log is one refactor away from a binary that silently answers
// "which registry am I using?" with nothing. One emission point cannot drift.
func logProvenance(s Source) {
	provenanceMu.Lock()
	defer provenanceMu.Unlock()
	line := s.String()
	if line == lastLoggedSource {
		return
	}
	lastLoggedSource = line
	fmt.Fprintln(ProvenanceWriter, line)
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// tryFile parses one candidate registry file. A file that exists but does not
// parse is reported as an error so the caller can degrade LOUDLY; a file that
// is simply absent is not an error, it is just this level declining.
func tryFile(path string) (*ModelsConfig, string, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", false, nil // absent: this level declines
	}
	cfg, err := LoadModelsConfig(path)
	if err != nil {
		return nil, "", true, err // present but broken: the caller must say so
	}
	return cfg, digest(raw), true, nil
}

// initModelsConfigFrom applies D1(a) precedence and returns the winning source.
// InitModelsConfig wraps it; tests call it directly.
func initModelsConfigFrom() (Source, error) {
	var degraded string

	// 1. Explicit path. If the operator named a file and it is broken, that is
	//    a hard error — silently ignoring an explicit instruction is worse than
	//    stopping.
	if p := os.Getenv(ModelsPathEnv); p != "" {
		cfg, ver, present, err := tryFile(p)
		if err != nil {
			return Source{}, fmt.Errorf("%s=%s could not be parsed: %w", ModelsPathEnv, p, err)
		}
		if !present {
			return Source{}, fmt.Errorf("%s=%s does not exist", ModelsPathEnv, p)
		}
		GlobalModelsConfig = cfg
		LoadedSource = Source{Kind: SourceExplicitPath, Path: p, Version: ver}
		logProvenance(LoadedSource)
		return LoadedSource, nil
	}

	// 2. Published registry (the gcsfuse mount). Broken here degrades to the
	//    floor rather than taking the fleet down — but never quietly.
	dir := os.Getenv(PublishedDirEnv)
	if dir == "" {
		dir = DefaultPublishedDir
	}
	pubPath := filepath.Join(dir, "models.yml")
	cfg, ver, present, err := tryFile(pubPath)
	switch {
	case err != nil:
		degraded = fmt.Sprintf("published registry at %s was rejected (%v); using the embedded floor", pubPath, err)
	case present:
		GlobalModelsConfig = cfg
		LoadedSource = Source{Kind: SourcePublished, Path: pubPath, Version: ver}
		logProvenance(LoadedSource)
		return LoadedSource, nil
	}

	// 3. Embedded floor.
	if len(embeddedModelsYAML) > 0 {
		var config ModelsConfig
		if err := yamlUnmarshal(embeddedModelsYAML, &config); err == nil {
			GlobalModelsConfig = &config
			LoadedSource = Source{
				Kind:     SourceEmbedded,
				Version:  digest(embeddedModelsYAML),
				Degraded: degraded,
			}
			logProvenance(LoadedSource)
			return LoadedSource, nil
		}
	}

	// 4. Development tree fallback: the repo checkout, where nothing is embedded
	//    because the test binary was built without the embed.
	for _, path := range []string{
		"internal/modelreg/models.yml",
		"../internal/modelreg/models.yml",
		"../modelreg/models.yml",
		"models.yml",
	} {
		cfg, ver, present, err := tryFile(path)
		if present && err == nil {
			GlobalModelsConfig = cfg
			LoadedSource = Source{Kind: SourceEmbedded, Path: path, Version: ver, Degraded: degraded}
			logProvenance(LoadedSource)
			return LoadedSource, nil
		}
	}

	return Source{}, fmt.Errorf("no models registry found: %s unset, no published registry at %s, "+
		"and no embedded or development copy", ModelsPathEnv, pubPath)
}
