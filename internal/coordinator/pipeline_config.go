package coordinator

import "fmt"

// M-PIPELINE-RECONCILIATION M4 (decision D2, ratified 2026-08-26):
// chain-as-data.
//
// Before this, adding a project to the dev pipeline meant cloning three agent
// entries and hand-rewiring their trigger_on_complete chain. The stapledon-*
// and twilight-* sextet was exactly that — six entries, zero messages ever
// received, drifting from the primary chain (no model pins, no approval
// labels) because nobody edits six copies in lockstep. A pipeline declares
// the stages ONCE; adding a project is one binding line.

// PipelineStage is one stage template. The expanded agent id and inbox are
// "<project>-<suffix>"; the label is "<binding label> <label_suffix>".
type PipelineStage struct {
	Suffix           string   `yaml:"suffix" json:"suffix"`
	LabelSuffix      string   `yaml:"label_suffix" json:"label_suffix"`
	Capabilities     []string `yaml:"capabilities" json:"capabilities"`
	Skill            string   `yaml:"skill" json:"skill"`
	Provider         string   `yaml:"provider" json:"provider,omitempty"` // default "claude"
	Model            string   `yaml:"model" json:"model,omitempty"`
	Timeout          string   `yaml:"timeout" json:"timeout,omitempty"`
	OutputMarkers    []string `yaml:"output_markers" json:"output_markers,omitempty"`
	ArtifactPatterns []string `yaml:"artifact_patterns" json:"artifact_patterns,omitempty"`
}

// PipelineBinding instantiates every stage of a pipeline for one project.
type PipelineBinding struct {
	Project     string `yaml:"project" json:"project"`
	Label       string `yaml:"label" json:"label"`
	Workspace   string `yaml:"workspace" json:"workspace"`
	MergeBranch string `yaml:"merge_branch" json:"merge_branch,omitempty"`
	// ExecutionLane / Repo pass through to every expanded agent (M3 fields).
	ExecutionLane ExecutionLane `yaml:"execution_lane" json:"execution_lane,omitempty"`
	Repo          string        `yaml:"repo" json:"repo,omitempty"`
}

// PipelineConfig is one named chain plus its project bindings.
type PipelineConfig struct {
	Name     string            `yaml:"name" json:"name"`
	Stages   []PipelineStage   `yaml:"stages" json:"stages"`
	Bindings []PipelineBinding `yaml:"bindings" json:"bindings"`
}

// ExpandPipelines materializes every binding of every pipeline into concrete
// AgentConfigs — the same shape the literal entries had, so nothing downstream
// changes. Collisions with literal agent ids are ERRORS: a pipeline silently
// shadowing (or shadowed by) a hand-written entry is exactly the ambiguity
// this feature deletes.
func (c *CoordinatorConfig) ExpandPipelines() ([]*AgentConfig, error) {
	literal := make(map[string]bool, len(c.Agents))
	for _, a := range c.Agents {
		literal[a.ID] = true
	}

	var out []*AgentConfig
	seen := map[string]string{} // expanded id -> pipeline name
	for _, pl := range c.Pipelines {
		if len(pl.Stages) == 0 {
			return nil, fmt.Errorf("pipeline %q declares no stages", pl.Name)
		}
		for _, b := range pl.Bindings {
			if b.Project == "" || b.Workspace == "" {
				return nil, fmt.Errorf("pipeline %q: a binding needs project and workspace (got project=%q)", pl.Name, b.Project)
			}
			for i, st := range pl.Stages {
				if st.Suffix == "" || st.Skill == "" {
					return nil, fmt.Errorf("pipeline %q: stage %d needs suffix and skill", pl.Name, i)
				}
				id := b.Project + "-" + st.Suffix
				if literal[id] {
					return nil, fmt.Errorf("pipeline %q expands %q, which collides with a literal agent entry — remove one", pl.Name, id)
				}
				if prev, dup := seen[id]; dup {
					return nil, fmt.Errorf("pipelines %q and %q both expand agent %q", prev, pl.Name, id)
				}
				seen[id] = pl.Name

				provider := st.Provider
				if provider == "" {
					provider = "claude"
				}
				var trigger []string
				if i+1 < len(pl.Stages) {
					trigger = []string{b.Project + "-" + pl.Stages[i+1].Suffix}
				}
				out = append(out, &AgentConfig{
					ID:                 id,
					Label:              b.Label + " " + st.LabelSuffix,
					Inbox:              id,
					Workspace:          b.Workspace,
					MergeBranch:        b.MergeBranch,
					ExecutionLane:      b.ExecutionLane,
					Repo:               b.Repo,
					Capabilities:       append([]string(nil), st.Capabilities...),
					Provider:           provider,
					Model:              st.Model,
					Timeout:            st.Timeout,
					TriggerOnComplete:  trigger,
					SessionContinuity:  true,
					MaxConcurrentTasks: 1,
					Invoke:             &InvokeConfig{Type: "skill", Name: st.Skill},
					OutputMarkers:      append([]string(nil), st.OutputMarkers...),
					ArtifactPatterns:   append([]string(nil), st.ArtifactPatterns...),
				})
			}
		}
	}
	return out, nil
}
