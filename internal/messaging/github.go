package messaging

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// GitHubClient provides integration with GitHub via the gh CLI.
// All methods require gh to be installed and authenticated.
type GitHubClient struct {
	config *GitHubConfig
	// For testing: override command execution
	execCommand func(name string, arg ...string) ([]byte, error)
}

// GitHubIssue represents a GitHub issue returned from the API.
type GitHubIssue struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	State     string   `json:"state"`
	Labels    []string `json:"labels"`
	CreatedAt string   `json:"createdAt"`
	Author    string   `json:"author"`
	URL       string   `json:"url"`
}

// ghIssueResponse is the raw response from gh issue list --json
type ghIssueResponse struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	CreatedAt string `json:"createdAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	URL string `json:"url"`
}

// NewGitHubClient creates a new GitHub client with the given configuration.
func NewGitHubClient(config *GitHubConfig) *GitHubClient {
	return &GitHubClient{
		config:      config,
		execCommand: defaultExecCommand,
	}
}

// defaultExecCommand executes a command and returns its output.
func defaultExecCommand(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	return cmd.CombinedOutput()
}

// GetConfig returns the GitHub configuration (may be nil if not configured).
func (c *GitHubClient) GetConfig() *GitHubConfig {
	return c.config
}

// CheckGHInstalled verifies that the gh CLI is installed and returns the version.
// Returns an error with installation instructions if not found.
func (c *GitHubClient) CheckGHInstalled() (string, error) {
	output, err := c.execCommand("gh", "--version")
	if err != nil {
		return "", fmt.Errorf("gh CLI not installed. Install it with:\n"+
			"  macOS:   brew install gh\n"+
			"  Linux:   sudo apt install gh  OR  sudo dnf install gh\n"+
			"  Windows: winget install GitHub.cli\n"+
			"  Other:   https://cli.github.com/\n\nError: %w", err)
	}

	// Parse version from "gh version X.Y.Z (date)"
	versionLine := strings.Split(string(output), "\n")[0]
	parts := strings.Fields(versionLine)
	if len(parts) >= 3 {
		return parts[2], nil
	}
	return strings.TrimSpace(string(output)), nil
}

// CheckGHAuth verifies that gh is authenticated and returns the active username.
// Returns an error with authentication instructions if not logged in.
func (c *GitHubClient) CheckGHAuth() (string, error) {
	output, err := c.execCommand("gh", "auth", "status")
	if err != nil {
		return "", fmt.Errorf("gh CLI not authenticated. Run:\n"+
			"  gh auth login\n\nError: %w", err)
	}

	// Parse username from "Logged in to github.com account USERNAME (keyring)"
	// or "Logged in to github.com as USERNAME"
	outputStr := string(output)

	// Try pattern: "account USERNAME"
	accountRe := regexp.MustCompile(`account\s+(\S+)`)
	if matches := accountRe.FindStringSubmatch(outputStr); len(matches) > 1 {
		return matches[1], nil
	}

	// Try pattern: "as USERNAME"
	asRe := regexp.MustCompile(`as\s+(\S+)`)
	if matches := asRe.FindStringSubmatch(outputStr); len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("could not parse username from gh auth status output:\n%s", outputStr)
}

// ValidateUser checks that the authenticated gh user matches the expected user.
// Returns nil if they match, or an error with instructions if they don't.
// This is a HARD FAIL - callers should not proceed if this returns an error.
func (c *GitHubClient) ValidateUser() error {
	if c.config == nil {
		return fmt.Errorf("GitHub config not loaded")
	}

	if c.config.ExpectedUser == "" {
		return fmt.Errorf("expected_user not configured in github section of ~/.ailang/config.yaml")
	}

	activeUser, err := c.CheckGHAuth()
	if err != nil {
		return err
	}

	if activeUser != c.config.ExpectedUser {
		return fmt.Errorf("GitHub account mismatch!\n"+
			"  Expected: %s\n"+
			"  Active:   %s\n\n"+
			"Switch accounts with:\n"+
			"  gh auth switch --user %s\n\n"+
			"Or update expected_user in ~/.ailang/config.yaml",
			c.config.ExpectedUser, activeUser, c.config.ExpectedUser)
	}

	return nil
}

// PreFlightChecks runs all pre-flight checks before GitHub operations.
// Returns nil if all checks pass, or the first error encountered.
func (c *GitHubClient) PreFlightChecks() error {
	// Check gh is installed
	if _, err := c.CheckGHInstalled(); err != nil {
		return err
	}

	// Validate user matches expected
	if err := c.ValidateUser(); err != nil {
		return err
	}

	return nil
}

// CreateIssueInput contains the parameters for creating a GitHub issue.
type CreateIssueInput struct {
	Title     string   // Issue title (will be prefixed with [from])
	Body      string   // Issue body content
	FromAgent string   // Agent name for attribution
	Category  string   // bug, feature, or general
	Repo      string   // Optional repo override (owner/repo)
	Labels    []string // Additional labels
}

// CreateIssue creates a new GitHub issue and returns the issue number.
// The title is prefixed with [from:agent-name] and a from:agent-name label is added.
// Labels are automatically created if they don't exist.
func (c *GitHubClient) CreateIssue(input CreateIssueInput) (int, error) {
	if err := c.PreFlightChecks(); err != nil {
		return 0, err
	}

	repo := input.Repo
	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return 0, fmt.Errorf("no repository specified: set --repo or configure default_repo in ~/.ailang/config.yaml")
	}

	// Format title with [from] prefix
	title := fmt.Sprintf("[%s] %s", input.FromAgent, input.Title)

	// Build label list
	labels := []string{fmt.Sprintf("from:%s", input.FromAgent)}

	// Add category label if specified
	if input.Category != "" {
		labels = append(labels, input.Category)
	}

	// Add configured create_labels
	if c.config != nil {
		labels = append(labels, c.config.CreateLabels...)
	}

	// Add any additional labels
	labels = append(labels, input.Labels...)

	// Ensure all labels exist before creating issue
	if err := c.EnsureLabelsForIssue(repo, labels); err != nil {
		return 0, fmt.Errorf("failed to ensure labels: %w", err)
	}

	// Build body with metadata footer
	body := input.Body
	if input.FromAgent != "" {
		body += fmt.Sprintf("\n\n---\n_Reported by: %s via ailang messages_", input.FromAgent)
	}

	// Build gh command
	args := []string{"issue", "create",
		"--repo", repo,
		"--title", title,
		"--body", body,
	}

	// Add labels
	for _, label := range labels {
		args = append(args, "--label", label)
	}

	output, err := c.execCommand("gh", args...)
	if err != nil {
		return 0, fmt.Errorf("failed to create issue: %w\nOutput: %s", err, string(output))
	}

	// Parse issue number from URL in output
	// Output format: "https://github.com/owner/repo/issues/123"
	outputStr := strings.TrimSpace(string(output))
	issueNum, err := parseIssueNumberFromURL(outputStr)
	if err != nil {
		return 0, fmt.Errorf("issue created but could not parse number from output: %s", outputStr)
	}

	return issueNum, nil
}

// parseIssueNumberFromURL extracts the issue number from a GitHub issue URL.
func parseIssueNumberFromURL(url string) (int, error) {
	re := regexp.MustCompile(`/issues/(\d+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return 0, fmt.Errorf("no issue number found in URL: %s", url)
	}
	return strconv.Atoi(matches[1])
}

