package apiserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The MCP tool and resource lists are fixed at boot, so the server must not
// advertise listChanged. Advertising it makes MCP 2025-11-25 clients open a
// subscriptions/listen stream that never closes, which on Cloud Run pins an
// instance in-flight per connected session (see NewMCPServer).
func TestMCPServerDoesNotAdvertiseListChanged(t *testing.T) {
	ms := NewMCPServer(&Server{modules: map[string]*ModuleInfo{}})

	ct, st := mcp.NewInMemoryTransports()
	ss, err := ms.mcpServer.Connect(context.Background(), st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()

	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(context.Background(), ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	caps := cs.InitializeResult().Capabilities
	if caps == nil {
		t.Fatal("no capabilities advertised")
	}
	if caps.Tools == nil || caps.Resources == nil {
		t.Fatalf("tools and resources capabilities must be advertised: %+v", caps)
	}
	if caps.Tools.ListChanged {
		t.Error("tools.listChanged advertised; clients would open a permanent subscriptions/listen stream")
	}
	if caps.Resources.ListChanged {
		t.Error("resources.listChanged advertised; clients would open a permanent subscriptions/listen stream")
	}
}
