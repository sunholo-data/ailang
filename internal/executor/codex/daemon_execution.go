package codex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// daemonExecution feeds the unchanged exec-event consumer without spawning a
// second refresh owner. Cancel reaches the protocol adapter, not the daemon.
func daemonExecution(ctx context.Context, path string, task *executor.Task, model, directive string, env []string) (io.ReadCloser, func() error, func() error, context.CancelFunc, error) {
	if task.AllowedTools != nil {
		return nil, nil, nil, nil, fmt.Errorf("Codex daemon adapter cannot enforce task tool allowlists")
	}
	cfg := map[string]any{}
	// Unlike exec, the daemon's parent environment predates this task. Pass the
	// already filtered child environment explicitly to thread-local shell policy.
	set := map[string]any{}
	for _, v := range env {
		n, val, ok := strings.Cut(v, "=")
		if ok {
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
	for _, s := range task.MCPServers {
		if !mcpNamePattern.MatchString(s.Name) || s.Command == "" {
			return nil, nil, nil, nil, fmt.Errorf("invalid Codex daemon MCP configuration")
		}
		prefix := "mcp_servers." + s.Name + "."
		cfg[prefix+"command"] = s.Command
		cfg[prefix+"args"] = s.Args
		mcpEnv := map[string]any{}
		for _, name := range s.EnvVars {
			val, ok := set[name]
			if !ok {
				return nil, nil, nil, nil, fmt.Errorf("Codex daemon MCP environment %s is unavailable", name)
			}
			mcpEnv[name] = val
		}
		cfg[prefix+"env"] = mcpEnv
		cfg[prefix+"env_vars"] = []string{}
		cfg[prefix+"required"] = s.Required
		for _, name := range s.EnvVars {
			if !envNamePattern.MatchString(name) {
				return nil, nil, nil, nil, fmt.Errorf("invalid Codex daemon MCP environment name")
			}
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	r, w := io.Pipe()
	done := make(chan error, 1)
	finished := make(chan struct{})
	var runErr error
	start := func() error {
		go func() {
			err := RunDaemonExec(runCtx, path, DaemonExecOptions{Model: model, Directive: directive, Workspace: task.Workspace, Sandbox: "danger-full-access", ApprovalPolicy: "never", Config: cfg, Env: env, Effort: task.Effort}, w)
			_ = w.Close()
			runErr = err
			done <- err
			close(finished)
		}()
		return nil
	}
	wait := func() error { return <-done }
	rwrap := &daemonReader{ReadCloser: r, cancel: cancel, finished: finished, runErr: &runErr}
	return rwrap, start, wait, cancel, nil
}

type daemonReader struct {
	io.ReadCloser
	cancel   context.CancelFunc
	finished <-chan struct{}
	runErr   *error
}

func (r *daemonReader) Close() error {
	r.cancel()
	<-r.finished
	err := r.ReadCloser.Close()
	if r.runErr != nil && errors.Is(*r.runErr, ErrProcessTerminationUnconfirmed) {
		return *r.runErr
	}
	return err
}
