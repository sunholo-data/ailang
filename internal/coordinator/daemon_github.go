package coordinator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"gopkg.in/yaml.v3"
)

// runGitHubSync runs periodic GitHub issue import in the background.
// This imports GitHub issues as messages, which then trigger task creation.
// Supports both single-repo (legacy) and multi-repo configurations.
func (d *Daemon) runGitHubSync() {
	cfg := d.coordConfig.GitHubSync

	// Get default repo from global github config
	defaultRepo := d.getDefaultRepo()

	// Get repos to sync (handles backwards compatibility)
	repos := cfg.GetRepos(defaultRepo)
	if len(repos) == 0 {
		d.logger.Println("GitHub sync: no repos configured or sync disabled")
		return
	}

	// Use the minimum interval across all repos
	minInterval := 5 * time.Minute
	for _, repo := range repos {
		if repo.Enabled && repo.IntervalSecs > 0 {
			interval := time.Duration(repo.IntervalSecs) * time.Second
			if interval < minInterval {
				minInterval = interval
			}
		}
	}
	if minInterval < time.Minute {
		minInterval = 5 * time.Minute // Minimum 5 minutes to avoid rate limits
	}

	d.logger.Printf("GitHub sync started (repos: %d, interval: %v)", len(repos), minInterval)
	for _, repo := range repos {
		if repo.Enabled {
			d.logger.Printf("  - %s -> %s", repo.Repo, repo.TargetInbox)
		}
	}

	ticker := time.NewTicker(minInterval)
	defer ticker.Stop()

	// Run immediately on startup
	d.syncAllRepos(repos)

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("GitHub sync stopping")
			return
		case <-ticker.C:
			d.syncAllRepos(repos)
		}
	}
}

// syncAllRepos syncs issues from all configured repos.
func (d *Daemon) syncAllRepos(repos []RepoSyncConfig) {
	for _, repo := range repos {
		if !repo.Enabled {
			continue
		}
		d.syncRepoIssues(repo)
	}
}

// syncRepoIssues imports GitHub issues from a specific repo as messages using the ailang CLI.
func (d *Daemon) syncRepoIssues(repo RepoSyncConfig) {
	d.logger.Printf("Syncing GitHub issues from %s...", SanitizeLog(repo.Repo))

	// Build the command: ailang messages import-github --repo <repo> [--inbox target] [--labels label1,label2]
	args := []string{"messages", "import-github", "--repo", repo.Repo}
	if repo.TargetInbox != "" {
		args = append(args, "--inbox", repo.TargetInbox)
	}
	if len(repo.WatchLabels) > 0 {
		labels := ""
		for i, label := range repo.WatchLabels {
			if i > 0 {
				labels += ","
			}
			labels += label
		}
		args = append(args, "--labels", labels)
	}

	// Use os.Args[0] to find the current binary — "ailang" may not be in PATH
	// on Cloud Run buildpack images.
	cmd := exec.CommandContext(d.ctx, os.Args[0], args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		d.logger.Printf("GitHub sync error for %s: %v\nOutput: %s", SanitizeLog(repo.Repo), err, SanitizeLog(string(output)))
		return
	}

	// Log result (trimmed to avoid log spam)
	result := string(output)
	if len(result) > 200 {
		result = result[:200] + "..."
	}
	if result != "" {
		d.logger.Printf("GitHub sync [%s]: %s", SanitizeLog(repo.Repo), SanitizeLog(result))
	}
}

// getDefaultRepo returns the default GitHub repo from the global github config.
func (d *Daemon) getDefaultRepo() string {
	// Load from full config file to get github.default_repo
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".ailang", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "sunholo-data/ailang" // Fallback default
	}

	// Parse just to get github.default_repo
	var fullConfig struct {
		GitHub struct {
			DefaultRepo string `yaml:"default_repo"`
		} `yaml:"github"`
	}
	if err := yaml.Unmarshal(data, &fullConfig); err != nil {
		return "sunholo-data/ailang"
	}
	if fullConfig.GitHub.DefaultRepo != "" {
		return fullConfig.GitHub.DefaultRepo
	}
	return "sunholo-data/ailang"
}

// runLabelResync periodically re-checks GitHub labels and updates message routing.
// This fixes messages that were imported before labels were added.
func (d *Daemon) runLabelResync() {
	cfg := d.coordConfig.GitHubSync
	if cfg == nil || !cfg.ResyncLabels {
		return
	}

	interval := time.Duration(cfg.ResyncIntervalSec) * time.Second
	if interval < 30*time.Minute {
		interval = 1 * time.Hour // Minimum 1 hour to avoid rate limits
	}

	d.logger.Printf("Label resync started (interval: %v)", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on startup
	d.resyncLabels()

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("Label resync stopping")
			return
		case <-ticker.C:
			d.resyncLabels()
		}
	}
}

