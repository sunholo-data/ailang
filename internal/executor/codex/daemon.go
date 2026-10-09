package codex

// The exec CLI embeds its own app-server in 0.162.0. Missions instead connect
// to one explicitly managed daemon; proxy clients never load or refresh auth.
import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrProcessTerminationUnconfirmed prevents credential lease release when an owner cannot be joined.
var ErrProcessTerminationUnconfirmed = errors.New("Codex refresh owner termination unconfirmed")

type DaemonExecOptions struct {
	Model, Directive, Workspace, Sandbox, ApprovalPolicy, Effort string
	Config                                                       map[string]any
	Env                                                          []string
	OutputSchema                                                 map[string]any
	Ephemeral                                                    bool
}

// RunDaemonExec emits the existing codex exec JSONL contract from one daemon
// thread over the daemon control socket. The daemon must already be running
// for CODEX_HOME. No private server
// fallback is permitted. Cancellation interrupts only this client's turn.
func RunDaemonExec(ctx context.Context, codexPath string, opts DaemonExecOptions, out io.Writer) error {
	home, err := daemonHome(opts.Env)
	if err != nil {
		return err
	}
	auth, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		return fmt.Errorf("mission Codex subscription login is missing or unreadable: %w", err)
	}
	if _, err := ParseSubscriptionAuth(auth); err != nil {
		return fmt.Errorf("mission Codex subscription login is invalid: %w", err)
	}
	opts.Config = daemonIsolatedConfig(opts.Config, opts.Env)
	conn, err := dialDaemon(ctx, opts.Env)
	if err != nil {
		return err
	}
	defer conn.Close()
	stream := &daemonWebSocketStream{conn: conn}
	return runDaemonProtocol(ctx, stream, stream, opts, out)
}

