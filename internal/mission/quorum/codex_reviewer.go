package quorum

// The OpenAI quorum seat over the ChatGPT SUBSCRIPTION (`codex exec`), the lane the
// mission roles already use (D-FLEET-20 A, Mark, attended 2026-11-16).
//
// The seat used to be the models.yml row gpt6-1-sol, which calls the metered OpenAI
// API. That org has no credits ("You have no credits remaining"), so every quorum that
// seated it recorded it absent and an OpenAI-free quorum filled in from the reserve.
// `codex exec --model gpt-6.1-sol` on the subscription works and costs nothing metered.
// Never route an OpenAI model through OpenRouter instead (test-enforced in modelreg).
//
// Same prompt and schema as every other seat. The review runs in an empty temporary
// directory with a read-only sandbox and no persisted session: a reviewer reads the doc
// it is handed and never touches the shared tree.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/codex"
)

// CodexReviewerPrefix marks a roster id that runs through the subscription codex
// lane. It is the mission's own lane spelling ("codex:gpt-6.1-sol"), so VendorOf,
// ModelAllowed and --author benching already understand it.
const CodexReviewerPrefix = "codex:"

// CodexReviewerModel is the model the OpenAI seat runs.
// gpt-6.1-sol -> gpt-6.2-sol 2026-11-17 (Mark, attended); needs codex-cli >= 0.169.0.
// gpt-6.2-sol -> gpt-6.2-astra 2026-11-25 (Mark, attended: pool "codex astra, gemini
// carbon, fable, glm 5-3, kimi"). Astra is admitted for the quorum seat only
// (QuorumOpenAIModels); the D-FLEET-12 mission-role allowlist is unchanged.
const CodexReviewerModel = "gpt-6.2-astra"

// CodexReviewerID is the roster id of the OpenAI seat.
const CodexReviewerID = CodexReviewerPrefix + CodexReviewerModel

// codexReviewTimeout bounds one review. A reasoning model's text verdict on a long
// doc takes minutes; a hung CLI must cost minutes, not the iteration.
const codexReviewTimeout = 10 * time.Minute

// codexReviewCall runs one review turn and returns the `codex exec --json` event
// stream. A variable so tests never start a real codex process.
var codexReviewCall = callCodexReview

func callCodexReview(ctx context.Context, model, prompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, codexReviewTimeout)
	defer cancel()
	dir, err := os.MkdirTemp("", "ailang-quorum-codex-*")
	if err != nil {
		return nil, fmt.Errorf("codex review workspace: %w", err)
	}
	defer os.RemoveAll(dir)
	var out []byte
	switch runtime := config.CodexRuntime(); runtime {
	case "daemon":
		// Missions talk to the one managed daemon for CODEX_HOME; RunDaemonExec refuses
		// a non-subscription auth.json and never starts a private auth manager.
		env, berr := executor.BuildEnvironment(executor.EnvironmentOptions{
			Task: &executor.Task{Workspace: dir, Model: model}, Executor: "codex", Model: model, Context: ctx})
		if berr != nil {
			return nil, fmt.Errorf("codex review environment: %w", berr)
		}
		var buf bytes.Buffer
		err = codex.RunDaemonExec(ctx, "codex", codexDaemonReviewOptions(model, dir, prompt, env), &buf)
		out = buf.Bytes()
	case "", "cli":
		if aerr := requireSubscriptionAuth(); aerr != nil {
			return nil, aerr
		}
		out, err = codexReviewCmd(ctx, model, dir, prompt).Output()
	default:
		return nil, fmt.Errorf("unsupported AILANG_CODEX_RUNTIME %q", runtime)
	}
	if err != nil {
		if ctx.Err() != nil {
			return out, fmt.Errorf("codex exec: no reply within %s", codexReviewTimeout)
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return out, fmt.Errorf("codex exec: %w: %.300s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return out, fmt.Errorf("codex exec: %w", err)
	}
	return out, nil
}

// requireSubscriptionAuth refuses an api-key auth.json, which would bill the metered
// API — the exact failure this seat moved away from.
func requireSubscriptionAuth() error {
	path, err := codex.InstalledAuthPath()
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("codex subscription login missing or unreadable (%s): %w", path, err)
	}
	if _, err := codex.ParseSubscriptionAuth(raw); err != nil {
		return err
	}
	return nil
}

