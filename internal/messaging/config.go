package messaging

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// GitHubConfig holds configuration for GitHub integration
type GitHubConfig struct {
	// DefaultRepo is the default repository for --github flag (e.g., "sunholo-data/ailang")
	DefaultRepo string `yaml:"default_repo"`

	// InboxRepos maps inbox names (or prefixes) to specific GitHub repositories.
	// This allows messages sent to different inboxes to create issues in the correct repo.
	// Supports exact matches and prefix matches (e.g., "pkg:" matches "pkg:sunholo/auth").
	InboxRepos map[string]string `yaml:"inbox_repos"`

	// ExpectedUser is the expected GitHub username (must match gh auth status)
	// This prevents accidentally creating issues under the wrong account
	ExpectedUser string `yaml:"expected_user"`

	// ExecTimeout bounds every `gh` shell-out this client makes. Zero means
	// defaultExecTimeout (30s), chosen to match the mission driver's existing
	// `_mc_bounded 30`. Without it a hung `gh` blocks the caller forever, which
	// on the mission path means a wedged fire (design doc V18 / objection P-1).
	ExecTimeout time.Duration `yaml:"exec_timeout"`

	// CreateLabels are labels to add when creating issues
	CreateLabels []string `yaml:"create_labels"`

	// WatchLabels are labels to watch for incoming messages (import)
	WatchLabels []string `yaml:"watch_labels"`

	// AutoImport enables automatic import on session start (default: true)
	AutoImport *bool `yaml:"auto_import"`

	// TrustedAuthors are the GitHub logins whose issues may carry DIRECTIVE weight:
	// agent-inbox routing, bug/feature categorisation (which feeds the auto-triage
	// router), and the `[agent-name]` title prefix that sets a message's sender.
	//
	// SECURITY: the repo is public, so anyone can open an issue, and the issue
	// TEMPLATES auto-apply `bug`/`enhancement` labels with no write access required.
	// Label filtering alone therefore proves nothing about who is speaking. Issues
	// from anyone else still import — losing public feedback would be worse — but
	// as INERT feedback: no category, no inbox routing, no spoofable sender.
	//
	// Empty list means only ExpectedUser is trusted, which is the conservative
	// default and matches the sole author of every issue in the repo today.
	TrustedAuthors []string `yaml:"trusted_authors"`
}

// IsTrustedAuthor reports whether a GitHub login may author a directive.
// GitHub logins are case-insensitive, so the comparison is too — otherwise
// "markedmondson1234" would be treated as a stranger, or worse, a lookalike
// login could be waved through by a case-sensitive miss.
func (c *GitHubConfig) IsTrustedAuthor(login string) bool {
	if c == nil || login == "" {
		return false
	}
	if len(c.TrustedAuthors) == 0 {
		return c.ExpectedUser != "" && strings.EqualFold(login, c.ExpectedUser)
	}
	for _, a := range c.TrustedAuthors {
		if strings.EqualFold(login, a) {
			return true
		}
	}
	return false
}

// RepoForInbox returns the GitHub repository for a given inbox name.
// Resolution order: exact match in inbox_repos, then prefix match, then default_repo.
func (c *GitHubConfig) RepoForInbox(inbox string) string {
	if c == nil {
		return ""
	}
	if len(c.InboxRepos) == 0 {
		return c.DefaultRepo
	}

	// Exact match first
	if repo, ok := c.InboxRepos[inbox]; ok {
		return repo
	}

	// Prefix match (e.g., "pkg:" matches "pkg:sunholo/auth")
	for prefix, repo := range c.InboxRepos {
		if strings.HasSuffix(prefix, ":") && strings.HasPrefix(inbox, prefix) {
			return repo
		}
	}

	return c.DefaultRepo
}

// IsAutoImportEnabled returns whether auto-import is enabled (defaults to true)
func (c *GitHubConfig) IsAutoImportEnabled() bool {
	if c.AutoImport == nil {
		return true // default
	}
	return *c.AutoImport
}

