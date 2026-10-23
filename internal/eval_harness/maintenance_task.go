package eval_harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// MaintenanceExecutorSettings freezes existing registry controls and teaching
// prompts. Shipped scripted-control settings intentionally cannot launch models.
type MaintenanceExecutorSettings struct {
	Schema              string                        `json:"schema"`
	ModelKey            string                        `json:"modelKey"`
	ModelControlsSHA256 string                        `json:"modelControlsSHA256"`
	HarnessVersion      string                        `json:"harnessVersion"`
	AllowedTools        []string                      `json:"allowedTools"`
	SystemPrompts       map[string]MaintenanceFileRef `json:"systemPrompts"`
}

// MaintenanceModelControlsSHA256 freezes resolved existing registry controls,
// including effective routing and budget defaults, for a future launch manifest.
func MaintenanceModelControlsSHA256(key string, maxTokens int) (string, error) {
	registry := modelreg.GlobalModelsConfig
	if registry == nil {
		return "", fmt.Errorf("model registry is not loaded")
	}
	row, ok := registry.Models[key]
	if !ok {
		return "", fmt.Errorf("explicit registry model key %q is missing", key)
	}
	controls := struct {
		Model       modelreg.ModelConfig
		Route       map[string]any
		MaxTokens   int
		HardSeconds int
		MaxCost     float64
	}{row, registry.EffectiveOpenRouterProvider(row.APIName), row.ResolvedMaxTokensPerBench(maxTokens), row.ResolvedHardTimeoutSecs(), row.ResolvedMaxCostUSD()}
	b, err := json.Marshal(controls)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

// NewMaintenanceTaskBuilder reuses task assembly without the one-shot workspace
// lifecycle. prepare attaches controller-owned containment. Actual qualification,
// rig/capability admission and launch authorization remain mandatory in the
// runner's BeforeDispatch guard. Construction performs no executor/model call.
func NewMaintenanceTaskBuilder(e *MaintenanceExecution, prepare func(*executor.Task) error) (MaintenanceTaskBuilder, error) {
	if e == nil || prepare == nil {
		return nil, fmt.Errorf("execution and trusted containment preparation required")
	}
	data, err := maintenanceReadBoundFile(e.Manifest.Settings)
	if err != nil {
		return nil, err
	}
	var settings MaintenanceExecutorSettings
	if err = maintenanceStrictJSON(data, &settings); err != nil {
		return nil, err
	}
	if settings.Schema != "maintenance.executor-settings/1" || settings.HarnessVersion != e.Plan.Harness || !strings.HasPrefix(settings.HarnessVersion, "pi@") || e.Manifest.Isolation.Backend != "pi" {
		return nil, fmt.Errorf("frozen pi executor settings/harness required")
	}
	if len(settings.AllowedTools) == 0 {
		return nil, fmt.Errorf("explicit native tool allowlist required")
	}
	seen := map[string]bool{}
	for _, tool := range settings.AllowedTools {
		if tool != "Read" && tool != "Write" && tool != "Edit" && tool != "Bash" && tool != "Grep" && tool != "Find" && tool != "LS" {
			return nil, fmt.Errorf("unqualified maintenance tool %q", tool)
		}
		if seen[tool] {
			return nil, fmt.Errorf("duplicate maintenance tool")
		}
		seen[tool] = true
	}
	check := func() (string, error) {
		digest, err := MaintenanceModelControlsSHA256(settings.ModelKey, e.Manifest.Limits.MaxTokens)
		if err != nil {
			return "", err
		}
		if digest != settings.ModelControlsSHA256 {
			return "", fmt.Errorf("frozen model/pricing/route/budget controls changed")
		}
		cli, model, err := modelreg.GlobalModelsConfig.GetExecutorForModel(settings.ModelKey)
		if err != nil {
			return "", err
		}
		if cli != "pi" || model != e.Plan.Model {
			return "", fmt.Errorf("resolved model/backend differs from plan")
		}
		return model, nil
	}
	if _, err = check(); err != nil {
		return nil, err
	}
	for language, ref := range settings.SystemPrompts {
		if !filepath.IsAbs(ref.Path) {
			ref.Path = filepath.Join(e.Root, ref.Path)
		}
		if _, err = maintenanceReadBoundFile(ref); err != nil {
			return nil, fmt.Errorf("%s teaching prompt: %w", language, err)
		}
		settings.SystemPrompts[language] = ref
	}
	return func(tr MaintenanceTrajectoryPlan, stage MaintenanceStagePlan, workspace, request string) (*executor.Task, error) {
		model, err := check()
		if err != nil {
			return nil, err
		}
		if _, err = maintenanceReadBoundFile(e.Manifest.Settings); err != nil {
			return nil, err
		}
		ref, ok := settings.SystemPrompts[tr.Language]
		if !ok {
			return nil, fmt.Errorf("missing frozen %s system prompt", tr.Language)
		}
		prompt, err := maintenanceReadBoundFile(ref)
		if err != nil {
			return nil, err
		}
		cfg := &MultiExecutorConfig{AgentBenchmarkConfig: AgentBenchmarkConfig{TimeoutSeconds: e.Manifest.Limits.MaxStageSeconds, MaxTokensPerBench: e.Manifest.Limits.MaxTokens}, ExecutorName: "pi", ModelName: settings.ModelKey, ConfigKey: settings.ModelKey, StageID: stage.ID}
		task, err := BuildEvalExecutorTask(&BenchmarkSpec{ID: tr.ID, Timeout: e.Manifest.Limits.MaxStageSeconds}, cfg, model, workspace, string(prompt), request)
		if err != nil {
			return nil, err
		}
		// Manifest bounds can tighten ordinary registry ceilings, not expand them.
		if ceiling := time.Duration(e.Manifest.Limits.MaxStageSeconds) * time.Second; task.Timeout <= 0 || task.Timeout > ceiling {
			task.Timeout = ceiling
		}
		if task.MaxTokensPerBench <= 0 || task.MaxTokensPerBench > e.Manifest.Limits.MaxTokens {
			task.MaxTokensPerBench = e.Manifest.Limits.MaxTokens
		}
		task.AllowedTools = append([]string{}, settings.AllowedTools...)
		task.IsolateFromAmbientContext = true
		task.IsolateFromProjectExtensions = true
		task.PersistentSystemPrompt = false
		task.Iteration = 1
		task.Metadata["maintenance_settings_digest"] = e.Plan.SettingsDigest
		task.Metadata["maintenance_system_prompt_digest"] = ref.SHA256
		task.Metadata["maintenance_expected_executor_version"] = settings.HarnessVersion
		task.Metadata["maintenance_request_limit_enforcement"] = "unavailable"
		if err = prepare(task); err != nil {
			return nil, err
		}
		return task, nil
	}, nil
}
