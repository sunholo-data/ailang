package apiserver

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
)

const a2aAgentSrc = `module agent

import std/clock (sleep)
import std/string (length, toUpper)

-- Answer in capitals. "x" divides by zero, so a test sees a failed task.
export func reply(text: string) -> string =
  if text == "x" then show(1 / (length(text) - 1)) else toUpper(text)

-- Wait, then echo: long enough for a heartbeat.
export func slow(text: string) -> string ! {Clock} { sleep(300); text }

export func two(a: string, b: string) -> string = a
export func count(n: int) -> int = n
`

func newA2AServer(t *testing.T, handler string) *httptest.Server {
	t.Helper()
	eff := effects.NewEffContext(nil)
	eff.Grant(effects.NewCapability("Clock"))
	id, err := ParseServerInfo("name=agent,title=Test agent,icon=/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	srv, ts := newSurfaceServer(t, Config{A2A: true, A2AMessage: handler, EffCtx: eff, ServerInfo: id},
		map[string]string{"agent.ail": a2aAgentSrc})
	if err := srv.ValidateA2A(); err != nil {
		t.Fatalf("ValidateA2A: %v", err)
	}
	return ts
}

func a2aPost(t *testing.T, url, method string, params any) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 7, "method": method, "params": params})
	resp, err := http.Post(url+"/a2a/", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func a2aUserMessage(text string) map[string]any {
	return map[string]any{"message": map[string]any{"role": "user", "messageId": "m1", "kind": "message",
		"parts": []any{map[string]any{"kind": "text", "text": text}}}}
}

func a2aArtifactText(t *testing.T, task map[string]any) string {
	t.Helper()
	arts, _ := task["artifacts"].([]any)
	if len(arts) != 1 {
		t.Fatalf("artifacts = %#v, want one", task["artifacts"])
	}
	parts := arts[0].(map[string]any)["parts"].([]any)
	return parts[0].(map[string]any)["text"].(string)
}

func TestA2AMessageCardIsV03Agent(t *testing.T) {
	ts := newA2AServer(t, "reply")
	for _, path := range []string{"/.well-known/agent-card.json", "/.well-known/agent.json"} {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+path, nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var card map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&card)
		resp.Body.Close()
		if card["protocolVersion"] != "0.3.0" || card["preferredTransport"] != "JSONRPC" {
			t.Fatalf("%s: protocolVersion/preferredTransport = %v/%v", path, card["protocolVersion"], card["preferredTransport"])
		}
		if u, _ := card["url"].(string); !strings.HasPrefix(u, "https://") || !strings.HasSuffix(u, "/a2a/") {
			t.Fatalf("%s: url = %q, want https://…/a2a/ behind a TLS proxy", path, u)
		}
		if card["iconUrl"] != strings.TrimSuffix(card["url"].(string), "/a2a/")+"/icon.png" {
			t.Fatalf("%s: iconUrl = %v, want /icon.png on the card's own host", path, card["iconUrl"])
		}
		if card["name"] != "Test agent" {
			t.Fatalf("%s: name = %v, want the --server-info title", path, card["name"])
		}
		if card["capabilities"].(map[string]any)["streaming"] != true {
			t.Fatalf("%s: streaming not advertised", path)
		}
		skills := card["skills"].([]any)
		if len(skills) != 1 {
			t.Fatalf("%s: skills = %d, want exactly the handler", path, len(skills))
		}
		sk := skills[0].(map[string]any)
		if sk["id"] != "reply" || !strings.HasPrefix(sk["description"].(string), "Answer in capitals.") {
			t.Fatalf("%s: skill = %#v, want id reply and the doc comment", path, sk)
		}
	}
}

func TestA2ACardWithoutHandlerKeepsExportSkills(t *testing.T) {
	ts := newA2AServer(t, "")
	var card map[string]any
	getJSON(t, ts.URL+"/.well-known/agent-card.json", &card)
	if card["protocolVersion"] != "0.3.0" {
		t.Fatalf("protocolVersion = %v", card["protocolVersion"])
	}
	if card["capabilities"].(map[string]any)["streaming"] != false {
		t.Fatal("no handler, but streaming advertised")
	}
	if n := len(card["skills"].([]any)); n < 2 {
		t.Fatalf("skills = %d, want one per export", n)
	}
	out := a2aPost(t, ts.URL, "message/send", a2aUserMessage("hi"))
	if e, _ := out["error"].(map[string]any); e == nil || e["code"] != float64(-32601) {
		t.Fatalf("message/send without a handler = %#v, want -32601", out)
	}
}

func TestA2AMessageSendCompletes(t *testing.T) {
	ts := newA2AServer(t, "reply")
	out := a2aPost(t, ts.URL, "message/send", a2aUserMessage("hello"))
	task := out["result"].(map[string]any)
	if task["kind"] != "task" || task["status"].(map[string]any)["state"] != "completed" {
		t.Fatalf("task = %#v, want a completed task", task)
	}
	if got := a2aArtifactText(t, task); got != "HELLO" {
		t.Fatalf("answer = %q, want HELLO", got)
	}
	// tasks/get reads the same task back.
	got := a2aPost(t, ts.URL, "tasks/get", map[string]any{"id": task["id"]})["result"].(map[string]any)
	if got["id"] != task["id"] || a2aArtifactText(t, got) != "HELLO" {
		t.Fatalf("tasks/get = %#v", got)
	}
	if e := a2aPost(t, ts.URL, "tasks/get", map[string]any{"id": "nope"})["error"].(map[string]any); e["code"] != float64(-32001) {
		t.Fatalf("unknown task = %#v, want -32001", e)
	}
}

