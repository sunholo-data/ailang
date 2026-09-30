package coordinator

// Resolve a pending approval card when its coordinator PR merges — and fire
// the handoff the approval would have fired. A PR closed without merging
// rejects the card, with no re-attempt and no handoff.
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

// settledPRLookup finds how a branch's PR ended: State "MERGED", or "CLOSED"
// when it was closed unmerged and nothing from the branch is still open; nil,
// nil while one is open or none exists. A variable so tests do not reach GitHub.
var settledPRLookup = lookupSettledPRREST

// githubAPIBase is GitHub's REST root; a variable so a test can serve it.
var githubAPIBase = "https://api.github.com"

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
	for i, task := range tasks {
		// Cloud Run sends SIGTERM when it scales an idle instance to zero,
		// often mid-sweep. Every remaining lookup then failed with "context
		// canceled", one warning per card — 234 in a week, which read as a
		// sweep that never worked while it was resolving cards on schedule.
		if ctx.Err() != nil {
			d.logger.Printf("landed-card sweep interrupted (%v) after %d of %d cards; the next run resumes", ctx.Err(), i, len(tasks))
			return
		}
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
		pr, err := settledPRLookup(ctx, token, repo, BranchForTask(task.ID))
		if err != nil {
			if ctx.Err() != nil {
				continue // interrupted: reported once, at the top of the loop
			}
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
		if pr.State == "CLOSED" {
			d.rejectClosedCard(ctx, task, approval, pr)
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

// rejectClosedCard rejects a card whose PR was closed unmerged. Closing is the
// decision, as merging is the approval: the operator already said no on
// GitHub. RetriggerOnReject stays false — a re-attempt would reopen the very
// work that was just turned down.
func (d *Daemon) rejectClosedCard(ctx context.Context, task *TaskRecord, approval *ApprovalRequestRecord, pr *LandedPR) {
	ok, reason := DecideClosedCard(task, approval, pr)
	if !ok {
		return
	}
	_, err := ProcessApprovalRequest(ctx, &ApprovalParams{
		TaskID:        task.ID,
		Action:        "reject",
		ApprovedBy:    fmt.Sprintf("pr-closed #%d", pr.Number),
		Channel:       "pr-closed",
		Feedback:      reason,
		Store:         d.taskStore,
		MsgStore:      d.msgStore,
		AgentRegistry: d.agentRegistry,
		ObsBackend:    d.obsBackend,
		SkipMerge:     true,
	})
	if err != nil {
		d.logger.Printf("ERROR: closed card %s (%s) could not be rejected: %v", task.ID, reason, err)
		return
	}
	d.logger.Printf("Closed card rejected: %s — %s", task.ID, reason)
}

// lookupSettledPRREST asks GitHub's REST API how branch's PR ended.
func lookupSettledPRREST(ctx context.Context, token, repo, branch string) (*LandedPR, error) {
	owner := repo[:strings.IndexByte(repo, '/')]
	// state=all: a closed PR only settles the card when nothing from the same
	// branch is still open, and that needs the open ones in the same listing.
	q := url.Values{"state": {"all"}, "head": {owner + ":" + branch}}
	apiURL := fmt.Sprintf("%s/repos/%s/pulls?%s", githubAPIBase, repo, q.Encode())
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
	return settledFromREST(prs, branch), nil
}

type restPR struct {
	Number   int     `json:"number"`
	State    string  `json:"state"` // "open" or "closed"
	MergedAt *string `json:"merged_at"`
	Head     struct {
		Ref string `json:"ref"`
	} `json:"head"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// settledFromREST reads how branch's PR ended from a REST listing, in order of
// precedence: any merged PR is MERGED; else any open PR means undecided (nil);
// else the newest closed-unmerged PR is CLOSED. Only exact head matches count.
func settledFromREST(prs []restPR, branch string) *LandedPR {
	var mine []restPR
	for _, p := range prs {
		if p.Head.Ref == branch {
			mine = append(mine, p)
		}
	}
	for _, p := range mine {
		if p.MergedAt != nil {
			// The list endpoint does not carry merged_by; say so rather than
			// attribute the merge to the PR's author.
			return &LandedPR{Number: p.Number, State: "MERGED", HeadRefName: p.Head.Ref, MergedAt: *p.MergedAt}
		}
	}
	var closed *LandedPR
	for _, p := range mine {
		if p.State == "open" {
			return nil
		}
		if closed == nil || p.Number > closed.Number {
			closed = &LandedPR{Number: p.Number, State: "CLOSED", HeadRefName: p.Head.Ref}
		}
	}
	return closed
}
