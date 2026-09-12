package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// `ailang coordinator agent-check` — does this agent's configuration match reality?
//
// Adding ONE agent (daneel-writer, 2026-09-11) took eight rounds, and every
// failure was silent. Each is a check below:
//
//	 1. the live bucket had the agent but not its ssh_key_secret/push_branch —
//	    a bucket write made before a later repo edit, never re-synced. THREE
//	    separate drifts in one day.                              -> checkDrift
//	 2. merge_branch: main, where the repo's default branch is dev and `main`
//	    does not exist. PR creation is best-effort, so the task would have
//	    reported success with no PR.                             -> checkBranch
//	 3. the fleet token is READ-ONLY on that repo. A `GET` returns 200 either
//	    way, so read access had been mistaken for write.         -> checkPush
//	 4. ssh_key_secret named a secret that did not exist yet.    -> checkSecret
//	 5. …and the executor's service account could not read it.   -> checkSecret
//	 6. auto_merge: true alongside a deploy key — a deploy key cannot use the
//	    GitHub API, so the flag could never fire.                -> checkCoherence
//	 7. artifact_patterns omitted log.md, which the skill writes every run, so
//	    the auto-merge scope guard would have refused every branch. -> checkSkill
//	 8. the config was written but the coordinator had not rolled, so routing
//	    still followed the previous registry.                    -> checkRolled
//
// THE RULE THIS COMMAND KEEPS: a check that cannot run reports UNKNOWN, never
// OK. Several of the above are only visible with permissions this CLI may not
// have — listing deploy keys needs repo admin, and GitHub answers 404 rather
// than 403, so "no keys found" and "not allowed to look" are the same response.
// Reporting that as a pass is how the original mistake happened.

type checkState int

const (
	statePass checkState = iota
	stateFail
	stateUnknown
)

type agentCheck struct {
	Name   string
	State  checkState
	Detail string
	Fix    string // the command or edit that resolves it
}

func (c agentCheck) marker() string {
	switch c.State {
	case statePass:
		return green("✓")
	case stateFail:
		return red("✗")
	default:
		return yellow("?")
	}
}

func coordinatorAgentCheck(args []string) error {
	var agentID string
	repoConfig := os.Getenv("AILANG_AGENT_CHECK_REPO_CONFIG")
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo-config":
			if i+1 < len(args) {
				repoConfig = args[i+1]
				i++
			}
		default:
			if !strings.HasPrefix(args[i], "-") && agentID == "" {
				agentID = args[i]
			}
		}
	}
	if agentID == "" {
		return fmt.Errorf("usage: ailang coordinator agent-check <agent-id> [--repo-config <config.cloud.yaml>]")
	}

	ctx := context.Background()

	// Judge against the LIVE plane, because that is what dispatches. The repo
	// copy is compared separately, as drift.
	live, _, liveErr := loadCloudInboxRegistry()
	if liveErr != nil {
		return fmt.Errorf("cannot read the live registry: %w", liveErr)
	}
	agent := live.GetAgentByID(agentID)
	if agent == nil {
		return fmt.Errorf("no agent %q in the LIVE registry — it may be in the repo but not deployed; check `ailang coordinator config get`", agentID)
	}

	fmt.Printf("agent-check: %s\n\n", agentID)

	checks := []agentCheck{
		checkDrift(agentID, repoConfig),
		checkUnknownKeys(repoConfig),
		checkBranch(ctx, agent),
		checkPush(ctx, agent),
		checkSecret(ctx, agent),
		checkCoherence(agent),
		checkSkill(ctx, agent),
		checkRolled(ctx),
	}

	var fails, unknowns int
	for _, c := range checks {
		fmt.Printf("  %s %-22s %s\n", c.marker(), c.Name, c.Detail)
		if c.Fix != "" && c.State != statePass {
			fmt.Printf("      %s %s\n", cyan("fix:"), c.Fix)
		}
		switch c.State {
		case stateFail:
			fails++
		case stateUnknown:
			unknowns++
		}
	}

	fmt.Println()
	switch {
	case fails > 0:
		fmt.Printf("%s %d check(s) failed — this agent will not work as configured.\n", red("✗"), fails)
		return fmt.Errorf("%d agent check(s) failed", fails)
	case unknowns > 0:
		// Non-zero deliberately: an unverifiable agent is not a verified one,
		// and the exit code is what a script reads.
		fmt.Printf("%s %d check(s) could not be verified. That is not a pass.\n", yellow("!"), unknowns)
		return fmt.Errorf("%d agent check(s) unverifiable", unknowns)
	default:
		fmt.Printf("%s every check passed.\n", green("✓"))
		return nil
	}
}

