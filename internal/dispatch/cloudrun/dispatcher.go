// Package cloudrun implements the CloudDispatcher interface using
// the Cloud Run Jobs Admin API v2. It triggers job executions with
// per-execution environment variable overrides.
package cloudrun

import (
	"context"
	"fmt"
	"strings"

	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/googleapis/gax-go/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// jobRunner abstracts the Cloud Run Jobs API client for testing.
type jobRunner interface {
	RunJob(ctx context.Context, req *runpb.RunJobRequest, opts ...gax.CallOption) (*run.RunJobOperation, error)
}

// Dispatcher implements coordinator.CloudDispatcher using Cloud Run Jobs API.
type Dispatcher struct {
	client    jobRunner
	projectID string
	region    string
	prefix    string
}

// Compile-time check that Dispatcher implements CloudDispatcher.
var _ coordinator.CloudDispatcher = (*Dispatcher)(nil)

// NewDispatcher creates a new Cloud Run Jobs dispatcher.
// It creates a gRPC client to the Cloud Run Admin API.
func NewDispatcher(ctx context.Context, projectID, region, prefix string) (*Dispatcher, error) {
	client, err := run.NewJobsClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Run Jobs client: %w", err)
	}

	return &Dispatcher{
		client:    client,
		projectID: projectID,
		region:    region,
		prefix:    prefix,
	}, nil
}

// newDispatcherWithClient creates a Dispatcher with a custom client (for testing).
func newDispatcherWithClient(client jobRunner, projectID, region, prefix string) *Dispatcher {
	return &Dispatcher{
		client:    client,
		projectID: projectID,
		region:    region,
		prefix:    prefix,
	}
}

// knownVariants is the set of valid executor_variant values.
// Each variant corresponds to a separate Cloud Run Job template in Terraform with the
// matching Docker image baked in. The Cloud Run Jobs API does not support per-execution
// image overrides, so variant selection happens via job name, not ContainerOverride.
// Job naming: {prefix}-agent-executor[-{variant}][-apikey]
var knownVariants = map[string]bool{
	"":          true, // defaults to "default"
	"default":   true,
	"go":        true,
	"gemini":    true,
	"gemini-go": true,
	"codex":     true,
	"codex-go":  true,
	"opencode":  true,
	"pi":        true,
	"pi-go":     true, // Dockerfile.agent-pi-go + job ailang-agent-executor-pi-go both exist
	"motoko":    true, // M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0): AILANG-native agent
	"eval":      true,
	"eval-go":   true,
}

// providersForVariant maps an executor variant to the executor binaries baked
// into that variant's image. Ground truth is docker/Dockerfile.agent-<variant>:
// the Cloud Run Jobs API cannot override an image per execution, so the variant
// IS the image, and the image decides which binary exists on $PATH.
//
// A nil value means "any provider" — only agent-eval installs every CLI.
var providersForVariant = map[string][]string{
	"":          {"claude"},   // Dockerfile.agent: @anthropic-ai/claude-code
	"default":   {"claude"},   //   ditto
	"go":        {"claude"},   // Dockerfile.agent-go: FROM agent
	"codex":     {"codex"},    // Dockerfile.agent-codex: @openai/codex
	"codex-go":  {"codex"},    // FROM agent-codex
	"gemini":    {"gemini"},   // Dockerfile.agent-gemini: @google/gemini-cli
	"gemini-go": {"gemini"},   // FROM agent-gemini
	"opencode":  {"opencode"}, // Dockerfile.agent-opencode: opencode-ai
	"pi":        {"pi"},       // Dockerfile.agent-pi: @mariozechner/pi-coding-agent
	"pi-go":     {"pi"},       // FROM agent-pi
	"motoko":    {"motoko"},   // Dockerfile.agent-motoko
	"eval":      nil,          // agent-eval: claude + gemini + codex + opencode + pi
	"eval-go":   nil,          // FROM agent-eval
}

// binarylessProviders reach a remote API and shell out to nothing, so they are
// runnable in any image and must never be refused on image grounds.
var binarylessProviders = map[string]bool{
	"managed_agents": true,
}

