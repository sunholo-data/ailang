package eval_harness

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// A Google row's gcp_project / gcp_location decide the project a STANDARD-mode
// eval bills. Before 2026-11-24 the row was ignored and the resolver's project
// (prod ailang-multivac on the rig) was billed instead.
func TestNewAIAgent_GoogleRowPinsVertexProject(t *testing.T) {
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })
	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"pinned":   {APIName: "gemini-x", Provider: "google", GCPProject: "row-project", GCPLocation: "europe-west4"},
		"unpinned": {APIName: "gemini-y", Provider: "google"},
	}}
	// The resolver's answer, standing in for ~/.ailang/config.yaml's prod project.
	t.Setenv("AILANG_CLOUD_PROJECT", "resolver-project")
	t.Setenv("GOOGLE_CLOUD_LOCATION", "us-central1")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")

	for _, c := range []struct{ model, project, location string }{
		{"pinned", "row-project", "europe-west4"},
		{"unpinned", "resolver-project", "us-central1"},
	} {
		agent, err := NewAIAgent(c.model, 0)
		if err != nil {
			t.Fatalf("%s: %v", c.model, err)
		}
		g, ok := agent.adapter.provider.(*gemini.Client)
		if !ok {
			t.Fatalf("%s: provider = %T, want a Vertex *gemini.Client", c.model, agent.adapter.provider)
		}
		if p, l := g.VertexTarget(); p != c.project || l != c.location {
			t.Errorf("%s: Vertex target = %s/%s, want %s/%s", c.model, p, l, c.project, c.location)
		}
	}
}

// The pin applies to Google rows only: another provider's gcp_project field is
// not a billing project for it.
func TestVertexPinFor_GoogleOnly(t *testing.T) {
	row := &ModelConfig{GCPProject: "p", GCPLocation: "global"}
	if got := len(vertexPinFor("google", row)); got != 2 {
		t.Errorf("google row: %d options, want 2", got)
	}
	for _, prov := range []string{"anthropic", "openai", "openrouter", ""} {
		if got := vertexPinFor(prov, row); got != nil {
			t.Errorf("%q row: %d options, want none", prov, len(got))
		}
	}
	if vertexPinFor("google", nil) != nil || vertexPinFor("google", &ModelConfig{}) != nil {
		t.Error("a row without gcp fields must add no options")
	}
}
