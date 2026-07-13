package daemon

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// EnvProject maps the --env flag to a (GCP project, prefix) tuple. Mirrors
// the per-environment terraform tfvars in ailang-multivac.
var EnvProject = map[string]struct {
	Project string
	Prefix  string
}{
	"dev":  {Project: "ailang-multivac-dev", Prefix: "ailang-dev"},
	"test": {Project: "ailang-multivac-test", Prefix: "ailang-test"},
	"prod": {Project: "ailang-multivac", Prefix: "ailang"},
}

// FileConfig is the on-disk shape of ~/.ailang/config/daemon.yaml. Optional;
// every field has a sensible default.
type FileConfig struct {
	Env           string `yaml:"env"`             // dev|test|prod (default: prod)
	Inbox         string `yaml:"inbox"`           // user inbox name (default: empty = surface all)
	TaskWindowSec int    `yaml:"task_window_sec"` // dedup window for task events (default: 60)
	MsgWindowSec  int    `yaml:"msg_window_sec"`  // dedup window for messages   (default: 300)
	DryRun        bool   `yaml:"dry_run"`         // skip notifier (default: false)
	ExcludesPath  string `yaml:"excludes_path"`   // override path to notify_excludes.conf

	// ExtraMessageEnvs lists ADDITIONAL cloud environments whose inbox-message
	// subscriptions this daemon should ALSO watch, beyond the primary Env.
	// Each entry is resolved through EnvProject to a (project, prefix) and the
	// daemon subscribes to that project's MessagesSub. Default empty =
	// single-project (backward-compatible). Example: env=dev +
	// extra_message_envs=[prod] makes one process watch BOTH dev and prod
	// inbox messages (fixing silent loss of external prod feedback). Task
	// events remain primary-env only. The `--also-subscribe` CLI flag appends
	// to this list.
	ExtraMessageEnvs []string `yaml:"extra_message_envs"`
}

// ExtraMessageSource is a resolved additional inbox-message source: the env
// label plus its (project, prefix, base sub name). The CLI uses these to build
// a project-scoped fetcher + subscriber per extra env.
type ExtraMessageSource struct {
	Env         string
	Project     string
	Prefix      string
	MessagesSub string
}

// ResolveExtraMessageSources maps each extra env label to its (project, prefix)
// via EnvProject, de-duplicating against the primary env (so `env: prod` +
// `extra_message_envs: [prod]` does not double-subscribe the same project) and
// against itself. Returns an error for an unknown env label — a typo must fail
// loudly rather than silently drop a feedback source.
func ResolveExtraMessageSources(primaryEnv string, extras []string) ([]ExtraMessageSource, error) {
	seen := map[string]bool{primaryEnv: true}
	var out []ExtraMessageSource
	for _, env := range extras {
		env = strings.TrimSpace(env)
		if env == "" || seen[env] {
			continue
		}
		mapping, ok := EnvProject[env]
		if !ok {
			return nil, fmt.Errorf("unknown extra_message_env %q (want dev|test|prod)", env)
		}
		seen[env] = true
		out = append(out, ExtraMessageSource{
			Env:         env,
			Project:     mapping.Project,
			Prefix:      mapping.Prefix,
			MessagesSub: "messages-laptop",
		})
	}
	return out, nil
}

// LoadFileConfig reads ~/.ailang/config/daemon.yaml if present. Returns a
// zero-value FileConfig with no error if the file does not exist.
func LoadFileConfig() (FileConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return FileConfig{}, fmt.Errorf("home dir: %w", err)
	}
	path := filepath.Join(home, ".ailang", "config", "daemon.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return FileConfig{}, nil
	}
	if err != nil {
		return FileConfig{}, fmt.Errorf("read daemon.yaml: %w", err)
	}
	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return FileConfig{}, fmt.Errorf("parse daemon.yaml: %w", err)
	}
	return fc, nil
}

// LoadExcludes reads one substring per non-blank, non-`#`-prefixed line. Path
// defaults to ~/.ailang/config/notify_excludes.conf. Missing file is not an
// error — returns nil slice.
func LoadExcludes(path string) ([]string, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("home dir: %w", err)
		}
		path = filepath.Join(home, ".ailang", "config", "notify_excludes.conf")
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open excludes: %w", err)
	}
	defer func() { _ = f.Close() }()

	var out []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan excludes: %w", err)
	}
	return out, nil
}

// ConfigForEnv builds a daemon Config from a --env value, applying file-config
// overrides where present. Returns the resolved (project, prefix) for the
// caller to plumb into the Pub/Sub client. Returns an error if env is unknown.
func ConfigForEnv(env string, fc FileConfig) (Config, string, string, error) {
	if env == "" {
		env = fc.Env
	}
	if env == "" {
		env = "prod"
	}
	mapping, ok := EnvProject[env]
	if !ok {
		return Config{}, "", "", fmt.Errorf("unknown env %q (want dev|test|prod)", env)
	}

	taskWindow := 60 * time.Second
	if fc.TaskWindowSec > 0 {
		taskWindow = time.Duration(fc.TaskWindowSec) * time.Second
	}
	msgWindow := 5 * time.Minute
	if fc.MsgWindowSec > 0 {
		msgWindow = time.Duration(fc.MsgWindowSec) * time.Second
	}

	excludes, err := LoadExcludes(fc.ExcludesPath)
	if err != nil {
		return Config{}, "", "", err
	}

	return Config{
		EventsSub:   "events-laptop",
		MessagesSub: "messages-laptop",
		TaskWindow:  taskWindow,
		MsgWindow:   msgWindow,
		Excludes:    excludes,
		DryRun:      fc.DryRun,
	}, mapping.Project, mapping.Prefix, nil
}
