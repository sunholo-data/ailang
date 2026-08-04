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
	config Config
	runner *apiserver.CallbackRunner
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
	return &Server{config: cfg, runner: runner}, nil
}

// MCPHandler returns the MCP endpoint handler. Wire behavior lands in M2.
func (s *Server) MCPHandler() http.Handler { return http.NotFoundHandler() }

// A2AHandler returns the A2A endpoint handler. Wire behavior lands in M3.
func (s *Server) A2AHandler() http.Handler { return http.NotFoundHandler() }

// Mount reserves the public protocol routes. Full routing lands in M3.
func (s *Server) Mount(mux *http.ServeMux) {
	if mux == nil {
		return
	}
	mux.Handle("/mcp/", s.MCPHandler())
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
