package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type chainReadBackend struct {
	observatory.Backend
	stage                 *observatory.ChainStage
	stageErr, evidenceErr error
	evidenceCalls         int
	opts                  observatory.ChainListOptions
}

func (b *chainReadBackend) GetStage(context.Context, string) (*observatory.ChainStage, error) {
	return b.stage, b.stageErr
}
func (b *chainReadBackend) GetSpanLitesByStageID(_ context.Context, _ string, l, o int) (*observatory.SpanLitePage, error) {
	b.evidenceCalls++
	return &observatory.SpanLitePage{Limit: l, Offset: o}, b.evidenceErr
}
func (b *chainReadBackend) GetChatMessagesByTaskID(context.Context, string) ([]*observatory.ChatMessage, error) {
	b.evidenceCalls++
	return nil, b.evidenceErr
}
func (b *chainReadBackend) GetChatMessagesBySession(context.Context, string, time.Time, time.Time) ([]*observatory.ChatMessage, error) {
	b.evidenceCalls++
	return nil, b.evidenceErr
}
func (b *chainReadBackend) ListChains(_ context.Context, o observatory.ChainListOptions) ([]*observatory.ChainSummary, error) {
	b.opts = o
	return nil, b.evidenceErr
}

func TestChainReadStageContract(t *testing.T) {
	for _, endpoint := range []string{"spans", "chat"} {
		for _, tc := range []struct {
			name, chain           string
			stageErr, evidenceErr error
			want                  int
			code                  string
		}{
			{name: "empty", chain: "chain", want: 200},
			{name: "wrong chain", chain: "other", want: 404, code: "not_found"},
			{name: "nil missing", want: 404, code: "not_found"},
			{name: "missing", stageErr: observatory.ErrNotFound, want: 404, code: "not_found"},
			{name: "stage unavailable", stageErr: status.Error(codes.Unavailable, "private details"), want: 503, code: "backend_unavailable"},
			{name: "index", chain: "chain", evidenceErr: status.Error(codes.FailedPrecondition, "private index URL"), want: 503, code: "query_not_ready"},
			{name: "denied", chain: "chain", evidenceErr: status.Error(codes.PermissionDenied, "private details"), want: 403, code: "permission_denied"},
			{name: "unauthenticated", stageErr: status.Error(codes.Unauthenticated, "private details"), want: 401, code: "unauthenticated"},
			{name: "internal", chain: "chain", evidenceErr: errors.New("private details"), want: 500, code: "query_failed"},
		} {
			t.Run(endpoint+"/"+tc.name, func(t *testing.T) {
				b := &chainReadBackend{stageErr: tc.stageErr, evidenceErr: tc.evidenceErr}
				if tc.chain != "" {
					b.stage = &observatory.ChainStage{ID: "stage", ChainID: tc.chain, TaskID: "task"}
				}
				s := &Server{obsBackend: b}
				w := httptest.NewRecorder()
				r := httptest.NewRequest("GET", "/api/chains/chain/stages/stage/"+endpoint, nil)
				if endpoint == "chat" {
					s.handleStageChat(w, r)
				} else {
					s.handleStageSpans(w, r)
				}
				if w.Code != tc.want {
					t.Fatalf("status=%d body=%s", w.Code, w.Body)
				}
				if tc.code != "" && !strings.Contains(w.Body.String(), `"error":"`+tc.code+`"`) {
					t.Errorf("missing code: %s", w.Body)
				}
				if tc.code != "" {
					var response struct {
						Retryable bool `json:"retryable"`
					}
					if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
						t.Fatal(err)
					}
					if response.Retryable != (tc.want == 503) {
						t.Errorf("retryability: %s", w.Body)
					}
				}
				if strings.Contains(w.Body.String(), "private") {
					t.Errorf("backend details leaked: %s", w.Body)
				}
				if tc.want == 200 && !strings.Contains(w.Body.String(), `":[]`) {
					t.Errorf("empty evidence not array: %s", w.Body)
				}
				if (tc.stageErr != nil || tc.chain != "chain") && b.evidenceCalls != 0 {
					t.Error("queried evidence before confirming stage ownership")
				}
			})
		}
	}
}
func TestChainReadListOptions(t *testing.T) {
	b := &chainReadBackend{}
	s := &Server{obsBackend: b}
	w := httptest.NewRecorder()
	s.handleListChains(w, httptest.NewRequest("GET", "/api/chains?workspace_id=ws&github_repo=owner/repo&agent_id=agent&limit=7&offset=3", nil))
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("response %d %s", w.Code, w.Body)
	}
	if b.opts.WorkspaceID != "ws" || b.opts.GitHubRepo != "owner/repo" || b.opts.AgentID != "agent" || b.opts.Limit != 7 || b.opts.Offset != 3 {
		t.Fatalf("options %+v", b.opts)
	}
}
func TestChainReadRejectsInvalidPage(t *testing.T) {
	for _, query := range []string{"limit=bad", "limit=0", "limit=-1", "limit=", "offset=-1", "offset=bad", "offset=", "limit=1&limit=2"} {
		for _, path := range []string{"/api/chains", "/api/chains/chain/stages/stage/spans"} {
			t.Run(path+"?"+query, func(t *testing.T) {
				b := &chainReadBackend{stage: &observatory.ChainStage{ChainID: "chain"}}
				s := &Server{obsBackend: b}
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, path+"?"+query, nil)
				if path == "/api/chains" {
					s.handleListChains(w, r)
				} else {
					s.handleStageSpans(w, r)
				}
				if w.Code != 400 {
					t.Fatalf("status=%d body=%s", w.Code, w.Body)
				}
			})
		}
	}
}

