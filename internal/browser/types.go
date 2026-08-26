// Package browser provides provider-neutral, secret-safe browser session
// lifecycle primitives for AILANG executors and evaluations.
package browser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const Redacted = "[REDACTED]"

// FailureCategory is the stable browser failure vocabulary used by evals.
type FailureCategory string

const (
	FailurePolicyDenied       FailureCategory = "browser_policy_denied"
	FailureCapacityExhausted  FailureCategory = "browser_capacity_exhausted"
	FailureProvision          FailureCategory = "browser_provision_failed"
	FailureConnect            FailureCategory = "browser_connect_failed"
	FailureActionTimeout      FailureCategory = "browser_action_timeout"
	FailureSessionTimeout     FailureCategory = "browser_session_timeout"
	FailureRemoteDisconnected FailureCategory = "browser_remote_disconnected"
	FailureArtifactExport     FailureCategory = "browser_artifact_export_failed"
	FailureCleanup            FailureCategory = "browser_cleanup_failed"
	FailureCostUnknown        FailureCategory = "browser_cost_unknown"
)

// Failure intentionally excludes provider response bodies and raw causes from
// its printable form. Adapters may log a separately sanitized diagnostic.
type Failure struct {
	Category FailureCategory
	Op       string
}

func NewFailure(category FailureCategory, op string, _ error) *Failure {
	return &Failure{Category: category, Op: op}
}

func (e *Failure) Error() string {
	if e == nil {
		return "browser operation failed"
	}
	if e.Op == "" {
		return string(e.Category)
	}
	return fmt.Sprintf("%s: %s failed", e.Category, e.Op)
}

