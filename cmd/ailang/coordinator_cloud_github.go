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
	"github.com/sunholo-data/ailang/internal/gitutil"
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
	repoOwner, repoName, err := gitutil.GitHubOwnerRepo(ctx, workDir)
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

	maybeEnableAutoMerge(ctx, token, repoOwner, repoName, prNum, workDir, baseBranch)
}

// maybeEnableAutoMerge turns on GitHub's native auto-merge, under TWO
// independent conditions that must both hold.
//
//  1. The agent is registry-configured for it (AILANG_AUTO_MERGE, set by the
//     dispatcher from agent.auto_merge — never from message content).
//  2. The branch actually changes nothing but documentation.
//
// The second is not redundant. Condition 1 says "this agent is allowed to
// auto-merge because it writes design docs"; condition 2 checks that this
// particular run DID. An agent that is meant to write markdown and emits Go is
// exactly the case where nobody should be merging on green — and it is a
// plausible failure, not a hypothetical: the prompt is model-driven.
//
// Native auto-merge rather than merging ourselves: GitHub waits for the required
// checks, honours branch protection, and if the checks never go green it simply
// never merges. There is no polling loop of ours to get wrong, and no path where
// we merge something protection would have refused.
func maybeEnableAutoMerge(ctx context.Context, token, owner, repo string, prNum int, workDir, baseBranch string) {
	if os.Getenv("AILANG_AUTO_MERGE") != "1" {
		return
	}

	docsOnly, reason, err := branchIsDocsOnly(ctx, workDir, baseBranch)
	if err != nil {
		// Could not tell -> do not enable. A PR left for a human is a delay; an
		// auto-merged one we could not classify is a change nobody reviewed.
		fmt.Fprintf(os.Stderr, "execute-job: auto-merge NOT enabled: cannot classify the diff: %v\n", err)
		return
	}
	if !docsOnly {
		fmt.Fprintf(os.Stderr, "execute-job: auto-merge NOT enabled: %s\n", reason)
		return
	}

	if err := enableGitHubAutoMerge(ctx, token, owner, repo, prNum); err != nil {
		// Not fatal: the PR exists and a human can merge it. Saying so is the
		// point — silence here would look identical to "it will merge itself".
		fmt.Fprintf(os.Stderr, "execute-job: auto-merge could not be enabled on #%d: %v\n", prNum, err)
		return
	}
	fmt.Printf("execute-job: auto-merge enabled on #%d (docs-only; GitHub will merge when required checks pass)\n", prNum)
}

// branchIsDocsOnly reports whether the branch changes only documentation.
//
// The predicate deliberately MATCHES the docs-only lane in .github/workflows/ci.yml.
// If the two ever disagree, a PR could take the fast CI lane and then be
// auto-merged on the strength of checks that never ran the thing it changed.
func branchIsDocsOnly(ctx context.Context, workDir, baseBranch string) (bool, string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "origin/"+baseBranch+"...HEAD")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return false, "", fmt.Errorf("git diff against origin/%s: %w", baseBranch, err)
	}
	files := strings.Fields(strings.TrimSpace(string(out)))
	if len(files) == 0 {
		return false, "the branch changes no files", nil
	}
	for _, f := range files {
		if !isDocsPath(f) {
			return false, fmt.Sprintf("%s is not a documentation file", f), nil
		}
	}
	return true, "", nil
}

