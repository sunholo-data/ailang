package apiserver

// A2A v0.3 message methods (M-EXECUTOR-GE-A2A M1). With --a2a-message the
// server names one export as its message handler: the caller's text goes in,
// the answer text comes out. That is the shape Gemini Enterprise sends (a
// user's request as text, not a function call), so any AILANG program can be
// a GE agent. message/send answers with a completed Task; message/stream
// sends the same Task as SSE events, with heartbeats while the handler runs;
// tasks/get reads a recent Task back.

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/embed"
)

// a2aHeartbeat bounds the silence on an open stream, so a proxy or the
// client does not idle it out while the handler works. A var for tests.
var a2aHeartbeat = 15 * time.Second

const (
	a2aProtocolVersion = "0.3.0"
	// a2aTaskTTL / a2aTaskCap bound the tasks/get store: a recent answer can
	// be read back, nothing accumulates.
	a2aTaskTTL = time.Hour
	a2aTaskCap = 1000
	// a2aMaxFailure bounds the handler error text a caller sees.
	a2aMaxFailure = 500
)

// a2aHandler is the resolved --a2a-message export.
type a2aHandler struct {
	modPath string
	export  ExportInfo
}

// resolveA2AMessage finds the --a2a-message export: "fn" when the name is
// unique across loaded modules, or "module/path.fn". It must take one string.
func (s *Server) resolveA2AMessage() (*a2aHandler, error) {
	name := s.a2aMessage
	if name == "" {
		return nil, nil
	}
	wantMod, fn := "", name
	if i := strings.LastIndex(name, "."); i >= 0 {
		wantMod, fn = strings.ReplaceAll(name[:i], ".", "/"), name[i+1:]
	}
	var found []*a2aHandler
	for _, info := range s.GetModules() {
		if info == nil || (wantMod != "" && info.Path != wantMod) {
			continue
		}
		for _, e := range info.Exports {
			if e.Name == fn {
				found = append(found, &a2aHandler{modPath: info.Path, export: e})
			}
		}
	}
	switch {
	case len(found) == 0:
		return nil, fmt.Errorf("--a2a-message %s: no such export", name)
	case len(found) > 1:
		return nil, fmt.Errorf("--a2a-message %s: exported by more than one module; name it as module/path.%s", name, fn)
	}
	h := found[0]
	if h.export.Arity != 1 || len(h.export.ParamTypes) != 1 || h.export.ParamTypes[0] != "string" {
		return nil, fmt.Errorf("--a2a-message %s: must take exactly one string (the caller's text), has type %s", name, h.export.Type)
	}
	return h, nil
}

// ValidateA2A refuses a --a2a-message that cannot be served. Under
// --mcp-connect-only the handler must be reachable only through the gated
// /a2a/: an ungated HTTP route or MCP tool to it would bypass the sign-in
// that pays for each call.
func (s *Server) ValidateA2A() error {
	if s.a2aMessage == "" {
		return nil
	}
	if !s.a2aEnabled {
		return errors.New("--a2a-message needs --a2a")
	}
	h, err := s.resolveA2AMessage()
	if err != nil {
		return err
	}
	if s.mcpConnectOnly && (!h.export.IsNoExpose || !h.export.IsNoMCP) {
		return fmt.Errorf("--a2a-message %s: under --mcp-connect-only the handler must be @noexpose @nomcp, so only the gated /a2a/ reaches it", s.a2aMessage)
	}
	s.a2aHandlerFn = h
	return nil
}

// a2aSkill is the card's one skill for a message handler. Its description is
// the handler's doc comment: what a model reads to decide when to delegate.
func (s *Server) a2aSkill(h *a2aHandler) map[string]any {
	id := s.serverIdentity()
	name := id.Title
	if name == "" {
		name = id.Name
	}
	desc := strings.TrimSpace(h.export.DocComment)
	if desc == "" {
		desc = name
	}
	return map[string]any{
		"id":          h.export.Name,
		"name":        name,
		"description": desc,
		"tags":        []string{id.Name},
		"inputModes":  []string{"text/plain"},
		"outputModes": []string{"text/plain"},
	}
}

