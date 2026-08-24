package serveapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func embeddedA2A(t *testing.T, host embeddedTestHost, timeout time.Duration, capacity int) http.Handler {
	t.Helper()
	runner, err := newCallbackRunner(timeout, capacity)
	if err != nil {
		t.Fatal(err)
	}
	return newEmbeddedA2AHandler(embeddedA2AConfig{
		AgentName: "sentinel-agent", AgentDescription: "sentinel-description", AgentVersion: "9.8.7",
		Runner: runner, Resolve: host.resolve, Tools: host.tools, Invoke: host.invoke,
	})
}

func sentinelTool(name, marker string) ToolDescriptor {
	return ToolDescriptor{Name: name, Description: "description-" + marker,
		InputSchema: json.RawMessage(`{"type":"object"}`), Tags: []string{"tag-" + marker}, Examples: []string{"example-" + marker}}
}

func a2aCard(t *testing.T, handler http.Handler, session string) map[string]any {
	t.Helper()
	r := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent.json", nil)
	req.Header.Set("X-Session", session)
	handler.ServeHTTP(r, req)
	var card map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &card); err != nil {
		t.Fatalf("card JSON: %v body=%q", err, r.Body.String())
	}
	if r.Code != http.StatusOK {
		t.Fatalf("card status=%d body=%q", r.Code, r.Body.String())
	}
	return card
}

func a2aSend(t *testing.T, handler http.Handler, session, skill string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":73,"method":"tasks/send","params":{"id":"task-sentinel","metadata":{"skill_id":"` + skill + `"},"message":{"role":"user","parts":[{"type":"data","data":{"args":["arg-sentinel"]}}]}}}`
	r := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/a2a/", strings.NewReader(body))
	req.Header.Set("X-Session", session)
	handler.ServeHTTP(r, req)
	var result map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &result); err != nil {
		t.Fatalf("task JSON: %v body=%q", err, r.Body.String())
	}
	return r, result
}

