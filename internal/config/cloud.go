package config

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Environment variables this package owns. Nothing else reads them.
const (
	EnvCloudProject       = "AILANG_CLOUD_PROJECT"
	EnvGoogleCloudProject = "GOOGLE_CLOUD_PROJECT"
	EnvCloudRegion        = "AILANG_CLOUD_REGION"
	EnvGoogleCloudRegion  = "GOOGLE_CLOUD_REGION"
	EnvNoMetadata         = "AILANG_NO_METADATA"
	EnvConfigFile         = "AILANG_CONFIG"
)

// ErrNoCloudProject is returned by CloudProject when every source is empty.
// Callers that want to keep running on a legacy default must say so through
// DeprecatedDefault; nothing in this package guesses.
var ErrNoCloudProject = errors.New("no cloud project: set " + EnvCloudProject +
	" (or " + EnvGoogleCloudProject + ", or pubsub.project_id in ~/.ailang/config.yaml)")

// Source names where a resolved value came from, so a command that acts on a
// plane can print it: a silently-discovered project is the same failure class
// as a silently-guessed store.
type Source string

// The sources CloudProject can report.
const (
	SourceAilangEnv  Source = EnvCloudProject
	SourceGoogleEnv  Source = EnvGoogleCloudProject
	SourceConfigFile Source = "config.yaml pubsub.project_id"
	SourceMetadata   Source = "gce-metadata"
)

// CloudProject resolves the GCP project this process acts on. See the package
// comment for the precedence. It never returns a default.
func CloudProject(ctx context.Context) (string, error) {
	p, _, err := CloudProjectSource(ctx)
	return p, err
}

// CloudProjectSource is CloudProject plus the Source the value came from.
func CloudProjectSource(ctx context.Context) (string, Source, error) {
	if v := strings.TrimSpace(os.Getenv(EnvCloudProject)); v != "" {
		return v, SourceAilangEnv, nil
	}
	if v := strings.TrimSpace(os.Getenv(EnvGoogleCloudProject)); v != "" {
		return v, SourceGoogleEnv, nil
	}
	if v := projectFromConfigFile(); v != "" {
		return v, SourceConfigFile, nil
	}
	if v := projectFromMetadata(ctx); v != "" {
		return v, SourceMetadata, nil
	}
	return "", "", ErrNoCloudProject
}

// Region resolves the cloud region: AILANG_CLOUD_REGION, then
// GOOGLE_CLOUD_REGION, then the deprecated fleet default (D3).
func Region() (string, error) {
	if v := strings.TrimSpace(os.Getenv(EnvCloudRegion)); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(os.Getenv(EnvGoogleCloudRegion)); v != "" {
		return v, nil
	}
	return DeprecatedDefault(EnvCloudRegion, "europe-west1")
}

// FilePath is the user config file: AILANG_CONFIG when set, else
// ~/.ailang/config.yaml. It returns "" when the home directory is unresolvable
// and no override is set — a config file is optional, so that is not an error
// here; callers that need the file decide.
func FilePath() string {
	if p := os.Getenv(EnvConfigFile); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".ailang", "config.yaml")
}

// projectFromConfigFile reads ONLY pubsub.project_id from the config file. A
// missing or unreadable file, or one without the key, is simply "not set".
func projectFromConfigFile() string {
	path := FilePath()
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path) //nolint:gosec // the user's own config path
	if err != nil {
		return ""
	}
	var doc struct {
		PubSub struct {
			ProjectID string `yaml:"project_id"`
		} `yaml:"pubsub"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return ""
	}
	return strings.TrimSpace(doc.PubSub.ProjectID)
}

// metadataBaseURL is the GCE metadata server; tests point it at an httptest
// server. The project-id path is appended.
var metadataBaseURL = "http://metadata.google.internal"

// metadataTimeout bounds the lookup. Off GCE the host does not resolve, which
// fails fast; on a network that black-holes it, this is the ceiling.
const metadataTimeout = 500 * time.Millisecond

var metadata struct {
	mu      sync.Mutex
	done    bool
	project string
}

// projectFromMetadata asks the metadata server once per process and caches
// the answer, empty or not.
func projectFromMetadata(ctx context.Context) string {
	if os.Getenv(EnvNoMetadata) != "" {
		return ""
	}
	metadata.mu.Lock()
	defer metadata.mu.Unlock()
	if metadata.done {
		return metadata.project
	}
	metadata.done = true
	metadata.project = fetchMetadataProject(ctx)
	return metadata.project
}

func fetchMetadataProject(ctx context.Context) string {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		metadataBaseURL+"/computeMetadata/v1/project/project-id", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// resetMetadataForTest clears the per-process metadata cache.
func resetMetadataForTest() {
	metadata.mu.Lock()
	defer metadata.mu.Unlock()
	metadata.done = false
	metadata.project = ""
}

// EnvTraceProject is the telemetry-specific override for the Cloud Trace
// export project. Setting it (or EnvGoogleCloudProject) is what ENABLES Cloud
// Trace export, so this is a switch as much as a value.
const EnvTraceProject = "OTLP_GOOGLE_CLOUD_PROJECT"

// TraceProjectFromEnv returns the project Cloud Trace export should target, or
// "" when export is not enabled. It reads the environment ONLY — deliberately
// not CloudProject's yaml/metadata fallbacks, because falling through would
// switch trace export on for every process on a GCE box or every machine whose
// ~/.ailang/config.yaml names a project. Enabling export is an explicit act.
func TraceProjectFromEnv() string {
	if p := os.Getenv(EnvTraceProject); p != "" {
		return p
	}
	return os.Getenv(EnvGoogleCloudProject)
}
