package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/gitexec"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

type AutoMergeMode int

const (
	MergeModeDocs AutoMergeMode = iota
	MergeModeCode
)

type codeMergeSettings struct {
	autoMerge        bool
	checks, patterns []string
	secret, identity string
	// localHead is the commit the wrapper classified and pushed. When set,
	// the PR head must match it before any review is posted.
	localHead string
}

func codeMergeSettingsFromEnv() codeMergeSettings {
	return codeMergeSettings{config.AutoMerge(), config.AutoMergeRequiredChecks(), artifactPatternsFromEnv(), config.ApproverSecret(), config.ApproverIdentity(), ""}
}
func (s codeMergeSettings) validate() error {
	return (&coordinator.AgentConfig{AutoMerge: s.autoMerge, AutoMergeCode: true, AutoMergeRequiredChecks: s.checks, ArtifactPatterns: s.patterns, AutoMergeApproverSecret: s.secret, AutoMergeApproverIdentity: s.identity}).ValidateAutoMergeCode()
}

// codeMergeREST never includes tokens or server response bodies in errors.
func codeMergeREST(ctx context.Context, token, method, path string, payload interface{}, out interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, method, "https://api.github.com"+path, body)
	if err != nil {
		return fmt.Errorf("GitHub request construction failed")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub %s request failed", method)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("GitHub %s returned HTTP %d", method, resp.StatusCode)
	}
	if out != nil {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out); err != nil {
			return fmt.Errorf("malformed GitHub %s response", method)
		}
	}
	return nil
}
func codeRepoPath(owner, repo string) string {
	return "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repo)
}

