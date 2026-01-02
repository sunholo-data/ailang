package coordinator

import (
	"os/exec"
	"time"
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