func TestA2AMessageSendAcceptsTypeParts(t *testing.T) {
	ts := newA2AServer(t, "reply")
	params := map[string]any{"message": map[string]any{"role": "user",
		"parts": []any{map[string]any{"type": "text", "text": "old"}, map[string]any{"kind": "text", "text": "new"}}}}
	task := a2aPost(t, ts.URL, "message/send", params)["result"].(map[string]any)
	if got := a2aArtifactText(t, task); got != "OLD\nNEW" {
		t.Fatalf("answer = %q, want both text parts joined", got)
	}
	empty := map[string]any{"message": map[string]any{"role": "user", "parts": []any{map[string]any{"kind": "data", "data": map[string]any{}}}}}
	if e := a2aPost(t, ts.URL, "message/send", empty)["error"].(map[string]any); e["code"] != float64(-32602) {
		t.Fatalf("no text part = %#v, want -32602", e)
	}
}

func TestA2AMessageHandlerFailureIsFailedTask(t *testing.T) {
	ts := newA2AServer(t, "reply")
	task := a2aPost(t, ts.URL, "message/send", a2aUserMessage("x"))["result"].(map[string]any)
	st := task["status"].(map[string]any)
	if st["state"] != "failed" {
		t.Fatalf("state = %v, want failed", st["state"])
	}
	msg := st["message"].(map[string]any)["parts"].([]any)[0].(map[string]any)["text"].(string)
	if !strings.Contains(msg, "division by zero") {
		t.Fatalf("failure text = %q, want the handler's error", msg)
	}
}

// a2aStream reads every SSE line of one message/stream call.
func a2aStream(t *testing.T, url, text string) (events []map[string]any, heartbeats int) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 9, "method": "message/stream", "params": a2aUserMessage(text)})
	resp, err := http.Post(url+"/a2a/", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, ": heartbeat"):
			heartbeats++
		case strings.HasPrefix(line, "data: "):
			var ev map[string]any
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
				t.Fatalf("bad event %q: %v", line, err)
			}
			if ev["id"] != float64(9) {
				t.Fatalf("event id = %v, want the request id", ev["id"])
			}
			events = append(events, ev["result"].(map[string]any))
		}
	}
	return events, heartbeats
}

func TestA2AMessageStreamEvents(t *testing.T) {
	ts := newA2AServer(t, "reply")
	events, _ := a2aStream(t, ts.URL, "stream")
	var kinds []string
	for _, e := range events {
		kinds = append(kinds, e["kind"].(string))
	}
	if strings.Join(kinds, ",") != "task,status-update,artifact-update,status-update" {
		t.Fatalf("event kinds = %v", kinds)
	}
	if events[1]["status"].(map[string]any)["state"] != "working" || events[1]["final"] != false {
		t.Fatalf("second event = %#v, want working, not final", events[1])
	}
	art := events[2]["artifact"].(map[string]any)["parts"].([]any)[0].(map[string]any)["text"]
	if art != "STREAM" || events[2]["lastChunk"] != true {
		t.Fatalf("artifact event = %#v", events[2])
	}
	last := events[3]
	if last["final"] != true || last["status"].(map[string]any)["state"] != "completed" {
		t.Fatalf("last event = %#v, want final completed", last)
	}
	taskID := events[0]["id"]
	for _, e := range events[1:] {
		if e["taskId"] != taskID {
			t.Fatalf("event taskId %v != task id %v", e["taskId"], taskID)
		}
	}
}

func TestA2AMessageStreamFailureAndHeartbeat(t *testing.T) {
	ts := newA2AServer(t, "reply")
	events, _ := a2aStream(t, ts.URL, "x")
	last := events[len(events)-1]
	if len(events) != 3 || last["status"].(map[string]any)["state"] != "failed" || last["final"] != true {
		t.Fatalf("events = %#v, want task, working, final failed (no artifact)", events)
	}

	old := a2aHeartbeat
	a2aHeartbeat = 20 * time.Millisecond
	t.Cleanup(func() { a2aHeartbeat = old })
	slow := newA2AServer(t, "slow")
	events, beats := a2aStream(t, slow.URL, "late")
	if beats < 2 {
		t.Fatalf("heartbeats = %d during a 300ms handler, want several", beats)
	}
	if events[len(events)-1]["status"].(map[string]any)["state"] != "completed" {
		t.Fatalf("slow handler did not complete: %#v", events)
	}
}

func TestValidateA2AMessage(t *testing.T) {
	cases := []struct {
		name, handler string
		cfg           Config
		want          string
	}{
		{"needs --a2a", "reply", Config{}, "needs --a2a"},
		{"missing export", "nope", Config{A2A: true}, "no such export"},
		{"two params", "two", Config{A2A: true}, "exactly one string"},
		{"int param", "count", Config{A2A: true}, "exactly one string"},
		{"module qualified", "agent.reply", Config{A2A: true}, ""},
		{"wrong module", "other.reply", Config{A2A: true}, "no such export"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.cfg.A2AMessage = c.handler
			srv, _ := newSurfaceServer(t, c.cfg, map[string]string{"agent.ail": a2aAgentSrc})
			err := srv.ValidateA2A()
			if c.want == "" {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want %q", err, c.want)
			}
		})
	}
}
