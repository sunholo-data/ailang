// Package auth provides persistent authenticated browser identity for AILANG
// browser sessions: immutable profile versions, credential-grade material
// handling, leases, and policy.
//
// It deliberately does NOT import internal/browser. The dependency runs the
// other way — internal/browser consumes this package for the session lifecycle —
// so the failure vocabulary and redaction constant are defined locally rather
// than reused from the parent package.
package auth

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

// Redacted mirrors browser.Redacted. It is duplicated rather than imported to
// keep the dependency direction one-way; a test in internal/browser asserts the
// two stay identical.
const Redacted = "[REDACTED]"

// VersionLatest is the only version token that is not a stored version. It is
// resolved to a concrete version once, before a run starts, and the concrete
// value is what gets recorded.
const VersionLatest = "latest"

// FailureCategory is the stable browser-auth failure vocabulary. These strings
// are queried by eval tooling; changing a spelling is a breaking change.
type FailureCategory string

const (
	FailureProfileNotFound      FailureCategory = "browser_auth_profile_not_found"
	FailureProfileExpired       FailureCategory = "browser_auth_profile_expired"
	FailureProfileRevoked       FailureCategory = "browser_auth_profile_revoked"
	FailureLeaseConflict        FailureCategory = "browser_auth_lease_conflict"
	FailureScopeDenied          FailureCategory = "browser_auth_scope_denied"
	FailureRefreshRequired      FailureCategory = "browser_auth_refresh_required"
	FailureMaterializeFailed    FailureCategory = "browser_auth_materialize_failed"
	FailureWritebackDenied      FailureCategory = "browser_auth_writeback_denied"
	FailureArtifactPolicyDenied FailureCategory = "browser_auth_artifact_policy_denied"
	FailureCleanupFailed        FailureCategory = "browser_auth_cleanup_failed"
)

// AllFailureCategories returns every allocated category. Used by tests and by
// eval tooling that enumerates the vocabulary.
func AllFailureCategories() []FailureCategory {
	return []FailureCategory{
		FailureProfileNotFound,
		FailureProfileExpired,
		FailureProfileRevoked,
		FailureLeaseConflict,
		FailureScopeDenied,
		FailureRefreshRequired,
		FailureMaterializeFailed,
		FailureWritebackDenied,
		FailureArtifactPolicyDenied,
		FailureCleanupFailed,
	}
}

// Failure is the public error. Like browser.Failure it excludes the underlying
// cause from its printable form: a vault error, a provider response body, or a
// decrypt error can all carry secret material. Adapters may log a separately
// sanitized diagnostic.
//
// Reason is an operator-safe token (for example "artifact_policy_absent"). It
// must never carry a value, only a classification.
type Failure struct {
	Category  FailureCategory
	Op        string
	Reason    string
	Retryable bool
}

func NewFailure(category FailureCategory, op string, _ error) *Failure {
	return &Failure{Category: category, Op: op}
}

// NewFailureReason attaches an operator-safe classification token.
func NewFailureReason(category FailureCategory, op, reason string) *Failure {
	return &Failure{Category: category, Op: op, Reason: reason}
}

