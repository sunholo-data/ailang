package feedbackgate

// autoPrefix is the category prefix the feedback publisher stamps when
// AutoDispatch is set (internal/feedback/publisher.go). A category WITHOUT
// this prefix is not authorized for agentic dispatch.
const autoPrefix = "auto:"

// Gate modes for FeedbackGateConfig.Mode. Env AILANG_FEEDBACK_GATE_MODE
// overrides the config value at the wiring layer (M4).
const (
	// ModeOff is a pass-through: the gate is skipped entirely. Equivalent to
	// the gate being disabled. (The M4 wiring also honors enabled=false.)
	ModeOff = "off"
	// ModeFileOnly runs deterministic rules + cooldown but disables the
	// classifier stage (no LLM calls). An emergency kill-switch for the
	// classifier while keeping the cheap defenses on.
	ModeFileOnly = "file-only"
	// ModeFull runs everything: rules + cooldown + classifier.
	ModeFull = "full"
)

// FeedbackGateConfig configures the gate. It is parsed from the
// coordinator.feedback_gate block of ~/.ailang/config.yaml. Opt-in: the M4
// wiring treats Enabled=false (the default) as a full pass-through.
//
// The Cooldown and Classifier fields are injected dependencies (not YAML). The
// coordinator wiring constructs them (Firestore-backed cooldown, real
// ai.Provider classifier) only in cloud mode; unit tests inject fakes. When
// nil, the corresponding stage is a no-op, which is what keeps Decide pure at
// M1.
type FeedbackGateConfig struct {
	// Enabled turns the gate on. Off by default — existing deployments see
	// zero behavior change until they explicitly enable it.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Mode is the operator kill-switch: off | file-only | full. Empty
	// defaults to ModeFull. Env AILANG_FEEDBACK_GATE_MODE overrides at M4.
	Mode string `yaml:"mode" json:"mode,omitempty"`

	// DryRun, when true, runs the full gate and records the verdict it WOULD
	// have applied, but the M4 wiring always dispatches anyway. For
	// false-positive tuning. Env AILANG_FEEDBACK_GATE_DRY_RUN overrides at M4.
	DryRun bool `yaml:"dry_run" json:"dry_run,omitempty"`

	// AllowedSenders is the sender allowlist. A From value not in this list
	// (and not matching the agent-* prefix) is rejected as untrusted. Empty
	// defaults to {"mcp-public"}. agent-* is always allowed regardless.
	AllowedSenders []string `yaml:"allowed_senders" json:"allowed_senders,omitempty"`

	// MaxBodyBytes rejects submissions whose trimmed body exceeds this size.
	// Zero defaults to 8192 (8KB).
	MaxBodyBytes int `yaml:"max_body_bytes" json:"max_body_bytes,omitempty"`

	// KnownCategories are the categories (stripped of the auto: prefix) that
	// are eligible for dispatch. A category outside this set is filed for
	// human routing. Empty defaults to {bug,feature,docs,limitation}.
	KnownCategories []string `yaml:"known_categories" json:"known_categories,omitempty"`

	// MaxDispatchPerHour / MaxDispatchPerDay bound per-contact dispatches (M2).
	// Zero defaults to 3 / 10 respectively.
	MaxDispatchPerHour int `yaml:"max_dispatch_per_hour" json:"max_dispatch_per_hour,omitempty"`
	MaxDispatchPerDay  int `yaml:"max_dispatch_per_day" json:"max_dispatch_per_day,omitempty"`

	// ClassifierModel is the model used for the M3 classifier. Empty defaults
	// to "claude-haiku-4-5".
	ClassifierModel string `yaml:"classifier_model" json:"classifier_model,omitempty"`

	// DailyBudgetUSD caps the classifier's daily spend (M5). At/over budget,
	// the classifier stage short-circuits to file (never dispatch). Zero
	// defaults to 5.0.
	DailyBudgetUSD float64 `yaml:"daily_budget_usd" json:"daily_budget_usd,omitempty"`

	// Cooldown is the injected per-contact sliding-window store (M2). Nil =>
	// cooldown stage skipped (pure M1).
	Cooldown CooldownStore `yaml:"-" json:"-"`

	// Classifier is the injected last-resort JSON classifier (M3). Nil =>
	// classifier stage skipped.
	Classifier *Classifier `yaml:"-" json:"-"`
}

// normalized returns a copy of the config with empty/zero fields filled with
// defaults, so the stages can rely on populated values.
func (c FeedbackGateConfig) normalized() FeedbackGateConfig {
	if c.Mode == "" {
		c.Mode = ModeFull
	}
	if len(c.AllowedSenders) == 0 {
		c.AllowedSenders = []string{"mcp-public"}
	}
	if c.MaxBodyBytes == 0 {
		c.MaxBodyBytes = 8192
	}
	if len(c.KnownCategories) == 0 {
		c.KnownCategories = []string{"bug", "feature", "docs", "limitation"}
	}
	if c.MaxDispatchPerHour == 0 {
		c.MaxDispatchPerHour = 3
	}
	if c.MaxDispatchPerDay == 0 {
		c.MaxDispatchPerDay = 10
	}
	if c.ClassifierModel == "" {
		c.ClassifierModel = "claude-haiku-4-5"
	}
	if c.DailyBudgetUSD == 0 {
		c.DailyBudgetUSD = 5.0
	}
	return c
}
