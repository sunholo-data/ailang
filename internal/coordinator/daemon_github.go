package coordinator

import (
	"os/exec"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// runGitHubSync runs periodic GitHub issue import in the background.
// This imports GitHub issues as messages, which then trigger task creation.
func (d *Daemon) runGitHubSync() {
	cfg := d.coordConfig.GitHubSync
	interval := time.Duration(cfg.IntervalSecs) * time.Second
	if interval < time.Minute {
		interval = 5 * time.Minute // Minimum 5 minutes to avoid rate limits
	}

	d.logger.Printf("GitHub sync started (interval: %v, labels: %v, target: %s)",
		interval, cfg.WatchLabels, cfg.TargetInbox)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on startup
	d.syncGitHubIssues()

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("GitHub sync stopping")
			return
		case <-ticker.C:
			d.syncGitHubIssues()
		}
	}
}

// syncGitHubIssues imports GitHub issues as messages using the ailang CLI.
func (d *Daemon) syncGitHubIssues() {
	cfg := d.coordConfig.GitHubSync
	d.logger.Println("Running GitHub issue sync...")

	// Build the command: ailang messages import-github [--inbox target] [--labels label1,label2]
	args := []string{"messages", "import-github"}
	if cfg.TargetInbox != "" {
		args = append(args, "--inbox", cfg.TargetInbox)
	}
	if len(cfg.WatchLabels) > 0 {
		labels := ""
		for i, label := range cfg.WatchLabels {
			if i > 0 {
				labels += ","
			}
			labels += label
		}
		args = append(args, "--labels", labels)
	}

	// Use exec to run the ailang command
	cmd := exec.CommandContext(d.ctx, "ailang", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		d.logger.Printf("GitHub sync error: %v\nOutput: %s", err, string(output))
		return
	}

	// Log result (trimmed to avoid log spam)
	result := string(output)
	if len(result) > 200 {
		result = result[:200] + "..."
	}
	if result != "" {
		d.logger.Printf("GitHub sync: %s", result)
	} else {
		d.logger.Println("GitHub sync complete (no new issues)")
	}
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

		// Fetch current labels from GitHub
		labels, err := d.fetchIssueLabels(*msg.GitHubIssue)
		if err != nil {
			d.logger.Printf("Label resync: failed to fetch labels for #%d: %v", *msg.GitHubIssue, err)
			continue
		}

		// Determine target inbox based on labels
		newInbox := d.routeByLabels(labels, msg.ToInbox)
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
func (d *Daemon) fetchIssueLabels(issueNumber int) ([]string, error) {
	// Use gh api with --jq to extract label names
	args := []string{"api",
		"-H", "Accept: application/vnd.github+json",
		"/repos/sunholo-data/ailang/issues/" + itoa(issueNumber),
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

// routeByLabels determines the target inbox based on GitHub labels.
func (d *Daemon) routeByLabels(labels []string, currentInbox string) string {
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

