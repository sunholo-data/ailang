package eval_harness

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
	"github.com/sunholo-data/ailang/internal/browser/browserbase"
	localbrowser "github.com/sunholo-data/ailang/internal/browser/local"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var browserSessionTracer = telemetry.Tracer("eval.browser_session")

const (
	defaultBrowserViewportWidth  = 1280
	defaultBrowserViewportHeight = 720
	defaultBrowserPolicyVersion  = "browser-policy-v1"
	providerDefaultUnpinned      = "provider-default-unpinned"
)

// BrowserSessionConfig is additive: an empty Provider preserves the existing
// non-browser eval path. ProviderInstance exists for hermetic tests and custom
// deployments; ordinary callers select local-playwright or browserbase.
type BrowserSessionConfig struct {
	Provider           string
	ProviderInstance   browser.SessionProvider
	BaseDir            string
	ArtifactDir        string
	NpxPath            string
	MCPVersion         string
	BrowserVersion     string
	PolicyVersion      string
	Region             string
	MaximumDuration    time.Duration
	ActionTimeout      time.Duration
	ViewportWidth      int
	ViewportHeight     int
	Headful            bool
	BrowserbaseBaseURL string
	APIKeyEnv          string
	ProjectIDEnv       string
	ChainID            string
	StageID            string
}

func executeWithBrowser(
	ctx context.Context,
	exec executor.Executor,
	task *executor.Task,
	handler executor.EventHandler,
	config BrowserSessionConfig,
) (*executor.Result, *browser.BrowserRunManifest, error) {
	if config.Provider == "" {
		if err := executor.ValidateTaskCapabilities(task, exec); err != nil {
			return nil, nil, err
		}
		result, err := exec.ExecuteStreaming(ctx, task, handler)
		return result, nil, err
	}
	if !executorHasCapability(exec, executor.CapMCP) {
		return nil, nil, fmt.Errorf("browser provider %q requires executor capability %q; executor %q does not advertise it", config.Provider, executor.CapMCP, exec.Name())
	}
	provider, err := resolveBrowserProvider(config)
	if err != nil {
		return nil, nil, err
	}
	ctx, span := telemetry.StartSpan(ctx, browserSessionTracer, "browser.session", trace.WithAttributes(
		attribute.String("browser.provider", provider.Name()),
		attribute.String("browser.run_id", task.ID),
		attribute.String("ailang.chain_id", config.ChainID),
		attribute.String("ailang.stage_id", config.StageID),
	))
	defer span.End()
	artifactDir := config.ArtifactDir
	if artifactDir == "" && task.Workspace != "" {
		artifactDir = filepath.Join(task.Workspace, "browser-artifacts")
	}
	mcpVersion := config.MCPVersion
	if mcpVersion == "" {
		mcpVersion = localbrowser.DefaultMCPVersion
	}
	browserVersion := config.BrowserVersion
	if browserVersion == "" {
		browserVersion = localbrowser.DefaultBrowserVersion
	}
	viewportWidth, viewportHeight := config.ViewportWidth, config.ViewportHeight
	if viewportWidth <= 0 || viewportHeight <= 0 {
		viewportWidth, viewportHeight = defaultBrowserViewportWidth, defaultBrowserViewportHeight
	}
	policyVersion := config.PolicyVersion
	if policyVersion == "" {
		policyVersion = defaultBrowserPolicyVersion
	}
	maximumDuration := config.MaximumDuration
	if maximumDuration <= 0 {
		maximumDuration = task.Timeout
	}
	controller := browser.NewController(provider, browser.ControllerOptions{CleanupTimeout: 30 * time.Second})
	run, err := controller.Start(ctx, browser.SessionSpec{
		RunID: task.ID, Provider: provider.Name(), ChainID: config.ChainID,
		StageID: config.StageID, Browser: "chromium", BrowserVersion: browserVersion,
		MCPVersion: mcpVersion, PolicyVersion: policyVersion,
		ViewportWidth: viewportWidth, ViewportHeight: viewportHeight,
		Locale: providerDefaultUnpinned, Timezone: providerDefaultUnpinned,
		Headless: !config.Headful, MaximumDuration: maximumDuration,
		ActionTimeout: config.ActionTimeout, ArtifactDir: artifactDir, Region: config.Region,
		RecordTrace: true, RecordVideo: true,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, string(browser.FailureProvision))
		return nil, nil, err
	}
	span.SetAttributes(attribute.String("browser.session_id", run.Session().ID))
	mcp, env := run.Connection()
	task.MCPServers = append(task.MCPServers, executor.MCPServerConfig{
		Name: mcp.Name, Command: mcp.Command, Args: append([]string(nil), mcp.Args...),
		EnvVars: append([]string(nil), mcp.EnvVars...), Required: mcp.Required,
	})
	if task.ExtraEnv == nil {
		task.ExtraEnv = make(map[string]string, len(env))
	}
	for key, value := range env {
		task.ExtraEnv[key] = value
	}
	if err := executor.ValidateTaskCapabilities(task, exec); err != nil {
		manifest, finishErr := run.Finish(ctx, browser.TerminationExecutorFailed)
		if finishErr != nil {
			return nil, &manifest, errors.Join(err, finishErr)
		}
		return nil, &manifest, err
	}
	result, executeErr := exec.ExecuteStreaming(ctx, task, handler)
	termination := browser.TerminationCompleted
	if executeErr != nil || (result != nil && !result.Success) {
		termination = browser.TerminationExecutorFailed
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		termination = browser.TerminationCancelled
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		termination = browser.TerminationTimeout
	}
	manifest, finishErr := run.Finish(ctx, termination)
	manifest.ManagedVessel = executorHasCapability(exec, executor.CapRemoteSandbox)
	if manifest.ManagedVessel {
		manifest.AgentScaffold = exec.Name()
		manifest.Comparable = false
		manifest.NonComparableReason = "managed_agent_scaffold"
	}
	span.SetAttributes(
		attribute.String("browser.termination", string(manifest.Termination)),
		attribute.Int("browser.action_count", manifest.Usage.ActionCount),
		attribute.Int("browser.artifact_count", len(manifest.Artifacts.Refs)),
		attribute.Bool("browser.artifacts_complete", manifest.Artifacts.Complete),
		attribute.String("browser.cost_source", manifest.Cost.Source),
		attribute.String("browser.inspection_ref", manifest.Inspection.Ref),
	)
	if finishErr != nil || executeErr != nil || termination != browser.TerminationCompleted {
		span.SetStatus(codes.Error, string(termination))
	} else {
		span.SetStatus(codes.Ok, string(termination))
	}
	if result != nil {
		if result.ProviderData == nil {
			result.ProviderData = make(map[string]any)
		}
		result.ProviderData["browser"] = manifest
	}
	if executeErr != nil {
		return result, &manifest, executeErr
	}
	if finishErr != nil {
		return result, &manifest, finishErr
	}
	return result, &manifest, nil
}

