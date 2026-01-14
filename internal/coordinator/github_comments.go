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
func (p *GitHubPoster) GetRecentHumanComments(issueNum int, since time.Time) ([]IssueComment, error) {
	// Get all comments from GitHub
	ghComments, err := p.client.GetIssueComments(p.repo, issueNum)
	if err != nil {
		return nil, err
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
func (p *GitHubPoster) PostFeedback(issueNum int, feedback string, iteration int, channel string) error {
	body := fmt.Sprintf("**Human Feedback**\n\n%s\n\n---\n_Source: %s | Iteration: %d/3_",
		feedback, channel, iteration)

	return p.PostComment(issueNum, body)
}
