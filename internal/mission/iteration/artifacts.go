package iteration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/executor/proctree"
	"github.com/sunholo-data/ailang/internal/gitutil"
	"github.com/sunholo-data/ailang/internal/mission/dispatch"
)

type ArtifactEvidence struct {
	Path   string `json:"path"`
	Blob   string `json:"blob"`
	SHA256 string `json:"sha256"`
	Mode   string `json:"mode"`
}
type Evidence struct {
	Result         StageResult        `json:"result"`
	ResultBytes    []byte             `json:"result_bytes"`
	ResultSHA256   string             `json:"result_sha256"`
	OutputRevision string             `json:"output_revision"`
	OutputTree     string             `json:"output_tree"`
	Artifacts      []ArtifactEvidence `json:"artifacts"`
	Checks         []CheckReceipt     `json:"checks"`
	AuthorModels   []string           `json:"author_models"`
	AuthorRoute    dispatch.Candidate `json:"author_route"`
}

func (e Evidence) Digest() string { return digest(e) }

// Git runs bounded local Git inspection. It disables replace objects and hooks;
// callers supply only frozen, validated revisions and literal pathspecs.
func Git(ctx context.Context, repo string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false", "--literal-pathspecs", "-C", repo}, args...)...)
	proctree.Configure(cmd)
	cmd.Env = append(os.Environ(), "GIT_NO_REPLACE_OBJECTS=1", "GIT_TERMINAL_PROMPT=0")
	out := &boundedOutput{limit: 16 * 1024 * 1024}
	stderr := &boundedOutput{limit: 4096}
	cmd.Stdout = out
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	if out.truncated {
		return "", fmt.Errorf("git output exceeds 16 MiB")
	}
	return out.String(), nil
}

type boundedOutput struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	remaining := b.limit - b.buf.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.buf.Write(p)
	return n, nil
}
func (b *boundedOutput) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.buf.String() }
func shaBytes(b []byte) string          { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

// ValidateRepository compares normalized origin identity and exact local base.
// It never creates state or contacts a remote.
func ValidateRepository(ctx context.Context, repo string, spec Spec) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := spec.Validate(); err != nil {
		return err
	}
	owner, name, err := gitutil.GitHubOwnerRepo(ctx, repo)
	if err != nil {
		return err
	}
	var origin string
	if owner != "" {
		origin = "github.com/" + owner + "/" + name
	} else {
		raw, err := Git(ctx, repo, "remote", "get-url", "origin")
		if err != nil {
			return err
		}
		origin = normalizeOrigin(strings.TrimSpace(raw))
	}
	if origin != normalizeOrigin(spec.Repository) {
		return fmt.Errorf("repository origin mismatch: got %q, expected %q", origin, spec.Repository)
	}
	actual, err := Git(ctx, repo, "rev-parse", "--verify", spec.BaseRevision+"^{commit}")
	if err != nil {
		return err
	}
	if strings.TrimSpace(actual) != spec.BaseRevision {
		return fmt.Errorf("base_revision does not identify exact commit")
	}
	return nil
}
func normalizeOrigin(raw string) string {
	raw = strings.TrimSuffix(strings.TrimSuffix(raw, "/"), ".git")
	if strings.HasPrefix(raw, "git@") {
		if i := strings.Index(raw, ":"); i >= 0 {
			return strings.TrimPrefix(raw[:i], "git@") + "/" + raw[i+1:]
		}
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return strings.ToLower(u.Host) + strings.TrimSuffix(u.Path, "/")
	}
	return raw
}

// ErrNeedsDecision marks a bound, preserved protocol result awaiting attended authority.
var ErrNeedsDecision = errors.New("stage requires an operator decision")

