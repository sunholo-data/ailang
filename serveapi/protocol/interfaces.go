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