type daemonMessage struct {
	ID     *int            `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func runDaemonProtocol(ctx context.Context, r io.Reader, w io.Writer, opts DaemonExecOptions, out io.Writer) (errOut error) {
	if opts.Model == "" {
		return errors.New("Codex daemon requires an explicit model")
	}
	if opts.Sandbox == "" {
		opts.Sandbox = "danger-full-access"
	}
	switch opts.Sandbox {
	case "read-only", "workspace-write", "danger-full-access":
	default:
		return errors.New("unsupported Codex daemon sandbox")
	}
	if opts.ApprovalPolicy == "" {
		opts.ApprovalPolicy = "never"
	}
	switch opts.ApprovalPolicy {
	case "never", "on-request", "untrusted":
	default:
		return errors.New("unsupported Codex daemon approval policy")
	}
	enc := json.NewEncoder(w)
	output := json.NewEncoder(out)
	send := func(id int, method string, params any) error {
		return enc.Encode(map[string]any{"id": id, "method": method, "params": params})
	}
	emit := func(v map[string]any) error { return output.Encode(v) }
	type received struct {
		message daemonMessage
		err     error
	}
	messages := make(chan received, 64)
	readerDone := make(chan struct{})
	defer close(readerDone)
	// Closing the proxy's pipes at return releases this reader, including cancel.
	go func() {
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64<<10), 16<<20)
		for sc.Scan() {
			var m daemonMessage
			if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
				messages <- received{err: errors.New("malformed Codex daemon protocol message")}
				return
			}
			select {
			case messages <- received{message: m}:
			case <-readerDone:
				return
			}
		}
		err := sc.Err()
		if err == nil {
			err = io.EOF
		}
		select {
		case messages <- received{err: err}:
		case <-readerDone:
		}
	}()
	if err := send(1, "initialize", map[string]any{"clientInfo": map[string]any{"name": "ailang-mission-exec", "version": "1"}, "capabilities": map[string]any{"experimentalApi": false}}); err != nil {
		return err
	}
	var threadID, turnID string
	terminal := false
	turnRequested := false
	defer func() {
		if errOut != nil && turnRequested && turnID == "" && !terminal {
			errOut = fmt.Errorf("%w: turn/start outcome unknown: %v", ErrProcessTerminationUnconfirmed, errOut)
			return
		}
		if errOut != nil && turnID != "" && !terminal {
			_ = send(99, "turn/interrupt", map[string]any{"threadId": threadID, "turnId": turnID})
			until := time.After(5 * time.Second)
			for {
				select {
				case msg := <-messages:
					if msg.err != nil {
						errOut = fmt.Errorf("%w: %v", ErrProcessTerminationUnconfirmed, errOut)
						return
					}
					if msg.message.Method == "turn/completed" {
						var p struct {
							ThreadID string `json:"threadId"`
							Turn     struct {
								ID string `json:"id"`
							} `json:"turn"`
						}
						if json.Unmarshal(msg.message.Params, &p) == nil && p.ThreadID == threadID && p.Turn.ID == turnID {
							return
						}
					}
				case <-until:
					errOut = fmt.Errorf("%w: %v", ErrProcessTerminationUnconfirmed, errOut)
					return
				}
			}
		}
	}()
	usage := map[string]any{"input_tokens": 0, "output_tokens": 0, "cached_input_tokens": 0}
	var canceled error
	cancelC := ctx.Done()
	var cancelDeadline <-chan time.Time
	started := false
	for {
		select {
		case <-cancelC:
			canceled = ctx.Err()
			cancelC = nil
			// Await any in-flight creation response, then interrupt the assigned turn.
			cancelDeadline = time.After(5 * time.Second)
			if threadID != "" && turnID != "" {
				if err := send(99, "turn/interrupt", map[string]any{"threadId": threadID, "turnId": turnID}); err != nil {
					return fmt.Errorf("Codex daemon cancellation delivery failed: %w", err)
				}
			}
		case <-cancelDeadline:
			return fmt.Errorf("Codex daemon turn cancellation unconfirmed; operator recovery required: %w", ErrProcessTerminationUnconfirmed)
		case msg := <-messages:
			if msg.err != nil {
				return fmt.Errorf("Codex daemon proxy disconnected: %w", msg.err)
			}
			m := msg.message
			if m.Error != nil {
				if m.ID != nil && *m.ID == 3 {
					turnRequested = false
				}
				return fmt.Errorf("Codex daemon request failed: %s", m.Error.Message)
			}
			if m.ID != nil {
				if m.Method != "" {
					if err := enc.Encode(map[string]any{"id": *m.ID, "error": map[string]any{"code": -32601, "message": "AILANG daemon adapter does not support this server request"}}); err != nil {
						return err
					}
					canceled = fmt.Errorf("unsupported Codex daemon server request %s", m.Method)
					cancelC = nil
					cancelDeadline = time.After(5 * time.Second)
					if threadID != "" && turnID != "" {
						if err := send(99, "turn/interrupt", map[string]any{"threadId": threadID, "turnId": turnID}); err != nil {
							return err
						}
					}
					continue
				}
				switch *m.ID {
				case 1:
					if err := enc.Encode(map[string]any{"method": "initialized"}); err != nil {
						return err
					}
					p := map[string]any{"model": opts.Model, "cwd": opts.Workspace, "sandbox": opts.Sandbox, "approvalPolicy": opts.ApprovalPolicy, "config": opts.Config, "ephemeral": opts.Ephemeral, "threadSource": "exec"}
					if err := send(2, "thread/start", p); err != nil {
						return err
					}
				case 2:
					var v struct {
						Thread struct {
							ID string `json:"id"`
						} `json:"thread"`
					}
					if json.Unmarshal(m.Result, &v) != nil || v.Thread.ID == "" {
						return errors.New("Codex daemon returned no thread id")
					}
					threadID = v.Thread.ID
					if err := emit(map[string]any{"type": "thread.started", "thread_id": threadID}); err != nil {
						return err
					}
					if canceled != nil {
						if err := send(4, "thread/unsubscribe", map[string]any{"threadId": threadID}); err != nil {
							return err
						}
						continue
					}
					p := map[string]any{"threadId": threadID, "input": []any{map[string]any{"type": "text", "text": opts.Directive}}, "model": opts.Model}
					if opts.Effort != "" {
						p["effort"] = opts.Effort
					}
					if opts.OutputSchema != nil {
						p["outputSchema"] = opts.OutputSchema
					}
					turnRequested = true
					if err := send(3, "turn/start", p); err != nil {
						return err
					}
				case 3:
					var v struct {
						Turn struct {
							ID string `json:"id"`
						} `json:"turn"`
					}
					if json.Unmarshal(m.Result, &v) != nil || v.Turn.ID == "" {
						return errors.New("Codex daemon returned no turn id")
					}
					turnID = v.Turn.ID
					if !started {
						if err := emit(map[string]any{"type": "turn.started"}); err != nil {
							return err
						}
						started = true
					}
					if canceled != nil {
						if err := send(99, "turn/interrupt", map[string]any{"threadId": threadID, "turnId": turnID}); err != nil {
							return err
						}
					}
				case 4:
					return canceled
				}
				continue
			}
			if m.Method == "" {
				continue
			}
			var p struct {
				ThreadID string         `json:"threadId"`
				TurnID   string         `json:"turnId"`
				Item     map[string]any `json:"item"`
				Turn     struct {
					ID     string `json:"id"`
					Status string `json:"status"`
					Error  *struct {
						Message string `json:"message"`
					} `json:"error"`
				} `json:"turn"`
			}
			if json.Unmarshal(m.Params, &p) != nil {
				return errors.New("malformed Codex daemon notification")
			}
			if p.ThreadID != threadID {
				continue
			}
			if p.TurnID != "" && turnID != "" && p.TurnID != turnID {
				continue
			}
			switch m.Method {
			case "item/started", "item/agentMessage/delta", "item/reasoning/textDelta", "item/reasoning/summaryTextDelta", "item/commandExecution/outputDelta", "item/fileChange/outputDelta":
				if canceled == nil && ctx.Err() == nil {
					if err := emit(map[string]any{"type": "daemon.progress", "method": m.Method}); err != nil {
						return err
					}
				}
			case "turn/started":
				if turnID == "" {
					turnID = p.Turn.ID
				}
				if !started && canceled == nil && ctx.Err() == nil {
					if err := emit(map[string]any{"type": "turn.started"}); err != nil {
						return err
					}
					started = true
				}
			case "thread/tokenUsage/updated":
				var u struct {
					TokenUsage struct {
						Total struct {
							Input  int `json:"inputTokens"`
							Output int `json:"outputTokens"`
							Cached int `json:"cachedInputTokens"`
						} `json:"total"`
					} `json:"tokenUsage"`
				}
				if json.Unmarshal(m.Params, &u) != nil {
					return errors.New("malformed Codex usage")
				}
				usage = map[string]any{"input_tokens": u.TokenUsage.Total.Input, "output_tokens": u.TokenUsage.Total.Output, "cached_input_tokens": u.TokenUsage.Total.Cached}
			case "item/completed":
				item := p.Item
				switch item["type"] {
				case "agentMessage":
					item["type"] = "agent_message"
				case "commandExecution":
					item["type"] = "command_execution"
					item["aggregated_output"] = item["aggregatedOutput"]
					item["exit_code"] = item["exitCode"]
				case "fileChange":
					item["type"] = "file_change"
				default:
					continue
				}
				if canceled != nil || ctx.Err() != nil {
					continue
				}
				if err := emit(map[string]any{"type": "item.completed", "item": item}); err != nil {
					return err
				}
			case "turn/completed":
				if turnID != "" && p.Turn.ID != turnID {
					continue
				}
				terminal = true
				if p.Turn.Status != "completed" && canceled == nil {
					canceled = fmt.Errorf("Codex daemon turn ended with status %s", p.Turn.Status)
				}
				if p.Turn.Error != nil {
					canceled = fmt.Errorf("Codex daemon turn failed: %s", p.Turn.Error.Message)
				}
				if canceled == nil {
					if err := emit(map[string]any{"type": "turn.completed", "usage": usage}); err != nil {
						return err
					}
				}
				if err := send(4, "thread/unsubscribe", map[string]any{"threadId": threadID}); err != nil {
					return err
				}
			}
			// Server requests (approval, MCP elicitation, custom tools) must not hang.
			// Approval policy never avoids approvals; unsupported requests fail closed.
		}
	}
}

// Every entry point, including the shell controller, supplies an exact filtered
// environment. A daemon's startup environment must never become task authority.
func daemonIsolatedConfig(input map[string]any, env []string) map[string]any {
	cfg := map[string]any{}
	for k, v := range input {
		cfg[k] = v
	}
	set := map[string]any{}
	for _, v := range env {
		n, val, ok := strings.Cut(v, "=")
		if ok {
			set[n] = val
		}
	}
	if explicit, ok := cfg["shell_environment_policy.set"].(map[string]any); ok {
		for n, v := range explicit {
			set[n] = v
		}
	}
	prefix := "shell_environment_policy.set."
	for k, v := range cfg {
		if strings.HasPrefix(k, prefix) {
			set[strings.TrimPrefix(k, prefix)] = v
			delete(cfg, k)
		}
	}
	// Preserve the real routing home even if an accidental override was supplied.
	for _, v := range env {
		n, val, ok := strings.Cut(v, "=")
		if ok && (n == "CODEX_HOME" || n == "AILANG_CODEX_RUNTIME") {
			set[n] = val
		}
	}
	cfg["shell_environment_policy.set"] = set
	cfg["shell_environment_policy.inherit"] = "none"
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	cfg["shell_environment_policy.include_only"] = names
	return cfg
}