// checkDrift compares the live registry against the repo source of truth.
//
// The bucket is a CACHE. `coordinator config set` writes it, and a config deploy
// regenerates it from the repo — so a bucket-only edit survives until the next
// deploy and then vanishes. Three edits were lost that way on 2026-09-11.
func checkDrift(agentID, repoConfig string) agentCheck {
	c := agentCheck{Name: "repo/live in sync"}
	if repoConfig == "" {
		c.State = stateUnknown
		c.Detail = "no --repo-config given, so drift against the source of truth is unchecked"
		c.Fix = "ailang coordinator agent-check " + agentID + " --repo-config <multivac>/config/config.cloud.yaml"
		return c
	}
	repoReg, err := coordinator.LoadAgentRegistryFrom(repoConfig)
	if err != nil {
		c.State = stateUnknown
		c.Detail = fmt.Sprintf("cannot read %s: %v", repoConfig, err)
		return c
	}
	repoAgent := repoReg.GetAgentByID(agentID)
	if repoAgent == nil {
		c.State = stateFail
		c.Detail = "live has this agent, the repo does NOT — the next config deploy will delete it"
		c.Fix = "add the agent to " + repoConfig + " and push"
		return c
	}
	live, _, err := loadCloudInboxRegistry()
	if err != nil {
		c.State = stateUnknown
		c.Detail = "cannot re-read the live registry"
		return c
	}
	if diff := diffAgents(live.GetAgentByID(agentID), repoAgent); diff != "" {
		c.State = stateFail
		c.Detail = "live and repo DISAGREE: " + diff
		c.Fix = "re-sync the bucket from the repo, then `coordinator config set` (which rolls)"
		return c
	}
	c.State = statePass
	c.Detail = "live matches " + repoConfig
	return c
}

// diffAgents reports the first meaningful difference, or "".
//
// Only the fields that change BEHAVIOUR. A comment or ordering difference is
// noise, and a drift check that cries wolf gets ignored — which is the same
// outcome as not having one.
func diffAgents(live, repo *coordinator.AgentConfig) string {
	if live == nil || repo == nil {
		return "one side is missing"
	}
	var out []string
	cmp := func(name, a, b string) {
		if a != b {
			out = append(out, fmt.Sprintf("%s live=%q repo=%q", name, a, b))
		}
	}
	cmp("workspace", live.Workspace, repo.Workspace)
	cmp("merge_branch", live.MergeBranch, repo.MergeBranch)
	cmp("model", live.Model, repo.Model)
	cmp("ssh_key_secret", live.SSHKeySecret, repo.SSHKeySecret)
	cmp("ssh_host_alias", live.SSHHostAlias, repo.SSHHostAlias)
	if live.AutoMerge != repo.AutoMerge {
		out = append(out, fmt.Sprintf("auto_merge live=%v repo=%v", live.AutoMerge, repo.AutoMerge))
	}
	if strings.Join(live.ArtifactPatterns, ",") != strings.Join(repo.ArtifactPatterns, ",") {
		out = append(out, "artifact_patterns differ")
	}
	if len(out) > 3 {
		out = append(out[:3], fmt.Sprintf("…and %d more", len(out)-3))
	}
	return strings.Join(out, "; ")
}

