package serveapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sunholo-data/ailang/serveapi/protocol"
	"github.com/sunholo-data/ailang/serveapi/protocol/mcphttp"
)

// embeddedMCPConfig supplies the request-scoped host operations used by the
// public serveapi facade without introducing an internal-to-public import.
type embeddedMCPConfig struct {
	AgentName    string
	AgentVersion string
	Runner       *callbackRunner
	Resolve      func(context.Context, *http.Request) (any, error)
	Tools        func(context.Context, any) ([]ToolDescriptor, error)
	Invoke       func(context.Context, any, string, json.RawMessage) (json.RawMessage, error)
}

// newEmbeddedMCPHandler adapts the facade's callbacks onto the stdlib-only
// dispatcher in serveapi/protocol/mcphttp, the single MCP implementation
// (ailang#885). serveapi links no MCP SDK.
func newEmbeddedMCPHandler(config embeddedMCPConfig) http.Handler {
	handler, err := mcphttp.NewHandler(mcphttp.Config{
		Agent:    protocol.AgentInfo{Name: config.AgentName, Version: config.AgentVersion},
		Resolver: resolverFunc(config.Resolve),
		Tools:    toolsFunc(config.Tools),
		Invoker:  invokerFunc(config.Invoke),
		Runner:   config.Runner,
	})
	if err != nil {
		// New validates every field before it gets here, so this is a programming
		// error in the facade, not a runtime condition to degrade around.
		panic(fmt.Sprintf("serveapi: embedded MCP handler: %v", err))
	}
	return handler
}

type resolverFunc func(context.Context, *http.Request) (any, error)

func (f resolverFunc) ResolveSession(ctx context.Context, r *http.Request) (protocol.Session, error) {
	return f(ctx, r)
}

type toolsFunc func(context.Context, any) ([]ToolDescriptor, error)

func (f toolsFunc) Tools(ctx context.Context, session protocol.Session) ([]protocol.ToolDescriptor, error) {
	return f(ctx, session)
}

type invokerFunc func(context.Context, any, string, json.RawMessage) (json.RawMessage, error)

func (f invokerFunc) Invoke(ctx context.Context, session protocol.Session, call protocol.Invocation) (protocol.InvocationResult, error) {
	value, err := f(ctx, session, call.Name, call.Arguments)
	return protocol.InvocationResult{Value: value}, err
}
