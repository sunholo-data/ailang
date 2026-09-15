package observatory

import "strings"

// The ONE span-name classifier and the ONE flat-list → tree linker. Before
// M-V1-SIMPLIFY-S3 M5 three classifiers (exact-match here, prefix-match in
// internal/server, substring-match in cmd/ailang) and four linkers disagreed
// about what a "turn" or a "tool" span was called, and one linker walked a
// map, so `ailang trace local` printed roots in a different order every run.

// ClassifySpanNodeType maps a span name onto the hierarchy node types the
// UI and CLI render. Rules are the union of what the three classifiers
// agreed on, keyed on the span names the executors actually emit:
//
//	coordinator.*                                 → coordinator
//	<harness>.execute (claude, gemini, pi, opencode, motoko, codex,
//	  managed_agents, cloud_job) and ailang.exec   → executor
//	exec.turn, exec.turn.N, turn.*                → turn
//	exec.tool_use*, claude_code.tool.*, *.tool.*, tool.* → tool
//	anything else                                 → other
func ClassifySpanNodeType(name string) SpanHierarchyNodeType {
	switch {
	case strings.HasPrefix(name, "coordinator."):
		return NodeTypeCoordinator
	case strings.HasSuffix(name, ".execute") || name == "ailang.exec":
		return NodeTypeExecutor
	case name == "exec.turn" || strings.HasPrefix(name, "exec.turn.") || strings.HasPrefix(name, "turn."):
		return NodeTypeTurn
	case strings.HasPrefix(name, "exec.tool_use") || strings.HasPrefix(name, "exec.tool") ||
		strings.HasPrefix(name, "tool.") || strings.Contains(name, ".tool."):
		return NodeTypeTool
	default:
		return NodeTypeOther
	}
}

// LinkSpans threads a flat span list into a forest by ParentSpanID.
//
// newNode builds the caller's node for a span; addChild attaches child to
// parent. A span with no parent, or whose parent is not in the list, is a
// root. Roots keep the input order and children keep the input order under
// their parent, so a sorted query stays sorted. byID indexes every node.
func LinkSpans[N any](spans []*Span, newNode func(*Span) N, addChild func(parent, child N)) (roots []N, byID map[string]N) {
	if len(spans) == 0 {
		return nil, nil
	}
	byID = make(map[string]N, len(spans))
	for _, span := range spans {
		byID[span.ID] = newNode(span)
	}
	for _, span := range spans {
		node := byID[span.ID]
		if span.ParentSpanID == "" {
			roots = append(roots, node)
			continue
		}
		if parent, ok := byID[span.ParentSpanID]; ok {
			addChild(parent, node)
		} else {
			roots = append(roots, node)
		}
	}
	return roots, byID
}

// BuildSpanNodeTree is LinkSpans over the observatory's own SpanNode.
func BuildSpanNodeTree(spans []*Span) []*SpanNode {
	roots, _ := LinkSpans(spans,
		func(s *Span) *SpanNode { return &SpanNode{Span: s} },
		func(p, c *SpanNode) { p.Children = append(p.Children, c) })
	return roots
}