// ValidateArtifacts inspects immutable Git objects and runs only frozen checks.
// Failed checks return partial evidence so callers can preserve their receipts.
func ValidateArtifacts(ctx context.Context, spec Spec, stage Stage, request dispatch.Request, report *dispatch.Report, workspace, verificationRoot string) (*Evidence, error) {
	if err := ValidateRepository(ctx, workspace, spec); err != nil {
		return nil, err
	}
	frozenStage := false
	for _, frozen := range spec.Stages {
		if frozen.ID == stage.ID && digest(frozen) == digest(stage) {
			frozenStage = true
		}
	}
	if !frozenStage {
		return nil, fmt.Errorf("stage policy differs from frozen work item")
	}
	if request.MissionID != spec.MissionID || request.WorkItemID != spec.WorkItemID || request.StageID != stage.ID || request.Role != stage.Role || request.Workspace != workspace {
		return nil, fmt.Errorf("request does not bind this stage workspace")
	}
	if !validRevision(request.InputRevision) {
		return nil, fmt.Errorf("request requires full input revision")
	}
	if report == nil || report.Version != 1 || report.Status != "execution_completed" || report.RequestDigest != request.Digest() || report.AttemptID != request.AttemptID {
		return nil, fmt.Errorf("successful matching dispatch report required")
	}
	route, err := completedRoute(report, request.Models)
	if err != nil {
		return nil, err
	}
	if err := checkAuthorTree(ctx, workspace); err != nil {
		return nil, err
	}
	resultPath := filepath.Join(workspace, ResultFile)
	info, err := os.Lstat(resultPath)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("stage-result.json must be a regular untracked file")
	}
	file, err := os.Open(resultPath)
	if err != nil {
		return nil, err
	}
	body, readErr := io.ReadAll(io.LimitReader(file, MaxResultBytes+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	result, err := DecodeStageResult(bytes.NewReader(body), stage.Role, spec.AcceptanceCriteria)
	if err != nil {
		return nil, err
	}
	if result.RequestDigest != request.Digest() || result.InputRevision != request.InputRevision {
		return nil, fmt.Errorf("result does not bind dispatched request and input revision")
	}
	head, err := Git(ctx, workspace, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(head) != result.OutputRevision {
		return nil, fmt.Errorf("workspace HEAD differs from result output revision")
	}
	if _, err := Git(ctx, workspace, "merge-base", "--is-ancestor", spec.BaseRevision, request.InputRevision); err != nil {
		return nil, fmt.Errorf("stage input is not descended from frozen base: %w", err)
	}
	if _, err := Git(ctx, workspace, "merge-base", "--is-ancestor", request.InputRevision, result.OutputRevision); err != nil {
		return nil, fmt.Errorf("output is not descended from stage input: %w", err)
	}
	if _, err := Git(ctx, workspace, "cat-file", "-e", result.OutputRevision+":"+ResultFile); err == nil {
		return nil, fmt.Errorf("protocol file must not be committed")
	}
	tree, err := Git(ctx, workspace, "rev-parse", result.OutputRevision+"^{tree}")
	if err != nil {
		return nil, err
	}
	e := &Evidence{Result: result, ResultBytes: body, ResultSHA256: shaBytes(body), OutputRevision: result.OutputRevision, OutputTree: strings.TrimSpace(tree), AuthorModels: []string{route.Model}, AuthorRoute: route}
	if result.Outcome == "needs_decision" {
		return e, ErrNeedsDecision
	}
	if result.Outcome != "produced" && result.Outcome != "pass" {
		return e, fmt.Errorf("stage outcome %q cannot be accepted", result.Outcome)
	}
	changes, err := Git(ctx, workspace, "diff", "--no-ext-diff", "--no-renames", "--name-only", "-z", request.InputRevision, result.OutputRevision, "--")
	if err != nil {
		return e, err
	}
	if changes == "" && stage.Role != "evaluator" {
		return e, fmt.Errorf("author output contains no product changes")
	}
	for _, p := range strings.Split(changes, "\x00") {
		if p == "" {
			continue
		}
		if !PathAllowed(p, spec.AllowedPaths) {
			return e, fmt.Errorf("changed path %q is outside allowed scope", p)
		}
		if err := checkObjectPath(ctx, workspace, result.OutputRevision, p, true); err != nil {
			return e, err
		}
	}
	artifacts := map[string]bool{}
	for _, p := range result.ArtifactPaths {
		if !PathAllowed(p, spec.AllowedPaths) {
			return e, fmt.Errorf("artifact path outside scope: %s", p)
		}
		a, err := inspectArtifact(ctx, workspace, result.OutputRevision, p)
		if err != nil {
			return e, err
		}
		e.Artifacts = append(e.Artifacts, a)
		artifacts[p] = true
	}
	for _, p := range stage.RequiredArtifacts {
		if !artifacts[p] {
			return e, fmt.Errorf("required artifact %q not reported", p)
		}
	}
	if err := verifyAuthorities(ctx, workspace, spec.BaseRevision, stage.AuthorityRefs); err != nil {
		return e, err
	}
	refs := append([]AuthorityRef(nil), stage.AuthorityRefs...)
	for _, prerequisite := range spec.Prerequisites {
		refs = append(refs, prerequisite.AuthorityRefs...)
	}
	for _, ref := range refs {
		artifact, err := inspectArtifact(ctx, workspace, result.OutputRevision, ref.Path)
		if err != nil || artifact.SHA256 != ref.SHA256 {
			return e, fmt.Errorf("candidate changed frozen authority %s", ref.Path)
		}
	}
	if err := verifyChecks(ctx, workspace, verificationRoot, spec.Verification, e); err != nil {
		return e, err
	}
	// Reinspect author HEAD, status and protocol bytes immediately before acceptance.
	if err := checkAuthorTree(ctx, workspace); err != nil {
		return e, err
	}
	head, err = Git(ctx, workspace, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != e.OutputRevision {
		return e, fmt.Errorf("candidate changed during verification")
	}
	info, err = os.Lstat(resultPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxResultBytes {
		return e, fmt.Errorf("stage result changed during verification")
	}
	latestFile, err := os.Open(resultPath)
	if err != nil {
		return e, err
	}
	latest, err := io.ReadAll(io.LimitReader(latestFile, MaxResultBytes+1))
	_ = latestFile.Close()
	if err != nil || !bytes.Equal(latest, body) {
		return e, fmt.Errorf("stage result changed during verification")
	}
	return e, nil
}
func completedRoute(report *dispatch.Report, models []string) (dispatch.Candidate, error) {
	var route dispatch.Candidate
	count := 0
	for _, a := range report.Attempts {
		if a.Status == "completed" {
			route = a.Candidate
			count++
		}
	}
	if count != 1 || route.Model == "" || route.Vendor == "" || route.Executor == "" {
		return route, fmt.Errorf("one completed route with actual model/vendor/executor provenance required")
	}
	for _, m := range models {
		if m == route.Model {
			return route, nil
		}
	}
	return route, fmt.Errorf("completed route was not a requested model")
}
func checkAuthorTree(ctx context.Context, workspace string) error {
	status, err := Git(ctx, workspace, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignored")
	if err != nil {
		return err
	}
	for _, entry := range strings.Split(status, "\x00") {
		if entry == "" {
			continue
		}
		if entry != "?? "+ResultFile {
			return fmt.Errorf("unexpected dirty/ignored author path: %q", entry)
		}
	}
	return nil
}
func inspectArtifact(ctx context.Context, repo, revision, p string) (ArtifactEvidence, error) {
	a := ArtifactEvidence{Path: p}
	if err := checkObjectPath(ctx, repo, revision, p, false); err != nil {
		return a, err
	}
	entry, err := Git(ctx, repo, "ls-tree", "-z", revision, "--", p)
	if err != nil {
		return a, err
	}
	parts := strings.SplitN(strings.TrimSuffix(entry, "\x00"), "\t", 2)
	if len(parts) != 2 {
		return a, fmt.Errorf("missing artifact %q", p)
	}
	fields := strings.Fields(parts[0])
	if len(fields) != 3 || fields[1] != "blob" {
		return a, fmt.Errorf("artifact must be a regular blob: %q", p)
	}
	a.Mode, a.Blob = fields[0], fields[2]
	body, err := Git(ctx, repo, "cat-file", "blob", a.Blob)
	if err != nil {
		return a, err
	}
	a.SHA256 = shaBytes([]byte(body))
	return a, nil
}
func checkObjectPath(ctx context.Context, repo, revision, p string, allowDeleted bool) error {
	parts := strings.Split(p, "/")
	for i := range parts {
		partial := strings.Join(parts[:i+1], "/")
		entry, err := Git(ctx, repo, "ls-tree", "-z", revision, "--", partial)
		if err != nil {
			return err
		}
		if entry == "" {
			if allowDeleted {
				return nil
			}
			return fmt.Errorf("missing Git path %q", partial)
		}
		fields := strings.Fields(strings.SplitN(entry, "\t", 2)[0])
		if len(fields) != 3 {
			return fmt.Errorf("invalid Git tree entry")
		}
		if fields[0] == "120000" || fields[0] == "160000" {
			return fmt.Errorf("symlink/submodule path is not an accepted artifact: %s", partial)
		}
	}
	return nil
}
