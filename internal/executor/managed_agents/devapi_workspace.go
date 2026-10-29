package managed_agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/gitexec"
)

// Workspace round trip for gemini_agents.
//
// UP: a git workspace is mounted as a `repository` source cloned from its
// GitHub origin, checked out at the merge-base with the remote, and everything
// local on top of that (unpushed commits, uncommitted edits, untracked files)
// travels as ONE inline patch built with a throwaway index — the caller's real
// index is never touched. A non-git workspace (eval benchmarks) travels as
// inline files.
//
// DOWN: the agent is told to finish by diffing against the recorded start
// commit into /workspace/.ailang/out.patch; the executor downloads it through
// the files API and `git apply`s it to the local workspace.

const (
	sandboxRepo     = "/workspace/repo"
	sandboxMeta     = "/workspace/.ailang"
	localPatchPath  = sandboxMeta + "/local.patch"
	startSHAPath    = sandboxMeta + "/start_sha"
	outPatchPath    = sandboxMeta + "/out.patch"
	maxInlineFile   = 1 << 20 // API limit: 1 MB per inline file
	maxInlineTotal  = 2 << 20 // API limit: 2 MB total inline
	maxPatchBytes   = 32 << 20
	sandboxGitIdent = "-c user.name=ailang -c user.email=ailang@localhost"
)

// workspacePlan is what goes up to the sandbox for one task.
type workspacePlan struct {
	RepoURL    string // https clone URL; "" for a non-git workspace
	BaseSHA    string // merge-base with the remote; "" for a non-git workspace
	LocalPatch string // binary diff base..working tree incl. untracked; may be ""
	Inline     []inlineSource
}

type inlineSource struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Target  string `json:"target"`
}

// planWorkspace inspects dir and decides how to mount it.
func planWorkspace(ctx context.Context, dir string) (*workspacePlan, error) {
	if _, err := gitOut(ctx, dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return planInlineWorkspace(dir)
	}
	origin, err := gitOut(ctx, dir, "remote", "get-url", "origin")
	if err != nil {
		return nil, fmt.Errorf("%s: workspace %s is a git repo without an 'origin' remote to mount from", DevAPIExecutorName, dir)
	}
	repoURL := httpsCloneURL(origin)
	if err := executor.ValidateCloneURL(repoURL); err != nil {
		return nil, fmt.Errorf("%s: origin %q is not mountable: %w", DevAPIExecutorName, origin, err)
	}
	base, err := remoteBase(ctx, dir)
	if err != nil {
		return nil, err
	}
	patch, err := localDelta(ctx, dir, base)
	if err != nil {
		return nil, err
	}
	if len(patch) > maxInlineFile {
		return nil, fmt.Errorf("%s: local changes on top of the pushed base %s are %d bytes; the inline limit is %d — push or shrink them", DevAPIExecutorName, base[:12], len(patch), maxInlineFile)
	}
	p := &workspacePlan{RepoURL: repoURL, BaseSHA: base, LocalPatch: patch}
	if patch != "" {
		p.Inline = []inlineSource{{Type: "inline", Content: patch, Target: localPatchPath}}
	}
	return p, nil
}

// httpsCloneURL rewrites git@host:owner/repo(.git) to https form; https stays.
func httpsCloneURL(origin string) string {
	origin = strings.TrimSpace(origin)
	if strings.HasPrefix(origin, "git@") {
		if host, path, ok := strings.Cut(strings.TrimPrefix(origin, "git@"), ":"); ok {
			return "https://" + host + "/" + path
		}
	}
	return origin
}

// remoteBase finds the newest commit of HEAD's history that exists on the
// remote: merge-base(HEAD, upstream), falling back to origin/HEAD. A missing
// remote ref is a named error — mounting an unpushed commit would 404 in the
// sandbox, after the money is spent.
func remoteBase(ctx context.Context, dir string) (string, error) {
	for _, ref := range []string{"@{upstream}", "origin/HEAD", "origin/dev", "origin/main"} {
		if _, err := gitOut(ctx, dir, "rev-parse", "--verify", "--quiet", ref); err != nil {
			continue
		}
		if mb, err := gitOut(ctx, dir, "merge-base", "HEAD", ref); err == nil && executor.IsValidSHA40(mb) {
			return mb, nil
		}
	}
	return "", fmt.Errorf("%s: no remote-tracking ref shares history with HEAD in the workspace (tried @{upstream}, origin/HEAD, origin/dev, origin/main) — fetch first", DevAPIExecutorName)
}