// checkBranch verifies the branch the agent targets actually exists.
//
// daneel-writer was configured merge_branch: main against a repo whose default
// is dev and which has no main at all. PR creation is best-effort in the
// wrapper, so the task would have reported success having opened nothing.
func checkBranch(ctx context.Context, a *coordinator.AgentConfig) agentCheck {
	c := agentCheck{Name: "target branch exists"}
	repo := ownerRepoOf(a)
	if repo == "" {
		c.State = stateUnknown
		c.Detail = "workspace is not a GitHub owner/repo coordinate"
		return c
	}
	want, kind := a.MergeBranch, "merge_branch"
	if want == "" {
		c.State = stateUnknown
		c.Detail = "merge_branch is not set"
		return c
	}

	body, code, err := githubGET(ctx, "/repos/"+repo)
	if err != nil || code != 200 {
		c.State = stateUnknown
		c.Detail = fmt.Sprintf("cannot read %s (HTTP %d) — note 404 here can mean NO ACCESS, not absent", repo, code)
		return c
	}
	var meta struct {
		DefaultBranch string `json:"default_branch"`
	}
	_ = json.Unmarshal(body, &meta)

	if _, bcode, _ := githubGET(ctx, "/repos/"+repo+"/branches/"+want); bcode == 200 {
		c.State = statePass
		c.Detail = fmt.Sprintf("%s=%s exists (repo default: %s)", kind, want, meta.DefaultBranch)
		return c
	}
	c.State = stateFail
	c.Detail = fmt.Sprintf("%s=%q does NOT exist on %s; its default branch is %q", kind, want, repo, meta.DefaultBranch)
	c.Fix = fmt.Sprintf("set %s: %s", kind, meta.DefaultBranch)
	return c
}

// checkPush answers the question a GET cannot: can this agent actually write?
//
// `GET /repos/...` returns 200 for a read-only collaborator, so a 200 was once
// taken as proof of push access on a repo that was deliberately read-only. The
// permissions block is the field that answers it.
func checkPush(ctx context.Context, a *coordinator.AgentConfig) agentCheck {
	c := agentCheck{Name: "can write to repo"}
	repo := ownerRepoOf(a)
	if repo == "" {
		c.State = stateUnknown
		c.Detail = "workspace is not a GitHub owner/repo coordinate"
		return c
	}

	// With a deploy key the fleet token's permission is irrelevant — the key is
	// the credential, and it is repo-scoped by GitHub.
	if a.SSHKeySecret != "" {
		c.State = stateUnknown
		c.Detail = "uses an SSH deploy key, so the fleet token's permission does not apply; the in-job pre-flight is what proves it"
		c.Fix = "the wrapper runs ls-remote + push --dry-run on first dispatch and fails loudly"
		return c
	}

	body, code, err := githubGET(ctx, "/repos/"+repo)
	if err != nil || code != 200 {
		c.State = stateUnknown
		c.Detail = fmt.Sprintf("cannot read %s (HTTP %d)", repo, code)
		return c
	}
	var meta struct {
		Permissions struct {
			Push  bool `json:"push"`
			Admin bool `json:"admin"`
		} `json:"permissions"`
	}
	_ = json.Unmarshal(body, &meta)
	if meta.Permissions.Push {
		c.State = statePass
		c.Detail = fmt.Sprintf("the fleet token has push on %s", repo)
		return c
	}
	c.State = stateFail
	c.Detail = fmt.Sprintf("the fleet token is READ-ONLY on %s (push=false) — every push will be denied", repo)
	c.Fix = "grant write, or give the agent an ssh_key_secret (a write-enabled deploy key) and set push_branch"
	return c
}

