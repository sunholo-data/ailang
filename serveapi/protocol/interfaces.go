package protocol

import (
	"context"
	"encoding/json"
	"net/http"
)

type Session any
type SessionResolver interface {
	ResolveSession(context.Context, *http.Request) (Session, error)
}
type ToolSource interface {
	Tools(context.Context, Session) ([]ToolDescriptor, error)
}
type Invoker interface {
	Invoke(context.Context, Session, Invocation) (InvocationResult, error)
}

// JSONRPCError may be implemented by an error returned from Invoker.Invoke.
// MCP answers that message, rather than the whole POST, with the exact code
// and message. Wrapped errors are supported. The hook requires a nonzero code
// and nonempty message; otherwise the frozen -32603 callback envelope applies.
// Hosts should use server-error codes -32000..-32099 or application-defined
// codes. Protocol-reserved codes -32700..-32600 pass through at the host's risk.
type JSONRPCError interface {
	error
	JSONRPCError() (code int, message string)
}

type Invocation struct {
	Name      string
	Arguments json.RawMessage
}
type InvocationResult struct{ Value json.RawMessage }
type AgentInfo struct {
	Name        string
	Description string
	Version     string
}
type AuthorizationError struct {
	Status int
	Err    error
}

func (e *AuthorizationError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return http.StatusText(e.Status)
}
func (e *AuthorizationError) Unwrap() error   { return e.Err }
func (e *AuthorizationError) HTTPStatus() int { return e.Status }
