package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/executor"
)

// openCascadePullRequest opens a PR via the GitHub REST API after the wrapper
// has pushed the agent's branch. Best-effort — if the token is missing, the
// repo URL isn't a GitHub HTTPS URL, or the PR already exists, we log and
// continue (the task itself succeeded; the PR is the surfacing mechanism for
// human review).
//
// We use direct REST calls (not the `gh` CLI) because the agent executor image
// doesn't ship with `gh` installed. GITHUB_TOKEN is already populated for git
// auth (see Step -1) so we reuse it for the REST call.
//
// Cascade context (root_package, root_version) is surfaced via env vars set
// by the dispatcher; if absent, we open a generic agent-task PR instead.
func openCascadePullRequest(ctx context.Context, workDir, branchName, taskID, agentID, baseBranch string) {
	if baseBranch == "" {
		baseBranch = "main"
	}

	// Trim whitespace — Secret Manager values sometimes have trailing newlines
	// which cause net/http to reject the Authorization header outright.
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		fmt.Fprintln(os.Stderr, "execute-job: pr create skipped: GITHUB_TOKEN not set")
		return
	}

	// Get the GitHub owner/repo by parsing the remote URL inside workDir.
	repoOwner, repoName, err := getGitHubOwnerRepo(ctx, workDir)
	if err != nil || repoOwner == "" || repoName == "" {
		fmt.Fprintf(os.Stderr, "execute-job: pr create skipped: cannot parse GitHub repo: %v\n", err)
		return
	}

	// Detect cascade vs generic agent task from env (set by dispatcher when
	// the inbound message had source=cascade + root_package attribute).
	rootPackage := os.Getenv("AILANG_CASCADE_ROOT_PACKAGE")
	title := fmt.Sprintf("[agent] %s: %s", agentID, taskID)
	body := fmt.Sprintf("Autonomous task `%s` completed and pushed by agent `%s`.\n\n"+
		"This PR was opened deterministically by the AILANG coordinator wrapper.\n\n"+
		"View execution chain: `ailang chains view %s`", taskID, agentID, taskID)
	labels := []string{"agent-task"}

	if rootPackage != "" {
		title = fmt.Sprintf("[cascade] bump %s (%s)", rootPackage, taskID)
		body = fmt.Sprintf("Cascade-driven dependency update from `%s`.\n\n"+
			"Triggered by autonomous task `%s` (agent `%s`).\n\n"+
			"This PR was opened deterministically by the AILANG coordinator wrapper.\n"+
			"Always-PR is the v1 design — no autonomous merge.\n\n"+
			"View execution chain: `ailang chains view %s`", rootPackage, taskID, agentID, taskID)
		labels = []string{"cascade", "agent-task"}
	}

	prNum, prURL, err := createGitHubPR(ctx, token, repoOwner, repoName, title, body, branchName, baseBranch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: pr create failed: %v\n", err)
		return
	}
	fmt.Printf("execute-job: opened PR #%d: %s\n", prNum, prURL)

	// Best-effort: apply labels in a follow-up call. PR creation succeeded
	// even if labelling fails.
	if err := addGitHubLabels(ctx, token, repoOwner, repoName, prNum, labels); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: pr labels skipped: %v\n", err)
	}
}

// getGitHubOwnerRepo parses `git remote get-url origin` to extract owner/repo
// for HTTPS GitHub URLs. Returns empty strings + nil error for non-GitHub repos.
func getGitHubOwnerRepo(ctx context.Context, workDir string) (string, string, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", workDir, "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("git remote: %w", err)
	}
	url := strings.TrimSpace(string(out))
	// Accept https://github.com/OWNER/REPO[.git] or git@github.com:OWNER/REPO[.git]
	url = strings.TrimSuffix(url, ".git")
	for _, prefix := range []string{"https://github.com/", "git@github.com:"} {
		if strings.HasPrefix(url, prefix) {
			rest := strings.TrimPrefix(url, prefix)
			parts := strings.SplitN(rest, "/", 2)
			if len(parts) == 2 {
				return parts[0], parts[1], nil
			}
		}
	}
	return "", "", nil
}