// ListIssuesByLabel returns all open issues matching the specified labels.
// If no labels provided, uses watch_labels from config.
func (c *GitHubClient) ListIssuesByLabel(repo string, labels []string) ([]GitHubIssue, error) {
	if err := c.PreFlightChecks(); err != nil {
		return nil, err
	}

	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return nil, fmt.Errorf("no repository specified")
	}

	// Use watch_labels from config if no labels provided
	if len(labels) == 0 && c.config != nil {
		labels = c.config.WatchLabels
	}

	args := []string{"issue", "list",
		"--repo", repo,
		"--state", "open",
		"--json", "number,title,body,state,labels,createdAt,author,url",
	}

	// Add label filters
	for _, label := range labels {
		args = append(args, "--label", label)
	}

	output, err := c.execCommand("gh", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w\nOutput: %s", err, string(output))
	}

	// Parse JSON response
	var rawIssues []ghIssueResponse
	if err := json.Unmarshal(output, &rawIssues); err != nil {
		return nil, fmt.Errorf("failed to parse issue list: %w", err)
	}

	// Convert to our GitHubIssue type
	issues := make([]GitHubIssue, len(rawIssues))
	for i, raw := range rawIssues {
		labelNames := make([]string, len(raw.Labels))
		for j, l := range raw.Labels {
			labelNames[j] = l.Name
		}
		issues[i] = GitHubIssue{
			Number:    raw.Number,
			Title:     raw.Title,
			Body:      raw.Body,
			State:     raw.State,
			Labels:    labelNames,
			CreatedAt: raw.CreatedAt,
			Author:    raw.Author.Login,
			URL:       raw.URL,
		}
	}

	return issues, nil
}