func (e *Failure) Error() string {
	if e == nil {
		return "browser auth operation failed"
	}
	message := string(e.Category)
	if e.Op != "" {
		message = fmt.Sprintf("%s: %s failed", e.Category, e.Op)
	}
	if e.Reason != "" {
		message = fmt.Sprintf("%s (%s)", message, e.Reason)
	}
	return message
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

// AuthProfileRef names a profile. Alias is operator-defined and safe to log;
// Version is either a concrete stored version or VersionLatest.
type AuthProfileRef struct {
	Alias   string `json:"alias"`
	Version string `json:"version"`
}

func (r AuthProfileRef) String() string { return r.Alias + "@" + r.Version }

func (r AuthProfileRef) IsLatest() bool { return r.Version == VersionLatest }

// ParseRef parses "alias" or "alias@version". A bare alias means latest.
func ParseRef(text string) (AuthProfileRef, error) {
	alias, version := text, VersionLatest
	if head, tail, found := strings.Cut(text, "@"); found {
		alias, version = head, tail
		if strings.Contains(version, "@") {
			return AuthProfileRef{}, fmt.Errorf("profile reference %q has more than one @", text)
		}
	}
	if err := validateAlias(alias); err != nil {
		return AuthProfileRef{}, err
	}
	if version != VersionLatest {
		if err := validateVersion(version); err != nil {
			return AuthProfileRef{}, err
		}
	}
	return AuthProfileRef{Alias: alias, Version: version}, nil
}

// safeToken accepts lowercase alphanumerics plus '-', '_', and '.'. It is
// deliberately narrow: aliases and versions reach audit records, log lines, and
// filesystem paths, and ".." must never be expressible.
func safeToken(text string) bool {
	if text == "" || text == "." || text == ".." {
		return false
	}
	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return !strings.Contains(text, "..")
}

func validateAlias(alias string) error {
	if !safeToken(alias) {
		return fmt.Errorf("profile alias %q must be lowercase [a-z0-9._-] and must not contain '..'", alias)
	}
	return nil
}

func validateVersion(version string) error {
	if version == VersionLatest {
		return fmt.Errorf("%q is reserved and cannot be a stored version", VersionLatest)
	}
	if !safeToken(version) {
		return fmt.Errorf("profile version %q must be lowercase [a-z0-9._-] and must not contain '..'", version)
	}
	return nil
}

// AccountClass records how much authority the underlying website account has.
// The username itself is never stored — only its class.
type AccountClass string

const (
	AccountReadonly   AccountClass = "readonly"
	AccountMutable    AccountClass = "mutable"
	AccountPrivileged AccountClass = "privileged"
)

func (c AccountClass) Valid() bool {
	switch c {
	case AccountReadonly, AccountMutable, AccountPrivileged:
		return true
	default:
		return false
	}
}

// EgressBoundary records whether destination policy is actually enforced below
// the browser. M-BROWSER-EGRESS-BOUNDARY is not implemented yet, so today the
// only reachable non-empty value is EgressOperatorAcknowledged — an explicit,
// audited operator decision to proceed without enforcement. The empty value
// fails closed.
type EgressBoundary string

const (
	EgressAbsent               EgressBoundary = ""
	EgressOperatorAcknowledged EgressBoundary = "operator_acknowledged_unenforced"
	EgressEnforced             EgressBoundary = "enforced"
)

// AuthProfilePolicy is the authority a profile grants. Every field fails closed
// when unset: no origins means no navigation, nil artifacts means no exports,
// and an absent egress boundary denies the session outright.
type AuthProfilePolicy struct {
	AllowedOrigins     []string       `json:"allowed_origins"`
	AccountClass       AccountClass   `json:"account_class"`
	MaxConcurrent      int            `json:"max_concurrent"`
	AllowArtifacts     []string       `json:"allow_artifacts"`
	AllowHumanTakeover bool           `json:"allow_human_takeover"`
	ExpiresAt          time.Time      `json:"expires_at,omitzero"`
	EgressBoundary     EgressBoundary `json:"egress_boundary,omitempty"`
}

// NormalizeOrigin reduces a URL to an exact origin: scheme://host[:port], lower
// cased, with the default port elided. Anything that is not exactly an origin —
// a path, a query, a wildcard, credentials, a non-http scheme — is rejected
// rather than coerced, because a coerced origin silently widens authority.
func NormalizeOrigin(text string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("origin is empty")
	}
	if strings.Contains(text, "*") {
		return "", fmt.Errorf("origin %q contains a wildcard; exact origins only", text)
	}
	parsed, err := url.Parse(text)
	if err != nil {
		return "", fmt.Errorf("origin %q is not a URL", text)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("origin %q must use http or https", text)
	}
	if parsed.User != nil {
		return "", fmt.Errorf("origin %q must not carry credentials", text)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("origin %q must not carry a query or fragment", text)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", fmt.Errorf("origin %q must not carry a path", text)
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("origin %q has no host", text)
	}
	port := parsed.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	if port != "" {
		return scheme + "://" + host + ":" + port, nil
	}
	return scheme + "://" + host, nil
}

// AllowsOrigin reports whether target's origin exactly matches an allowed one.
// A target that cannot be normalized is denied.
func (p AuthProfilePolicy) AllowsOrigin(target string) bool {
	parsed, err := url.Parse(target)
	if err != nil {
		return false
	}
	origin, err := NormalizeOrigin(parsed.Scheme + "://" + parsed.Host)
	if err != nil {
		return false
	}
	return slices.Contains(p.AllowedOrigins, origin)
}

// ArtifactPolicyPresent distinguishes "the operator made no decision" (nil) from
// "the operator decided to allow nothing" (empty, non-nil). The first denies the
// session; the second allows the session with no exports.
func (p AuthProfilePolicy) ArtifactPolicyPresent() bool { return p.AllowArtifacts != nil }

func (p AuthProfilePolicy) AllowsArtifact(class string) bool {
	return slices.Contains(p.AllowArtifacts, class)
}

func (p AuthProfilePolicy) EgressBoundaryPresent() bool { return p.EgressBoundary != EgressAbsent }

func (p AuthProfilePolicy) EgressBoundaryEnforced() bool { return p.EgressBoundary == EgressEnforced }

// Validate rejects a policy that could not be honored. It does not decide
// whether a session may run — that is the preflight's job.
func (p AuthProfilePolicy) Validate() error {
	if !p.AccountClass.Valid() {
		return fmt.Errorf("account class %q is not one of readonly, mutable, privileged", p.AccountClass)
	}
	if p.MaxConcurrent < 1 {
		return fmt.Errorf("max_concurrent must be at least 1, got %d", p.MaxConcurrent)
	}
	for _, origin := range p.AllowedOrigins {
		if _, err := NormalizeOrigin(origin); err != nil {
			return err
		}
	}
	return nil
}

