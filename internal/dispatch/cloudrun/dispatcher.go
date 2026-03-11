// Package cloudrun implements the CloudDispatcher interface using
// the Cloud Run Jobs Admin API v2. It triggers job executions with
// per-execution environment variable overrides.
package cloudrun

import (
	"context"
	"fmt"

	run "cloud.google.com/go/run/apiv2"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/googleapis/gax-go/v2"

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
	jobName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s-agent-executor",
		d.projectID, d.region, d.prefix)

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