// createGitHubPR makes a POST /repos/{owner}/{repo}/pulls call.
// Returns the PR number, its html_url, and any error.
func createGitHubPR(ctx context.Context, token, owner, repo, title, body, head, base string) (int, string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)
	payload := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}
	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return 0, "", fmt.Errorf("github api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	var prResp struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(respBody, &prResp); err != nil {
		return 0, "", fmt.Errorf("decode pr response: %w", err)
	}
	return prResp.Number, prResp.HTMLURL, nil
}

// addGitHubLabels makes a POST /repos/{owner}/{repo}/issues/{prNum}/labels call.
// PRs and issues share the labels endpoint.
func addGitHubLabels(ctx context.Context, token, owner, repo string, prNum int, labels []string) error {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/labels", owner, repo, prNum)
	payload := map[string][]string{"labels": labels}
	bodyBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github labels api returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return nil
}

// discoverChangedFilesFromCommit uses ArtifactDiscovery to find files created/modified by the agent.
func discoverChangedFilesFromCommit(workDir, clonePoint string) []string {
	ad := coordinator.NewArtifactDiscovery(workDir, nil)
	if clonePoint != "" {
		ad.WithBaseCommit(clonePoint)
	}
	files, err := ad.DiscoverChangedFiles()
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: warning: artifact discovery failed: %v\n", err)
		return nil
	}
	if len(files) > 0 {
		fmt.Printf("execute-job: discovered %d changed files\n", len(files))
	}
	return files
}

// injectAgentsMD copies AGENTS.md from the plugin directory into the workspace
// if the workspace doesn't already have one.
func injectAgentsMD(pluginDir, workDir string) {
	src := filepath.Join(pluginDir, "AGENTS.md")
	dst := filepath.Join(workDir, "AGENTS.md")

	if _, err := os.Stat(dst); err == nil {
		fmt.Printf("execute-job: AGENTS.md already exists in repo, skipping injection\n")
		return
	}

	srcData, err := os.ReadFile(src)
	if err != nil {
		return
	}

	if err := os.WriteFile(dst, srcData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to inject AGENTS.md: %v\n", err)
		return
	}
	fmt.Printf("execute-job: injected AGENTS.md from plugin into workspace\n")
}