// resyncLabels checks imported messages for label changes and updates routing.
func (d *Daemon) resyncLabels() {
	d.logger.Println("Running label resync...")

	// Open message store
	dbPath := messaging.GetDefaultDatabasePath()
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		d.logger.Printf("Label resync: failed to open store: %v", err)
		return
	}
	defer store.Close()

	// Get messages imported from GitHub in the last 7 days
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	messages, err := store.ListInboxMessages(messaging.InboxListOptions{
		FromAgent: "github",
		Limit:     100,
	})
	if err != nil {
		d.logger.Printf("Label resync: failed to list messages: %v", err)
		return
	}

	updated := 0
	for _, msg := range messages {
		// Skip messages older than 7 days
		if msg.CreatedAt.Before(cutoff) {
			continue
		}

		// Skip messages without GitHub issue
		if msg.GitHubIssue == nil {
			continue
		}

		// Determine repo for this message
		repo := msg.GitHubRepo
		if repo == "" {
			repo = d.getDefaultRepo() // Fallback for old messages
		}

		// Fetch current labels from GitHub
		labels, err := d.fetchIssueLabels(repo, *msg.GitHubIssue)
		if err != nil {
			d.logger.Printf("Label resync: failed to fetch labels for %s#%d: %v", repo, *msg.GitHubIssue, err)
			continue
		}

		// Find repo config for label routing rules
		repoConfig := d.findRepoConfig(repo)

		// Determine target inbox based on labels
		newInbox := d.routeByLabels(labels, msg.ToInbox, repoConfig)
		if newInbox != msg.ToInbox {
			if err := store.ForwardInboxMessage(msg.ID, newInbox); err != nil {
				d.logger.Printf("Label resync: failed to forward #%d: %v", *msg.GitHubIssue, err)
				continue
			}
			d.logger.Printf("Label resync: forwarded #%d from '%s' to '%s'", *msg.GitHubIssue, msg.ToInbox, newInbox)
			updated++
		}
	}

	if updated > 0 {
		d.logger.Printf("Label resync: updated %d message(s)", updated)
	} else {
		d.logger.Println("Label resync: no changes needed")
	}
}

// fetchIssueLabels fetches labels for a GitHub issue using the gh CLI.
func (d *Daemon) fetchIssueLabels(repo string, issueNumber int) ([]string, error) {
	// Use gh api with --jq to extract label names
	args := []string{"api",
		"-H", "Accept: application/vnd.github+json",
		"/repos/" + repo + "/issues/" + itoa(issueNumber),
		"--jq", ".labels[].name",
	}
	cmd := exec.CommandContext(d.ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse labels (one per line)
	var labels []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			labels = append(labels, line)
		}
	}
	return labels, nil
}

// itoa converts int to string (avoiding strconv import for simple case)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// routeByLabels determines the target inbox based on GitHub labels and repo config.
// Uses per-repo label routing if configured, otherwise falls back to default rules.
func (d *Daemon) routeByLabels(labels []string, currentInbox string, repoConfig *RepoSyncConfig) string {
	// Try per-repo label routing first
	if repoConfig != nil && len(repoConfig.LabelRouting) > 0 {
		for _, label := range labels {
			for _, route := range repoConfig.LabelRouting {
				if strings.HasPrefix(label, route.LabelPrefix) {
					return route.Target
				}
			}
		}
	}

	// Fall back to default coordinator:* label routing (for backwards compatibility)
	for _, label := range labels {
		switch {
		case label == "coordinator:bug" || label == "coordinator:feature":
			return "design-doc-creator"
		case label == "coordinator:docs" || label == "coordinator:research":
			return "coordinator"
		case strings.HasPrefix(label, "coordinator:"):
			// Other coordinator:* labels go to coordinator
			return "coordinator"
		}
	}
	// No routing label found, keep current inbox
	return currentInbox
}

// findRepoConfig finds the repo config for a given repo name.
func (d *Daemon) findRepoConfig(repoName string) *RepoSyncConfig {
	if d.coordConfig.GitHubSync == nil {
		return nil
	}
	for i := range d.coordConfig.GitHubSync.Repos {
		if d.coordConfig.GitHubSync.Repos[i].Repo == repoName {
			return &d.coordConfig.GitHubSync.Repos[i]
		}
	}
	return nil
}