func TestChainReadStagePageAndPath(t *testing.T) {
	b := &chainReadBackend{stage: &observatory.ChainStage{ID: "stage", ChainID: "chain"}}
	s := &Server{obsBackend: b}
	w := httptest.NewRecorder()
	s.handleStageSpans(w, httptest.NewRequest("GET", "/api/chains/chain/stages/stage/spans?limit=7&offset=9", nil))
	var page observatory.SpanLitePage
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || page.Limit != 7 || page.Offset != 9 {
		t.Fatalf("response: %d %s", w.Code, w.Body)
	}
	for _, path := range []string{"/api/chains//stages/stage/spans", "/api/chains/chain/stages//spans", "/api/chains/chain/stages/stage/extra/spans"} {
		w = httptest.NewRecorder()
		s.handleStageSpans(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 400 {
			t.Fatalf("path %s: %d", path, w.Code)
		}
	}
}

func TestChainReadListFailure(t *testing.T) {
	b := &chainReadBackend{evidenceErr: status.Error(codes.FailedPrecondition, "private index")}
	w := httptest.NewRecorder()
	(&Server{obsBackend: b}).handleListChains(w, httptest.NewRequest("GET", "/api/chains", nil))
	if w.Code != 503 || !strings.Contains(w.Body.String(), `"error":"query_not_ready"`) {
		t.Fatalf("response %d %s", w.Code, w.Body)
	}
}

func TestChainReadSince(t *testing.T) {
	for _, query := range []string{"since=bad", "since=-1", "since=0", "since=", "since=1&since=2", "since=1.5", "since=9223372036854775808", "since=2562048"} {
		t.Run(query, func(t *testing.T) {
			b := &chainReadBackend{}
			w := httptest.NewRecorder()
			(&Server{obsBackend: b}).handleListChains(w, httptest.NewRequest("GET", "/api/chains?"+query, nil))
			if w.Code != 400 {
				t.Fatalf("response %d %s", w.Code, w.Body)
			}
		})
	}
	for _, query := range []string{"", "?since=24", "?since=2562047"} {
		t.Run("valid"+query, func(t *testing.T) {
			b := &chainReadBackend{}
			w := httptest.NewRecorder()
			before := time.Now()
			(&Server{obsBackend: b}).handleListChains(w, httptest.NewRequest("GET", "/api/chains"+query, nil))
			if w.Code != 200 {
				t.Fatalf("response %d %s", w.Code, w.Body)
			}
			if query == "" {
				if b.opts.CreatedAfter != nil {
					t.Fatal("absent since imposed a date filter")
				}
				return
			}
			hours := 24
			if query == "?since=2562047" {
				hours = 2562047
			}
			shift := -time.Duration(hours) * time.Hour
			if b.opts.CreatedAfter == nil || b.opts.CreatedAfter.Before(before.Add(shift)) || b.opts.CreatedAfter.After(time.Now().Add(shift)) {
				t.Fatalf("wrong since filter: %v", b.opts.CreatedAfter)
			}
		})
	}
}
