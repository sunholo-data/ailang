package messaging

import (
	"fmt"
	"strings"
)

// EnvelopeOption is a functional option for building an envelope.
type EnvelopeOption func(b *EnvelopeBuilder)

// EnvelopeBuilder computes envelope vectors from contextual inputs.
// Usage:
//
//	env, err := NewEnvelopeBuilder(embedder).
//	    WithCodeContext([]string{"internal/types/unify.go"}, []string{"func unify(a, b Type)..."}).
//	    WithResolution("fix: handle recursive types", "diff --git a/...").
//	    Build(msg)
type EnvelopeBuilder struct {
	embedder Embedder

	// Slot source data (populated by With* methods)
	codeFiles    []string
	codeSnippets []string

	contextFiles  []string
	contextErrors []string
	contextTools  []string

	skillPhases   []string
	skillNodes    []string
	skillPatterns []string

	resolutionCommitMsg string
	resolutionDiff      string
}

// NewEnvelopeBuilder creates a builder that will compute envelope vectors using the given embedder.
func NewEnvelopeBuilder(embedder Embedder) *EnvelopeBuilder {
	return &EnvelopeBuilder{embedder: embedder}
}

// WithCodeContext sets file paths and code snippets for the "code" envelope slot.
// This captures what code is affected by the message.
func (b *EnvelopeBuilder) WithCodeContext(filePaths []string, codeSnippets []string) *EnvelopeBuilder {
	b.codeFiles = filePaths
	b.codeSnippets = codeSnippets
	return b
}

// WithSessionContext sets recent files, errors, and tools for the "context" slot.
// This captures what the sender was working on.
func (b *EnvelopeBuilder) WithSessionContext(recentFiles, recentErrors, recentTools []string) *EnvelopeBuilder {
	b.contextFiles = recentFiles
	b.contextErrors = recentErrors
	b.contextTools = recentTools
	return b
}

// WithSkillHints sets compiler phases, AST node types, and file patterns for the "skill" slot.
// This captures what expertise is needed to handle the message.
func (b *EnvelopeBuilder) WithSkillHints(phases, nodeTypes, filePatterns []string) *EnvelopeBuilder {
	b.skillPhases = phases
	b.skillNodes = nodeTypes
	b.skillPatterns = filePatterns
	return b
}

// WithResolution sets the commit message and diff for the "resolution" slot.
// This captures how a task was resolved (called post-completion).
func (b *EnvelopeBuilder) WithResolution(commitMsg, diff string) *EnvelopeBuilder {
	b.resolutionCommitMsg = commitMsg
	b.resolutionDiff = diff
	return b
}

// Build computes the envelope for the given message.
// The "intent" slot is always computed from title + payload prefix.
// Other slots are only computed if the corresponding With* method was called.
func (b *EnvelopeBuilder) Build(msg *InboxMessage) (*Envelope, error) {
	if b.embedder == nil {
		return nil, fmt.Errorf("embedder is required to build envelope")
	}

	env := NewEnvelope()
	model := b.embedder.ModelName()

	// Intent: always computed from title + first 200 chars of payload
	intentText := msg.Title
	if msg.Payload != "" {
		suffix := msg.Payload
		if len(suffix) > 200 {
			suffix = suffix[:200]
		}
		intentText += " " + suffix
	}
	intentVec, err := b.embedder.Embed(intentText)
	if err != nil {
		return nil, fmt.Errorf("failed to embed intent: %w", err)
	}
	env.Set(SlotIntent, intentVec, model)

	// Code: from file paths + code snippets
	if len(b.codeFiles) > 0 || len(b.codeSnippets) > 0 {
		codeText := buildCodeText(b.codeFiles, b.codeSnippets)
		if codeText != "" {
			codeVec, err := b.embedder.Embed(codeText)
			if err == nil {
				env.Set(SlotCode, codeVec, model)
			}
		}
	}

	// Context: from recent files, errors, tools
	if len(b.contextFiles) > 0 || len(b.contextErrors) > 0 || len(b.contextTools) > 0 {
		ctxText := buildContextText(b.contextFiles, b.contextErrors, b.contextTools)
		if ctxText != "" {
			ctxVec, err := b.embedder.Embed(ctxText)
			if err == nil {
				env.Set(SlotContext, ctxVec, model)
			}
		}
	}

	// Skill: from phases, node types, file patterns
	if len(b.skillPhases) > 0 || len(b.skillNodes) > 0 || len(b.skillPatterns) > 0 {
		skillText := buildSkillText(b.skillPhases, b.skillNodes, b.skillPatterns)
		if skillText != "" {
			skillVec, err := b.embedder.Embed(skillText)
			if err == nil {
				env.Set(SlotSkill, skillVec, model)
			}
		}
	}

	// Resolution: from commit message + diff
	if b.resolutionCommitMsg != "" || b.resolutionDiff != "" {
		resText := buildResolutionText(b.resolutionCommitMsg, b.resolutionDiff)
		if resText != "" {
			resVec, err := b.embedder.Embed(resText)
			if err == nil {
				env.Set(SlotResolution, resVec, model)
			}
		}
	}

	return env, nil
}

// buildCodeText creates embeddable text from code context.
func buildCodeText(files, snippets []string) string {
	var parts []string
	if len(files) > 0 {
		parts = append(parts, "Files: "+strings.Join(files, ", "))
	}
	for _, s := range snippets {
		if len(s) > 1000 {
			s = s[:1000]
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "\n")
}

// buildContextText creates embeddable text from session context.
func buildContextText(files, errors, tools []string) string {
	var parts []string
	if len(files) > 0 {
		parts = append(parts, "Recent files: "+strings.Join(files, ", "))
	}
	if len(errors) > 0 {
		parts = append(parts, "Recent errors: "+strings.Join(errors, "; "))
	}
	if len(tools) > 0 {
		parts = append(parts, "Recent tools: "+strings.Join(tools, ", "))
	}
	return strings.Join(parts, "\n")
}

// buildSkillText creates embeddable text from skill hints.
func buildSkillText(phases, nodes, patterns []string) string {
	var parts []string
	if len(phases) > 0 {
		parts = append(parts, "Compiler phases: "+strings.Join(phases, ", "))
	}
	if len(nodes) > 0 {
		parts = append(parts, "AST node types: "+strings.Join(nodes, ", "))
	}
	if len(patterns) > 0 {
		parts = append(parts, "File patterns: "+strings.Join(patterns, ", "))
	}
	return strings.Join(parts, "\n")
}

// buildResolutionText creates embeddable text from resolution data.
func buildResolutionText(commitMsg, diff string) string {
	var parts []string
	if commitMsg != "" {
		parts = append(parts, "Commit: "+commitMsg)
	}
	if diff != "" {
		// Truncate diff to keep embedding input reasonable
		if len(diff) > 2000 {
			diff = diff[:2000]
		}
		parts = append(parts, "Diff:\n"+diff)
	}
	return strings.Join(parts, "\n")
}
