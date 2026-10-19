package coordinator

import (
	"context"
	"errors"
	"time"
)

// ShadowVerdict is the closed set of comparison outcomes (axiom A11).
type ShadowVerdict string

const (
	ShadowVerdictPending       ShadowVerdict = ""               // not graded yet
	ShadowVerdictPrimaryBetter ShadowVerdict = "primary_better" //
	ShadowVerdictShadowBetter  ShadowVerdict = "shadow_better"  //
	ShadowVerdictEquivalent    ShadowVerdict = "equivalent"     //
	ShadowVerdictBothFailed    ShadowVerdict = "both_failed"    //
	ShadowVerdictUngraded      ShadowVerdict = "ungraded"       // with UngradedReason
)

// Sides of a comparison.
const (
	ShadowSidePrimary = "primary"
	ShadowSideShadow  = "shadow"
)

// ShadowRunResult is one side's outcome, as its completion reported it. The
// cloud path keeps run metrics only on observatory stages, and a shadow has no
// stage (design V18), so this record is the only place a shadow's cost and diff
// survive.
type ShadowRunResult struct {
	TaskID          string    `json:"task_id"`
	AgentID         string    `json:"agent_id,omitempty"`
	Provider        string    `json:"provider,omitempty"`
	Model           string    `json:"model,omitempty"`
	Status          string    `json:"status"`
	BaseCommit      string    `json:"base_commit,omitempty"`
	HeadCommit      string    `json:"head_commit,omitempty"`
	ChangedFiles    []string  `json:"changed_files,omitempty"`
	DiffStat        string    `json:"diff_stat,omitempty"`
	Diff            string    `json:"diff,omitempty"` // capped by the job (maxCloudDiffBytes)
	CostUSD         float64   `json:"cost_usd,omitempty"`
	InputTokens     int       `json:"input_tokens,omitempty"`
	OutputTokens    int       `json:"output_tokens,omitempty"`
	NumTurns        int       `json:"num_turns,omitempty"`
	ToolCallCount   int       `json:"tool_call_count,omitempty"`
	DurationMS      int64     `json:"duration_ms,omitempty"`
	SessionID       string    `json:"session_id,omitempty"`
	ArtifactGCSPath string    `json:"artifact_gcs_path,omitempty"`
	ErrorMsg        string    `json:"error_msg,omitempty"`
	CompletedAt     time.Time `json:"completed_at"`
}

// ShadowComparison pairs a primary task with its shadow, keyed by the primary's
// ID. Each side is written by its own completion, in whichever order they land;
// the grade layers (M5, M6) fill in the rest.
type ShadowComparison struct {
	ParentID string           `json:"parent_id"`
	Primary  *ShadowRunResult `json:"primary,omitempty"`
	Shadow   *ShadowRunResult `json:"shadow,omitempty"`
	// BaseDrift is set when both sides are in and cloned different commits:
	// the shadow started from different code, and the report counts it.
	BaseDrift      bool          `json:"base_drift,omitempty"`
	Verdict        ShadowVerdict `json:"verdict,omitempty"`
	UngradedReason string        `json:"ungraded_reason,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// Complete reports whether both sides have landed.
func (c *ShadowComparison) Complete() bool { return c.Primary != nil && c.Shadow != nil }

// MergeShadowRun returns c with one side set and BaseDrift recomputed. Pure, so
// both stores merge the same way; a redelivered completion rewrites its side with
// the same values.
func MergeShadowRun(c ShadowComparison, side string, run ShadowRunResult, now time.Time) (ShadowComparison, error) {
	switch side {
	case ShadowSidePrimary:
		c.Primary = &run
	case ShadowSideShadow:
		c.Shadow = &run
	default:
		return c, errors.New("shadow comparison: unknown side " + side)
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	c.BaseDrift = c.Complete() && c.Primary.BaseCommit != "" && c.Shadow.BaseCommit != "" &&
		c.Primary.BaseCommit != c.Shadow.BaseCommit
	return c, nil
}

// ErrShadowComparisonNotFound is returned by GetShadowComparison for an unknown parent.
var ErrShadowComparisonNotFound = errors.New("shadow comparison not found")

// ShadowComparisonStore is implemented by the SQLite and Firestore task stores.
// It is an optional interface, asserted on the task store, so no other Store
// implementation has to grow it.
type ShadowComparisonStore interface {
	// RecordShadowRun merges one side into the parent's comparison, creating it
	// if needed. Atomic per parent: the two sides can complete concurrently on
	// different coordinator instances.
	RecordShadowRun(ctx context.Context, parentID, side string, run ShadowRunResult) (*ShadowComparison, error)
	GetShadowComparison(ctx context.Context, parentID string) (*ShadowComparison, error)
}
