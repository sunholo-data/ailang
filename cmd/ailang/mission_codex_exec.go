package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/executor/codex"
)

// missionCodexExec uses one managed auth owner. It deliberately never falls back
// to codex exec, which would start another auth manager against the same home.
func missionCodexExec(args []string) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			fmt.Println("ailang mission codex-exec --model MODEL [-C DIR] [-c KEY=TOML] [--json] [--sandbox MODE] [--dangerously-bypass-approvals-and-sandbox] PROMPT\nUses the managed daemon for CODEX_HOME; never starts an independent auth runtime.")
			return nil
		}
	}
	opts, jsonOutput, err := parseMissionCodexExecArgs(args)
	if err != nil {
		return err
	}
	if opts.Directive == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		opts.Directive = string(b)
	}
	opts.Env, err = executor.BuildEnvironment(executor.EnvironmentOptions{Task: &executor.Task{Workspace: opts.Workspace, Model: opts.Model}, Executor: "codex", Model: opts.Model, Context: context.Background()})
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var out io.Writer = os.Stdout
	if !jsonOutput {
		out = &codexTextWriter{out: os.Stdout}
	}
	return codex.RunDaemonExec(ctx, "codex", opts, out)
}

func parseMissionCodexExecArgs(args []string) (codex.DaemonExecOptions, bool, error) {
	opts := codex.DaemonExecOptions{Sandbox: "read-only", ApprovalPolicy: "never", Config: map[string]any{}}
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		value := func() (string, error) {
			i++
			if i >= len(args) {
				return "", fmt.Errorf("%s requires a value", arg)
			}
			return args[i], nil
		}
		switch arg {
		case "--json":
			jsonOutput = true
		case "--skip-git-repo-check": // app-server thread/start has no git-repo gate
		case "--dangerously-bypass-approvals-and-sandbox":
			opts.Sandbox = "danger-full-access"
			opts.ApprovalPolicy = "never"
		case "--model", "-m":
			v, e := value()
			if e != nil {
				return opts, false, e
			}
			opts.Model = v
		case "--cd", "-C":
			v, e := value()
			if e != nil {
				return opts, false, e
			}
			opts.Workspace = v
		case "--sandbox", "-s":
			v, e := value()
			if e != nil {
				return opts, false, e
			}
			opts.Sandbox = v
		case "--ask-for-approval", "-a":
			v, e := value()
			if e != nil {
				return opts, false, e
			}
			opts.ApprovalPolicy = v
		case "--ephemeral":
			opts.Ephemeral = true
		case "--config", "-c":
			v, e := value()
			if e != nil {
				return opts, false, e
			}
			key, raw, ok := strings.Cut(v, "=")
			if !ok || strings.TrimSpace(key) == "" {
				return opts, false, fmt.Errorf("config requires KEY=TOML")
			}
			var decoded map[string]any
			if _, e := toml.Decode("value="+raw, &decoded); e != nil {
				return opts, false, fmt.Errorf("invalid TOML for config key %s", key)
			}
			opts.Config[key] = decoded["value"]
			if key == "model_reasoning_effort" {
				effort, ok := decoded["value"].(string)
				if !ok {
					return opts, false, fmt.Errorf("reasoning effort must be a string")
				}
				opts.Effort = effort
			}
		case "--":
			if i+2 != len(args) {
				return opts, false, fmt.Errorf("exactly one prompt is required")
			}
			opts.Directive = args[i+1]
			i = len(args)
		default:
			if strings.HasPrefix(arg, "-") && arg != "-" {
				return opts, false, fmt.Errorf("unsupported daemon exec option %q", arg)
			}
			if opts.Directive != "" {
				return opts, false, fmt.Errorf("exactly one prompt is required")
			}
			opts.Directive = arg
		}
	}
	if opts.Model == "" || opts.Directive == "" {
		return opts, false, fmt.Errorf("--model and a prompt are required")
	}
	switch opts.Sandbox {
	case "read-only", "workspace-write", "danger-full-access":
	default:
		return opts, false, fmt.Errorf("unsupported sandbox mode %q", opts.Sandbox)
	}
	switch opts.ApprovalPolicy {
	case "never", "on-request", "untrusted":
	default:
		return opts, false, fmt.Errorf("unsupported approval policy %q", opts.ApprovalPolicy)
	}
	return opts, jsonOutput, nil
}

// codexTextWriter renders completed message/tool output for existing driver logs.
// Partial writes are buffered; malformed JSON is an error rather than silent loss.
type codexTextWriter struct {
	out     io.Writer
	pending []byte
}

func (w *codexTextWriter) Write(p []byte) (int, error) {
	w.pending = append(w.pending, p...)
	for {
		end := bytes.IndexByte(w.pending, '\n')
		if end < 0 {
			break
		}
		line := w.pending[:end]
		w.pending = w.pending[end+1:]
		var event struct {
			Type string `json:"type"`
			Item struct {
				Type   string `json:"type"`
				Text   string `json:"text"`
				Output string `json:"aggregated_output"`
			} `json:"item"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(line, &event); err != nil {
			return 0, fmt.Errorf("invalid daemon execution event: %w", err)
		}
		text := ""
		if event.Type == "item.completed" {
			switch event.Item.Type {
			case "agent_message":
				text = event.Item.Text
			case "command_execution":
				text = event.Item.Output
			}
		}
		if event.Type == "error" {
			text = event.Message
		}
		if text != "" {
			if _, err := fmt.Fprintln(w.out, text); err != nil {
				return 0, err
			}
		}
	}
	return len(p), nil
}
