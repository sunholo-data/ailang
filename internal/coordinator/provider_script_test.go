package coordinator

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParsePayloadToEnv(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantEnv []string
		wantErr bool
	}{
		{
			name:    "empty payload",
			payload: "",
			wantEnv: nil,
			wantErr: false,
		},
		{
			name:    "simple string values",
			payload: `{"model": "gpt5", "benchmarks": "all"}`,
			wantEnv: []string{"MODEL=gpt5", "BENCHMARKS=all"},
			wantErr: false,
		},
		{
			name:    "boolean values",
			payload: `{"parallel": true, "verbose": false}`,
			wantEnv: []string{"PARALLEL=true", "VERBOSE=false"},
			wantErr: false,
		},
		{
			name:    "numeric values",
			payload: `{"count": 42, "rate": 3.14}`,
			wantEnv: []string{"COUNT=42", "RATE=3.14"},
			wantErr: false,
		},
		{
			name:    "nested object",
			payload: `{"db": {"host": "localhost", "port": 5432}}`,
			wantEnv: []string{"DB_HOST=localhost", "DB_PORT=5432"},
			wantErr: false,
		},
		{
			name:    "array values",
			payload: `{"models": ["gpt5", "claude", "gemini"]}`,
			wantEnv: []string{"MODELS=gpt5,claude,gemini"},
			wantErr: false,
		},
		{
			name:    "null value",
			payload: `{"optional": null}`,
			wantEnv: []string{"OPTIONAL="},
			wantErr: false,
		},
		{
			name:    "snake_case key",
			payload: `{"my_key": "value"}`,
			wantEnv: []string{"MY_KEY=value"},
			wantErr: false,
		},
		{
			name:    "camelCase key",
			payload: `{"myKey": "value"}`,
			wantEnv: []string{"MYKEY=value"},
			wantErr: false,
		},
		{
			name:    "key with special chars",
			payload: `{"my-key.name": "value"}`,
			wantEnv: []string{"MY_KEY_NAME=value"},
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			payload: `not valid json`,
			wantEnv: nil,
			wantErr: true,
		},
		{
			name:    "deeply nested",
			payload: `{"level1": {"level2": {"level3": "deep"}}}`,
			wantEnv: []string{"LEVEL1_LEVEL2_LEVEL3=deep"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePayloadToEnv(tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePayloadToEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Check that all expected env vars are present
			gotMap := make(map[string]bool)
			for _, e := range got {
				gotMap[e] = true
			}
			for _, want := range tt.wantEnv {
				if !gotMap[want] {
					t.Errorf("ParsePayloadToEnv() missing expected env var %q, got %v", want, got)
				}
			}
		})
	}
}

func TestToEnvKey(t *testing.T) {
	tests := []struct {
		prefix string
		key    string
		want   string
	}{
		{"", "model", "MODEL"},
		{"", "MY_KEY", "MY_KEY"},
		{"DB", "host", "DB_HOST"},
		{"", "my-key", "MY_KEY"},
		{"", "my.key", "MY_KEY"},
		{"", "camelCase", "CAMELCASE"},
		{"PREFIX", "suffix", "PREFIX_SUFFIX"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"_"+tt.key, func(t *testing.T) {
			got := toEnvKey(tt.prefix, tt.key)
			if got != tt.want {
				t.Errorf("toEnvKey(%q, %q) = %q, want %q", tt.prefix, tt.key, got, tt.want)
			}
		})
	}
}

