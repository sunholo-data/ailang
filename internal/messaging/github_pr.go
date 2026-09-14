package messaging

// Pull-request operations.
//
// The coordinator has created PRs since cloud execution existed and has never
// had any way to RESOLVE one. Issues it can open, comment on, label and close;
// pull requests it could only make. The consequence is mechanical: every cloud
// task leaves a branch and a PR behind, and nothing ever revisits them.
//
// Measured 2026-09-14: 24 open `coordinator/*` PRs, of which 22 belonged to
// tasks that were already rejected or failed, and one belonged to a task an
// operator had explicitly APPROVED twenty minutes earlier. Approval sets
// SkipMerge for a cloud task — correctly, there is no local worktree — and that
// was read as "do nothing", so approved and abandoned looked identical from
// GitHub.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// PullRequest is the subset of a PR the coordinator reasons about.
type PullRequest struct {
	Number      int      `json:"number"`
	State       string   `json:"state"`
	HeadRefName string   `json:"headRefName"`
	BaseRefName string   `json:"baseRefName"`
	Title       string   `json:"title"`
	Files       []string `json:"-"`
}

type ghPRResponse struct {
	Number      int    `json:"number"`
	State       string `json:"state"`
	HeadRefName string `json:"headRefName"`
	BaseRefName string `json:"baseRefName"`
	Title       string `json:"title"`
	Files       []struct {
		Path string `json:"path"`
	} `json:"files"`
}

// ListOpenPRsWithPrefix returns every OPEN pull request whose head branch starts
// with prefix, in ONE API call.
//
// One call, not one per task. The obvious shape — ask GitHub about each task's
// branch — is O(tasks) round trips and took minutes against 500 tasks before
// printing anything. Open PRs are the small set; tasks are the large one, so
// the query belongs on the small side.
func (c *GitHubClient) ListOpenPRsWithPrefix(repo, prefix string) ([]PullRequest, error) {
	if err := c.PreFlightChecks(); err != nil {
		return nil, err
	}
	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return nil, fmt.Errorf("no repository specified")
	}
	out, err := c.execCommandCtx("gh", "pr", "list",
		"--repo", repo, "--state", "open", "--limit", "200",
		"--json", "number,state,headRefName,baseRefName,title,files")
	if err != nil {
		return nil, fmt.Errorf("listing open PRs in %s: %w\nOutput: %s", repo, err, string(out))
	}
	var raw []ghPRResponse
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("decoding PR list for %s: %w", repo, err)
	}
	var prs []PullRequest
	for _, p := range raw {
		if prefix != "" && !strings.HasPrefix(p.HeadRefName, prefix) {
			continue
		}
		pr := PullRequest{
			Number: p.Number, State: p.State, HeadRefName: p.HeadRefName,
			BaseRefName: p.BaseRefName, Title: p.Title,
		}
		for _, f := range p.Files {
			pr.Files = append(pr.Files, f.Path)
		}
		prs = append(prs, pr)
	}
	return prs, nil
}

// FindPRForBranch returns the OPEN pull request whose head is branch, or nil.
//
// Nil-with-no-error is the ordinary case, not a fault: a task that changed
// nothing never opened one, and a direct-push agent never opens one at all.
func (c *GitHubClient) FindPRForBranch(repo, branch string) (*PullRequest, error) {
	if err := c.PreFlightChecks(); err != nil {
		return nil, err
	}
	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return nil, fmt.Errorf("no repository specified")
	}
	out, err := c.execCommandCtx("gh", "pr", "list",
		"--repo", repo, "--head", branch, "--state", "open",
		"--json", "number,state,headRefName,baseRefName,title,files")
	if err != nil {
		return nil, fmt.Errorf("listing PRs for %s: %w\nOutput: %s", branch, err, string(out))
	}
	var prs []ghPRResponse
	if err := json.Unmarshal(out, &prs); err != nil {
		return nil, fmt.Errorf("decoding PR list for %s: %w", branch, err)
	}
	if len(prs) == 0 {
		return nil, nil
	}
	p := prs[0]
	pr := &PullRequest{
		Number: p.Number, State: p.State, HeadRefName: p.HeadRefName,
		BaseRefName: p.BaseRefName, Title: p.Title,
	}
	for _, f := range p.Files {
		pr.Files = append(pr.Files, f.Path)
	}
	return pr, nil
}

// MergePR squash-merges and deletes the head branch, waiting for required
// checks if the base branch demands them.
//
// Squash deliberately: an agent's branch is a working record — clone, retry,
// fixup — and the unit that belongs on the base branch is the change, not the
// process that produced it.
//
// `--auto` is what makes this work on a PROTECTED branch. Without it an
// immediate merge is refused outright:
//
//	X Pull request sunholo-data/ailang#1196 is not mergeable: the base branch
//	  policy prohibits the merge.
//
// measured 2026-09-14 on the first approval that tried. The approval is a
// decision about the WORK, and it is made minutes before CI can possibly have
// run — so "merge when the requirements are met" is the honest reading of it,
// where "refuse because they are not met yet" is a race the operator would have
// to lose on purpose.
//
// --auto degrades correctly on an UNPROTECTED branch: GitHub merges immediately
// when there is nothing to wait for. And it never bypasses a check — a PR whose
// CI fails simply stays open, which is the outcome anyone would want.
// --admin is deliberately NOT used: an approval is permission to land the work,
// not permission to skip the repository's own gates.
func (c *GitHubClient) MergePR(repo string, number int, subject, body string) error {
	if err := c.PreFlightChecks(); err != nil {
		return err
	}
	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return fmt.Errorf("no repository specified")
	}
	args := []string{"pr", "merge", "--repo", repo, strconv.Itoa(number),
		"--squash", "--delete-branch", "--auto"}
	if subject != "" {
		args = append(args, "--subject", subject)
	}
	if body != "" {
		args = append(args, "--body", body)
	}
	if out, err := c.execCommandCtx("gh", args...); err != nil {
		return fmt.Errorf("merging PR #%d: %w\nOutput: %s", number, err, string(out))
	}
	return nil
}

// ClosePR closes without merging and deletes the head branch, recording why.
//
// The comment is required in practice rather than by signature: a closed PR
// with no reason is indistinguishable from an abandoned one, which is the state
// this whole file exists to end.
func (c *GitHubClient) ClosePR(repo string, number int, comment string) error {
	if err := c.PreFlightChecks(); err != nil {
		return err
	}
	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return fmt.Errorf("no repository specified")
	}
	args := []string{"pr", "close", "--repo", repo, strconv.Itoa(number), "--delete-branch"}
	if comment != "" {
		args = append(args, "--comment", comment)
	}
	if out, err := c.execCommandCtx("gh", args...); err != nil {
		return fmt.Errorf("closing PR #%d: %w\nOutput: %s", number, err, string(out))
	}
	return nil
}
