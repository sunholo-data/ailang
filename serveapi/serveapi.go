// Package serveapi exposes AILANG-owned protocol handlers while leaving
// identity, session, capability, and execution policy to the embedding host.
package serveapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/apiserver"
)

const (
	DefaultCallbackTimeout        = 5 * time.Second
	DefaultMaxConcurrentCallbacks = 64
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

type ToolDescriptor struct {
	Name         string
	Description  string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
	Tags         []string
	Examples     []string
}

type Invocation struct {
	Name      string
	Arguments json.RawMessage
}

type InvocationResult struct {
	Value json.RawMessage
}

type AgentInfo struct {
	Name        string
	Description string
	Version     string
}

// AuthorizationError maps resolver rejection to an HTTP 401 or 403 response.
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

type Config struct {
	Resolver               SessionResolver
	Tools                  ToolSource
	Invoker                Invoker
	Agent                  AgentInfo
	CallbackTimeout        time.Duration
	MaxConcurrentCallbacks int
}

// Server is an embeddable protocol surface. Wire adapters are completed in
// later milestones; M1 establishes and validates the host contract.
type Server struct {
	config     Config
	runner     *apiserver.CallbackRunner
	mcpHandler http.Handler
	a2aHandler http.Handler
}

func New(cfg Config) (*Server, error) {
	if isNilInterface(cfg.Resolver) {
		return nil, errors.New("session resolver is required")
	}
	if isNilInterface(cfg.Tools) {
		return nil, errors.New("tool source is required")
	}
	if isNilInterface(cfg.Invoker) {
		return nil, errors.New("invoker is required")
	}
	if strings.TrimSpace(cfg.Agent.Name) == "" {
		return nil, errors.New("agent name is required")
	}
	if strings.TrimSpace(cfg.Agent.Version) == "" {
		return nil, errors.New("agent version is required")
	}
	if cfg.CallbackTimeout < 0 {
		return nil, errors.New("callback timeout must not be negative")
	}
	if cfg.MaxConcurrentCallbacks < 0 {
		return nil, errors.New("maximum concurrent callbacks must not be negative")
	}
	if cfg.CallbackTimeout == 0 {
		cfg.CallbackTimeout = DefaultCallbackTimeout
	}
	if cfg.MaxConcurrentCallbacks == 0 {
		cfg.MaxConcurrentCallbacks = DefaultMaxConcurrentCallbacks
	}
	runner, err := apiserver.NewCallbackRunner(cfg.CallbackTimeout, cfg.MaxConcurrentCallbacks)
	if err != nil {
		return nil, fmt.Errorf("configure callback runner: %w", err)
	}
	s := &Server{config: cfg, runner: runner}
	s.mcpHandler = apiserver.NewEmbeddedMCPHandler(apiserver.EmbeddedMCPConfig{
		AgentName: cfg.Agent.Name, AgentVersion: cfg.Agent.Version, Runner: runner,
		Resolve: func(ctx context.Context, request *http.Request) (any, error) {
			return cfg.Resolver.ResolveSession(ctx, request)
		},
		Tools: func(ctx context.Context, session any) ([]apiserver.ToolDescriptor, error) {
			descriptors, err := cfg.Tools.Tools(ctx, session)
			if err != nil {
				return nil, err
			}
			result := make([]apiserver.ToolDescriptor, len(descriptors))
			for i, descriptor := range descriptors {
				result[i] = apiserver.ToolDescriptor(descriptor)
			}
			return result, nil
		},
		Invoke: func(ctx context.Context, session any, name string, arguments json.RawMessage) (json.RawMessage, error) {
			result, err := cfg.Invoker.Invoke(ctx, session, Invocation{Name: name, Arguments: arguments})
			return result.Value, err
		},
	})
	s.a2aHandler = apiserver.NewEmbeddedA2AHandler(apiserver.EmbeddedA2AConfig{
		AgentName: cfg.Agent.Name, AgentDescription: cfg.Agent.Description, AgentVersion: cfg.Agent.Version,
		Runner: runner,
		Resolve: func(ctx context.Context, request *http.Request) (any, error) {
			return cfg.Resolver.ResolveSession(ctx, request)
		},
		Tools: func(ctx context.Context, session any) ([]apiserver.ToolDescriptor, error) {
			descriptors, err := cfg.Tools.Tools(ctx, session)
			if err != nil {
				return nil, err
			}
			result := make([]apiserver.ToolDescriptor, len(descriptors))
			for i, descriptor := range descriptors {
				result[i] = apiserver.ToolDescriptor(descriptor)
			}
			return result, nil
		},
		Invoke: func(ctx context.Context, session any, name string, arguments json.RawMessage) (json.RawMessage, error) {
			result, err := cfg.Invoker.Invoke(ctx, session, Invocation{Name: name, Arguments: arguments})
			return result.Value, err
		},
	})
	return s, nil
}

// MCPHandler returns the request-scoped stateless MCP endpoint handler.
func (s *Server) MCPHandler() http.Handler { return s.mcpHandler }

// A2AHandler returns the request-scoped A2A endpoint handler.
func (s *Server) A2AHandler() http.Handler { return s.a2aHandler }

// Mount reserves the public protocol routes. Full routing lands in M3.
func (s *Server) Mount(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.Handle("/mcp/", http.StripPrefix("/mcp", s.MCPHandler()))
	mux.Handle("/.well-known/agent.json", s.A2AHandler())
	mux.Handle("/a2a/", s.A2AHandler())
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