func TestEmbeddedA2ACardExactRequestLocalSurfaces(t *testing.T) {
	sessionA, sessionB := &embeddedTestSession{"A"}, &embeddedTestSession{"B"}
	host := embeddedTestHost{
		resolve: func(_ context.Context, r *http.Request) (any, error) {
			if r.Header.Get("X-Session") == "B" {
				return sessionB, nil
			}
			return sessionA, nil
		},
		tools: func(_ context.Context, session any) ([]ToolDescriptor, error) {
			if session == sessionB {
				return []ToolDescriptor{sentinelTool("shared", "shared"), sentinelTool("beta_only", "beta")}, nil
			}
			return []ToolDescriptor{sentinelTool("shared", "shared"), sentinelTool("alpha_only", "alpha")}, nil
		},
		invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"ok":true}`), nil
		},
	}
	handler := embeddedA2A(t, host, time.Second, 8)
	for _, tc := range []struct {
		session string
		want    []ToolDescriptor
		foreign string
	}{
		{"A", []ToolDescriptor{sentinelTool("alpha_only", "alpha"), sentinelTool("shared", "shared")}, "beta_only"},
		{"B", []ToolDescriptor{sentinelTool("beta_only", "beta"), sentinelTool("shared", "shared")}, "alpha_only"},
	} {
		card := a2aCard(t, handler, tc.session)
		skills := card["skills"].([]any)
		if len(skills) != len(tc.want) {
			t.Fatalf("session %s skills=%#v", tc.session, skills)
		}
		for i, want := range tc.want {
			got := skills[i].(map[string]any)
			if got["id"] != want.Name || got["name"] != want.Name || got["description"] != want.Description ||
				!reflect.DeepEqual(got["tags"], []any{want.Tags[0]}) || !reflect.DeepEqual(got["examples"], []any{want.Examples[0]}) {
				t.Fatalf("session %s skill %d=%#v want=%#v", tc.session, i, got, want)
			}
			if got["name"] == tc.foreign {
				t.Fatalf("foreign sentinel %q leaked", tc.foreign)
			}
		}
	}
}

func TestEmbeddedA2ADispatchAuthorizationAndSessionIdentity(t *testing.T) {
	sessionA, sessionB := &embeddedTestSession{"A"}, &embeddedTestSession{"B"}
	var calls atomic.Int32
	var gotSession any
	host := embeddedTestHost{
		resolve: func(_ context.Context, r *http.Request) (any, error) {
			if r.Header.Get("X-Session") == "B" {
				return sessionB, nil
			}
			return sessionA, nil
		},
		tools: func(_ context.Context, session any) ([]ToolDescriptor, error) {
			if session == sessionB {
				return []ToolDescriptor{objectTool("beta_only")}, nil
			}
			return []ToolDescriptor{objectTool("alpha_only")}, nil
		},
		invoke: func(_ context.Context, session any, _ string, arguments json.RawMessage) (json.RawMessage, error) {
			calls.Add(1)
			gotSession = session
			if string(arguments) != `["arg-sentinel"]` {
				t.Errorf("arguments=%s", arguments)
			}
			return json.RawMessage(`{"ok":true}`), nil
		},
	}
	handler := embeddedA2A(t, host, time.Second, 8)
	_, allowed := a2aSend(t, handler, "B", "beta_only")
	if calls.Load() != 1 || gotSession != sessionB || allowed["result"] == nil {
		t.Fatalf("allowed calls=%d session=%p result=%#v", calls.Load(), gotSession, allowed)
	}
	calls.Store(0)
	_, denied := a2aSend(t, handler, "A", "beta_only")
	errBody := denied["error"].(map[string]any)
	if calls.Load() != 0 || errBody["code"] != float64(-32602) {
		t.Fatalf("denied calls=%d response=%#v", calls.Load(), denied)
	}
}

func TestEmbeddedA2AFrozenCallbackEnvelopes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		card        bool
		callbackErr error
		capacity    bool
		wantStatus  int
		want        string
	}{
		{"task timeout", false, context.DeadlineExceeded, false, http.StatusOK, "host callback timed out"},
		{"card timeout", true, context.DeadlineExceeded, false, http.StatusGatewayTimeout, "host callback timed out"},
		{"task canceled", false, context.Canceled, false, http.StatusOK, "host callback canceled"},
		{"card canceled", true, context.Canceled, false, http.StatusInternalServerError, "host callback canceled"},
		{"task overload", false, nil, true, http.StatusOK, "host callback capacity exceeded"},
		{"card overload", true, nil, true, http.StatusServiceUnavailable, "host callback capacity exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			block := make(chan struct{})
			started := make(chan struct{})
			var startOnce sync.Once
			host := embeddedTestHost{resolve: func(context.Context, *http.Request) (any, error) {
				if tc.capacity {
					// Signal that the capacity token is already held, so the
					// overload request below cannot win a race with us. A sleep
					// here would be a timing heuristic, not synchronization.
					startOnce.Do(func() { close(started) })
					<-block
					return "late", nil
				}
				return nil, tc.callbackErr
			}, tools: func(context.Context, any) ([]ToolDescriptor, error) { return []ToolDescriptor{objectTool("tool")}, nil },
				invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) {
					return json.RawMessage(`{}`), nil
				}}
			handler := embeddedA2A(t, host, 20*time.Millisecond, 1)
			if tc.capacity {
				go func() {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					handler.ServeHTTP(httptest.NewRecorder(), req)
				}()
				select {
				case <-started:
				case <-time.After(5 * time.Second):
					t.Fatal("blocking callback never acquired the capacity token")
				}
				defer close(block)
			}
			var recorder *httptest.ResponseRecorder
			if tc.card {
				recorder = httptest.NewRecorder()
				handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
			} else {
				recorder, _ = a2aSend(t, handler, "A", "tool")
			}
			if recorder.Code != tc.wantStatus || !strings.Contains(recorder.Body.String(), tc.want) {
				t.Fatalf("status=%d body=%q want status=%d message=%q", recorder.Code, recorder.Body.String(), tc.wantStatus, tc.want)
			}
			if tc.card && recorder.Code == http.StatusOK {
				t.Fatal("card used task POST mapping")
			}
			if !tc.card && recorder.Code != http.StatusOK {
				t.Fatal("task used card mapping")
			}
		})
	}
}

func TestEmbeddedA2AInterleavedRequestLocalSurfaces(t *testing.T) {
	var observed atomic.Int32
	host := embeddedTestHost{
		resolve: func(_ context.Context, r *http.Request) (any, error) { return r.Header.Get("X-Session"), nil },
		tools: func(_ context.Context, session any) ([]ToolDescriptor, error) {
			observed.Add(1)
			return []ToolDescriptor{sentinelTool(strings.ToLower(session.(string))+"_only", session.(string))}, nil
		},
		invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) {
			return nil, errors.New("unused")
		},
	}
	handler := embeddedA2A(t, host, time.Second, 16)
	var wg sync.WaitGroup
	failures := make(chan string, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session := "A"
			foreign := "b_only"
			if i%2 == 1 {
				session, foreign = "B", "a_only"
			}
			card := a2aCard(t, handler, session)
			encoded, _ := json.Marshal(card["skills"])
			if strings.Contains(string(encoded), foreign) {
				failures <- string(encoded)
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	if observed.Load() != 100 {
		t.Fatalf("observed request count=%d want=100", observed.Load())
	}
	for failure := range failures {
		t.Fatalf("foreign sentinel: %s", failure)
	}
}

func TestEmbeddedA2ABlockedPrincipalDoesNotBlockAnother(t *testing.T) {
	startedA := make(chan struct{})
	releaseA := make(chan struct{})
	host := embeddedTestHost{
		resolve: func(_ context.Context, r *http.Request) (any, error) { return r.Header.Get("X-Session"), nil },
		tools: func(_ context.Context, session any) ([]ToolDescriptor, error) {
			if session == "A" {
				close(startedA)
				<-releaseA
			}
			return []ToolDescriptor{sentinelTool(strings.ToLower(session.(string))+"_only", session.(string))}, nil
		},
		invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) {
			return nil, errors.New("unused")
		},
	}
	handler := embeddedA2A(t, host, time.Second, 2)
	doneA := make(chan struct{})
	go func() { defer close(doneA); a2aCard(t, handler, "A") }()
	<-startedA
	cardB := a2aCard(t, handler, "B")
	encodedB, _ := json.Marshal(cardB["skills"])
	if !strings.Contains(string(encodedB), "b_only") || strings.Contains(string(encodedB), "a_only") {
		t.Fatalf("B surface while A blocked: %s", encodedB)
	}
	close(releaseA)
	<-doneA
}

// TestEmbeddedA2AEmptyAndInvalidResultsAreDistinguishable covers the two ways a
// host can return an unusable result. They must NOT collapse into one message:
// "no result" points the embedder at its Invoker, "invalid JSON" at its encoder.
// The valid arm is the known-positive control — without it, a mutation that made
// EVERY result an error would still satisfy the two failure arms.
func TestEmbeddedA2AEmptyAndInvalidResultsAreDistinguishable(t *testing.T) {
	for _, tc := range []struct {
		name    string
		result  json.RawMessage
		want    string
		wantErr bool
	}{
		{"nil result", nil, "host callback returned no result", true},
		{"empty result", json.RawMessage(``), "host callback returned no result", true},
		{"malformed result", json.RawMessage(`{not json`), "host callback returned invalid JSON", true},
		{"valid result", json.RawMessage(`{"ok":true}`), `"state":"completed"`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := embeddedTestHost{
				resolve: func(_ context.Context, r *http.Request) (any, error) { return r.Header.Get("X-Session"), nil },
				tools: func(context.Context, any) ([]ToolDescriptor, error) {
					return []ToolDescriptor{objectTool("tool")}, nil
				},
				invoke: func(context.Context, any, string, json.RawMessage) (json.RawMessage, error) {
					return tc.result, nil
				},
			}
			recorder, _ := a2aSend(t, embeddedA2A(t, host, time.Second, 4), "A", "tool")
			if recorder.Code != http.StatusOK {
				t.Fatalf("status=%d want 200 (A2A task POST is always 200)", recorder.Code)
			}
			body := recorder.Body.String()
			if !strings.Contains(body, tc.want) {
				t.Fatalf("body=%q want %q", body, tc.want)
			}
			if gotErr := strings.Contains(body, `"code":-32603`); gotErr != tc.wantErr {
				t.Fatalf("body=%q: -32603 present=%v want=%v", body, gotErr, tc.wantErr)
			}
		})
	}
}
