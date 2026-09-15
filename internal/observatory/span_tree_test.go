package observatory

import "testing"

func TestClassifySpanNodeType(t *testing.T) {
	cases := map[string]SpanHierarchyNodeType{
		"coordinator.task.execute": NodeTypeCoordinator,
		"coordinator.dispatch":     NodeTypeCoordinator,
		"claude.execute":           NodeTypeExecutor,
		"gemini.execute":           NodeTypeExecutor,
		"pi.execute":               NodeTypeExecutor,
		"opencode.execute":         NodeTypeExecutor,
		"motoko.execute":           NodeTypeExecutor,
		"codex.execute":            NodeTypeExecutor,
		"managed_agents.execute":   NodeTypeExecutor,
		"ailang.exec":              NodeTypeExecutor,
		"exec.turn":                NodeTypeTurn,
		"exec.turn.3":              NodeTypeTurn,
		"turn.3":                   NodeTypeTurn,
		"exec.tool_use":            NodeTypeTool,
		"exec.tool_use.Read":       NodeTypeTool,
		"claude_code.tool.Bash":    NodeTypeTool,
		"gemini.tool.call":         NodeTypeTool,
		"tool.grep":                NodeTypeTool,
		"ailang.run":               NodeTypeOther,
		"http.request":             NodeTypeOther,
		"":                         NodeTypeOther,
	}
	for name, want := range cases {
		if got := ClassifySpanNodeType(name); got != want {
			t.Errorf("ClassifySpanNodeType(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestLinkSpansKeepsInputOrderAndOrphansAsRoots(t *testing.T) {
	spans := []*Span{
		{ID: "b", ParentSpanID: "a"},
		{ID: "a"},
		{ID: "c", ParentSpanID: "a"},
		{ID: "d", ParentSpanID: "missing"},
		{ID: "e", ParentSpanID: "c"},
	}
	roots := BuildSpanNodeTree(spans)
	if len(roots) != 2 || roots[0].Span.ID != "a" || roots[1].Span.ID != "d" {
		t.Fatalf("roots = %v, want [a d] in input order", ids(roots))
	}
	a := roots[0]
	if got := ids(a.Children); len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("a.children = %v, want [b c] in input order", got)
	}
	if got := ids(a.Children[1].Children); len(got) != 1 || got[0] != "e" {
		t.Fatalf("c.children = %v, want [e]", got)
	}
	if r := BuildSpanNodeTree(nil); r != nil {
		t.Errorf("empty input must yield nil, got %v", r)
	}
}

func ids(nodes []*SpanNode) []string {
	out := make([]string, len(nodes))
	for i, n := range nodes {
		out[i] = n.Span.ID
	}
	return out
}