func TestScriptProvider_Execute(t *testing.T) {
	provider := NewScriptProvider()

	t.Run("simple echo command", func(t *testing.T) {
		task := &AnalyzedTask{
			Task: &Task{
				ID:        "test-task-1",
				MessageID: "msg-1",
				Content:   `{"greeting": "hello"}`,
			},
		}
		opts := &ExecuteOptions{
			Workspace: "/tmp",
			InvokeConfig: &InvokeConfig{
				Type:           "script",
				Command:        "echo $GREETING",
				EnvFromPayload: true,
			},
		}

		result, err := provider.Execute(context.Background(), task, opts)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Execute() success = false, error = %s", result.Error)
		}

		if !strings.Contains(result.Output, "hello") {
			t.Errorf("Execute() output = %q, want to contain 'hello'", result.Output)
		}

		if result.Cost != 0.0 {
			t.Errorf("Execute() cost = %v, want 0.0 (scripts are free)", result.Cost)
		}

		if result.Provider != "script" {
			t.Errorf("Execute() provider = %q, want 'script'", result.Provider)
		}
	})

	t.Run("AILANG context variables injected", func(t *testing.T) {
		task := &AnalyzedTask{
			Task: &Task{
				ID:        "task-abc123",
				MessageID: "msg-xyz789",
				Content:   "",
			},
		}
		opts := &ExecuteOptions{
			Workspace: "/tmp", // Use existing directory
			InvokeConfig: &InvokeConfig{
				Type:    "script",
				Command: "echo TASK=$AILANG_TASK_ID MSG=$AILANG_MESSAGE_ID WS=$AILANG_WORKSPACE",
			},
		}

		result, err := provider.Execute(context.Background(), task, opts)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Execute() success = false, error = %s", result.Error)
		}

		if !strings.Contains(result.Output, "TASK=task-abc123") {
			t.Errorf("Execute() output missing AILANG_TASK_ID")
		}
		if !strings.Contains(result.Output, "MSG=msg-xyz789") {
			t.Errorf("Execute() output missing AILANG_MESSAGE_ID")
		}
		if !strings.Contains(result.Output, "WS=/tmp") {
			t.Errorf("Execute() output missing AILANG_WORKSPACE, got: %s", result.Output)
		}
	})

	t.Run("exit code determines success", func(t *testing.T) {
		task := &AnalyzedTask{
			Task: &Task{ID: "test-fail"},
		}
		opts := &ExecuteOptions{
			Workspace: "/tmp",
			InvokeConfig: &InvokeConfig{
				Type:    "script",
				Command: "exit 1",
			},
		}

		result, err := provider.Execute(context.Background(), task, opts)
		if err != nil {
			t.Fatalf("Execute() error = %v (should not return error for non-zero exit)", err)
		}

		if result.Success {
			t.Errorf("Execute() success = true, want false for exit code 1")
		}

		if !strings.Contains(result.Error, "exit") {
			t.Errorf("Execute() error = %q, want to contain 'exit'", result.Error)
		}
	})

	t.Run("timeout kills script", func(t *testing.T) {
		task := &AnalyzedTask{
			Task: &Task{ID: "test-timeout"},
		}
		opts := &ExecuteOptions{
			Workspace: "/tmp",
			InvokeConfig: &InvokeConfig{
				Type:    "script",
				Command: "sleep 10",
				Timeout: "100ms",
			},
		}

		start := time.Now()
		result, err := provider.Execute(context.Background(), task, opts)
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if result.Success {
			t.Errorf("Execute() success = true, want false for timeout")
		}

		if !strings.Contains(result.Error, "timed out") {
			t.Errorf("Execute() error = %q, want to contain 'timed out'", result.Error)
		}

		if elapsed > 1*time.Second {
			t.Errorf("Execute() took %v, want < 1s (timeout should kill it)", elapsed)
		}
	})

	t.Run("missing command field", func(t *testing.T) {
		task := &AnalyzedTask{
			Task: &Task{ID: "test-missing"},
		}
		opts := &ExecuteOptions{
			Workspace: "/tmp",
			InvokeConfig: &InvokeConfig{
				Type:    "script",
				Command: "", // Missing!
			},
		}

		_, err := provider.Execute(context.Background(), task, opts)
		if err == nil {
			t.Errorf("Execute() error = nil, want error for missing command")
		}
	})

	t.Run("wrong invoke type", func(t *testing.T) {
		task := &AnalyzedTask{
			Task: &Task{ID: "test-wrong-type"},
		}
		opts := &ExecuteOptions{
			Workspace: "/tmp",
			InvokeConfig: &InvokeConfig{
				Type: "skill", // Wrong type!
				Name: "some-skill",
			},
		}

		_, err := provider.Execute(context.Background(), task, opts)
		if err == nil {
			t.Errorf("Execute() error = nil, want error for wrong invoke type")
		}
	})

	t.Run("custom shell", func(t *testing.T) {
		task := &AnalyzedTask{
			Task: &Task{ID: "test-bash"},
		}
		opts := &ExecuteOptions{
			Workspace: "/tmp",
			InvokeConfig: &InvokeConfig{
				Type:    "script",
				Command: "echo $BASH_VERSION",
				Shell:   "/bin/bash",
			},
		}

		result, err := provider.Execute(context.Background(), task, opts)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Execute() with bash shell failed: %s", result.Error)
		}
	})

	t.Run("multiline script", func(t *testing.T) {
		task := &AnalyzedTask{
			Task: &Task{
				ID:      "test-multiline",
				Content: `{"step": "one"}`,
			},
		}
		opts := &ExecuteOptions{
			Workspace: "/tmp",
			InvokeConfig: &InvokeConfig{
				Type: "script",
				Command: `
					set -e
					echo "Step: $STEP"
					echo "RESULT: success"
				`,
				EnvFromPayload: true,
			},
		}

		result, err := provider.Execute(context.Background(), task, opts)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		if !result.Success {
			t.Errorf("Execute() success = false, error = %s", result.Error)
		}

		if !strings.Contains(result.Output, "Step: one") {
			t.Errorf("Execute() output missing 'Step: one'")
		}
		if !strings.Contains(result.Output, "RESULT: success") {
			t.Errorf("Execute() output missing 'RESULT: success'")
		}
	})
}

func TestScriptProvider_Name(t *testing.T) {
	provider := NewScriptProvider()
	if got := provider.Name(); got != "script" {
		t.Errorf("Name() = %q, want 'script'", got)
	}
}

func TestScriptProvider_CanHandle(t *testing.T) {
	provider := NewScriptProvider()
	task := &AnalyzedTask{
		Task: &Task{ID: "test"},
	}
	// Script tasks are explicitly routed, not auto-detected
	if got := provider.CanHandle(task); got != false {
		t.Errorf("CanHandle() = %v, want false (scripts are explicitly routed)", got)
	}
}
