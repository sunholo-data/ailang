// Package dispatch provides opt-in mission role execution. A successful execution
// is not artifact acceptance and never advances a coordinator task on its own.
package dispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Request is supplied by a trusted local caller, not accepted from an inbox.
// InputRevision and AuthorModels are caller-supplied provenance, not attestation.
type Request struct {
	Version        int      `json:"version"`
	MissionID      string   `json:"mission_id"`
	WorkItemID     string   `json:"work_item_id"`
	StageID        string   `json:"stage_id"`
	AttemptID      string   `json:"attempt_id"`
	Role           string   `json:"role"`
	Workspace      string   `json:"workspace"`
	InputRevision  string   `json:"input_revision"`
	Instructions   string   `json:"instructions"`
	Models         []string `json:"models"`
	AuthorModels   []string `json:"author_models,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	MaxTokens      int      `json:"max_tokens"`
	MaxCostUSD     float64  `json:"max_cost_usd"`
}

func DecodeRequest(r io.Reader) (Request, error) {
	var req Request
	d := json.NewDecoder(io.LimitReader(r, 1024*1024+1))
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		return req, fmt.Errorf("decode role request: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return req, fmt.Errorf("role request must contain exactly one JSON object")
	}
	return req, req.Validate()
}

func (r Request) Validate() error {
	if r.Version != 1 {
		return fmt.Errorf("unsupported role request version %d (want 1)", r.Version)
	}
	for _, id := range []string{r.MissionID, r.WorkItemID, r.StageID, r.AttemptID} {
		if !validID(id) {
			return fmt.Errorf("invalid mission/work-item/stage/attempt id %q", id)
		}
	}
	switch r.Role {
	case "designer", "planner", "executor", "evaluator":
	default:
		return fmt.Errorf("unknown mission role %q", r.Role)
	}
	if !filepath.IsAbs(r.Workspace) {
		return fmt.Errorf("workspace must be absolute")
	}
	info, err := os.Stat(r.Workspace)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("workspace must be an existing directory: %s", r.Workspace)
	}
	if strings.TrimSpace(r.Instructions) == "" || len(r.Instructions) > 512*1024 || strings.TrimSpace(r.InputRevision) == "" {
		return fmt.Errorf("nonempty instructions (at most 512 KiB) and input_revision are required")
	}
	if r.TimeoutSeconds < 1 || r.TimeoutSeconds > 1800 || r.MaxTokens <= 0 || r.MaxCostUSD <= 0 || math.IsNaN(r.MaxCostUSD) || math.IsInf(r.MaxCostUSD, 0) {
		return fmt.Errorf("timeout_seconds must be 1..1800; max_tokens and finite max_cost_usd must be positive")
	}
	if len(r.Models) == 0 || len(r.Models) > 16 {
		return fmt.Errorf("models must contain 1..16 explicit registry keys")
	}
	seen := map[string]bool{}
	for _, name := range r.Models {
		if strings.TrimSpace(name) == "" || seen[name] {
			return fmt.Errorf("empty or duplicate model candidate %q", name)
		}
		seen[name] = true
	}
	if len(r.AuthorModels) > 16 || (r.Role == "evaluator" && len(r.AuthorModels) == 0) {
		return fmt.Errorf("evaluator requires 1..16 author_models for different-vendor independence")
	}
	return nil
}

func validID(s string) bool {
	if s == "" || len(s) > 128 || s == "." || s == ".." {
		return false
	}
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.') {
			return false
		}
	}
	return true
}

// Digest is defined for validated requests only; all request fields are bound.
func (r Request) Digest() string {
	b, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
