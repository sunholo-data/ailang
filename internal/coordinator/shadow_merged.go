package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// Design layer 2 (M-AILANG-LANE-EXECUTOR-SHADOW M6): once the primary's PR
// settles, compare both candidates with what was actually accepted. This half
// only READS GitHub; the merged PR's tests are run by the grade job.

const (
	// shadowMergedSweepInterval throttles the PR lookups, like the landed-card sweep.
	shadowMergedSweepInterval = 10 * time.Minute
	// shadowMergedGiveUpAfter stops waiting for a PR nobody settles.
	shadowMergedGiveUpAfter = 14 * 24 * time.Hour
)

// mergedPRDetailsLookup reads a merged PR's head, merge commit and changed
// paths. A variable so tests do not reach GitHub.
var mergedPRDetailsLookup = lookupMergedPRDetailsREST

// sweepShadowMerged advances comparisons waiting on their primary's PR.
func (d *Daemon) sweepShadowMerged(store ShadowComparisonStore, now time.Time) {
	if now.Sub(d.lastShadowMergedSweep) < shadowMergedSweepInterval {
		return
	}
	d.lastShadowMergedSweep = now
	waiting, err := store.ListShadowComparisonsByGradeState(d.ctx, ShadowGradeGraded, 50)
	if err != nil || len(waiting) == 0 {
		return
	}
	token := config.GitHubToken()
	if token == "" {
		d.logger.Printf("Warning: shadow merged-fix sweep cannot run — GITHUB_TOKEN is unset")
		return
	}
	for _, c := range waiting {
		if d.ctx.Err() != nil {
			return
		}
		merged, err := d.shadowMergedFix(c, token, now)
		if err != nil {
			d.logger.Printf("Warning: shadow merged-fix %s: %v", c.ParentID, err)
			continue
		}
		if merged == nil {
			continue // still open
		}
		next := ShadowGradeReview
		if merged.State == "merged" && len(merged.TestFiles) > 0 {
			next = ShadowGradeMergedTests
		}
		d.settleShadow(store, c.ParentID, ShadowGradeGraded, "", func(cur *ShadowComparison, _ string) {
			cur.Merged, cur.GradeState = merged, next
		})
	}
}

// shadowMergedFix returns how the primary's PR ended, or nil while it is open.
func (d *Daemon) shadowMergedFix(c *ShadowComparison, token string, now time.Time) (*ShadowMergedFix, error) {
	agent := d.shadowAgent(c)
	repo := ""
	if agent != nil {
		repo = agent.ResolveRepo()
	}
	if strings.Count(repo, "/") != 1 {
		return &ShadowMergedFix{State: "unsettled"}, nil // no repo to ask: nothing will ever settle it
	}
	pr, err := settledPRLookup(d.ctx, token, repo, BranchForTask(c.ParentID))
	if err != nil {
		return nil, err
	}
	switch {
	case pr == nil && now.Sub(c.CreatedAt) > shadowMergedGiveUpAfter:
		return &ShadowMergedFix{State: "unsettled"}, nil
	case pr == nil:
		return nil, nil
	case pr.State == "CLOSED":
		return &ShadowMergedFix{State: "closed", Number: pr.Number}, nil
	}
	head, mergeSHA, files, err := mergedPRDetailsLookup(d.ctx, token, repo, pr.Number)
	if err != nil {
		return nil, err
	}
	m := &ShadowMergedFix{State: "merged", Number: pr.Number, HeadSHA: head, MergeCommit: mergeSHA, Files: files}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.ail") {
			m.TestFiles = append(m.TestFiles, f)
		}
	}
	m.Circular = c.Primary != nil && head != "" && head == c.Primary.HeadCommit
	if c.Primary != nil {
		m.PrimaryOverlap = jaccard(c.Primary.ChangedFiles, files)
	}
	if c.Shadow != nil {
		m.ShadowOverlap = jaccard(c.Shadow.ChangedFiles, files)
	}
	return m, nil
}

// jaccard is |a∩b| / |a∪b| over paths; 0 when both are empty.
func jaccard(a, b []string) float64 {
	set := map[string]int{}
	for _, x := range a {
		set[x] |= 1
	}
	for _, x := range b {
		set[x] |= 2
	}
	if len(set) == 0 {
		return 0
	}
	both := 0
	for _, v := range set {
		if v == 3 {
			both++
		}
	}
	return float64(both) / float64(len(set))
}

// lookupMergedPRDetailsREST reads a PR's head SHA, merge commit and changed
// file paths (paged, up to GitHub's 3000-file listing limit).
func lookupMergedPRDetailsREST(ctx context.Context, token, repo string, number int) (head, merge string, files []string, err error) {
	var pr struct {
		MergeCommitSHA string `json:"merge_commit_sha"`
		Head           struct {
			SHA string `json:"sha"`
		} `json:"head"`
	}
	if err := githubGET(ctx, token, fmt.Sprintf("%s/repos/%s/pulls/%d", githubAPIBase, repo, number), &pr); err != nil {
		return "", "", nil, err
	}
	for page := 1; page <= 30; page++ {
		var batch []struct {
			Filename string `json:"filename"`
		}
		if err := githubGET(ctx, token, fmt.Sprintf("%s/repos/%s/pulls/%d/files?per_page=100&page=%d", githubAPIBase, repo, number, page), &batch); err != nil {
			return "", "", nil, err
		}
		for _, f := range batch {
			files = append(files, f.Filename)
		}
		if len(batch) < 100 {
			break
		}
	}
	sort.Strings(files)
	return pr.Head.SHA, pr.MergeCommitSHA, files, nil
}

func githubGET(ctx context.Context, token, apiURL string, out any) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", apiURL, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