// checkVariantProviderAgreement refuses a dispatch whose executor binary cannot
// exist in the image it would run in.
//
// ExecutorVariant selects the Cloud Run Job, and therefore the image. The
// separate AILANG_PROVIDER env var selects which executor runs INSIDE it
// (cmd/ailang/coordinator_cloud.go passes it to executor.GetExecutor). Nothing
// tied the two together, so a mismatch was discoverable only by the container,
// at the END of its setup. Measured 2026-08-28: task-a0628a5f dispatched to
// ailang-agent-executor-codex, logged "running opencode executor (unified path)"
// and died on
//
//	exec: "opencode": executable file not found in $PATH
//
// AFTER cloning 24,032 files and cutting its branch. Every such dispatch burns a
// full container start and repo clone to learn something knowable before launch.
// Refusing here turns a silent late failure into a loud early one.
func checkVariantProviderAgreement(variant, provider string) error {
	if provider == "" || binarylessProviders[provider] {
		return nil
	}
	allowed, known := providersForVariant[variant]
	if !known || allowed == nil {
		// Unknown variants are rejected by jobSuffixForVariant; nil means the
		// image carries every CLI.
		return nil
	}
	for _, p := range allowed {
		if p == provider {
			return nil
		}
	}
	return fmt.Errorf("executor_variant %q runs image agent-%s, which has %v on $PATH, but provider is %q: "+
		"the job would clone the repo and then fail with %q: executable file not found in $PATH. "+
		"Fix the agent's provider/executor_variant pair in config.cloud.yaml",
		variant, variantImageName(variant), allowed, provider, provider)
}

// variantImageName renders the image basename for a variant, for error text.
func variantImageName(variant string) string {
	if variant == "" || variant == "default" {
		return "agent"
	}
	return variant
}

// jobSuffixForVariant returns the Cloud Run Job name suffix for a variant + auth mode pair.
// Examples:
//
//	("", "oauth")      → "agent-executor"
//	("go", "oauth")    → "agent-executor-go"
//	("go", "apikey")   → "agent-executor-go-apikey"
//	("codex", "oauth") → "agent-executor-codex"
func jobSuffixForVariant(variant, authMode string) (string, error) {
	if !knownVariants[variant] {
		return "", fmt.Errorf("unknown executor_variant %q — check config.cloud.yaml", variant)
	}
	suffix := "agent-executor"
	if variant != "" && variant != "default" {
		suffix += "-" + variant
	}
	if authMode == "apikey" {
		suffix += "-apikey"
	}
	return suffix, nil
}

