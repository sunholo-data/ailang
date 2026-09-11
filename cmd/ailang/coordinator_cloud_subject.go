package main

import (
	"encoding/json"
	"strings"
	"unicode"
)

// Turning a task directive into a subject line a person can read.
//
// The commit subject was `Task <id>: <the entire directive>` — the whole prompt,
// on one line, with no length bound. What that produced, measured 2026-09-11:
//
//	PR #1143  Task task-080f4657: {"project":"ailang","target":"ailang","request":"…
//	PR #1033  Task task-88a9fa95: `ailang lock` stamps a wall-clock `generated_at` …
//
// The second reads well only because that request happened to arrive as prose.
// A structured request lands as raw JSON in the subject line, and the PR title —
// derived from nothing at all — was `[agent] design-doc-creator: task-080f4657`.
// Neither says what the change is, so every PR costs a click.
//
// This is NOT a tradeoff against determinism, which was the question worth
// asking. The subject stays a pure function of committed inputs — no model, no
// network, same directive in, same subject out. Deterministic and uninformative
// were never the same constraint; the old code simply threw the informative part
// away.

// directiveSubjectMax is the subject budget. 68 leaves room for a
// `[agent] <id>: ` prefix inside git's conventional 72.
const directiveSubjectMax = 68

// jsonSubjectKeys are the fields a structured request might carry its human
// description in, most specific first.
//
// Ordered rather than "first string field": a JSON object's key order is not
// stable through Go's decoder, so picking arbitrarily would make the subject
// non-deterministic for the same input — the one property this must not lose.
var jsonSubjectKeys = []string{"request", "title", "summary", "description", "task", "goal"}

// summarizeDirective derives a short, human subject from a task directive.
//
// Returns "" when nothing usable can be derived; callers fall back to the task
// id rather than inventing a description.
func summarizeDirective(directive string) string {
	s := strings.TrimSpace(directive)
	if s == "" {
		return ""
	}

	// A structured request: pull the human field out rather than printing the
	// payload. Tried before line-splitting, because pretty-printed JSON would
	// otherwise yield "{" as its first line.
	if extracted := subjectFromJSON(s); extracted != "" {
		s = extracted
	}

	// An already-prefixed directive would otherwise produce
	// "Task X: Task X: …" once the caller adds its own prefix.
	s = stripTaskPrefix(s)

	// First non-empty line: a directive's later paragraphs are context, and a
	// subject that swallows them is the unbounded case all over again.
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			s = t
			break
		}
	}

	s = strings.Join(strings.Fields(s), " ") // collapse runs of whitespace
	s = strings.TrimSpace(stripTaskPrefix(s))
	if s == "" {
		return ""
	}
	return truncateOnWord(s, directiveSubjectMax)
}

// subjectFromJSON returns a human field from a JSON object directive, or "".
func subjectFromJSON(s string) string {
	if !strings.HasPrefix(s, "{") {
		return ""
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		// Not valid JSON after all — a directive that merely starts with a
		// brace is prose, and prose is handled by the caller.
		return ""
	}
	for _, k := range jsonSubjectKeys {
		if v, ok := obj[k].(string); ok {
			if t := strings.TrimSpace(v); t != "" {
				return t
			}
		}
	}
	return ""
}

// stripTaskPrefix removes a leading "Task <id>: ".
func stripTaskPrefix(s string) string {
	const marker = "Task "
	if !strings.HasPrefix(s, marker) {
		return s
	}
	rest := s[len(marker):]
	colon := strings.Index(rest, ": ")
	// Bounded: a long "Task …" sentence that merely contains a colon later on is
	// prose, not a prefix, and must survive intact.
	if colon <= 0 || colon > 48 {
		return s
	}
	return strings.TrimSpace(rest[colon+2:])
}

// truncateOnWord shortens to at most max runes, breaking at a word boundary.
func truncateOnWord(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	cut := r[:max]
	// Back up to the last space so a subject never ends mid-word.
	for i := len(cut) - 1; i > max/2; i-- {
		if unicode.IsSpace(cut[i]) {
			return strings.TrimRight(string(cut[:i]), " \t.,;:—-") + "…"
		}
	}
	return strings.TrimRight(string(cut), " \t.,;:—-") + "…"
}

// agentPRTitle is the PR title: descriptive when we can be, and always
// deterministic.
//
// The task id moves into the body. It is a lookup key, not a description, and
// it was occupying the one line a reader actually sees.
func agentPRTitle(agentID, taskID, directive string) string {
	if subject := summarizeDirective(directive); subject != "" {
		return "[agent] " + agentID + ": " + subject
	}
	return "[agent] " + agentID + ": " + taskID
}

// agentCommitSubject is the git subject line for the wrapper's commit.
func agentCommitSubject(taskID, directive string) string {
	if subject := summarizeDirective(directive); subject != "" {
		return subject
	}
	return "Task " + taskID
}