// requiredChecksPresentOnBase verifies existence only. The deployment ruleset
// must require these names on PRs; native auto-merge delegates waiting to GitHub.
func requiredChecksPresentOnBase(ctx context.Context, token, owner, repo, base string, checks []string) ([]string, error) {
	if len(checks) == 0 {
		return nil, fmt.Errorf("code auto-merge requires non-empty required checks")
	}
	for _, name := range checks {
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("empty required check name")
		}
	}
	root := codeRepoPath(owner, repo)
	var commit struct {
		SHA string `json:"sha"`
	}
	if err := codeMergeREST(ctx, token, "GET", root+"/commits/"+url.PathEscape(base), nil, &commit); err != nil {
		return nil, err
	}
	if commit.SHA == "" {
		return nil, fmt.Errorf("base HEAD response has no SHA")
	}
	names := map[string]bool{}
	complete := false
	for page := 1; page <= 1000; page++ {
		var result struct {
			Total *int `json:"total_count"`
			Runs  *[]struct {
				Name string `json:"name"`
			} `json:"check_runs"`
		}
		path := fmt.Sprintf("%s/commits/%s/check-runs?filter=all&per_page=100&page=%d", root, url.PathEscape(commit.SHA), page)
		if err := codeMergeREST(ctx, token, "GET", path, nil, &result); err != nil {
			return nil, err
		}
		if result.Total == nil || result.Runs == nil || *result.Total < 0 {
			return nil, fmt.Errorf("malformed base check-runs response")
		}
		for _, run := range *result.Runs {
			if run.Name == "" {
				return nil, fmt.Errorf("base check-run has no name")
			}
			names[run.Name] = true
		}
		seen := (page-1)*100 + len(*result.Runs)
		if seen >= *result.Total {
			complete = true
			break
		}
		if len(*result.Runs) != 100 {
			return nil, fmt.Errorf("incomplete base check-runs page %d", page)
		}
	}
	if !complete {
		return nil, fmt.Errorf("base check-runs pagination exceeded limit")
	}
	var missing []string
	for _, name := range checks {
		if !names[name] {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

// preflightCodeApprover verifies both identities before native enable.
// The review is pinned to the final pushed commit, so a concurrent push does
// not receive an approval for code the wrapper did not inspect.
func preflightCodeApprover(ctx context.Context, approverToken, authorToken, owner, repo string, pr int, expected string) (string, error) {
	var user struct {
		Login string `json:"login"`
	}
	if err := codeMergeREST(ctx, approverToken, "GET", "/user", nil, &user); err != nil {
		return "", fmt.Errorf("approver identity: %w", err)
	}
	if user.Login == "" || !strings.EqualFold(user.Login, expected) {
		return "", fmt.Errorf("approver login does not match configured identity %q", expected)
	}
	var pull struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	path := fmt.Sprintf("%s/pulls/%d", codeRepoPath(owner, repo), pr)
	if err := codeMergeREST(ctx, authorToken, "GET", path, nil, &pull); err != nil {
		return "", fmt.Errorf("PR author: %w", err)
	}
	if pull.User.Login == "" || pull.Head.SHA == "" {
		return "", fmt.Errorf("PR response missing author or head SHA")
	}
	if strings.EqualFold(user.Login, pull.User.Login) {
		return "", fmt.Errorf("the approver must not be the PR author")
	}
	return pull.Head.SHA, nil
}
func approvePRAsNonAuthor(ctx context.Context, token, owner, repo string, pr int, head, note string) error {
	var review struct {
		ID    int64  `json:"id"`
		State string `json:"state"`
	}
	payload := map[string]string{"event": "APPROVE", "body": note, "commit_id": head}
	if err := codeMergeREST(ctx, token, "POST", fmt.Sprintf("%s/pulls/%d/reviews", codeRepoPath(owner, repo), pr), payload, &review); err != nil {
		return err
	}
	if review.ID == 0 || review.State != "APPROVED" {
		return fmt.Errorf("GitHub did not confirm the approving review")
	}
	return nil
}

const codeMergeAuditStart = "<!-- ailang-auto-merge-code -->"
const codeMergeAuditEnd = "<!-- /ailang-auto-merge-code -->"

func (s codeMergeSettings) audit(state string) string {
	return fmt.Sprintf("%s\n**Auto-merge (code)**: %s; scope: %s; required checks: %s; approver: %s (non-author). Refs #1599.\n%s\n", codeMergeAuditStart, state, strings.Join(s.patterns, ", "), strings.Join(s.checks, ", "), s.identity, codeMergeAuditEnd)
}

// updateCodeMergeAudit preserves the existing PR's request and retry history.
func updateCodeMergeAudit(ctx context.Context, token, owner, repo string, pr int, note string) error {
	path := fmt.Sprintf("%s/pulls/%d", codeRepoPath(owner, repo), pr)
	var pull struct {
		Body *string `json:"body"`
	}
	if err := codeMergeREST(ctx, token, "GET", path, nil, &pull); err != nil {
		return err
	}
	body := ""
	if pull.Body != nil {
		body = *pull.Body
	}
	if start := strings.Index(body, codeMergeAuditStart); start >= 0 {
		end := strings.Index(body[start:], codeMergeAuditEnd)
		if end < 0 {
			return fmt.Errorf("existing code auto-merge audit has no closing marker")
		}
		body = body[:start] + body[start+end+len(codeMergeAuditEnd):]
	}
	return codeMergeREST(ctx, token, "PATCH", path, map[string]string{"body": strings.TrimSpace(body) + "\n\n" + note}, nil)
}
func disableGitHubAutoMerge(ctx context.Context, token, owner, repo string, pr int) error {
	id, err := pullRequestNodeID(ctx, token, owner, repo, pr)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]interface{}{"query": `mutation($id:ID!){disablePullRequestAutoMerge(input:{pullRequestId:$id}){clientMutationId}}`, "variables": map[string]string{"id": id}})
	body, err := githubGraphQL(ctx, token, payload)
	if err != nil {
		return err
	}
	var response struct {
		Data   map[string]json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("malformed auto-merge disable response")
	}
	if len(response.Errors) > 0 || len(response.Data["disablePullRequestAutoMerge"]) == 0 || string(response.Data["disablePullRequestAutoMerge"]) == "null" {
		return fmt.Errorf("GitHub did not confirm auto-merge disable")
	}
	return nil
}