// isDocsPath mirrors the ci.yml docs-only lane exactly.
//
// NOT every .md: CLAUDE.md and .claude/**/*.md change agent behaviour, and
// docs/ is the website with its own build. Only design docs and changelogs.
func isDocsPath(f string) bool {
	if !strings.HasSuffix(f, ".md") {
		return false
	}
	return strings.HasPrefix(f, "design_docs/") || strings.HasPrefix(f, "changelogs/")
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
// if the workspace doesn't already have one, and EXCLUDES the injected copy from
// git.
//
// The exclusion is the point. AGENTS.md is the HARNESS's instruction file, not
// agent output — but it landed in the workspace untracked and unignored, so the
// commit step swept it up. Four ailang-parse PRs (#26, #27, #28, #30) each
// carried an identical `AGENTS.md +60/-0`, and two carried nothing else: a run
// that produced no work looked like a change. That is worse than an empty PR,
// because it hides the real defect behind a plausible diff.
//
// The cascade path had spotted the same clutter and skipped injection when
// AILANG_CASCADE_ROOT_PACKAGE was set. That protected one caller and left every
// other task committing the file — patching the symptom where it was noticed
// instead of fixing it where it lives (CLAUDE.md Principle 3). With the
// exclusion here, that special case is no longer needed.
func injectAgentsMD(pluginDir, workDir string) {
	src := filepath.Join(pluginDir, "AGENTS.md")
	dst := filepath.Join(workDir, "AGENTS.md")

	// A repo that ships its own AGENTS.md keeps it: it is tracked content and
	// must stay committable. Only the copy WE inject gets excluded.
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

	// Exclude BEFORE the agent runs, so no commit can race it.
	if err := excludeFromGit(workDir, "AGENTS.md"); err != nil {
		// Loud, because the failure mode is silent PR pollution rather than a
		// crash: the job still works, and every PR it makes is misleading.
		fmt.Fprintf(os.Stderr,
			"warning: injected AGENTS.md but could NOT exclude it from git (%v) — "+
				"it may be committed into this task's PR and make an empty run look like a change\n", err)
		return
	}
	fmt.Printf("execute-job: injected AGENTS.md from plugin into workspace (excluded from git)\n")
}

// excludeFromGit adds a path to the repository's .git/info/exclude — the
// per-clone ignore list.
//
// info/exclude rather than .gitignore on purpose: .gitignore is tracked content,
// so writing to it would itself be a change the agent commits, which is the very
// problem being fixed. info/exclude is local to the clone and invisible to the
// diff.
//
// Resolved via `git rev-parse --git-dir` rather than assuming <workDir>/.git,
// because in a worktree .git is a FILE pointing elsewhere and the naive path
// would silently write a regular file that git never reads.
func excludeFromGit(workDir, pattern string) error {
	out, err := exec.Command("git", "-C", workDir, "rev-parse", "--git-dir").Output()
	if err != nil {
		return fmt.Errorf("resolve git dir: %w", err)
	}
	gitDir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(workDir, gitDir)
	}

	infoDir := filepath.Join(gitDir, "info")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", infoDir, err)
	}
	excludePath := filepath.Join(infoDir, "exclude")

	existing, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", excludePath, err)
	}
	// Idempotent: a job may re-run in a reused workspace.
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == pattern {
			return nil
		}
	}

	body := string(existing)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	body += pattern + "\n"
	return os.WriteFile(excludePath, []byte(body), 0o644)
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

// enableGitHubAutoMerge turns on auto-merge for a PR.
//
// GraphQL, because there is no REST endpoint for this: `enablePullRequestAutoMerge`
// is the only API that sets it. It needs the PR's node_id, which the REST create
// response does not give us in a form we kept, so we fetch it first.
//
// SQUASH deliberately: an agent branch is a working history ("fix lint", "address
// review"), and dev keeps one commit per landed change.
func enableGitHubAutoMerge(ctx context.Context, token, owner, repo string, prNum int) error {
	nodeID, err := pullRequestNodeID(ctx, token, owner, repo, prNum)
	if err != nil {
		return err
	}

	const mutation = `mutation($id:ID!){enablePullRequestAutoMerge(input:{pullRequestId:$id,mergeMethod:SQUASH}){clientMutationId}}`
	payload, _ := json.Marshal(map[string]interface{}{
		"query":     mutation,
		"variables": map[string]string{"id": nodeID},
	})

	body, err := githubGraphQL(ctx, token, payload)
	if err != nil {
		return err
	}

	// GraphQL reports failures in the body with HTTP 200, so a status check
	// alone would call every refusal a success — including "auto-merge is not
	// allowed on this repository", which is precisely what we need to hear.
	var resp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decoding the auto-merge response: %w", err)
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("%s", resp.Errors[0].Message)
	}
	return nil
}

// pullRequestNodeID fetches the GraphQL node id for a PR number.
func pullRequestNodeID(ctx context.Context, token, owner, repo string, prNum int) (string, error) {
	const query = `query($o:String!,$r:String!,$n:Int!){repository(owner:$o,name:$r){pullRequest(number:$n){id}}}`
	payload, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"o": owner, "r": repo, "n": prNum,
		},
	})
	body, err := githubGraphQL(ctx, token, payload)
	if err != nil {
		return "", err
	}
	var resp struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ID string `json:"id"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decoding the node-id response: %w", err)
	}
	if len(resp.Errors) > 0 {
		return "", fmt.Errorf("%s", resp.Errors[0].Message)
	}
	if resp.Data.Repository.PullRequest.ID == "" {
		return "", fmt.Errorf("no node id returned for #%d", prNum)
	}
	return resp.Data.Repository.PullRequest.ID, nil
}

// githubGraphQL posts a GraphQL request and returns the raw body.
func githubGraphQL(ctx context.Context, token string, payload []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.github.com/graphql", strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("github graphql: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}