func IsFailure(err error, category FailureCategory) bool {
	for err != nil {
		if failure, ok := err.(*Failure); ok {
			return failure.Category == category
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}
	return false
}

// SessionSpec contains normalized, non-secret session policy.
type SessionSpec struct {
	RunID           string        `json:"run_id"`
	Provider        string        `json:"provider,omitempty"`
	ChainID         string        `json:"chain_id,omitempty"`
	StageID         string        `json:"stage_id,omitempty"`
	Browser         string        `json:"browser,omitempty"`
	BrowserVersion  string        `json:"browser_version,omitempty"`
	MCPVersion      string        `json:"mcp_version,omitempty"`
	PolicyVersion   string        `json:"policy_version,omitempty"`
	ViewportWidth   int           `json:"viewport_width,omitempty"`
	ViewportHeight  int           `json:"viewport_height,omitempty"`
	Locale          string        `json:"locale,omitempty"`
	Timezone        string        `json:"timezone,omitempty"`
	Headless        bool          `json:"headless"`
	MaximumDuration time.Duration `json:"maximum_duration,omitempty"`
	IdleTimeout     time.Duration `json:"idle_timeout,omitempty"`
	ActionTimeout   time.Duration `json:"action_timeout,omitempty"`
	AllowedOrigins  []string      `json:"allowed_origins,omitempty"`
	BlockedOrigins  []string      `json:"blocked_origins,omitempty"`
	// ProfileRef is the RESOLVED authenticated profile reference, "alias@version".
	// It is never "alias@latest": latest is resolved to a concrete version before
	// the run starts so the result records what actually ran.
	ProfileRef string `json:"profile_ref,omitempty"`
	// ProfileHash fingerprints the stored canonical material (the sealed blob for
	// local profiles, the context reference for hosted ones). It identifies a
	// version; it does not reveal one.
	ProfileHash string `json:"profile_hash,omitempty"`
	// AuthLeaseID correlates this session with the lease that authorized it. It is
	// an opaque safe identifier and grants nothing on its own.
	AuthLeaseID string `json:"auth_lease_id,omitempty"`
	// AuthMode is "read" or "refresh". Ordinary eval and agent sessions are always
	// "read" and discard their state.
	AuthMode       string `json:"auth_mode,omitempty"`
	ArtifactDir    string `json:"-"`
	Region         string `json:"region,omitempty"`
	RecordTrace    bool   `json:"record_trace,omitempty"`
	RecordVideo    bool   `json:"record_video,omitempty"`
	AllowDownloads bool   `json:"allow_downloads,omitempty"`
	AllowUploads   bool   `json:"allow_uploads,omitempty"`
	HumanTakeover  bool   `json:"human_takeover,omitempty"`
}

type Session struct {
	ID          string    `json:"id"`
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"created_at"`
	StateDir    string    `json:"-"`
	ArtifactDir string    `json:"-"`
}

// MCPServerSpec is safe to serialize: values are never included here. EnvVars
// names environment variables the executor should forward to the MCP child.
type MCPServerSpec struct {
	Name     string   `json:"name"`
	Command  string   `json:"command"`
	Args     []string `json:"args,omitempty"`
	EnvVars  []string `json:"env_vars,omitempty"`
	Required bool     `json:"required,omitempty"`
}

// SensitiveConnection keeps secret values out of fmt, JSON, logs, errors, and
// durable task configuration. Materialize is the sole explicit extraction API.
type SensitiveConnection struct {
	mcp MCPServerSpec
	env map[string]string
}

func NewSensitiveConnection(mcp MCPServerSpec, env map[string]string) SensitiveConnection {
	cloned := make(map[string]string, len(env))
	for key, value := range env {
		cloned[key] = value
	}
	mcp.Args = append([]string(nil), mcp.Args...)
	mcp.EnvVars = append([]string(nil), mcp.EnvVars...)
	return SensitiveConnection{mcp: mcp, env: cloned}
}

func (c SensitiveConnection) String() string {
	return fmt.Sprintf("browser connection %s (%s)", Redacted, c.mcp.Name)
}

func (c SensitiveConnection) Error() string { return c.String() }

func (c SensitiveConnection) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		MCP      MCPServerSpec `json:"mcp"`
		Material string        `json:"material"`
	}{MCP: c.mcp, Material: Redacted})
}

func (c SensitiveConnection) Materialize() (MCPServerSpec, map[string]string) {
	mcp := c.mcp
	mcp.Args = append([]string(nil), c.mcp.Args...)
	mcp.EnvVars = append([]string(nil), c.mcp.EnvVars...)
	env := make(map[string]string, len(c.env))
	for key, value := range c.env {
		env[key] = value
	}
	return mcp, env
}

type InspectionRef struct {
	Available bool   `json:"available"`
	Ref       string `json:"ref,omitempty"`
}

type ArtifactRef struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	URLRef string `json:"url_ref,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

type ArtifactManifest struct {
	Complete bool          `json:"complete"`
	Refs     []ArtifactRef `json:"refs,omitempty"`
}

type Usage struct {
	DurationMS      int64 `json:"duration_ms,omitempty"`
	ActionCount     int   `json:"action_count,omitempty"`
	DisconnectCount int   `json:"disconnect_count,omitempty"`
	ReconnectCount  int   `json:"reconnect_count,omitempty"`
	ProxyBytes      int64 `json:"proxy_bytes,omitempty"`
}

type Cost struct {
	USD      *float64 `json:"usd"`
	Currency string   `json:"currency"`
	Source   string   `json:"source"`
}

type Termination string

const (
	TerminationCompleted      Termination = "completed"
	TerminationExecutorFailed Termination = "executor_failed"
	TerminationCancelled      Termination = "cancelled"
	TerminationTimeout        Termination = "timeout"
)

type BrowserRunManifest struct {
	RunID                 string           `json:"run_id"`
	ChainID               string           `json:"chain_id,omitempty"`
	StageID               string           `json:"stage_id,omitempty"`
	Provider              string           `json:"provider"`
	ProviderSessionID     string           `json:"provider_session_id"`
	ToolSurface           string           `json:"tool_surface"`
	Browser               string           `json:"browser,omitempty"`
	BrowserVersion        string           `json:"browser_version,omitempty"`
	MCPVersion            string           `json:"mcp_version,omitempty"`
	PolicyVersion         string           `json:"policy_version,omitempty"`
	ProfileHash           string           `json:"profile_hash,omitempty"`
	AuthProfileAlias      string           `json:"auth_profile_alias,omitempty"`
	AuthProfileVersion    string           `json:"auth_profile_version,omitempty"`
	AuthLeaseID           string           `json:"auth_lease_id,omitempty"`
	AuthMode              string           `json:"auth_mode,omitempty"`
	AuthErrorCategory     string           `json:"auth_error_category,omitempty"`
	ViewportWidth         int              `json:"viewport_width,omitempty"`
	ViewportHeight        int              `json:"viewport_height,omitempty"`
	Locale                string           `json:"locale,omitempty"`
	Timezone              string           `json:"timezone,omitempty"`
	Headless              bool             `json:"headless"`
	StartedAt             time.Time        `json:"started_at"`
	EndedAt               time.Time        `json:"ended_at"`
	Termination           Termination      `json:"termination"`
	Usage                 Usage            `json:"usage"`
	Cost                  Cost             `json:"cost"`
	Artifacts             ArtifactManifest `json:"artifacts"`
	Inspection            InspectionRef    `json:"inspection"`
	ArtifactErrorCategory FailureCategory  `json:"artifact_error_category,omitempty"`
	CleanupErrorCategory  FailureCategory  `json:"cleanup_error_category,omitempty"`
	ManagedVessel         bool             `json:"managed_vessel"`
	AgentScaffold         string           `json:"agent_scaffold,omitempty"`
	Comparable            bool             `json:"comparable"`
	NonComparableReason   string           `json:"non_comparable_reason,omitempty"`
}

// sensitiveKeyMarkers are matched as substrings of a normalized key. The list is
// deliberately over-inclusive: over-redacting a diagnostic costs a debugging
// round-trip, under-redacting one publishes a credential. The second group was
// added by M-BROWSER-AUTH-PROFILES, which classifies saved browser state as a
// credential equal to the password that produced it.
var sensitiveKeyMarkers = []string{
	"apikey", "token", "secret", "password", "authorization", "cookie",
	"header", "connecturl", "endpoint", "profiledata", "signingkey",

	"storagestate", "contextid", "sealed", "ciphertext", "material",
	"passkey", "otpseed", "recoverycode", "privatekey", "localstorage",
	"indexeddb", "credential",
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	for _, marker := range sensitiveKeyMarkers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

// SanitizeDiagnostics recursively redacts values under secret-bearing keys.
func SanitizeDiagnostics(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			if isSensitiveKey(key) {
				out[key] = Redacted
			} else {
				out[key] = SanitizeDiagnostics(child)
			}
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = SanitizeDiagnostics(child)
		}
		return out
	default:
		return value
	}
}
