package eval_analyzer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/sunholo/ailang/internal/eval_harness"
)

// DesignDocData contains all data for rendering a design document
type DesignDocData struct {
	// Header
	Title          string
	Date           string
	Frequency      int
	BenchmarkCount int
	Priority       string
	EstimatedLOC   string
	EstimatedTime  string
	Category       string
	Impact         string

	// Evidence
	Benchmarks        string
	Models            string
	TotalFailures     int
	FailurePercentage string
	ErrorExamples     []ErrorExample

	// Analysis
	ProblemStatement   string
	RootCause          string
	Solution           string
	ImplementationPlan string

	// Technical Design
	APIChanges        string
	TypeSystemChanges string
	RuntimeChanges    string

	// Implementation
	Tasks []Task

	// Testing
	UnitTests        string
	IntegrationTests string
	NewBenchmarks    string
	SuccessCriteria  []string

	// References
	SimilarFeatures   string
	RelatedDesignDocs string

	// Impact
	SuccessRateBefore     string
	TokenEfficiencyBefore string
	SuccessRateAfter      string
	TokenEfficiencyAfter  string

	// Metadata
	GeneratedDate  string
	GeneratorModel string
}

// ErrorExample represents a single error case in the design doc
type ErrorExample struct {
	Index int
	Error string
	Code  string
	Lang  string
}

// Task represents an implementation task
type Task struct {
	Number      int
	Title       string
	LOC         string
	Time        string
	Description string
}

// DesignGenerator generates design documents from issue reports
type DesignGenerator struct {
	aiAgent  *eval_harness.AIAgent
	model    string
	template *template.Template
}

// NewDesignGenerator creates a new design document generator
func NewDesignGenerator(model string, seed int64) (*DesignGenerator, error) {
	// Create AI agent for GPT-5
	agent, err := eval_harness.NewAIAgent(model, seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI agent: %w", err)
	}

	// Load template
	tmplPath := filepath.Join("internal", "eval_analyzer", "templates", "design_template.md")
	tmplData, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("design").Parse(string(tmplData))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &DesignGenerator{
		aiAgent:  agent,
		model:    model,
		template: tmpl,
	}, nil
}

