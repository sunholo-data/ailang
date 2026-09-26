// Package iteration defines frozen mission work inputs and acceptance contracts.
package iteration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"path"
	"strings"
)

const (
	MaxSpecBytes   = 1024 * 1024
	MaxResultBytes = 64 * 1024
	ResultFile     = "stage-result.json"
)

type Limits struct {
	TimeoutSeconds int     `json:"timeout_seconds"`
	MaxTokens      int     `json:"max_tokens"`
	MaxCostUSD     float64 `json:"max_cost_usd"`
}
type Criterion struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type ArtifactRef struct {
	Commit string `json:"commit"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type AuthorityRef struct {
	Revision       string `json:"revision"`
	Path           string `json:"path"`
	Locator        string `json:"locator"`
	SHA256         string `json:"sha256"`
	ArtifactDigest string `json:"artifact_digest"`
}
type Prerequisite struct {
	Role          string         `json:"role"`
	Artifact      ArtifactRef    `json:"artifact"`
	AuthorityRefs []AuthorityRef `json:"authority_refs"`
	AuthorModels  []string       `json:"author_models"`
}
type Verification struct {
	ID             string   `json:"id"`
	Argv           []string `json:"argv"`
	Cwd            string   `json:"cwd"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}
type Stage struct {
	ID                string         `json:"id"`
	Role              string         `json:"role"`
	Instructions      string         `json:"instructions"`
	RequiredArtifacts []string       `json:"required_artifacts"`
	AuthorityRefs     []AuthorityRef `json:"authority_refs"`
	Limits            Limits         `json:"limits"`
}
type Spec struct {
	Version            int            `json:"version"`
	MissionID          string         `json:"mission_id"`
	WorkItemID         string         `json:"work_item_id"`
	Repository         string         `json:"repository"`
	BaseRevision       string         `json:"base_revision"`
	ReviewBaseRevision string         `json:"review_base_revision,omitempty"`
	Brief              string         `json:"brief"`
	AllowedPaths       []string       `json:"allowed_paths"`
	Workflow           string         `json:"workflow"`
	Stages             []Stage        `json:"stages"`
	Prerequisites      []Prerequisite `json:"prerequisites"`
	Verification       []Verification `json:"verification"`
	Limits             Limits         `json:"limits"`
	AcceptanceCriteria []Criterion    `json:"acceptance_criteria"`
}

func Decode(r io.Reader) (Spec, error) {
	var s Spec
	if err := decodeStrict(r, MaxSpecBytes, &s); err != nil {
		return s, err
	}
	return s, s.Validate()
}

func (s Spec) Validate() error {
	if s.Version != 1 || s.Workflow != "full-v1" {
		return fmt.Errorf("version 1 and workflow full-v1 are required")
	}
	if !validID(s.MissionID) || !validID(s.WorkItemID) {
		return fmt.Errorf("invalid mission_id or work_item_id")
	}
	if strings.TrimSpace(s.Repository) == "" || strings.TrimSpace(s.Repository) != s.Repository || strings.ContainsAny(s.Repository, "\x00\r\n\t ") {
		return fmt.Errorf("repository must be an explicit normalized origin")
	}
	if s.ReviewBaseRevision != "" && !validRevision(s.ReviewBaseRevision) {
		return fmt.Errorf("review_base_revision must be a full Git commit ID")
	}
	if !validRevision(s.BaseRevision) {
		return fmt.Errorf("base_revision must be a full Git commit ID")
	}
	if strings.TrimSpace(s.Brief) == "" || len(s.Brief) > 512*1024 {
		return fmt.Errorf("brief must be nonempty and at most 512 KiB")
	}
	if err := validatePaths(s.AllowedPaths, true, true); err != nil {
		return fmt.Errorf("allowed_paths: %w", err)
	}
	if err := s.Limits.validate(7200); err != nil {
		return err
	}
	if len(s.Stages) < 1 || len(s.Stages) > 4 || len(s.Prerequisites) > 3 {
		return fmt.Errorf("require 1..4 stages and at most 3 prerequisites")
	}
	if s.ReviewBaseRevision != "" && (len(s.Stages) != 1 || s.Stages[0].Role != "evaluator" || len(s.Prerequisites) != 3 || s.Prerequisites[2].Role != "executor") {
		return fmt.Errorf("review_base_revision is only valid for evaluator-only work with imported executor evidence")
	}
	roles := []string{"designer", "planner", "executor", "evaluator"}
	if len(s.Stages)+len(s.Prerequisites) != len(roles) {
		return fmt.Errorf("full-v1 must account for all four roles")
	}
	for i, p := range s.Prerequisites {
		if p.Role != roles[i] {
			return fmt.Errorf("prerequisites must supply preceding roles in order")
		}
		if !validRevision(p.Artifact.Commit) || !validPath(p.Artifact.Path, false) || !validHash(p.Artifact.SHA256) {
			return fmt.Errorf("invalid prerequisite artifact")
		}
		if len(p.AuthorModels) < 1 || len(p.AuthorModels) > 16 {
			return fmt.Errorf("prerequisite requires 1..16 author model registry identities")
		}
		seen := map[string]bool{}
		for _, m := range p.AuthorModels {
			if strings.TrimSpace(m) == "" || seen[m] {
				return fmt.Errorf("invalid or duplicate author model")
			}
			seen[m] = true
		}
		if len(p.AuthorityRefs) == 0 {
			return fmt.Errorf("prerequisite requires approval authority")
		}
		if err := validateAuthorities(p.AuthorityRefs); err != nil {
			return err
		}
		for _, a := range p.AuthorityRefs {
			if a.ArtifactDigest != p.Artifact.SHA256 {
				return fmt.Errorf("authority does not bind prerequisite artifact")
			}
		}
	}
	seen := map[string]bool{}
	for i, st := range s.Stages {
		if !validID(st.ID) || seen[st.ID] {
			return fmt.Errorf("invalid or duplicate stage ID")
		}
		seen[st.ID] = true
		if st.Role != roles[len(s.Prerequisites)+i] {
			return fmt.Errorf("stages must follow remaining full-v1 role order")
		}
		if strings.TrimSpace(st.Instructions) == "" || len(st.Instructions) > 512*1024 {
			return fmt.Errorf("stage instructions must be nonempty and at most 512 KiB")
		}
		if err := st.Limits.validate(1800); err != nil {
			return err
		}
		if st.Limits.TimeoutSeconds > s.Limits.TimeoutSeconds || st.Limits.MaxTokens > s.Limits.MaxTokens || st.Limits.MaxCostUSD > s.Limits.MaxCostUSD {
			return fmt.Errorf("stage limits exceed iteration limits")
		}
		if err := validatePaths(st.RequiredArtifacts, false, st.Role != "evaluator"); err != nil {
			return fmt.Errorf("required_artifacts: %w", err)
		}
		for _, p := range st.RequiredArtifacts {
			if !PathAllowed(p, s.AllowedPaths) {
				return fmt.Errorf("required artifact %q is outside allowed_paths", p)
			}
		}
		if err := validateAuthorities(st.AuthorityRefs); err != nil {
			return err
		}
	}
	if len(s.Verification) > 32 {
		return fmt.Errorf("at most 32 verification checks")
	}
	seen = map[string]bool{}
	for _, v := range s.Verification {
		if !validID(v.ID) || seen[v.ID] || len(v.Argv) == 0 || strings.TrimSpace(v.Argv[0]) == "" {
			return fmt.Errorf("verification requires unique ID and nonempty argv")
		}
		seen[v.ID] = true
		for _, arg := range v.Argv {
			if strings.ContainsRune(arg, 0) {
				return fmt.Errorf("verification argv contains NUL")
			}
		}
		if v.Cwd != "." && !validPath(v.Cwd, false) {
			return fmt.Errorf("verification cwd must be canonical repo-relative")
		}
		if v.TimeoutSeconds < 1 || v.TimeoutSeconds > 600 {
			return fmt.Errorf("verification timeout must be 1..600 seconds")
		}
	}
	if len(s.AcceptanceCriteria) == 0 {
		return fmt.Errorf("acceptance_criteria must be nonempty")
	}
	seen = map[string]bool{}
	for _, c := range s.AcceptanceCriteria {
		if !validID(c.ID) || seen[c.ID] || strings.TrimSpace(c.Text) == "" {
			return fmt.Errorf("criteria require unique IDs and nonempty text")
		}
		seen[c.ID] = true
	}
	return nil
}