// checkSecret verifies the named secret exists and the executor can read it.
func checkSecret(ctx context.Context, a *coordinator.AgentConfig) agentCheck {
	c := agentCheck{Name: "ssh key secret"}
	if a.SSHKeySecret == "" {
		c.State = statePass
		c.Detail = "none configured (uses the fleet token)"
		return c
	}
	project := os.Getenv("AILANG_CLOUD_PROJECT")
	if project == "" {
		project = "ailang-multivac"
	}

	if out, err := exec.CommandContext(ctx, "gcloud", "secrets", "describe", a.SSHKeySecret,
		"--project", project, "--format=value(name)").CombinedOutput(); err != nil {
		c.State = stateFail
		c.Detail = fmt.Sprintf("secret %q does not exist in %s: %s", a.SSHKeySecret, project, firstLine(string(out)))
		c.Fix = fmt.Sprintf("gcloud secrets create %s --project %s --data-file <key>", a.SSHKeySecret, project)
		return c
	}

	// Existence is not access. The job reads it as its own service account.
	out, err := exec.CommandContext(ctx, "gcloud", "secrets", "get-iam-policy", a.SSHKeySecret,
		"--project", project, "--format=value(bindings.members)").CombinedOutput()
	if err != nil {
		c.State = stateUnknown
		c.Detail = "secret exists; cannot read its IAM policy to confirm the executor can access it"
		return c
	}
	const sa = "ailang-agent@" + "ailang-multivac.iam.gserviceaccount.com"
	if !strings.Contains(string(out), sa) {
		c.State = stateFail
		c.Detail = fmt.Sprintf("secret exists but %s is NOT granted access — the job will fail fetching it", sa)
		c.Fix = fmt.Sprintf("gcloud secrets add-iam-policy-binding %s --project %s --member=serviceAccount:%s --role=roles/secretmanager.secretAccessor", a.SSHKeySecret, project, sa)
		return c
	}
	c.State = statePass
	c.Detail = fmt.Sprintf("%s exists and %s can read it", a.SSHKeySecret, sa)
	return c
}

// checkCoherence catches settings that cannot both be true.
func checkCoherence(a *coordinator.AgentConfig) agentCheck {
	c := agentCheck{Name: "settings coherent"}
	var problems []string

	// A deploy key is SSH-only. It cannot open a PR or enable auto-merge, so
	// auto_merge alongside one is a flag nothing can act on.
	if a.SSHKeySecret != "" && a.AutoMerge {
		problems = append(problems, "auto_merge with an ssh deploy key: a deploy key cannot use the GitHub API, so it can never fire")
	}
	// Direct push is skip_approval + merge_branch (daemon_tasks_exec.go:211) —
	// there is no push_branch field, though the name reads like one. A direct
	// push opens no PR, so there is nothing for auto-merge to merge.
	if a.SkipApproval && a.MergeBranch != "" && a.AutoMerge {
		problems = append(problems, "auto_merge with skip_approval+merge_branch: that combination pushes DIRECTLY, so no PR is opened")
	}
	// Auto-merge is bounded by the declared artifacts; without them the guard
	// refuses everything, which reads as "auto-merge silently does nothing".
	if a.AutoMerge && len(a.ArtifactPatterns) == 0 {
		problems = append(problems, "auto_merge with no DECLARED artifact_patterns: the default is `**/*`, so the scope guard would bound nothing — the wrapper refuses instead")
	}
	if a.SSHKeySecret != "" && a.SSHHostAlias == "" {
		problems = append(problems, "ssh_key_secret without ssh_host_alias: the alias is the bound that stops the key reaching other repos")
	}
	if len(problems) == 0 {
		c.State = statePass
		c.Detail = "no contradictory settings"
		return c
	}
	c.State = stateFail
	c.Detail = problems[0]
	if len(problems) > 1 {
		c.Detail += fmt.Sprintf(" (+%d more)", len(problems)-1)
	}
	c.Fix = "remove the flag that cannot fire rather than leaving it set-and-ignored"
	return c
}

