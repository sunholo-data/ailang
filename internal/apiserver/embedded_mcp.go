package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EmbeddedMCPConfig supplies the request-scoped host operations used by the
// public serveapi facade without introducing an internal-to-public import.
type EmbeddedMCPConfig struct {
	AgentName    string
	AgentVersion string
	Runner       *CallbackRunner
	Resolve      func(context.Context, *http.Request) (any, error)
	Tools        func(context.Context, any) ([]ToolDescriptor, error)
	Invoke       func(context.Context, any, string, json.RawMessage) (json.RawMessage, error)
}

type embeddedMCPHandler struct {
	config    EmbeddedMCPConfig
	transport http.Handler
}

type embeddedMCPContext struct {
	surface *AuthorizedSurface
	session any
	failure *embeddedCallbackFailure
}

type embeddedCallbackFailure struct {
	mu      sync.Mutex
	message string
}

type embeddedMCPContextKey struct{}

// NewEmbeddedMCPHandler builds the stateless SDK transport once. The server
// returned to it is still new for every authorized POST.
func NewEmbeddedMCPHandler(config EmbeddedMCPConfig) http.Handler {
	h := &embeddedMCPHandler{config: config}
	h.transport = mcp.NewStreamableHTTPHandler(h.serverForRequest,
		&mcp.StreamableHTTPOptions{Stateless: true})
	return h
}

func (h *embeddedMCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.transport.ServeHTTP(w, r)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, mcp.DefaultMaxRequestBodyBytes+1))
	if err != nil || len(body) > mcp.DefaultMaxRequestBodyBytes {
		writeMCPEnvelope(w, requestID(body), "invalid MCP request body")
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	id := requestID(body)

	session, err := RunCallback(r.Context(), h.config.Runner, func(ctx context.Context) (any, error) {
		return h.config.Resolve(ctx, r)
	})
	if err != nil {
		if status := authorizationStatus(err); status != 0 {
			http.Error(w, err.Error(), status)
			return
		}
		writeMCPCallbackError(w, id, err)
		return
	}

	descriptors, err := RunCallback(r.Context(), h.config.Runner, func(ctx context.Context) ([]ToolDescriptor, error) {
		return h.config.Tools(ctx, session)
	})
	if err != nil {
		writeMCPCallbackError(w, id, err)
		return
	}
	surface, err := callerSurface(descriptors)
	if err != nil {
		writeMCPEnvelope(w, id, err.Error())
		return
	}

	failure := &embeddedCallbackFailure{}
	ctx := context.WithValue(r.Context(), embeddedMCPContextKey{}, embeddedMCPContext{surface, session, failure})
	r = r.WithContext(ctx)
	r.Body = io.NopCloser(bytes.NewReader(body))
	h.serveTransport(w, r, id)
}

func (h *embeddedMCPHandler) serveTransport(w http.ResponseWriter, r *http.Request, id json.RawMessage) {
	buffer := newBufferedResponseWriter()
	defer func() {
		if recover() != nil {
			writeMCPEnvelope(w, id, "host tool registration failed")
		}
	}()
	h.transport.ServeHTTP(buffer, r)
	requestContext := r.Context().Value(embeddedMCPContextKey{}).(embeddedMCPContext)
	requestContext.failure.mu.Lock()
	message := requestContext.failure.message
	requestContext.failure.mu.Unlock()
	if message != "" {
		writeMCPEnvelope(w, id, message)
		return
	}
	for name, values := range buffer.header {
		w.Header()[name] = append([]string(nil), values...)
	}
	// The SDK sets Content-Type on every response path it has today, but this wrapper
	// replays its headers wholesale, so that guarantee is inherited rather than ours. An
	// unlabelled body here would be content-sniffed by the browser, and this body can
	// contain reflected request data. Assert the invariant locally instead (#603).
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(buffer.status)
	_, _ = w.Write(buffer.body.Bytes())
}

type bufferedResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header), status: http.StatusOK}
}

func (w *bufferedResponseWriter) Header() http.Header            { return w.header }
func (w *bufferedResponseWriter) WriteHeader(status int)         { w.status = status }
func (w *bufferedResponseWriter) Write(data []byte) (int, error) { return w.body.Write(data) }
func (w *bufferedResponseWriter) Flush()                         {}

func (h *embeddedMCPHandler) serverForRequest(r *http.Request) *mcp.Server {
	requestContext, ok := r.Context().Value(embeddedMCPContextKey{}).(embeddedMCPContext)
	if !ok {
		return nil
	}
	server := mcp.NewServer(&mcp.Implementation{
		Name: h.config.AgentName, Version: h.config.AgentVersion,
	}, nil)
	for _, descriptor := range requestContext.surface.All() {
		descriptor := descriptor
		server.AddTool(&mcp.Tool{
			Name: descriptor.Name, Description: descriptor.Description,
			InputSchema: descriptor.InputSchema, OutputSchema: descriptor.OutputSchema,
		}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := RunCallback(ctx, h.config.Runner, func(callCtx context.Context) (json.RawMessage, error) {
				return h.config.Invoke(callCtx, requestContext.session, descriptor.Name, req.Params.Arguments)
			})
			if err != nil {
				requestContext.failure.mu.Lock()
				requestContext.failure.message = callbackMessage(err)
				requestContext.failure.mu.Unlock()
				return mcpError(callbackMessage(err)), nil
			}
			return &mcp.CallToolResult{
				Content:           []mcp.Content{&mcp.TextContent{Text: string(result)}},
				StructuredContent: result,
			}, nil
		})
	}
	return server
}

func writeMCPCallbackError(w http.ResponseWriter, id json.RawMessage, err error) {
	writeMCPEnvelope(w, id, callbackMessage(err))
}

// mcpError creates an MCP tool error result shared by standalone and embedded servers.
func mcpError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}
