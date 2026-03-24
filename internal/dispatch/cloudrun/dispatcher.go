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

	"github.com/sunholo/ailang/internal/coordinator"
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

// Dispatch triggers a Cloud Run Job execution with per-execution env var overrides.
// The job is identified by the pattern: projects/{project}/locations/{region}/jobs/{prefix}-agent-executor
// This matches the Terraform-defined job name in cloud_run_jobs.tf.
func (d *Dispatcher) Dispatch(ctx context.Context, params coordinator.DispatchParams) error {
	// M-CLOUD-DUAL-AUTH: Select job template based on auth mode.
	// "apikey" uses agent-executor-apikey (no OAuth token, user provides API key).
	// Default uses agent-executor (OAuth credentials from Secret Manager).
	jobSuffix := "agent-executor"
	if params.AuthMode == "apikey" {
		jobSuffix = "agent-executor-apikey"
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
	_, err := d.client.RunJob(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to trigger Cloud Run Job %s: %w", jobName, err)
	}

	return nil
}