// --- the task store (tasks/get) ---

type a2aTaskStore struct {
	mu    sync.Mutex
	tasks map[string]a2aStored
	order []string
}

type a2aStored struct {
	task map[string]any
	at   time.Time
}

func (st *a2aTaskStore) put(id string, task map[string]any) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.tasks == nil {
		st.tasks = map[string]a2aStored{}
	}
	if _, ok := st.tasks[id]; !ok {
		st.order = append(st.order, id)
	}
	st.tasks[id] = a2aStored{task: task, at: time.Now()}
	for len(st.order) > 0 {
		oldest := st.order[0]
		if len(st.order) <= a2aTaskCap && time.Since(st.tasks[oldest].at) < a2aTaskTTL {
			break
		}
		delete(st.tasks, oldest)
		st.order = st.order[1:]
	}
}

func (st *a2aTaskStore) get(id string) (map[string]any, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	t, ok := st.tasks[id]
	if !ok || time.Since(t.at) >= a2aTaskTTL {
		return nil, false
	}
	return t.task, true
}

// --- request parsing ---

type a2aMessageParams struct {
	Message struct {
		Role      string            `json:"role"`
		MessageID string            `json:"messageId"`
		ContextID string            `json:"contextId"`
		TaskID    string            `json:"taskId"`
		Parts     []json.RawMessage `json:"parts"`
	} `json:"message"`
}

// a2aText joins the message's text parts. v0.3 tags a part with "kind";
// earlier versions used "type", still accepted.
func a2aText(parts []json.RawMessage) string {
	var texts []string
	for _, raw := range parts {
		var p struct {
			Kind string `json:"kind"`
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(raw, &p) != nil {
			continue
		}
		if (p.Kind == "text" || p.Type == "text") && p.Text != "" {
			texts = append(texts, p.Text)
		}
	}
	return strings.Join(texts, "\n")
}

func a2aID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func a2aNow() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// a2aCall is one parsed message/send or message/stream request.
type a2aCall struct {
	text, taskID, contextID string
	userMsg                 map[string]any
}

func parseA2AMessage(req *a2aRequest) (*a2aCall, error) {
	var p a2aMessageParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %v", err)
	}
	text := a2aText(p.Message.Parts)
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("message has no text part")
	}
	c := &a2aCall{text: text, taskID: a2aID(), contextID: p.Message.ContextID}
	if c.contextID == "" {
		c.contextID = a2aID()
	}
	msgID := p.Message.MessageID
	if msgID == "" {
		msgID = a2aID()
	}
	c.userMsg = map[string]any{
		"kind": "message", "role": "user", "messageId": msgID,
		"taskId": c.taskID, "contextId": c.contextID,
		"parts": []map[string]any{{"kind": "text", "text": text}},
	}
	return c, nil
}

// --- calling the handler ---

// runA2AHandler calls the handler with the caller's text. The call's effect
// context carries the request context, so a Net or AI call it makes stops
// when the caller goes away.
func (s *Server) runA2AHandler(r *http.Request, text string) (string, error) {
	h := s.a2aHandlerFn
	args, err := bindCallArgs(h.export, []any{text})
	if err != nil {
		return "", err
	}
	res, err := s.engine.CallPrepared(func(eff interface{}) {
		if ec, ok := eff.(*effects.EffContext); ok {
			ec.GoCtx = r.Context()
		}
	}, h.modPath, h.export.Name, args...)
	if err != nil && !isCleanExit(err) {
		return "", err
	}
	if err != nil {
		return "", errors.New("handler exited without an answer")
	}
	v, err := embed.ToGo(res)
	if err != nil {
		return "", fmt.Errorf("result conversion: %v", err)
	}
	if str, ok := v.(string); ok {
		return str, nil
	}
	b, _ := json.Marshal(v)
	return string(b), nil
}

func a2aArtifact(answer string) map[string]any {
	return map[string]any{
		"artifactId": a2aID(),
		"name":       "answer",
		"parts":      []map[string]any{{"kind": "text", "text": answer}},
	}
}