// writeTaskArtifacts writes execution artifacts to the GCS-mounted artifact directory.
//
// The artifact bucket is mounted read-write at /artifacts via Cloud Run volume mount
// (configured in Terraform). Files written here go directly to GCS — no upload client needed.
//
// Writes:
//   - /artifacts/tasks/{taskID}/transcript.txt — plain-text turn summary
//   - /artifacts/tasks/{taskID}/metrics.json   — extended metrics (cache tokens, files)
//   - /artifacts/tasks/{taskID}/session.jsonl  — Claude Code JSONL history (copied from CLAUDE_CONFIG_DIR)
//
// The JSONL copy is necessary because gcsfuse uses "legacy staged writes" for files
// that Claude appends to incrementally — the staged write may not flush before the
// container exits. Explicitly re-writing the file via os.WriteFile guarantees flush.
//
// Returns the GCS path prefix ("tasks/{taskID}") for linking from Firestore.
// Failures are non-fatal and logged to stderr.
func writeTaskArtifacts(taskID string, result *executor.Result) string {
	artifactDir := filepath.Join("/artifacts", "tasks", taskID)
	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "execute-job: warning: could not create artifact dir %s: %v\n", artifactDir, err)
		return ""
	}

	prefix := fmt.Sprintf("tasks/%s", taskID)

	if result == nil {
		return prefix
	}

	// 1. Plain-text transcript
	if result.Transcript != "" {
		p := filepath.Join(artifactDir, "transcript.txt")
		if err := os.WriteFile(p, []byte(result.Transcript), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "execute-job: warning: failed to write transcript.txt: %v\n", err)
		}
	}

	// 2. Extended metrics (cache tokens, files — not in TaskCompletion message)
	metrics := map[string]any{
		"task_id":               taskID,
		"session_id":            result.SessionID,
		"num_turns":             result.NumTurns,
		"tool_call_count":       result.ToolCallCount,
		"input_tokens":          result.InputTokens,
		"output_tokens":         result.OutputTokens,
		"cache_read_tokens":     result.CacheReadInputTokens,
		"cache_creation_tokens": result.CacheCreationInputTokens,
		"cost_usd":              result.CostUSD,
		"duration_ms":           result.DurationMS,
		"files_created":         result.FilesCreated,
		"files_modified":        result.FilesModified,
		"written_at":            time.Now().UTC().Format(time.RFC3339),
	}
	if metricsJSON, err := json.Marshal(metrics); err == nil {
		p := filepath.Join(artifactDir, "metrics.json")
		if err := os.WriteFile(p, metricsJSON, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "execute-job: warning: failed to write metrics.json: %v\n", err)
		}
	}

	// 3. Copy Claude Code session JSONL to session.jsonl.
	// Claude writes the JSONL to CLAUDE_CONFIG_DIR/projects/{path}/{sessionID}.jsonl via gcsfuse.
	// gcsfuse uses legacy staged writes for incrementally-appended files, which may not flush
	// before the container exits. Re-writing via os.WriteFile guarantees the data reaches GCS.
	if result.SessionID != "" {
		claudeConfigDir := os.Getenv("CLAUDE_CONFIG_DIR")
		if claudeConfigDir == "" {
			claudeConfigDir = filepath.Join("/artifacts", "tasks", taskID, "claude")
		}
		jsonlPath := findSessionJSONL(claudeConfigDir, result.SessionID)
		if jsonlPath != "" {
			if data, err := os.ReadFile(jsonlPath); err == nil && len(data) > 0 {
				dst := filepath.Join(artifactDir, "session.jsonl")
				if err := os.WriteFile(dst, data, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "execute-job: warning: failed to write session.jsonl: %v\n", err)
				} else {
					fmt.Printf("execute-job: session.jsonl written (%d bytes)\n", len(data))
				}
			}
		}
	}

	fmt.Printf("execute-job: artifacts written to /artifacts/%s\n", prefix)
	return prefix
}

// findSessionJSONL searches claudeConfigDir for a JSONL file matching sessionID.
// Claude Code writes to: {claudeConfigDir}/projects/{encoded-path}/{sessionID}.jsonl
func findSessionJSONL(claudeConfigDir, sessionID string) string {
	target := sessionID + ".jsonl"
	var found string
	filepath.WalkDir(claudeConfigDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Base(path) == target {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func printExecuteJobHelp() {
	fmt.Println("Usage: ailang coordinator execute-job")
	fmt.Println("")
	fmt.Println("Execute a task in a Cloud Run Job container. Configuration is read")
	fmt.Println("from environment variables set by Eventarc/Pub/Sub trigger.")
	fmt.Println("")
	fmt.Println("Required Environment Variables:")
	fmt.Println("  AILANG_TASK_ID          Task ID to execute")
	fmt.Println("  AILANG_AGENT_ID         Agent ID (e.g., sprint-executor)")
	fmt.Println("  AILANG_CLOUD_PROJECT    GCP project for Pub/Sub")
	fmt.Println("")
	fmt.Println("Optional Environment Variables:")
	fmt.Println("  AILANG_WORKSPACE        Workspace identifier (default: default)")
	fmt.Println("  AILANG_PROVIDER         Executor: claude or gemini (default: claude)")
	fmt.Println("  AILANG_REPO_URL         Git repo URL to clone")
	fmt.Println("  AILANG_BRANCH           Base branch (default: dev)")
	fmt.Println("  AILANG_PUSH_BRANCH      Push directly to this branch (skip coordinator/ branch)")
	fmt.Println("  AILANG_DIRECTIVE        Task prompt/directive")
	fmt.Println("  AILANG_TOPIC_PREFIX     Topic prefix (default: ailang)")
	fmt.Println("  AILANG_PLUGIN_REPO      Git URL for shared skills plugin (--plugin-dir)")
	fmt.Println("  AILANG_MODEL            AI model override (e.g., sonnet, opus)")
	fmt.Println("  AILANG_MAX_COST_USD     Per-task cost budget in USD (0 = unlimited)")
}
