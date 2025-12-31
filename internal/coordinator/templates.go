package coordinator

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

// CommentData contains all data available to comment templates.
type CommentData struct {
	// Task information
	TaskID    string
	Title     string
	Type      string
	Priority  int
	Stage     string
	Status    string
	Agent     string
	Workspace string

	// GitHub context
	IssueNumber int
	Repository  string

	// Timing
	StartedAt   *time.Time
	CompletedAt *time.Time
	Duration    time.Duration

	// Results
	Success      bool
	Output       string
	Error        string
	Cost         float64
	TokensUsed   int
	InputTokens  int
	OutputTokens int

	// Artifacts
	DesignDocPath  string
	SprintPlanPath string
	WorktreePath   string
	BranchName     string
	FilesCreated   []string
	FilesModified  []string

	// Custom data
	Extra map[string]interface{}
}

// CommentTemplates holds all the comment templates.
var CommentTemplates = struct {
	Working           *template.Template
	DesignDocComplete *template.Template
	SprintPlanReady   *template.Template
	ImplementComplete *template.Template
	MergeComplete     *template.Template
	NeedsRevision     *template.Template
	Error             *template.Template
}{
	Working:           template.Must(template.New("working").Parse(workingTemplate)),
	DesignDocComplete: template.Must(template.New("design").Parse(designDocCompleteTemplate)),
	SprintPlanReady:   template.Must(template.New("sprint").Parse(sprintPlanReadyTemplate)),
	ImplementComplete: template.Must(template.New("implement").Parse(implementCompleteTemplate)),
	MergeComplete:     template.Must(template.New("merge").Parse(mergeCompleteTemplate)),
	NeedsRevision:     template.Must(template.New("revision").Parse(needsRevisionTemplate)),
	Error:             template.Must(template.New("error").Parse(errorTemplate)),
}

const workingTemplate = `**🤖 Agent Working**

I've picked up this issue and am working on it.

| Field | Value |
|-------|-------|
| **Task ID** | ` + "`{{.TaskID}}`" + ` |
| **Agent** | {{.Agent}} |
| **Stage** | {{.Stage}} |
| **Status** | In Progress |

You'll receive updates as I make progress.`

const designDocCompleteTemplate = `**📋 Design Document Ready**

I've created a design document for this issue.

### Summary

| Field | Value |
|-------|-------|
| **Task ID** | ` + "`{{.TaskID}}`" + ` |
| **Design Doc** | [{{.DesignDocPath}}]({{.DesignDocPath}}) |
| **Duration** | {{.Duration}} |
{{if gt .Cost 0.0}}| **Cost** | ${{printf "%.4f" .Cost}} |{{end}}
{{if gt .TokensUsed 0}}| **Tokens** | {{.TokensUsed}} ({{.InputTokens}} in / {{.OutputTokens}} out) |{{end}}

### Next Steps

1. **Review the design document** linked above
2. **Add the ` + "`design-approved`" + ` label** to this issue to proceed to sprint planning
3. **Add the ` + "`needs-revision`" + ` label** if changes are needed

Once approved, I'll automatically create a sprint plan for implementation.`

const sprintPlanReadyTemplate = `**📊 Sprint Plan Ready**

I've created a sprint plan for implementing this feature.

### Summary

| Field | Value |
|-------|-------|
| **Task ID** | ` + "`{{.TaskID}}`" + ` |
| **Sprint Plan** | [{{.SprintPlanPath}}]({{.SprintPlanPath}}) |
| **Duration** | {{.Duration}} |
{{if gt .Cost 0.0}}| **Cost** | ${{printf "%.4f" .Cost}} |{{end}}
{{if gt .TokensUsed 0}}| **Tokens** | {{.TokensUsed}} ({{.InputTokens}} in / {{.OutputTokens}} out) |{{end}}

### Next Steps

1. **Review the sprint plan** linked above
2. **Add the ` + "`sprint-approved`" + ` label** to this issue to start implementation
3. **Add the ` + "`needs-revision`" + ` label** if changes are needed

Once approved, I'll automatically begin implementing the sprint plan.`

const implementCompleteTemplate = `**✅ Implementation Complete**

I've completed the implementation for this issue.

### Summary

| Field | Value |
|-------|-------|
| **Task ID** | ` + "`{{.TaskID}}`" + ` |
| **Branch** | ` + "`{{.BranchName}}`" + ` |
| **Worktree** | ` + "`{{.WorktreePath}}`" + ` |
| **Duration** | {{.Duration}} |
{{if gt .Cost 0.0}}| **Cost** | ${{printf "%.4f" .Cost}} |{{end}}
{{if gt .TokensUsed 0}}| **Tokens** | {{.TokensUsed}} ({{.InputTokens}} in / {{.OutputTokens}} out) |{{end}}

{{if .FilesCreated}}
### Files Created
{{range .FilesCreated}}
- ` + "`{{.}}`" + `
{{end}}
{{end}}

{{if .FilesModified}}
### Files Modified
{{range .FilesModified}}
- ` + "`{{.}}`" + `
{{end}}
{{end}}

### Next Steps

1. **Review the changes** in the branch linked above
2. **Run tests locally** if you want to verify: ` + "`git worktree list`" + `
3. **Add the ` + "`merge-approved`" + ` label** to merge the changes
4. **Add the ` + "`needs-revision`" + ` label** if changes are needed

Once approved, I'll merge the changes and close this issue.`