// codexReviewCmd builds the CLI-runtime command. Split out so a test can pin the call
// site: read-only sandbox, no persisted session, no OPENAI_API_KEY (subscription only),
// spawn overrides off (D-FLEET-12), and the prompt on stdin, never argv.
func codexReviewCmd(ctx context.Context, model, dir, prompt string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "codex", "exec",
		"--json",
		"--model", model,
		"--sandbox", "read-only",
		"--skip-git-repo-check",
		"--ephemeral",
		"-c", SpawnOverrideKey+"=false",
		"-")
	cmd.Dir = dir
	cmd.Env = withoutEnv(os.Environ(), "OPENAI_API_KEY")
	cmd.Stdin = strings.NewReader(prompt)
	cmd.WaitDelay = 5 * time.Second
	return cmd
}

// codexDaemonReviewOptions is the daemon-runtime equivalent of codexReviewCmd.
func codexDaemonReviewOptions(model, dir, prompt string, env []string) codex.DaemonExecOptions {
	return codex.DaemonExecOptions{
		Model: model, Directive: prompt, Workspace: dir,
		Sandbox: "read-only", ApprovalPolicy: "never", Ephemeral: true,
		Config: map[string]any{SpawnOverrideKey: false}, Env: withoutEnv(env, "OPENAI_API_KEY"),
	}
}

// codexStream is what a review needs from the `codex exec --json` events: the last
// agent message, the turn's token usage, and any error the stream reported.
type codexStream struct {
	message             string
	tokensIn, tokensOut int
	errMsg              string
	sawMessage          bool
}

func parseCodexStream(raw []byte) codexStream {
	var s codexStream
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var ev struct {
			Type string `json:"type"`
			Item struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
			Usage *struct {
				Input  int `json:"input_tokens"`
				Output int `json:"output_tokens"`
			} `json:"usage"`
			Message string `json:"message"`
			Error   struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(line, &ev) != nil {
			continue
		}
		switch ev.Type {
		case "item.completed":
			if ev.Item.Type == "agent_message" {
				s.message, s.sawMessage = ev.Item.Text, true
			}
		case "turn.completed":
			if ev.Usage != nil {
				// cached_input_tokens is a SUBSET of input_tokens (Responses API).
				s.tokensIn += ev.Usage.Input
				s.tokensOut += ev.Usage.Output
			}
		case "error":
			s.errMsg = ev.Message
		case "turn.failed":
			s.errMsg = ev.Error.Message
		}
	}
	return s
}

// RunCodexSubscriptionReviewer reviews the doc as the OpenAI seat. Like RunReviewer it
// never returns nil: a failure is a named absence. CostUSD stays 0 — the call bills the
// subscription, not the API. Tokens come from the stream's own usage block.
func RunCodexSubscriptionReviewer(model, docPath, docBody string) *ReviewerOutcome {
	out := &ReviewerOutcome{Model: CodexReviewerPrefix + model}
	prompt := systemPrompt + "\n\n" + BuildPrompt(docPath, docBody) +
		"\n\nRespond with ONLY a JSON object matching this schema, no prose, no code fence:\n" + reviewSchema
	raw, err := codexReviewCall(context.Background(), model, prompt)
	s := parseCodexStream(raw)
	out.TokensIn, out.TokensOut = s.tokensIn, s.tokensOut
	if err != nil || s.errMsg != "" {
		out.AbsentReason = ReasonUnreachable
		switch {
		case err != nil && s.errMsg != "":
			out.Err = fmt.Sprintf("%v; stream error: %.300s", err, s.errMsg)
		case err != nil:
			out.Err = err.Error()
		default:
			out.Err = fmt.Sprintf("codex exec reported an error: %.300s", s.errMsg)
		}
		return out
	}
	if !s.sawMessage {
		out.AbsentReason = ReasonInvalid
		out.Err = fmt.Sprintf("codex exec returned no agent message (raw: %.200q)", raw)
		return out
	}
	result, perr := ParseReviewResult(s.message)
	if perr != nil {
		recordInvalid(out, s.message)
		out.Err = perr.Error()
		return out
	}
	out.Present = true
	out.Result = result
	return out
}