// EmbeddingsYAMLConfig holds YAML configuration for embeddings
type EmbeddingsYAMLConfig struct {
	// Provider: "ollama", "openai", "gemini", or "none"
	Provider string `yaml:"provider"`

	// Ollama-specific settings
	Ollama struct {
		Model     string `yaml:"model"`
		Endpoint  string `yaml:"endpoint"`
		Dimension int    `yaml:"dimension"`
		Timeout   string `yaml:"timeout"` // e.g., "30s"
		BatchSize int    `yaml:"batch_size"`
	} `yaml:"ollama"`

	// OpenAI-specific settings (M-SEMANTIC-ENVELOPE)
	OpenAI struct {
		APIKey    string `yaml:"api_key"`   // Falls back to OPENAI_API_KEY env
		Model     string `yaml:"model"`     // e.g. "text-embedding-3-small"
		Dimension int    `yaml:"dimension"` // 0 = model default
		Timeout   string `yaml:"timeout"`
	} `yaml:"openai"`

	// Gemini-specific settings (M-SEMANTIC-ENVELOPE)
	Gemini struct {
		APIKey    string `yaml:"api_key"`   // Falls back to GOOGLE_API_KEY env
		Model     string `yaml:"model"`     // e.g. "text-embedding-004"
		Dimension int    `yaml:"dimension"` // 0 = model default
		Timeout   string `yaml:"timeout"`
	} `yaml:"gemini"`

	// Search behavior
	Search struct {
		DefaultMode       string  `yaml:"default_mode"` // "simhash" or "neural"
		AutoEmbedOnInsert bool    `yaml:"auto_embed_on_insert"`
		SimhashThreshold  float64 `yaml:"simhash_threshold"`
		NeuralThreshold   float64 `yaml:"neural_threshold"`
	} `yaml:"search"`
}

// PubSubConfig holds Pub/Sub messaging configuration (M-PUBSUB).
type PubSubConfig struct {
	Enabled            bool   `yaml:"enabled"`
	ProjectID          string `yaml:"project_id"`          // Defaults to AILANG_CLOUD_PROJECT
	TopicPrefix        string `yaml:"topic_prefix"`        // Defaults to "ailang"
	LaptopSubscription bool   `yaml:"laptop_subscription"` // Create pull subscription for laptop
}

// Config holds the full AILANG configuration
type Config struct {
	GitHub     *GitHubConfig         `yaml:"github"`
	Embeddings *EmbeddingsYAMLConfig `yaml:"embeddings"`
	PubSub     *PubSubConfig         `yaml:"pubsub"`
}

// GetConfigPath returns the path of the AILANG config file: AILANG_CONFIG
// when set, else ~/.ailang/config.yaml (config.FilePath). This package used
// to spell the home path itself and so ignored AILANG_CONFIG while the
// coordinator honoured it — two loaders, two answers.
func GetConfigPath() string {
	return config.FilePath()
}

// LoadConfig loads the messaging sections (github, embeddings, pubsub) of
// the config file through the one loader in internal/config. Returns nil,
// nil when there is no config file (not an error); a file that does not
// parse is an error.
func LoadConfig() (*Config, error) {
	f, err := config.Load()
	if err != nil {
		if errors.Is(err, config.ErrConfigNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	for key, out := range map[string]any{
		"github":     &cfg.GitHub,
		"embeddings": &cfg.Embeddings,
		"pubsub":     &cfg.PubSub,
	} {
		if _, err := f.Section(key, out); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}
	return &cfg, nil
}

// LoadGitHubConfig loads just the GitHub configuration
// Returns nil if GitHub config is not present (not an error)
func LoadGitHubConfig() (*GitHubConfig, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	return config.GitHub, nil
}

// LoadEmbeddingsConfig loads the embeddings configuration
// Returns nil if embeddings config is not present (not an error)
func LoadEmbeddingsConfig() (*EmbeddingsYAMLConfig, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, nil
	}
	return config.Embeddings, nil
}

// ValidateGitHubConfig validates the GitHub configuration
// Returns an error if required fields are missing
func ValidateGitHubConfig(config *GitHubConfig) error {
	if config == nil {
		return fmt.Errorf("github configuration not found in ~/.ailang/config.yaml")
	}

	if config.ExpectedUser == "" {
		return fmt.Errorf("github.expected_user is required in ~/.ailang/config.yaml")
	}

	if config.DefaultRepo == "" {
		return fmt.Errorf("github.default_repo is required in ~/.ailang/config.yaml")
	}

	return nil
}