func a2aStatus(c *a2aCall, state, failure string) map[string]any {
	st := map[string]any{"state": state, "timestamp": a2aNow()}
	if failure != "" {
		st["message"] = map[string]any{
			"kind": "message", "role": "agent", "messageId": a2aID(),
			"taskId": c.taskID, "contextId": c.contextID,
			"parts": []map[string]any{{"kind": "text", "text": clipText(failure, a2aMaxFailure)}},
		}
	}
	return st
}

func clipText(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// a2aTask is the finished Task for one call.
func a2aTask(c *a2aCall, answer string, callErr error) map[string]any {
	t := map[string]any{
		"kind": "task", "id": c.taskID, "contextId": c.contextID,
		"history": []map[string]any{c.userMsg},
	}
	if callErr != nil {
		t["status"] = a2aStatus(c, "failed", callErr.Error())
		t["artifacts"] = []map[string]any{}
		return t
	}
	t["status"] = a2aStatus(c, "completed", "")
	t["artifacts"] = []map[string]any{a2aArtifact(answer)}
	return t
}

// --- the methods ---

func (s *Server) handleA2AMessageSend(w http.ResponseWriter, r *http.Request, req *a2aRequest) {
	if s.a2aHandlerFn == nil {
		a2aError(w, req.ID, -32601, "message/send needs a message handler (serve-api --a2a-message)")
		return
	}
	c, err := parseA2AMessage(req)
	if err != nil {
		a2aError(w, req.ID, -32602, err.Error())
		return
	}
	answer, callErr := s.runA2AHandler(r, c.text)
	task := a2aTask(c, answer, callErr)
	s.a2aTasks.put(c.taskID, task)
	a2aResult(w, req.ID, task)
}

func (s *Server) handleA2AMessageStream(w http.ResponseWriter, r *http.Request, req *a2aRequest) {
	if s.a2aHandlerFn == nil {
		a2aError(w, req.ID, -32601, "message/stream needs a message handler (serve-api --a2a-message)")
		return
	}
	c, err := parseA2AMessage(req)
	if err != nil {
		a2aError(w, req.ID, -32602, err.Error())
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		a2aError(w, req.ID, -32603, "streaming unsupported by this connection")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	send := func(result any) {
		b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}
	send(map[string]any{
		"kind": "task", "id": c.taskID, "contextId": c.contextID,
		"status": a2aStatus(c, "submitted", ""), "history": []map[string]any{c.userMsg},
	})
	send(map[string]any{"kind": "status-update", "taskId": c.taskID, "contextId": c.contextID,
		"status": a2aStatus(c, "working", ""), "final": false})

	type outcome struct {
		answer string
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		a, e := s.runA2AHandler(r, c.text)
		done <- outcome{a, e}
	}()
	tick := time.NewTicker(a2aHeartbeat)
	defer tick.Stop()
	var out outcome
wait:
	for {
		select {
		case out = <-done:
			break wait
		case <-tick.C:
			// An SSE comment: every client ignores it, every proxy sees traffic.
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
	task := a2aTask(c, out.answer, out.err)
	s.a2aTasks.put(c.taskID, task)
	if out.err == nil {
		send(map[string]any{"kind": "artifact-update", "taskId": c.taskID, "contextId": c.contextID,
			"artifact": task["artifacts"].([]map[string]any)[0], "append": false, "lastChunk": true})
	}
	send(map[string]any{"kind": "status-update", "taskId": c.taskID, "contextId": c.contextID,
		"status": task["status"], "final": true})
}

func (s *Server) handleA2ATasksGet(w http.ResponseWriter, req *a2aRequest) {
	var p struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil || p.ID == "" {
		a2aError(w, req.ID, -32602, "tasks/get needs params.id")
		return
	}
	task, ok := s.a2aTasks.get(p.ID)
	if !ok {
		a2aError(w, req.ID, -32001, "task not found")
		return
	}
	a2aResult(w, req.ID, task)
}
