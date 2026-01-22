package coordinator

import (
	"fmt"

	"github.com/sunholo/ailang/internal/messaging"
)

// LabelInProgress is the claim label used to prevent race conditions
// when multiple coordinator instances try to pick up the same issue.
const LabelInProgress = "coordinator:in-progress"

// GitHubPoster provides GitHub integration for the coordinator.
// It wraps the messaging GitHubClient to post comments, manage labels,
// and close issues as part of the autonomous workflow.
type GitHubPoster struct {
	client *messaging.GitHubClient
	repo   string
}

// NewGitHubPoster creates a new GitHub poster for coordinator tasks.
// Enables auto-switch so the coordinator can automatically switch GitHub accounts
// if there's a mismatch with the expected user.
func NewGitHubPoster() (*GitHubPoster, error) {
	config, err := messaging.LoadGitHubConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load GitHub config: %w", err)
	}

	client := messaging.NewGitHubClient(config)
	// Enable auto-switch for daemon/background operations
	// This allows the coordinator to seamlessly switch GitHub accounts
	// if the user has switched to a different account temporarily
	client.SetAutoSwitch(true)

	repo := ""
	if config != nil {
		repo = config.DefaultRepo
	}

	return &GitHubPoster{
		client: client,
		repo:   repo,
	}, nil
}

// PostComment posts a comment to a GitHub issue.
func (p *GitHubPoster) PostComment(issueNum int, body string) error {
	return p.client.AddComment(p.repo, issueNum, body)
}

// AddLabel adds a label to a GitHub issue.
// Creates the label if it doesn't exist.
func (p *GitHubPoster) AddLabel(issueNum int, label string) error {
	// Ensure label exists first
	if err := p.EnsureLabel(label); err != nil {
		return err
	}
	return p.client.AddLabelToIssue(p.repo, issueNum, label)
}

// RemoveLabel removes a label from a GitHub issue.
func (p *GitHubPoster) RemoveLabel(issueNum int, label string) error {
	return p.client.RemoveLabelFromIssue(p.repo, issueNum, label)
}

// CloseIssue closes a GitHub issue with an optional comment.
func (p *GitHubPoster) CloseIssue(issueNum int, comment string) error {
	return p.client.CloseIssue(p.repo, issueNum, comment)
}

// GetLabels returns the current labels on an issue.
func (p *GitHubPoster) GetLabels(issueNum int) ([]string, error) {
	return p.client.GetIssueLabels(p.repo, issueNum)
}

// HasLabel checks if an issue has a specific label.
func (p *GitHubPoster) HasLabel(issueNum int, label string) (bool, error) {
	labels, err := p.GetLabels(issueNum)
	if err != nil {
		return false, err
	}
	for _, l := range labels {
		if l == label {
			return true, nil
		}
	}
	return false, nil
}

// EnsureLabel creates a label if it doesn't exist.
func (p *GitHubPoster) EnsureLabel(label string) error {
	// Define colors for coordinator labels
	labelColors := map[string]string{
		// Routing labels
		"coordinator:bug":      "D73A4A", // Red
		"coordinator:feature":  "A2EEEF", // Cyan
		"coordinator:docs":     "0E8A16", // Green
		"coordinator:research": "FBCA04", // Yellow
		"coordinator:refactor": "7057FF", // Purple
		"coordinator:test":     "E4E669", // Light yellow

		// Claim label (prevents race conditions across coordinator instances)
		"coordinator:in-progress": "1D76DB", // Blue - task being worked on

		// Status labels
		"needs-design-approval": "B60205", // Dark red
		"needs-sprint-approval": "B60205", // Dark red
		"needs-merge-approval":  "B60205", // Dark red
		"needs-revision":        "FFA500", // Orange

		// Approval labels
		"design-approved": "0E8A16", // Green
		"sprint-approved": "0E8A16", // Green
		"merge-approved":  "0E8A16", // Green
	}

	labelDescriptions := map[string]string{
		// Routing labels
		"coordinator:bug":      "Bug fix - auto-routes to coordinator daemon",
		"coordinator:feature":  "New feature - auto-routes to coordinator daemon",
		"coordinator:docs":     "Documentation task - auto-routes to coordinator daemon",
		"coordinator:research": "Research task - auto-routes to coordinator daemon",
		"coordinator:refactor": "Refactoring task - auto-routes to coordinator daemon",
		"coordinator:test":     "Test writing task - auto-routes to coordinator daemon",

		// Claim label
		"coordinator:in-progress": "Task claimed by a coordinator instance - prevents duplicate work",

		// Status labels
		"needs-design-approval": "Awaiting human approval of design document",
		"needs-sprint-approval": "Awaiting human approval of sprint plan",
		"needs-merge-approval":  "Awaiting human approval to merge changes",
		"needs-revision":        "Requires revision based on feedback",

		// Approval labels
		"design-approved": "Human approved the design document",
		"sprint-approved": "Human approved the sprint plan",
		"merge-approved":  "Human approved the implementation for merge",
	}

	color, ok := labelColors[label]
	if !ok {
		color = "C5DEF5" // Default light blue
	}

	description, ok := labelDescriptions[label]
	if !ok {
		description = "Coordinator auto-routing label"
	}

	return p.client.EnsureLabel(p.repo, label, description, color)
}

// PostWorkingStatus posts a "working on it" comment to the issue.
func (p *GitHubPoster) PostWorkingStatus(issueNum int, taskID, agent string) error {
	body := fmt.Sprintf(`**🤖 Agent Working**

I've picked up this issue and am working on it.

| Field | Value |
|-------|-------|
| **Task ID** | `+"`%s`"+` |
| **Agent** | %s |
| **Status** | In Progress |

You'll receive updates as I make progress.`, taskID, agent)

	return p.PostComment(issueNum, body)
}

// Repo returns the configured repository.
func (p *GitHubPoster) Repo() string {
	return p.repo
}

// Client returns the underlying GitHub client.
func (p *GitHubPoster) Client() *messaging.GitHubClient {
	return p.client
}

// ClaimIssue attempts to claim an issue by adding the in-progress label.
// Returns an error if the issue is already claimed by another coordinator.
// This prevents race conditions when multiple coordinators poll GitHub.
func (p *GitHubPoster) ClaimIssue(issueNum int) error {
	// Check if already claimed
	claimed, err := p.HasLabel(issueNum, LabelInProgress)
	if err != nil {
		return fmt.Errorf("failed to check claim status: %w", err)
	}
	if claimed {
		return fmt.Errorf("issue #%d already claimed by another coordinator", issueNum)
	}

	// Claim it by adding the label
	if err := p.AddLabel(issueNum, LabelInProgress); err != nil {
		return fmt.Errorf("failed to claim issue: %w", err)
	}

	return nil
}

// ReleaseIssue releases a claimed issue by removing the in-progress label.
// Called when a task completes, fails, or is cancelled.
func (p *GitHubPoster) ReleaseIssue(issueNum int) error {
	return p.RemoveLabel(issueNum, LabelInProgress)
}

// IsIssueClaimed checks if an issue is already claimed by any coordinator.
func (p *GitHubPoster) IsIssueClaimed(issueNum int) (bool, error) {
	return p.HasLabel(issueNum, LabelInProgress)
}
