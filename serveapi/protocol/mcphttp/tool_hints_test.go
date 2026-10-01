package mcphttp

import (
	"encoding/json"
	"testing"

	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// The stdlib dispatcher must put title and annotations on the wire exactly as
// the go-sdk path in internal/apiserver does — the two are one concept with
// two implementations, and directory reviewers read whichever one is serving.
func TestListToolsEmitsTitleAndAnnotations(t *testing.T) {
	hints, err := protocol.ResolveToolHints([]string{"readOnly", "openWorld"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	annotated := tool("parse")
	annotated.Title = "Parse document"
	annotated.Annotations = hints
	surface, err := protocol.CallerSurface([]protocol.ToolDescriptor{annotated, tool("plain")})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(listTools(surface))
	want := `[{"name":"parse","title":"Parse document","description":"parse","inputSchema":{"type":"object"},` +
		`"annotations":{"readOnlyHint":true,"idempotentHint":false,"openWorldHint":true}},` +
		`{"name":"plain","description":"plain","inputSchema":{"type":"object"}}]`
	if string(b) != want {
		t.Errorf("tools/list\n got %s\nwant %s", b, want)
	}
}
