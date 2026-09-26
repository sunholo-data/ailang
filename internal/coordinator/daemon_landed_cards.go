package coordinator

// Resolve a pending approval card when its coordinator PR merges — and fire
// the handoff the approval would have fired.
//
// The PR -> card direction, run by the daemon. `ailang coordinator prs --landed`
// is the manual form and exists for the backlog; this is the standing one.
// Operator ruling 2026-09-23: a merge IS the approval, so the chain continues
// (design -> sprint-planner, sprint -> sprint-executor) without a second click.
//
// Three things shape it, all measured rather than assumed:
//
//   - REST, not GitHubClient. The coordinator image is a plain Go buildpack
//     with no `gh` binary, and GitHubClient shells out to `gh` — it would fail
//     pre-flight on every call, i.e. do nothing, in the one place this runs.
//   - Polling, not webhooks. The only repo webhook (sunholo-data/ailang) points
//     at the DEV coordinator and sends issue events only; daneel, parse, email
//     and decisions send nothing.
//   - Throttled, and bursty. Prod scales to zero with cpu_idle, so the tick
//     loop runs only while something keeps the instance up — measured
//     2026-09-23 as bursts with gaps up to ~90 min. Worst-case latency from
//     merge to handoff is therefore about an hour, not ten minutes.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// landedCardSweepInterval bounds GitHub calls: one per pending card per run.
const landedCardSweepInterval = 10 * time.Minute

// landedCardSweepSince excludes the backlog. Every card older than this was
// already reviewed by hand on 2026-09-23 and found superseded — docs landed and
// were later reverted, or already planned — so firing their handoffs would
// start sprints nobody wants. Those are resolved WITHOUT handoffs by
// `ailang coordinator prs --landed --apply`.
var landedCardSweepSince = time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)

// mergedPRLookup finds the merged PR for a branch; nil, nil when there is none.
// A variable so tests do not reach GitHub.
var mergedPRLookup = lookupMergedPRREST

// sweepLandedCards is called from the poll loop. Cloud only: that is where
// cloud tasks' cards live, and a local task merges through its worktree.
func (d *Daemon) sweepLandedCards() {
	if !IsCloudMode() || d.taskStore == nil || d.agentRegistry == nil {
		return
	}
	if time.Since(d.lastLandedCardSweep) < landedCardSweepInterval {
		return
	}
	d.lastLandedCardSweep = time.Now()

	token := config.GitHubToken()
	if token == "" {
		// LOUD, every run: a sweep that cannot look is not one that found nothing.
		d.logger.Printf("Warning: landed-card sweep cannot run — GITHUB_TOKEN is unset")
		return
	}
	d.runLandedCardSweep(d.ctx, token)
}

func (d *Daemon) runLandedCardSweep(ctx context.Context, token string) {
	tasks, err := d.taskStore.ListTasks(ctx, &TaskFilter{Status: []TaskStatus{TaskStatusPendingApproval}})
	if err != nil {
		d.logger.Printf("Warning: landed-card sweep could not list tasks: %v", err)
		return
	}
	for _, task := range tasks {
		if task.CreatedAt.Before(landedCardSweepSince) {
			continue
		}
		agent := d.agentRegistry.GetAgentByID(task.AgentID)
		if agent == nil {
			continue // another plane's task: not ours to decide
		}
		repo := task.GithubRepo
		if repo == "" {
			repo = agent.ResolveRepo()
		}
		if strings.Count(repo, "/") != 1 {
			continue
		}
		pr, err := mergedPRLookup(ctx, token, repo, BranchForTask(task.ID))
		if err != nil {
			d.logger.Printf("Warning: landed-card sweep cannot read PRs for %s in %s: %v", task.ID, repo, err)
			continue
		}
		if pr == nil {
			continue
		}
		approval, err := d.taskStore.GetApprovalRequestByTaskAnyStatus(ctx, task.ID)
		if err != nil {
			d.logger.Printf("Warning: landed-card sweep cannot read approval for %s: %v", task.ID, err)
			continue
		}
		ok, reason := DecideLandedCard(task, approval, pr)
		if !ok {
			continue
		}
		result, err := ProcessApprovalRequest(ctx, &ApprovalParams{
			TaskID:        task.ID,
			Action:        "approve",
			ApprovedBy:    LandedApprover(pr),
			Channel:       "pr-merged",
			Store:         d.taskStore,
			MsgStore:      d.msgStore,
			AgentRegistry: d.agentRegistry,
			ObsBackend:    d.obsBackend,
			SkipMerge:     true, // merged on GitHub already
		})
		if err != nil {
			d.logger.Printf("ERROR: landed card %s (%s) could not be resolved: %v", task.ID, reason, err)
			continue
		}
		d.logger.Printf("Landed card resolved: %s — %s — %s", task.ID, reason, result.Message)
	}
}

// lookupMergedPRREST asks GitHub's REST API for a merged PR from branch.
func lookupMergedPRREST(ctx context.Context, token, repo, branch string) (*LandedPR, error) {
	owner := repo[:strings.IndexByte(repo, '/')]
	q := url.Values{"state": {"closed"}, "head": {owner + ":" + branch}}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/pulls?%s", repo, q.Encode())
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", apiURL, resp.Status)
	}
	var prs []restPR
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, fmt.Errorf("decoding PR list: %w", err)
	}
	return mergedFromREST(prs, branch), nil
}

type restPR struct {
	Number   int     `json:"number"`
	MergedAt *string `json:"merged_at"`
	Head     struct {
		Ref string `json:"ref"`
	} `json:"head"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// mergedFromREST picks the merged PR for branch out of a REST listing. A
// closed-unmerged PR has merged_at null and is not a landing.
func mergedFromREST(prs []restPR, branch string) *LandedPR {
	for _, p := range prs {
		if p.MergedAt == nil || p.Head.Ref != branch {
			continue
		}
		// The list endpoint does not carry merged_by; say so rather than
		// attribute the merge to the PR's author.
		return &LandedPR{Number: p.Number, State: "MERGED", HeadRefName: p.Head.Ref, MergedAt: *p.MergedAt}
	}
	return nil
}