// sharedSkillsPlugin is the plugin every executor image pre-clones
// (docker/Dockerfile.agent-base, M-CLOUD-PLUGIN-SKILLS). A skill there is
// available to EVERY agent regardless of its workspace.
const sharedSkillsPlugin = "sunholo-data/ailang_bootstrap"

// checkSkill verifies an invoke.type=skill agent has a skill to run.
//
// TWO places resolve, and checking only the first is a false negative: the
// workspace clone (.claude/skills/<name>/SKILL.md) and the shared plugin baked
// into agent-base. Found 2026-09-12 registering design-doc-creator-daneel —
// sunholo-data/daneel has no .claude/ at all, yet the agent runs, because
// design-doc-creator lives in the plugin. Reporting that as a failure would
// have sent someone to commit a duplicate skill into a repo that does not need
// one.
func checkSkill(ctx context.Context, a *coordinator.AgentConfig) agentCheck {
	c := agentCheck{Name: "skill present"}
	if a.Invoke == nil || a.Invoke.Type != "skill" {
		c.State = statePass
		c.Detail = "not a skill-invoked agent"
		return c
	}
	repo := ownerRepoOf(a)
	if repo == "" {
		c.State = stateUnknown
		c.Detail = "workspace is not a GitHub owner/repo coordinate"
		return c
	}

	workspacePath := ".claude/skills/" + a.Invoke.Name + "/SKILL.md"
	pluginPath := "skills/" + a.Invoke.Name + "/SKILL.md"

	_, wsCode, _ := githubGET(ctx, "/repos/"+repo+"/contents/"+workspacePath)
	plCode := 0
	if wsCode != 200 {
		_, plCode, _ = githubGET(ctx, "/repos/"+sharedSkillsPlugin+"/contents/"+pluginPath)
	}
	return skillVerdict(a.Invoke.Name, repo, wsCode, plCode)
}

// skillVerdict is the decision, split from the fetching so it can be tested.
//
// The distinction it exists to keep: a definite 404 from BOTH places is a real
// miss, and anything else — a 403 on a private repo, a 5xx, a rate limit — is
// UNKNOWN. Reading "I could not look" as "it is not there" is the failure mode
// this whole command was written against.
func skillVerdict(skill, repo string, wsCode, plCode int) agentCheck {
	c := agentCheck{Name: "skill present"}
	workspacePath := ".claude/skills/" + skill + "/SKILL.md"
	pluginPath := "skills/" + skill + "/SKILL.md"

	switch {
	case wsCode == 200:
		c.State = statePass
		c.Detail = workspacePath + " exists in " + repo
	case plCode == 200:
		c.State = statePass
		c.Detail = pluginPath + " exists in the shared plugin " + sharedSkillsPlugin + " (pre-cloned into every executor image)"
	case wsCode == 404 && plCode == 404:
		c.State = stateFail
		c.Detail = fmt.Sprintf("%s is in neither %s nor the shared plugin %s — the dispatch will have no skill to run",
			skill, repo, sharedSkillsPlugin)
		c.Fix = "commit the skill to the workspace repo, or to " + sharedSkillsPlugin + "/skills/ if every agent should have it"
	default:
		c.State = stateUnknown
		c.Detail = fmt.Sprintf("cannot check the skill (workspace HTTP %d, shared plugin HTTP %d)", wsCode, plCode)
	}
	return c
}