// Digest binds all frozen fields. Invalid values have no digest.
func (s Spec) Digest() string {
	if s.Validate() != nil {
		return ""
	}
	return digest(s)
}
func digest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func (l Limits) validate(maxTimeout int) error {
	if l.TimeoutSeconds < 1 || l.TimeoutSeconds > maxTimeout || l.MaxTokens <= 0 || l.MaxCostUSD <= 0 || math.IsNaN(l.MaxCostUSD) || math.IsInf(l.MaxCostUSD, 0) {
		return fmt.Errorf("timeout must be 1..%d seconds; tokens and finite cost must be positive", maxTimeout)
	}
	return nil
}
func validID(s string) bool {
	if s == "" || len(s) > 128 || s == "." || s == ".." {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}
func validHex(s string, n int) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == n && strings.ToLower(s) == s
}
func validHash(s string) bool     { return validHex(s, 32) }
func validRevision(s string) bool { return validHex(s, 20) || validHex(s, 32) }
func validPath(p string, prefix bool) bool {
	if p == "" || strings.ContainsAny(p, "\\:*?[]\x00\r\n") || strings.HasPrefix(p, "/") {
		return false
	}
	if prefix {
		p = strings.TrimSuffix(p, "/")
	}
	if p == "." || p == ".." || strings.HasPrefix(p, "../") || path.Clean(p) != p {
		return false
	}
	return true
}
func validatePaths(paths []string, prefix, required bool) error {
	if len(paths) > 256 || (required && len(paths) == 0) {
		return fmt.Errorf("requires %s paths, at most 256", map[bool]string{true: "nonempty", false: "optional"}[required])
	}
	seen := map[string]bool{}
	for _, p := range paths {
		if !validPath(p, prefix) || p == ResultFile || seen[p] {
			return fmt.Errorf("invalid, reserved or duplicate path %q", p)
		}
		seen[p] = true
	}
	return nil
}

// PathAllowed uses exact file names or explicit directory prefixes, never globs.
func PathAllowed(p string, allowed []string) bool {
	if !validPath(p, false) || p == ResultFile {
		return false
	}
	for _, a := range allowed {
		if p == a || strings.HasSuffix(a, "/") && strings.HasPrefix(p, a) {
			return true
		}
	}
	return false
}
func validateAuthorities(refs []AuthorityRef) error {
	if len(refs) > 256 {
		return fmt.Errorf("at most 256 authority references")
	}
	for _, a := range refs {
		if !validRevision(a.Revision) || !validPath(a.Path, false) || strings.TrimSpace(a.Locator) == "" || !validHash(a.SHA256) || !validHash(a.ArtifactDigest) {
			return fmt.Errorf("invalid authority reference")
		}
	}
	return nil
}