// Dispatch triggers a Cloud Run Job execution with per-execution env var overrides.
// The job is identified by the pattern: projects/{project}/locations/{region}/jobs/{prefix}-agent-executor
// This matches the Terraform-defined job name in cloud_run_jobs.tf.
func (d *Dispatcher) Dispatch(ctx context.Context, params coordinator.DispatchParams) error {
	// Defend the external-effect boundary even when a caller bypasses the
	// coordinator daemon. Validation happens before job-name construction and,
	// critically, before RunJob.
	if err := coordinator.ValidateExecutionRoute(params.AgentID, params.Provider, params.ExecutorVariant); err != nil {
		return err
	}

	// M-EXECUTOR-VARIANTS + M-CLOUD-DUAL-AUTH: select the Cloud Run Job template.
	// Each variant has its own job with the corresponding Docker image baked in.
	// Auth mode selects between OAuth and API-key job templates within each variant.
	jobSuffix, err := jobSuffixForVariant(params.ExecutorVariant, params.AuthMode)
	if err != nil {
		return err
	}
	if err := checkVariantProviderAgreement(params.ExecutorVariant, params.Provider); err != nil {
		return err
	}
	jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s-%s",
		d.projectID, d.region, d.prefix, jobSuffix)

	envOverrides := []*runpb.EnvVar{
		{Name: "AILANG_TASK_ID", Values: &runpb.EnvVar_Value{Value: params.TaskID}},
		{Name: "AILANG_AGENT_ID", Values: &runpb.EnvVar_Value{Value: params.AgentID}},
		{Name: "AILANG_WORKSPACE", Values: &runpb.EnvVar_Value{Value: params.Workspace}},
		{Name: "AILANG_PROVIDER", Values: &runpb.EnvVar_Value{Value: params.Provider}},
		{Name: "AILANG_DIRECTIVE", Values: &runpb.EnvVar_Value{Value: params.Directive}},
		{Name: "AILANG_REPO_URL", Values: &runpb.EnvVar_Value{Value: params.RepoURL}},
		{Name: "AILANG_BRANCH", Values: &runpb.EnvVar_Value{Value: params.Branch}},
	}
	// For skip_approval agents, push directly to target branch instead of coordinator/{taskID}.
	if params.PushBranch != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_PUSH_BRANCH", Values: &runpb.EnvVar_Value{Value: params.PushBranch},
		})
	}
	// Pass plugin repo for shared skills (M-CLOUD-PLUGIN-SKILLS, v0.9.1)
	if params.PluginRepo != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_PLUGIN_REPO", Values: &runpb.EnvVar_Value{Value: params.PluginRepo},
		})
	}
	// Pass model override from agent config so cloud executor uses the right model
	// (without this, the executor defaults to "haiku" which is too weak for coding)
	if params.Model != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_MODEL", Values: &runpb.EnvVar_Value{Value: params.Model},
		})
	}
	// Pass executor timeout from agent config (M-CLOUD-OAUTH)
	// Without this, the executor defaults to 5m which is too short for complex tasks
	if params.Timeout != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_TIMEOUT", Values: &runpb.EnvVar_Value{Value: params.Timeout},
		})
	}
	// M-CLOUD-PROGRESS-TRACKING: Pass per-task cost budget for mid-execution enforcement.
	if params.MaxCostUSD > 0 {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_MAX_COST_USD", Values: &runpb.EnvVar_Value{Value: fmt.Sprintf("%.4f", params.MaxCostUSD)},
		})
	}
	// M-GIT-GUARDRAILS: Pass per-agent git mode for PreToolUse hook enforcement.
	// When set, overrides the Terraform-level AILANG_GIT_MODE default.
	if params.GitMode != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_GIT_MODE", Values: &runpb.EnvVar_Value{Value: params.GitMode},
		})
	}
	// M-HARNESS-COMMIT-CONTRACT: Pass site metadata for structured commit messages.
	if params.SiteSlug != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_SITE_SLUG", Values: &runpb.EnvVar_Value{Value: params.SiteSlug},
		})
	}
	if params.BriefID != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_BRIEF_ID", Values: &runpb.EnvVar_Value{Value: params.BriefID},
		})
	}
	// M-PKG-AUTONOMOUS-UPDATES: Pass monorepo subdirectory for package agents.
	if params.Subdirectory != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_SUBDIRECTORY", Values: &runpb.EnvVar_Value{Value: params.Subdirectory},
		})
	}
	// M-PKG-CASCADE-DETERMINISTIC-FIRST: cascade envelope env vars. The Cloud
	// Run Job wrapper reads these to choose deterministic-bump vs AI-escalation.
	if params.RootPackage != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_CASCADE_ROOT_PACKAGE", Values: &runpb.EnvVar_Value{Value: params.RootPackage},
		})
	}
	if params.RootChangeClass != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_CASCADE_CHANGE_CLASS", Values: &runpb.EnvVar_Value{Value: params.RootChangeClass},
		})
	}
	if params.FromVersion != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_CASCADE_FROM_VERSION", Values: &runpb.EnvVar_Value{Value: params.FromVersion},
		})
	}
	if params.ToVersion != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_CASCADE_TO_VERSION", Values: &runpb.EnvVar_Value{Value: params.ToVersion},
		})
	}
	if params.FromInterfaceHash != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_CASCADE_FROM_INTERFACE_HASH", Values: &runpb.EnvVar_Value{Value: params.FromInterfaceHash},
		})
	}
	if params.ToInterfaceHash != "" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_CASCADE_TO_INTERFACE_HASH", Values: &runpb.EnvVar_Value{Value: params.ToInterfaceHash},
		})
	}
	if params.EffectsWidened {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_CASCADE_EFFECTS_WIDENED", Values: &runpb.EnvVar_Value{Value: "true"},
		})
	}
	// M-CLOUD-PROGRESS-TRACKING M4: Inject W3C trace context for Cloud Trace linking.
	// This propagates the coordinator's span context to the Cloud Run Job so
	// job spans appear as children of the coordinator dispatch span.
	traceCarrier := make(map[string]string)
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(traceCarrier))
	for key, value := range traceCarrier {
		envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: envKey, Values: &runpb.EnvVar_Value{Value: value},
		})
	}

	// M-CLOUD-DUAL-AUTH: Inject API key and auth mode marker for apikey mode.
	// The Cloud Run Job reads ANTHROPIC_API_KEY natively (Claude Code supports it).
	// AILANG_AUTH_MODE tells the executor to skip OAuth credentials file writing.
	if params.AuthMode == "apikey" {
		envOverrides = append(envOverrides, &runpb.EnvVar{
			Name: "AILANG_AUTH_MODE", Values: &runpb.EnvVar_Value{Value: "apikey"},
		})
		if params.APIKey != "" {
			envOverrides = append(envOverrides, &runpb.EnvVar{
				Name: "ANTHROPIC_API_KEY", Values: &runpb.EnvVar_Value{Value: params.APIKey},
			})
		}
	}

	req := &runpb.RunJobRequest{
		Name: jobName,
		Overrides: &runpb.RunJobRequest_Overrides{
			ContainerOverrides: []*runpb.RunJobRequest_Overrides_ContainerOverride{{
				Env: envOverrides,
			}},
		},
	}

	// RunJob returns a long-running operation. We only check the initial error —
	// job completion is reported via Pub/Sub completions topic, not by polling.
	_, err = d.client.RunJob(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to trigger Cloud Run Job %s: %w", jobName, err)
	}

	return nil
}
