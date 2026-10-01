package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// authRecorder is an approval service that records the Authorization header
// of every create and poll, and refuses requests whose bearer is not want.
type authRecorder struct {
	mu      sync.Mutex
	want    string
	headers []string
}

func (a *authRecorder) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		a.mu.Lock()
		a.headers = append(a.headers, r.Method+" "+got)
		a.mu.Unlock()
		if got != "Bearer "+a.want {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "secret-1"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
	})
}

func authApprover(srv *httptest.Server, opts ...CloudApproverOption) *CloudSecretApprover {
	base := []CloudApproverOption{
		WithApproverDeadline(time.Second),
		WithApproverPollInterval(5 * time.Millisecond),
		WithApproverHTTPClient(srv.Client()),
	}
	return NewCloudSecretApprover(srv.URL, append(base, opts...)...)
}

// TestCloudApprover_MintsIDTokenForCreateAndPoll: with no static token the
// approver sends a minted ID token on BOTH the create and the poll.
func TestCloudApprover_MintsIDTokenForCreateAndPoll(t *testing.T) {
	rec := &authRecorder{want: "minted-id-token"}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	mints := 0
	a := authApprover(srv, WithApproverIDTokenSource(func(context.Context) (string, error) {
		mints++
		return "minted-id-token", nil
	}))
	if err := a.Approve(context.Background(), "op://Prod/k", "use"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.headers) < 2 || !strings.HasPrefix(rec.headers[0], "POST ") || !strings.HasPrefix(rec.headers[1], "GET ") {
		t.Fatalf("expected a create then a poll, got %v", rec.headers)
	}
	if mints < 2 {
		t.Errorf("expected a mint per request (the source caches), got %d", mints)
	}
}

// TestCloudApprover_StaticTokenWins: AILANG_APPROVAL_TOKEN, when configured,
// is what goes on the wire; the minter is not consulted.
func TestCloudApprover_StaticTokenWins(t *testing.T) {
	rec := &authRecorder{want: "shared-secret"}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	a := authApprover(srv,
		WithApproverAuthToken("shared-secret"),
		WithApproverIDTokenSource(func(context.Context) (string, error) {
			t.Error("minter must not run when a static token is set")
			return "minted", nil
		}),
	)
	if err := a.Approve(context.Background(), "op://Prod/k", "use"); err != nil {
		t.Fatalf("approve: %v", err)
	}
}

// TestCloudApprover_MintFailureExplains401: when no credential could be
// minted the request still goes out (a dashboard with intake auth off would
// answer), and the resulting 401 says why there was no credential.
func TestCloudApprover_MintFailureExplains401(t *testing.T) {
	rec := &authRecorder{want: "anything"}
	srv := httptest.NewServer(rec.handler())
	defer srv.Close()

	a := authApprover(srv, WithApproverIDTokenSource(func(context.Context) (string, error) {
		return "", errors.New("idtoken: unsupported credentials type")
	}))
	err := a.Approve(context.Background(), "op://Prod/k", "use")
	if err == nil {
		t.Fatal("expected a denial on 401")
	}
	for _, want := range []string{"401", "could not mint an ID token", "AILANG_APPROVAL_TOKEN"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q: %v", want, err)
		}
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.headers) != 1 || rec.headers[0] != "POST " {
		t.Fatalf("expected one unauthenticated create, got %v", rec.headers)
	}
}