// localDelta is the binary diff from base to the working tree INCLUDING
// untracked, non-ignored files, built in a throwaway index.
func localDelta(ctx context.Context, dir, base string) (string, error) {
	idx, err := os.CreateTemp("", "gemini-agents-index-*")
	if err != nil {
		return "", err
	}
	idxPath := idx.Name()
	idx.Close()
	defer os.Remove(idxPath)
	env := []string{"GIT_INDEX_FILE=" + idxPath}
	if _, err := gitOutEnv(ctx, dir, env, "read-tree", "HEAD"); err != nil {
		return "", fmt.Errorf("%s: read-tree: %w", DevAPIExecutorName, err)
	}
	if _, err := gitOutEnv(ctx, dir, env, "add", "-A"); err != nil {
		return "", fmt.Errorf("%s: stage working tree in temp index: %w", DevAPIExecutorName, err)
	}
	out, err := gitRawEnv(ctx, dir, env, "diff", "--cached", "--binary", base)
	if err != nil {
		return "", fmt.Errorf("%s: diff against base: %w", DevAPIExecutorName, err)
	}
	return out, nil
}

// planInlineWorkspace uploads a small non-git workspace file by file.
func planInlineWorkspace(dir string) (*workspacePlan, error) {
	p := &workspacePlan{}
	total := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		if len(b) > maxInlineFile {
			return fmt.Errorf("%s: %s is %d bytes, over the 1 MB inline limit", DevAPIExecutorName, rel, len(b))
		}
		total += len(b)
		if total > maxInlineTotal {
			return fmt.Errorf("%s: non-git workspace exceeds the 2 MB inline total; make it a git repo with a GitHub origin", DevAPIExecutorName)
		}
		p.Inline = append(p.Inline, inlineSource{Type: "inline", Content: string(b), Target: sandboxRepo + "/" + filepath.ToSlash(rel)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// environment renders the interaction environment for the plan. Egress is
// always on: the sandbox must fetch the base SHA (public repo, no secrets).
func (p *workspacePlan) environment() (json.RawMessage, error) {
	type source struct {
		Type    string `json:"type"`
		Source  string `json:"source,omitempty"`
		Content string `json:"content,omitempty"`
		Target  string `json:"target"`
	}
	var srcs []source
	if p.RepoURL != "" {
		srcs = append(srcs, source{Type: "repository", Source: p.RepoURL, Target: sandboxRepo})
	}
	for _, in := range p.Inline {
		srcs = append(srcs, source{Type: in.Type, Content: in.Content, Target: in.Target})
	}
	env := map[string]any{
		"type":    "remote",
		"network": map[string]any{"allowlist": []map[string]string{{"domain": "*"}}},
	}
	if len(srcs) > 0 {
		env["sources"] = srcs
	}
	return json.Marshal(env)
}

// setupScript prepares the sandbox repo and records the start commit.
func (p *workspacePlan) setupScript() string {
	var b strings.Builder
	fmt.Fprintf(&b, "mkdir -p %s %s && cd %s", sandboxMeta, sandboxRepo, sandboxRepo)
	if p.RepoURL != "" {
		fmt.Fprintf(&b, " && (git remote get-url origin >/dev/null 2>&1 || git remote add origin %s)", p.RepoURL)
		fmt.Fprintf(&b, " && git fetch -q --depth 50 origin %s && git checkout -q --detach %s", p.BaseSHA, p.BaseSHA)
		if p.LocalPatch != "" {
			fmt.Fprintf(&b, " && git apply --binary %s && git add -A && git %s commit -q -m 'local state'", localPatchPath, sandboxGitIdent)
		}
	} else {
		fmt.Fprintf(&b, " && git init -q && git add -A && git %s commit -q --allow-empty -m base", sandboxGitIdent)
	}
	fmt.Fprintf(&b, " && git rev-parse HEAD > %s && echo AILANG_SETUP_OK", startSHAPath)
	return b.String()
}

// finishScript writes the agent's changes as one binary patch.
func finishScript() string {
	return fmt.Sprintf("cd %s && git add -A && git diff --cached --binary \"$(cat %s)\" > %s && echo AILANG_PATCH_WRITTEN",
		sandboxRepo, startSHAPath, outPatchPath)
}

// wrapDirective frames the caller's directive with the setup and finish steps.
func (p *workspacePlan) wrapDirective(directive string) string {
	return "Your working copy is " + sandboxRepo + ". FIRST run this exact command and stop if it fails:\n\n" +
		p.setupScript() + "\n\nThen do the task below, working in " + sandboxRepo + ".\n\n" +
		"--- TASK ---\n" + directive + "\n--- END TASK ---\n\n" +
		"LAST, after all edits, run this exact command (it exports your changes):\n\n" + finishScript() + "\n"
}

// applyPatch applies a downloaded patch to the local workspace.
func applyPatch(ctx context.Context, dir string, patch []byte) error {
	cmd := gitexec.CommandContext(ctx, "apply", "--binary", "--whitespace=nowarn", "-")
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: git apply: %v: %s", DevAPIExecutorName, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func gitOut(ctx context.Context, dir string, args ...string) (string, error) {
	s, err := gitOutEnv(ctx, dir, nil, args...)
	return s, err
}

func gitOutEnv(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	s, err := gitRawEnv(ctx, dir, env, args...)
	return strings.TrimSpace(s), err
}

func gitRawEnv(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	cmd := gitexec.CommandContext(ctx, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// finishDevAPI brings the agent's edits back to the local workspace and deletes
// the sandbox. It mutates res/pd: a missing setup marker, an unappliable patch
// or a download failure turns a "completed" interaction into a failed task —
// a run whose edits never landed is not a success.
func (e *Executor) finishDevAPI(ctx context.Context, task *executor.Task, plan *workspacePlan,
	body *interactionRequest, envID string, res *executor.Result, pd map[string]any) {
	defer func() {
		// Fresh context: the task context may already be spent.
		dctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := e.deleteEnv(dctx, envID); err != nil {
			pd["gemini_agents_env_delete_error"] = err.Error()
		}
	}()
	if plan == nil || !res.Success {
		return
	}
	fail := func(err error) {
		res.Success = false
		res.FinishReason = "error"
		res.Error = err.Error()
	}
	if _, err := e.getEnvFile(ctx, envID, startSHAPath); err != nil {
		if errors.Is(err, errFileNotFound) {
			err = fmt.Errorf("%s: the sandbox setup command never ran (%s missing) — edits cannot be exported", DevAPIExecutorName, startSHAPath)
		}
		fail(err)
		return
	}
	patch, err := e.getEnvFile(ctx, envID, outPatchPath)
	if errors.Is(err, errFileNotFound) {
		// The agent skipped the export step: one short follow-up turn in the
		// same sandbox runs it. Its cost is added to the result.
		pd["gemini_agents_export_followup"] = true
		if ferr := e.exportFollowUp(ctx, task, body, envID, res); ferr != nil {
			fail(ferr)
			return
		}
		patch, err = e.getEnvFile(ctx, envID, outPatchPath)
	}
	if err != nil {
		fail(err)
		return
	}
	if len(patch) > maxPatchBytes {
		fail(fmt.Errorf("%s: exported patch exceeds %d bytes", DevAPIExecutorName, maxPatchBytes))
		return
	}
	pd["gemini_agents_patch_bytes"] = len(patch)
	if len(bytes.TrimSpace(patch)) == 0 {
		return // the agent changed nothing
	}
	if err := applyPatch(ctx, task.Workspace, patch); err != nil {
		if f, ferr := os.CreateTemp("", "gemini-agents-*.patch"); ferr == nil {
			_, _ = f.Write(patch)
			f.Close()
			pd["gemini_agents_unapplied_patch"] = f.Name()
		}
		fail(err)
	}
}

// exportFollowUp runs finishScript in the existing sandbox.
func (e *Executor) exportFollowUp(ctx context.Context, task *executor.Task, prev *interactionRequest,
	envID string, res *executor.Result) error {
	envRaw, _ := json.Marshal(envID) // reuse: the environment is a bare id string
	body := *prev
	body.Environment = envRaw
	body.SystemInstruction = ""
	body.Input = []inputBlock{{Type: "user_input", Content: []contentBlock{{Type: "text",
		Text: "Run this exact command and report its output, nothing else:\n\n" + finishScript()}}}}
	rd, err := e.send(ctx, "", "", &body)
	if err != nil {
		return fmt.Errorf("%s: export follow-up: %w", DevAPIExecutorName, err)
	}
	defer rd.Close()
	st := &streamState{}
	_ = parseSSE(rd, func(ev sseEvent) error { return foldEvent(st, ev) })
	if st.Status != "completed" {
		return fmt.Errorf("%s: export follow-up ended with status %q", DevAPIExecutorName, st.Status)
	}
	fresh := st.Usage.FreshInputTokens()
	res.InputTokens += fresh
	res.OutputTokens += st.Usage.TotalOutputTokens
	res.ReasonTokens += st.Usage.TotalThoughtTokens
	res.CacheReadInputTokens += st.Usage.TotalCachedTokens
	res.CostUSD += executor.ResolveCostModel(task, e.CostModel()).CalculateCost(executor.TokenUsage{
		InputTokens:          fresh,
		OutputTokens:         st.Usage.TotalOutputTokens + st.Usage.TotalThoughtTokens,
		CacheReadInputTokens: st.Usage.TotalCachedTokens,
	})
	return nil
}
