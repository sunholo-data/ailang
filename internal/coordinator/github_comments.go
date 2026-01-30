package coordinator

import (
	"fmt"
	"strings"
	"time"
)

// IssueComment represents a comment on a GitHub issue with bot detection.
type IssueComment struct {
	ID        int64
	Body      string
	Author    string
	CreatedAt time.Time
	IsBot     bool
}

// DefaultBotPatterns contains common bot username patterns to filter.
var DefaultBotPatterns = []string{
	"[bot]",
	"github-actions",
	"dependabot",
	"renovate",
	"codecov",
	"stale",
	"ailang-agent",
	"sunholo-voight-kampff", // Our own agent account
}

// IsBotUser checks if a username matches known bot patterns.
func IsBotUser(username string, additionalPatterns ...string) bool {
	lowerName := strings.ToLower(username)

	// Check default patterns
	for _, pattern := range DefaultBotPatterns {
		if strings.Contains(lowerName, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check additional patterns
	for _, pattern := range additionalPatterns {
		if strings.Contains(lowerName, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// GetRecentHumanComments fetches comments from a GitHub issue,
// filtering out bot comments and only returning those after the given time.
// Returns error if issueNum is invalid (<=0) or repo is not configured.
func (p *GitHubPoster) GetRecentHumanComments(issueNum int, since time.Time) ([]IssueComment, error) {
	// Validate inputs
	if p == nil {
		return nil, fmt.Errorf("GitHubPoster is nil")
	}
	if p.repo == "" {
		return nil, fmt.Errorf("repository not configured")
	}
	if issueNum <= 0 {
		return nil, fmt.Errorf("issue number must be positive, got %d", issueNum)
	}

	// Get all comments from GitHub
	ghComments, err := p.client.GetIssueComments(p.repo, issueNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue comments: %w", err)
	}

	// Filter and convert
	var result []IssueComment
	for _, c := range ghComments {
		// Parse timestamp
		createdAt, err := time.Parse(time.RFC3339, c.CreatedAt)
		if err != nil {
			// If we can't parse, skip this comment
			continue
		}

		// Skip comments before the since time
		if createdAt.Before(since) {
			continue
		}

		// Check if author is a bot
		isBot := IsBotUser(c.Author)

		// Only include human comments
		if isBot {
			continue
		}

		result = append(result, IssueComment{
			ID:        c.ID,
			Body:      c.Body,
			Author:    c.Author,
			CreatedAt: createdAt,
			IsBot:     false,
		})
	}

	return result, nil
}

// GetLatestHumanComment returns the most recent human comment, or nil if none.
func (p *GitHubPoster) GetLatestHumanComment(issueNum int, since time.Time) (*IssueComment, error) {
	comments, err := p.GetRecentHumanComments(issueNum, since)
	if err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return nil, nil
	}

	// Return the latest (last in chronological order)
	return &comments[len(comments)-1], nil
}

// ExtractFeedbackFromComments combines all human comments into a single feedback string.
func ExtractFeedbackFromComments(comments []IssueComment) string {
	if len(comments) == 0 {
		return ""
	}

	var parts []string
	for _, c := range comments {
		// Format: "author (time): body"
		parts = append(parts, c.Body)
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// PostFeedback posts a feedback comment to a GitHub issue.
// Includes iteration context for tracking.
// Returns error if issueNum is invalid, iteration is out of bounds, or channel is empty.
// Deprecated: Use PostFeedbackInRepo with an explicit repo parameter.
func (p *GitHubPoster) PostFeedback(issueNum int, feedback string, iteration int, channel string) error {
	return p.PostFeedbackInRepo("", issueNum, feedback, iteration, channel)
}

// PostFeedbackInRepo posts a feedback comment to a GitHub issue in a specific repo.
// If repo is empty, falls back to the default repo.
// Returns error if issueNum is invalid, iteration is out of bounds, or channel is empty.
func (p *GitHubPoster) PostFeedbackInRepo(repo string, issueNum int, feedback string, iteration int, channel string) error {
	// Validate inputs
	if issueNum <= 0 {
		return fmt.Errorf("issue number must be positive, got %d", issueNum)
	}
	if iteration < 1 || iteration > 3 {
		return fmt.Errorf("iteration must be between 1 and 3, got %d", iteration)
	}
	if channel == "" {
		return fmt.Errorf("channel cannot be empty")
	}

	body := fmt.Sprintf("**Human Feedback**\n\n%s\n\n---\n_Source: %s | Iteration: %d/3_",
		feedback, channel, iteration)

	if err := p.PostCommentInRepo(repo, issueNum, body); err != nil {
		return fmt.Errorf("failed to post feedback: %w", err)
	}
	return nil
}