// Normalized returns a copy whose origins are normalized and deduplicated.
func (p AuthProfilePolicy) Normalized() (AuthProfilePolicy, error) {
	out := p
	out.AllowedOrigins = nil
	seen := make(map[string]bool, len(p.AllowedOrigins))
	for _, origin := range p.AllowedOrigins {
		normalized, err := NormalizeOrigin(origin)
		if err != nil {
			return AuthProfilePolicy{}, err
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		out.AllowedOrigins = append(out.AllowedOrigins, normalized)
	}
	if p.AllowArtifacts != nil {
		out.AllowArtifacts = append([]string{}, p.AllowArtifacts...)
	}
	return out, nil
}

// SafeProfile is the public projection of a profile version. Every field is safe
// to serialize into results, audit records, and manifests.
type SafeProfile struct {
	Alias            string            `json:"alias"`
	Version          string            `json:"version"`
	Sequence         int               `json:"sequence"`
	ProfileHash      string            `json:"profile_hash"`
	Provider         string            `json:"provider"`
	Policy           AuthProfilePolicy `json:"policy"`
	CreatedAt        time.Time         `json:"created_at"`
	PreviousVersion  string            `json:"previous_version,omitempty"`
	RetiredAt        time.Time         `json:"retired_at,omitzero"`
	RevokedAt        time.Time         `json:"revoked_at,omitzero"`
	RevocationReason string            `json:"revocation_reason,omitempty"`

	// ExpiresAtOrZero mirrors Policy.ExpiresAt so lifecycle checks do not have
	// to reach through the policy. Zero means "no declared expiry".
	ExpiresAtOrZero time.Time `json:"expires_at,omitzero"`
}

func (p SafeProfile) Ref() AuthProfileRef {
	return AuthProfileRef{Alias: p.Alias, Version: p.Version}
}

func (p SafeProfile) Revoked() bool { return !p.RevokedAt.IsZero() }

func (p SafeProfile) Retired() bool { return !p.RetiredAt.IsZero() }

func (p SafeProfile) Expired(now time.Time) bool {
	return !p.ExpiresAtOrZero.IsZero() && !now.Before(p.ExpiresAtOrZero)
}

// LeaseMode separates ordinary reads from the audited refresh operation.
type LeaseMode string

const (
	LeaseRead    LeaseMode = "read"
	LeaseRefresh LeaseMode = "refresh"
)

func (m LeaseMode) Exclusive() bool { return m == LeaseRefresh }

// Writes reports whether this mode may publish a new canonical version.
func (m LeaseMode) Writes() bool { return m == LeaseRefresh }

func (m LeaseMode) Valid() bool { return m == LeaseRead || m == LeaseRefresh }

// RunIdentity is the audit subject: who took the lease and for what run.
type RunIdentity struct {
	RunID     string `json:"run_id"`
	ChainID   string `json:"chain_id,omitempty"`
	StageID   string `json:"stage_id,omitempty"`
	Principal string `json:"principal"`
}

// ProfileLease is a safe, serializable record of held authority. SafeID is an
// opaque identifier for correlation; it grants nothing on its own.
type ProfileLease struct {
	SafeID      string      `json:"safe_id"`
	Alias       string      `json:"alias"`
	Version     string      `json:"version"`
	ProfileHash string      `json:"profile_hash"`
	Mode        LeaseMode   `json:"mode"`
	Run         RunIdentity `json:"run"`
	AcquiredAt  time.Time   `json:"acquired_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
}

func (l ProfileLease) Ref() AuthProfileRef {
	return AuthProfileRef{Alias: l.Alias, Version: l.Version}
}

func (l ProfileLease) Expired(now time.Time) bool {
	return !l.ExpiresAt.IsZero() && !now.Before(l.ExpiresAt)
}

// AuthProfileBroker is the boundary the eval harness and browser controller use.
// The model never reaches any of these methods: it receives an already
// authenticated browser and a safe alias/hash in the result.
type AuthProfileBroker interface {
	// Resolve converts alias@version (possibly latest) into immutable safe
	// metadata, or a terminal failure for missing, expired, or revoked profiles.
	Resolve(ctx context.Context, ref AuthProfileRef) (SafeProfile, error)

	// Acquire takes a lease. Refresh mode is exclusive; read mode is bounded by
	// the policy's MaxConcurrent.
	Acquire(ctx context.Context, profile SafeProfile, run RunIdentity, mode LeaseMode) (ProfileLease, error)

	// Materialize produces disposable, provider-specific material under dst. The
	// returned value is opaque; only the provider adapter extracts it.
	Materialize(ctx context.Context, lease ProfileLease, provider string, dst string) (SensitiveProfileMaterial, error)

	// Release destroys materialization and frees the lease. It is idempotent.
	Release(ctx context.Context, lease ProfileLease) error

	// Revoke marks a version permanently unusable.
	Revoke(ctx context.Context, ref AuthProfileRef, reason string) error
}