const mergeCompleteTemplate = `**🎉 Merged and Complete**

The implementation has been merged successfully!

### Summary

| Field | Value |
|-------|-------|
| **Task ID** | ` + "`{{.TaskID}}`" + ` |
| **Branch** | ` + "`{{.BranchName}}`" + ` (merged to dev) |
| **Total Duration** | {{.Duration}} |
{{if gt .Cost 0.0}}| **Total Cost** | ${{printf "%.4f" .Cost}} |{{end}}
{{if gt .TokensUsed 0}}| **Total Tokens** | {{.TokensUsed}} |{{end}}

### Artifacts

{{if .DesignDocPath}}- **Design Doc**: [{{.DesignDocPath}}]({{.DesignDocPath}}){{end}}
{{if .SprintPlanPath}}- **Sprint Plan**: [{{.SprintPlanPath}}]({{.SprintPlanPath}}){{end}}

### Closing

This issue has been fully addressed and is now closed.

---
*🤖 Completed autonomously by the AILANG Coordinator*`

const needsRevisionTemplate = `**⚠️ Revision Requested**

Human feedback has been received and revisions are needed.

| Field | Value |
|-------|-------|
| **Task ID** | ` + "`{{.TaskID}}`" + ` |
| **Stage** | {{.Stage}} |
| **Status** | Paused - Awaiting Revision |

### What Happens Now

The pipeline is paused. To continue:

1. **Review the feedback** and make necessary changes
2. **Remove the ` + "`needs-revision`" + ` label**
3. **Add the appropriate approval label** to resume:
   - ` + "`design-approved`" + ` for design stage
   - ` + "`sprint-approved`" + ` for sprint stage
   - ` + "`merge-approved`" + ` for implementation stage

I'll resume work once the revised artifact is approved.`

const errorTemplate = `**❌ Error Occurred**

An error occurred while processing this task.

| Field | Value |
|-------|-------|
| **Task ID** | ` + "`{{.TaskID}}`" + ` |
| **Stage** | {{.Stage}} |
| **Status** | Failed |

### Error Details

` + "```" + `
{{.Error}}
` + "```" + `

### Recovery Options

1. **Review the error message** above
2. **Check the logs** for more details
3. **Create a new task** if this one cannot be recovered
4. **Contact the maintainers** if this is a bug

---
*🤖 AILANG Coordinator encountered an error*`

// RenderComment renders a comment template with the given data.
func RenderComment(tmpl *template.Template, data *CommentData) (string, error) {
	if data == nil {
		return "", fmt.Errorf("comment data is nil")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// RenderWorkingComment renders the "working" status comment.
func RenderWorkingComment(taskID, agent, stage string) (string, error) {
	return RenderComment(CommentTemplates.Working, &CommentData{
		TaskID: taskID,
		Agent:  agent,
		Stage:  stage,
	})
}

// RenderDesignDocComment renders the design doc complete comment.
func RenderDesignDocComment(data *CommentData) (string, error) {
	return RenderComment(CommentTemplates.DesignDocComplete, data)
}

// RenderSprintPlanComment renders the sprint plan ready comment.
func RenderSprintPlanComment(data *CommentData) (string, error) {
	return RenderComment(CommentTemplates.SprintPlanReady, data)
}

// RenderImplementCompleteComment renders the implementation complete comment.
func RenderImplementCompleteComment(data *CommentData) (string, error) {
	return RenderComment(CommentTemplates.ImplementComplete, data)
}

// RenderMergeCompleteComment renders the merge complete comment.
func RenderMergeCompleteComment(data *CommentData) (string, error) {
	return RenderComment(CommentTemplates.MergeComplete, data)
}

// RenderRevisionComment renders the needs revision comment.
func RenderRevisionComment(taskID, stage string) (string, error) {
	return RenderComment(CommentTemplates.NeedsRevision, &CommentData{
		TaskID: taskID,
		Stage:  stage,
	})
}

// RenderErrorComment renders the error comment.
func RenderErrorComment(taskID, stage, errMsg string) (string, error) {
	return RenderComment(CommentTemplates.Error, &CommentData{
		TaskID: taskID,
		Stage:  stage,
		Error:  errMsg,
	})
}
