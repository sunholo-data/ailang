package main

// The agent's own account of what it did, carried on the completion.
//
// A `no_changes` completion used to say only that nothing happened. On
// 2026-09-14 sprint-executor spent 33,737 input tokens establishing that it
// could not start, said so in one clear sentence, and the message that reached
// the plane carried an empty error_msg and changed_files:null. The reason was
// in a GCS transcript nobody was going to read.
//
// An agent that declines for a GOOD reason must not look like one that silently
// did nothing. That distinction is most of what trusting a pipeline is.

import "strings"

// completionSummaryMax bounds what rides on every completion message.
//
// 1200 bytes: enough for a conclusion plus the output markers under it, small
// enough that a burst of completions cannot become the reason the message plane
// is slow. The full transcript stays in GCS, named by ArtifactGCSPath.
const completionSummaryMax = 1200

// completionSummary takes the TAIL of the transcript, because an agent's
// conclusion is at the end — the beginning is it reading CLAUDE.md.
//
// Cut on a line boundary so the summary never opens mid-sentence, and mark the
// cut, because a silently truncated explanation is its own small lie.
func completionSummary(transcript string) string {
	t := strings.TrimSpace(transcript)
	if t == "" {
		return ""
	}
	if len(t) <= completionSummaryMax {
		return t
	}
	tail := t[len(t)-completionSummaryMax:]
	// Forward to the next newline so the first line is whole. If the tail has no
	// newline at all it is one long line; keep it rather than returning nothing.
	if i := strings.IndexByte(tail, '\n'); i >= 0 && i < len(tail)-1 {
		tail = tail[i+1:]
	}
	return "…(transcript truncated; full text at the artifact path)\n" + strings.TrimSpace(tail)
}