// checkRolled reports whether the coordinator has restarted since the config
// was last written. A gcsfuse mount updates the FILE immediately; the registry
// is built once at startup, so routing follows the config loaded then.
func checkRolled(ctx context.Context) agentCheck {
	c := agentCheck{Name: "coordinator current"}
	project, region, service := coordinatorService()
	out, err := exec.CommandContext(ctx, "gcloud", "run", "revisions", "list",
		"--service", service, "--project", project, "--region", region,
		"--limit", "1", "--format=value(metadata.creationTimestamp)").CombinedOutput()
	if err != nil {
		c.State = stateUnknown
		c.Detail = "cannot read the coordinator's current revision"
		return c
	}
	revAt, perr := time.Parse(time.RFC3339, firstLine(strings.TrimSpace(string(out))))
	if perr != nil {
		c.State = stateUnknown
		c.Detail = "cannot parse the revision timestamp"
		return c
	}
	cfgAt, cerr := configLastModified(ctx)
	if cerr != nil {
		c.State = stateUnknown
		c.Detail = "cannot read the config's last-modified time"
		return c
	}
	if revAt.After(cfgAt) {
		c.State = statePass
		c.Detail = fmt.Sprintf("rolled %s after the last config write", revAt.Sub(cfgAt).Round(time.Second))
		return c
	}
	c.State = stateFail
	c.Detail = fmt.Sprintf("config written %s AFTER the running revision — the coordinator is using the previous registry", cfgAt.Sub(revAt).Round(time.Second))
	c.Fix = "ailang coordinator config roll"
	return c
}

// configLastModified reads the config object's update time.
func configLastModified(ctx context.Context) (time.Time, error) {
	bucket, object := configLocation()
	out, err := exec.CommandContext(ctx, "gsutil", "stat", fmt.Sprintf("gs://%s/%s", bucket, object)).CombinedOutput()
	if err != nil {
		return time.Time{}, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "Update time:") {
			v := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			for _, layout := range []string{"Mon, 02 Jan 2006 15:04:05 GMT", time.RFC1123} {
				if t, err := time.Parse(layout, v); err == nil {
					return t, nil
				}
			}
		}
	}
	return time.Time{}, fmt.Errorf("no update time in gsutil stat output")
}

// ownerRepoOf normalises an agent's workspace to owner/repo.
func ownerRepoOf(a *coordinator.AgentConfig) string {
	return gitHubOwnerRepoFromURL(strings.TrimSpace(a.Workspace))
}

// githubGET is a read-only API call using the fleet token.
func githubGET(ctx context.Context, path string) ([]byte, int, error) {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		if out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output(); err == nil {
			token = strings.TrimSpace(string(out))
		}
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com"+path, nil)
	if err != nil {
		return nil, 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	var buf [1 << 18]byte
	n, _ := resp.Body.Read(buf[:])
	return buf[:n], resp.StatusCode, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// checkUnknownKeys reports config keys the registry does not read.
//
// This is the general form of the specific bug that prompted it:
// `push_branch: dev` was written into an agent entry and looks decisive, but
// AgentConfig has no such field — direct push is actually skip_approval +
// merge_branch (daemon_tasks_exec.go:211). YAML silently drops unknown keys, so
// the entry read as configured and the key did nothing. It happened to be
// harmless there; a mistyped `auto_merge` or `ssh_key_secret` would not be.
//
// yaml.v3's KnownFields turns "silently ignored" into "named".
func checkUnknownKeys(repoConfig string) agentCheck {
	c := agentCheck{Name: "no dead config keys"}
	if repoConfig == "" {
		c.State = stateUnknown
		c.Detail = "needs --repo-config to parse the source YAML strictly"
		return c
	}
	data, err := os.ReadFile(repoConfig)
	if err != nil {
		c.State = stateUnknown
		c.Detail = fmt.Sprintf("cannot read %s", repoConfig)
		return c
	}
	unknown, err := coordinator.UnknownConfigKeys(data)
	if err != nil {
		c.State = stateUnknown
		c.Detail = "strict parse failed: " + firstLine(err.Error())
		return c
	}
	if len(unknown) == 0 {
		c.State = statePass
		c.Detail = "every key in the config is read by the registry"
		return c
	}
	c.State = stateFail
	shown := unknown
	if len(shown) > 4 {
		shown = append(shown[:4], fmt.Sprintf("…+%d", len(unknown)-4))
	}
	c.Detail = "keys nothing reads: " + strings.Join(shown, ", ")
	c.Fix = "remove them, or add the field — a key that looks decisive and is ignored is worse than an absent one"
	return c
}