// Generate creates a design document from an issue report
func (g *DesignGenerator) Generate(ctx context.Context, issue IssueReport, totalFailures int) (string, error) {
	// Build prompt for GPT-5 to analyze the issue and generate design content
	prompt := g.buildPrompt(issue, totalFailures)

	// Call GPT-5
	result, err := g.aiAgent.GenerateCode(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate design content: %w", err)
	}

	// Parse GPT-5 output into structured data
	data := g.parseGPTOutput(result.Code, issue, totalFailures)

	// Render template
	var buf bytes.Buffer
	if err := g.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

// buildPrompt constructs the prompt for GPT-5
func (g *DesignGenerator) buildPrompt(issue IssueReport, totalFailures int) string {
	// Load context files
	claudeMd := loadFile("CLAUDE.md")
	readmeMd := loadFile("README.md")

	// Find similar design docs
	similarDocs := g.findSimilarDesigns(issue.Category)

	prompt := fmt.Sprintf(`You are an expert programming language designer working on AILANG, an AI-first functional programming language.

# Context

## AILANG Overview & Architecture
%s

## Current Implementation Status & Roadmap
%s

## Reference: Similar Implemented Features

The following are examples of completed design docs from similar features. Use these as templates for structure, level of detail, and implementation approach.

%s

# Task

Analyze the following issue discovered from AI evaluation benchmarks and create a detailed design document.

## Issue Report

**Category**: %s
**Title**: %s
**Language**: %s
**Frequency**: %d failures out of %d total failures (%.1f%%)
**Affected Benchmarks**: %s
**Models**: %s

### Error Examples

%s

### Failed Code Examples

%s

# Instructions

Generate a comprehensive design document with the following sections:

1. **Problem Statement** (2-3 paragraphs)
   - What specifically is failing?
   - Why is it failing (root cause)?
   - What are the user expectations?

2. **Root Cause Analysis** (1-2 paragraphs)
   - Technical explanation of why this pattern fails
   - What's missing from the language implementation?

3. **Proposed Solution** (2-3 paragraphs)
   - High-level approach to fix
   - Design decisions and tradeoffs
   - How it fits into AILANG's architecture

4. **Implementation Plan** (5-10 specific tasks)
   - Break down into concrete implementation steps
   - Estimate LOC for each task
   - Estimate time (hours/days)
   - Specify which files/packages to modify

5. **Testing Strategy**
   - Unit tests needed
   - Integration tests needed
   - New benchmarks to prevent regression

6. **Success Criteria** (5-8 specific checkboxes)
   - Measurable outcomes
   - What "done" looks like

7. **Estimated Impact**
   - Projected improvement in AI success rate
   - Projected improvement in token efficiency

Output ONLY the content for these sections in markdown format. Do NOT include the template structure, just the content that will fill in {{.ProblemStatement}}, {{.RootCause}}, etc.

Use this format:

PROBLEM_STATEMENT:
[your content here]

ROOT_CAUSE:
[your content here]

SOLUTION:
[your content here]

IMPLEMENTATION_PLAN:
[your content here]

TASKS:
1. Task Name (~LOC, time) - Description
2. Task Name (~LOC, time) - Description
...

UNIT_TESTS:
[your content here]

INTEGRATION_TESTS:
[your content here]

NEW_BENCHMARKS:
[your content here]

SUCCESS_CRITERIA:
- [ ] Criterion 1
- [ ] Criterion 2
...

ESTIMATED_LOC:
[total estimated lines of code]

ESTIMATED_TIME:
[total estimated time]

API_CHANGES:
[description or "None"]

TYPE_SYSTEM_CHANGES:
[description or "None"]

RUNTIME_CHANGES:
[description or "None"]

SUCCESS_RATE_BEFORE:
[percentage]

TOKEN_EFFICIENCY_BEFORE:
[description]

SUCCESS_RATE_AFTER:
[percentage]

TOKEN_EFFICIENCY_AFTER:
[description]
`,
		truncate(claudeMd, 8000),    // Increased: Full CLAUDE.md context is critical
		truncate(readmeMd, 4000),    // Increased: More implementation status
		truncate(similarDocs, 3000), // Increased: Full design doc examples
		issue.Category,
		issue.Title,
		issue.Lang,
		issue.Frequency,
		totalFailures,
		float64(issue.Frequency)/float64(totalFailures)*100.0,
		strings.Join(issue.Benchmarks, ", "),
		strings.Join(issue.Models, ", "),
		g.formatErrors(issue.ErrorMessages),
		g.formatCode(issue.Examples, issue.Lang),
	)

	return prompt
}

// parseGPTOutput extracts structured data from GPT-5's response
func (g *DesignGenerator) parseGPTOutput(output string, issue IssueReport, totalFailures int) *DesignDocData {
	// Parse sections from GPT output
	sections := parseSections(output)

	// Build error examples
	errorExamples := []ErrorExample{}
	for i, errMsg := range issue.ErrorMessages {
		code := ""
		if i < len(issue.Examples) {
			code = issue.Examples[i]
		}
		errorExamples = append(errorExamples, ErrorExample{
			Index: i + 1,
			Error: truncate(errMsg, 500),
			Code:  truncate(code, 1000),
			Lang:  issue.Lang,
		})
		if i >= 2 { // Limit to 3 examples
			break
		}
	}

	// Parse tasks
	tasks := parseTasks(sections["TASKS"])

	// Parse success criteria
	successCriteria := parseCheckboxes(sections["SUCCESS_CRITERIA"])

	// Calculate priority
	priority := calculatePriority(issue.Impact, issue.Frequency, totalFailures)

	return &DesignDocData{
		Title:                 issue.Title,
		Date:                  time.Now().Format("2006-01-02"),
		Frequency:             issue.Frequency,
		BenchmarkCount:        len(issue.Benchmarks),
		Priority:              priority,
		EstimatedLOC:          sections["ESTIMATED_LOC"],
		EstimatedTime:         sections["ESTIMATED_TIME"],
		Category:              issue.Category,
		Impact:                issue.Impact,
		Benchmarks:            strings.Join(issue.Benchmarks, ", "),
		Models:                strings.Join(issue.Models, ", "),
		TotalFailures:         totalFailures,
		FailurePercentage:     fmt.Sprintf("%.1f", float64(issue.Frequency)/float64(totalFailures)*100.0),
		ErrorExamples:         errorExamples,
		ProblemStatement:      sections["PROBLEM_STATEMENT"],
		RootCause:             sections["ROOT_CAUSE"],
		Solution:              sections["SOLUTION"],
		ImplementationPlan:    sections["IMPLEMENTATION_PLAN"],
		APIChanges:            sections["API_CHANGES"],
		TypeSystemChanges:     sections["TYPE_SYSTEM_CHANGES"],
		RuntimeChanges:        sections["RUNTIME_CHANGES"],
		Tasks:                 tasks,
		UnitTests:             sections["UNIT_TESTS"],
		IntegrationTests:      sections["INTEGRATION_TESTS"],
		NewBenchmarks:         sections["NEW_BENCHMARKS"],
		SuccessCriteria:       successCriteria,
		SimilarFeatures:       g.findSimilarFeatures(issue.Category),
		RelatedDesignDocs:     g.findRelatedDocs(issue.Category),
		SuccessRateBefore:     sections["SUCCESS_RATE_BEFORE"],
		TokenEfficiencyBefore: sections["TOKEN_EFFICIENCY_BEFORE"],
		SuccessRateAfter:      sections["SUCCESS_RATE_AFTER"],
		TokenEfficiencyAfter:  sections["TOKEN_EFFICIENCY_AFTER"],
		GeneratedDate:         time.Now().Format("2006-01-02 15:04:05"),
		GeneratorModel:        g.model,
	}
}

// Helper functions

func (g *DesignGenerator) formatErrors(errors []string) string {
	if len(errors) == 0 {
		return "No error messages captured"
	}

	var buf bytes.Buffer
	for i, err := range errors {
		buf.WriteString(fmt.Sprintf("**Error %d:**\n```\n%s\n```\n\n", i+1, truncate(err, 500)))
		if i >= 2 { // Limit to 3 examples
			break
		}
	}
	return buf.String()
}

func (g *DesignGenerator) formatCode(examples []string, lang string) string {
	if len(examples) == 0 {
		return "No code examples captured"
	}

	var buf bytes.Buffer
	for i, code := range examples {
		buf.WriteString(fmt.Sprintf("**Example %d:**\n```%s\n%s\n```\n\n", i+1, lang, truncate(code, 1000)))
		if i >= 2 { // Limit to 3 examples
			break
		}
	}
	return buf.String()
}

func (g *DesignGenerator) findSimilarDesigns(category string) string {
	// Load relevant implemented design docs for context
	implementedDir := "design_docs/implemented"

	var docs []string

	// Key reference implementations to always include
	keyDocs := []string{
		"v0_3_0/M-R4_recursion.md",
		"v0_3_0/M-R5_records.md",
		"v0_2_0/m_r2_effect_system.md",
		"v0_3_0/M-R8_block_expressions.md",
	}

	for _, docPath := range keyDocs {
		fullPath := filepath.Join(implementedDir, docPath)
		content := loadFile(fullPath)
		if content != "" && !strings.Contains(content, "Could not load") {
			// Extract key sections: Problem Statement + Implementation Plan
			summary := extractDesignDocSummary(content)
			docs = append(docs, fmt.Sprintf("### %s\n\n%s\n", filepath.Base(docPath), summary))
		}
	}

	// If category-specific docs needed, search by category
	if category == "type_error" || category == "compile_error" {
		// Add type system docs
		typeDoc := loadFile(filepath.Join(implementedDir, "v0_3_0/M-R7_type_fixes.md"))
		if typeDoc != "" && !strings.Contains(typeDoc, "Could not load") {
			docs = append(docs, fmt.Sprintf("**M-R7_type_fixes.md**:\n%s\n", truncate(typeDoc, 300)))
		}
	}

	if len(docs) == 0 {
		return "See design_docs/implemented/ for reference implementations (M-R4: Recursion, M-R5: Records, M-R2: Effects)"
	}

	return strings.Join(docs, "\n---\n")
}

func (g *DesignGenerator) findSimilarFeatures(category string) string {
	return "See design_docs/implemented/ for reference implementations"
}

func (g *DesignGenerator) findRelatedDocs(category string) string {
	return "CLAUDE.md, README.md, design_docs/planned/v0_4_0_net_enhancements.md"
}

// extractDesignDocSummary extracts key sections from a design doc
func extractDesignDocSummary(content string) string {
	lines := strings.Split(content, "\n")

	var result []string
	inSection := false
	sectionName := ""
	var sectionLines []string

	// Extract Problem Statement and Implementation Plan sections
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for section headers
		if strings.HasPrefix(trimmed, "## Problem Statement") {
			inSection = true
			sectionName = "Problem"
			sectionLines = []string{}
		} else if strings.HasPrefix(trimmed, "## Implementation Plan") || strings.HasPrefix(trimmed, "## Design") {
			// Save previous section
			if len(sectionLines) > 0 {
				result = append(result, fmt.Sprintf("**%s**: %s", sectionName, strings.Join(sectionLines, " ")))
			}
			inSection = true
			sectionName = "Implementation"
			sectionLines = []string{}
		} else if strings.HasPrefix(trimmed, "## ") {
			// End of section
			if len(sectionLines) > 0 {
				result = append(result, fmt.Sprintf("**%s**: %s", sectionName, strings.Join(sectionLines, " ")))
			}
			inSection = false
			sectionLines = []string{}
		} else if inSection && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			// Accumulate section content
			sectionLines = append(sectionLines, trimmed)

			// Limit to first 3 sentences
			if len(sectionLines) >= 3 {
				result = append(result, fmt.Sprintf("**%s**: %s", sectionName, strings.Join(sectionLines, " ")))
				inSection = false
				sectionLines = []string{}
			}
		}
	}

	if len(result) == 0 {
		// Fallback: just return first 500 chars
		return truncate(content, 500)
	}

	return truncate(strings.Join(result, "\n\n"), 800)
}

func loadFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("(Could not load %s)", path)
	}
	return string(data)
}

func parseSections(output string) map[string]string {
	sections := make(map[string]string)

	// Split by section markers
	lines := strings.Split(output, "\n")
	currentSection := ""
	var currentContent []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a section header
		if strings.HasSuffix(trimmed, ":") && len(strings.Fields(trimmed)) <= 3 {
			// Save previous section
			if currentSection != "" {
				sections[currentSection] = strings.TrimSpace(strings.Join(currentContent, "\n"))
			}

			// Start new section
			currentSection = strings.TrimSuffix(trimmed, ":")
			currentContent = []string{}
		} else if currentSection != "" {
			currentContent = append(currentContent, line)
		}
	}

	// Save last section
	if currentSection != "" {
		sections[currentSection] = strings.TrimSpace(strings.Join(currentContent, "\n"))
	}

	return sections
}

func parseTasks(tasksText string) []Task {
	var tasks []Task

	lines := strings.Split(tasksText, "\n")
	taskNum := 1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Parse: "1. Task Name (~LOC, time) - Description"
		// Simple extraction for now
		tasks = append(tasks, Task{
			Number:      taskNum,
			Title:       trimmed,
			LOC:         "TBD",
			Time:        "TBD",
			Description: "",
		})
		taskNum++
	}

	return tasks
}

func parseCheckboxes(criteriaText string) []string {
	var criteria []string

	lines := strings.Split(criteriaText, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [ ]") {
			criteria = append(criteria, strings.TrimSpace(strings.TrimPrefix(trimmed, "- [ ]")))
		}
	}

	return criteria
}

func calculatePriority(impact string, frequency int, totalFailures int) string {
	percentage := float64(frequency) / float64(totalFailures) * 100.0

	if impact == "critical" || percentage > 50.0 {
		return "P0 (Critical - Must Ship)"
	} else if impact == "high" || percentage > 25.0 {
		return "P1 (High Priority)"
	} else if impact == "medium" || percentage > 10.0 {
		return "P2 (Medium Priority)"
	}

	return "P3 (Low Priority)"
}