func fetchCodeApproverSecret(ctx context.Context, name string) (string, error) {
	project, err := config.CloudProject(ctx)
	if err != nil {
		return "", err
	}
	return fetchSecret(ctx, project, name)
}
func executeCodeAutoMerge(ctx context.Context, token, owner, repo string, pr int, base string, s codeMergeSettings, fetch func(context.Context, string) (string, error)) error {
	if err := s.validate(); err != nil {
		return err
	}
	missing, err := requiredChecksPresentOnBase(ctx, token, owner, repo, base, s.checks)
	if err != nil {
		return fmt.Errorf("required checks on %s: %w", base, err)
	}
	if len(missing) > 0 {
		return fmt.Errorf("required check(s) missing on %s: %s", base, strings.Join(missing, ", "))
	}
	approverToken, err := fetch(ctx, s.secret)
	// Secret-provider errors may include sensitive details: do not echo them.
	if err != nil {
		return fmt.Errorf("cannot fetch approver secret %q; check job service-account access", s.secret)
	}
	approverToken = strings.TrimSpace(approverToken)
	if approverToken == "" {
		return fmt.Errorf("approver secret %q is empty", s.secret)
	}
	head, err := preflightCodeApprover(ctx, approverToken, token, owner, repo, pr, s.identity)
	if err != nil {
		return err
	}
	if s.localHead != "" && !strings.EqualFold(head, s.localHead) {
		return fmt.Errorf("PR head %s does not match the local HEAD %s; refusing to approve code the wrapper did not inspect", head, s.localHead)
	}
	intent := s.audit("configured/requested; enable and approval pending")
	if err := updateCodeMergeAudit(ctx, token, owner, repo, pr, intent); err != nil {
		return fmt.Errorf("record code auto-merge intent: %w", err)
	}
	if err := addGitHubLabels(ctx, token, owner, repo, pr, []string{"auto-merge-code"}); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: auto-merge-code label failed on #%d; body audit retained\n", pr)
	}
	fmt.Printf("execute-job: code auto-merge requested on #%d (approver=%s; checks=%s; scope=%s)\n", pr, s.identity, strings.Join(s.checks, ", "), strings.Join(s.patterns, ", "))
	if err := enableGitHubAutoMerge(ctx, token, owner, repo, pr); err != nil {
		return fmt.Errorf("native auto-merge enable: %w", err)
	}
	if err := approvePRAsNonAuthor(ctx, approverToken, owner, repo, pr, head, intent); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		cleanupErr := disableGitHubAutoMerge(cleanupCtx, token, owner, repo, pr)
		state := "approval failed; native auto-merge disabled"
		if cleanupErr != nil {
			state = "approval failed; disable cleanup failed; native auto-merge may remain enabled"
		}
		if auditErr := updateCodeMergeAudit(cleanupCtx, token, owner, repo, pr, s.audit(state)); auditErr != nil {
			fmt.Fprintf(os.Stderr, "execute-job: failure audit update failed on #%d: %v\n", pr, auditErr)
		}
		if cleanupErr != nil {
			return fmt.Errorf("approval failed: %w; disable cleanup failed: %v", err, cleanupErr)
		}
		return fmt.Errorf("approval failed: %w; native auto-merge disabled", err)
	}
	if err := updateCodeMergeAudit(ctx, token, owner, repo, pr, s.audit("enabled and approved")); err != nil {
		return fmt.Errorf("native auto-merge enabled and review posted, but success audit failed: %w", err)
	}
	fmt.Printf("execute-job: auto-merge enabled and approved on #%d (code; approver=%s; required checks=%s; scope=%s) — GitHub will merge when required checks pass\n", pr, s.identity, strings.Join(s.checks, ", "), strings.Join(s.patterns, ", "))
	return nil
}
func maybeEnableCodeAutoMerge(ctx context.Context, token, owner, repo string, pr int, workDir, base string) {
	s := codeMergeSettingsFromEnv()
	if err := s.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: auto-merge NOT enabled: %v\n", err)
		return
	}
	ok, reason, err := branchIsAutoMergeable(ctx, workDir, base, s.patterns, MergeModeCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: auto-merge NOT enabled: cannot classify the diff: %v\n", err)
		return
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "execute-job: auto-merge NOT enabled: %s\n", reason)
		return
	}
	localHead, err := gitexec.CommandContext(ctx, "-C", workDir, "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(localHead)) == "" {
		fmt.Fprintf(os.Stderr, "execute-job: auto-merge NOT enabled: cannot resolve local HEAD\n")
		return
	}
	s.localHead = strings.TrimSpace(string(localHead))
	if err := executeCodeAutoMerge(ctx, token, owner, repo, pr, base, s, fetchCodeApproverSecret); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: code auto-merge incomplete on #%d: %v\n", pr, err)
	}
}