func resolveBrowserProvider(config BrowserSessionConfig) (browser.SessionProvider, error) {
	if config.ProviderInstance != nil {
		if config.ProviderInstance.Name() != config.Provider {
			return nil, fmt.Errorf("browser provider instance is %q, config selects %q", config.ProviderInstance.Name(), config.Provider)
		}
		return config.ProviderInstance, nil
	}
	switch config.Provider {
	case localbrowser.ProviderName:
		return localbrowser.New(localbrowser.Config{BaseDir: config.BaseDir, NpxPath: config.NpxPath, MCPVersion: config.MCPVersion})
	case browserbase.ProviderName:
		apiKeyEnv := config.APIKeyEnv
		if apiKeyEnv == "" {
			apiKeyEnv = "BROWSERBASE_API_KEY"
		}
		projectIDEnv := config.ProjectIDEnv
		if projectIDEnv == "" {
			projectIDEnv = "BROWSERBASE_PROJECT_ID"
		}
		return browserbase.New(browserbase.Config{
			APIKey: os.Getenv(apiKeyEnv), ProjectID: os.Getenv(projectIDEnv),
			BaseURL: config.BrowserbaseBaseURL, NpxPath: config.NpxPath, MCPVersion: config.MCPVersion,
		})
	default:
		return nil, fmt.Errorf("unknown browser provider %q (supported: %s, %s)", config.Provider, localbrowser.ProviderName, browserbase.ProviderName)
	}
}