// GetIssue retrieves a single issue by number.
func (c *GitHubClient) GetIssue(repo string, number int) (*GitHubIssue, error) {
	if err := c.PreFlightChecks(); err != nil {
		return nil, err
	}

	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return nil, fmt.Errorf("no repository specified")
	}

	args := []string{"issue", "view",
		"--repo", repo,
		strconv.Itoa(number),
		"--json", "number,title,body,state,labels,createdAt,author,url",
	}

	output, err := c.execCommand("gh", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue #%d: %w\nOutput: %s", number, err, string(output))
	}

	var raw ghIssueResponse
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}

	labelNames := make([]string, len(raw.Labels))
	for i, l := range raw.Labels {
		labelNames[i] = l.Name
	}

	return &GitHubIssue{
		Number:    raw.Number,
		Title:     raw.Title,
		Body:      raw.Body,
		State:     raw.State,
		Labels:    labelNames,
		CreatedAt: raw.CreatedAt,
		Author:    raw.Author.Login,
		URL:       raw.URL,
	}, nil
}

// CloseIssue closes an issue with an optional comment.
func (c *GitHubClient) CloseIssue(repo string, number int, comment string) error {
	if err := c.PreFlightChecks(); err != nil {
		return err
	}

	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return fmt.Errorf("no repository specified")
	}

	// Add comment if provided
	if comment != "" {
		commentArgs := []string{"issue", "comment",
			"--repo", repo,
			strconv.Itoa(number),
			"--body", comment,
		}
		if output, err := c.execCommand("gh", commentArgs...); err != nil {
			return fmt.Errorf("failed to add comment to issue #%d: %w\nOutput: %s", number, err, string(output))
		}
	}

	// Close the issue
	args := []string{"issue", "close",
		"--repo", repo,
		strconv.Itoa(number),
	}

	output, err := c.execCommand("gh", args...)
	if err != nil {
		return fmt.Errorf("failed to close issue #%d: %w\nOutput: %s", number, err, string(output))
	}

	return nil
}

// AddComment adds a comment to an existing issue.
func (c *GitHubClient) AddComment(repo string, number int, body string) error {
	if err := c.PreFlightChecks(); err != nil {
		return err
	}

	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return fmt.Errorf("no repository specified")
	}

	args := []string{"issue", "comment",
		"--repo", repo,
		strconv.Itoa(number),
		"--body", body,
	}

	output, err := c.execCommand("gh", args...)
	if err != nil {
		return fmt.Errorf("failed to add comment to issue #%d: %w\nOutput: %s", number, err, string(output))
	}

	return nil
}

// EnsureLabel creates a label if it doesn't already exist.
// Returns nil if label exists or was created successfully.
func (c *GitHubClient) EnsureLabel(repo, name, description, color string) error {
	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}
	if repo == "" {
		return fmt.Errorf("no repository specified")
	}

	// Try to create the label - gh label create with --force is idempotent
	// It will create or update the label as needed
	args := []string{"label", "create", name,
		"--repo", repo,
		"--description", description,
		"--color", color,
		"--force", // Create or update label
	}

	_, err := c.execCommand("gh", args...)
	if err != nil {
		// Log but don't fail - label might exist with different permissions
		// The issue creation will fail later if the label truly doesn't exist
		return nil
	}
	return nil
}

// EnsureLabelsForIssue ensures all labels needed for an issue exist.
func (c *GitHubClient) EnsureLabelsForIssue(repo string, labels []string) error {
	if repo == "" && c.config != nil {
		repo = c.config.DefaultRepo
	}

	// Define colors for different label types
	labelColors := map[string]string{
		"bug":            "D73A4A", // Red
		"feature":        "A2EEEF", // Cyan
		"general":        "C5DEF5", // Light blue
		"ailang-message": "0052CC", // Blue
	}
	defaultColor := "5319E7" // Purple for from:* labels

	for _, label := range labels {
		color := defaultColor
		description := "Auto-created by ailang messages"

		// Use specific colors for known labels
		if c, ok := labelColors[label]; ok {
			color = c
		}

		// Add descriptive text for known label types
		if strings.HasPrefix(label, "from:") {
			agentName := strings.TrimPrefix(label, "from:")
			description = fmt.Sprintf("Message from %s agent", agentName)
		} else if label == "bug" {
			description = "Bug report"
		} else if label == "feature" {
			description = "Feature request"
		} else if label == "ailang-message" {
			description = "Message from AILANG messaging system"
		}

		if err := c.EnsureLabel(repo, label, description, color); err != nil {
			return fmt.Errorf("failed to ensure label %q: %w", label, err)
		}
	}

	return nil
}
