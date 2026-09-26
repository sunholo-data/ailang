package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoJSON_DecodesAndDefaultsHeaders(t *testing.T) {
	var gotCT, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT, gotAuth = r.Header.Get("Content-Type"), r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"ok":true,"n":3}`))
	}))
	defer srv.Close()

	var out struct {
		OK bool `json:"ok"`
		N  int  `json:"n"`
	}
	res, err := DoJSON(context.Background(), JSONCall{
		Provider: "test",
		URL:      srv.URL,
		Headers:  http.Header{"Authorization": []string{"Bearer k"}},
		Body:     []byte(`{}`),
	}, &out)
	if err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.N != 3 || res.StatusCode != 200 || len(res.Body) == 0 {
		t.Fatalf("decode: %+v res=%+v", out, res)
	}
	if gotCT != "application/json" || gotAuth != "Bearer k" {
		t.Fatalf("headers: ct=%q auth=%q", gotCT, gotAuth)
	}
}

// Non-2xx: the ProviderError keeps the status and hoisted envelope message
// (what Generate callers return), and its Err is the classification (what
// Step callers get from ClassifyError) — a 429 is a rate limit, a spent
// quota is not.
func TestDoJSON_ClassifiesNon2xx(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		body     string
		wantCode string
		wantMsg  string
		retry    bool
	}{
		{"429", 429, `{"error":{"message":"Rate limit reached for requests","type":"rate_limit_error"}}`, CodeRateLimit, "Rate limit reached for requests", true},
		{"429-quota", 429, `{"error":{"message":"you (marked) have reached your session usage limit, upgrade for higher limits","type":"api_error"}}`, CodeBudgetExhausted, "session usage limit", false},
		{"401", 401, `{"error":{"message":"Invalid API key"}}`, CodeAuthFailed, "Invalid API key", false},
		{"502-raw", 502, `upstream gateway timeout`, CodeInternal, "upstream gateway timeout", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			res, err := DoJSON(context.Background(), JSONCall{Provider: "p", URL: srv.URL, Body: []byte(`{}`)}, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if res == nil || res.StatusCode != tc.status {
				t.Fatalf("result = %+v", res)
			}
			var perr *ProviderError
			if !errors.As(err, &perr) || perr.StatusCode != tc.status || !strings.Contains(perr.Message, tc.wantMsg) {
				t.Fatalf("ProviderError = %+v", perr)
			}
			e := ClassifyError(err)
			if e.Code != tc.wantCode || e.Retryable != tc.retry {
				t.Fatalf("classified %s/%v, want %s/%v", e.Code, e.Retryable, tc.wantCode, tc.retry)
			}
			if ShouldRetry(err) != tc.retry {
				t.Fatalf("ShouldRetry = %v, want %v", ShouldRetry(err), tc.retry)
			}
		})
	}
}

func TestDoJSON_ErrorMessageHook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"detail":"custom shape"}`))
	}))
	defer srv.Close()
	_, err := DoJSON(context.Background(), JSONCall{
		Provider:     "p",
		URL:          srv.URL,
		ErrorMessage: func(b []byte) string { return "hoisted:" + string(b) },
	}, nil)
	var perr *ProviderError
	if !errors.As(err, &perr) || !strings.HasPrefix(perr.Message, "hoisted:") {
		t.Fatalf("got %v", err)
	}
}

func TestDoJSON_Undecodable2xxIsProtocolError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`this is not JSON`))
	}))
	defer srv.Close()
	var out map[string]any
	res, err := DoJSON(context.Background(), JSONCall{Provider: "p", URL: srv.URL}, &out)
	if err == nil || res == nil || res.StatusCode != 200 {
		t.Fatalf("err=%v res=%+v", err, res)
	}
	if e := ClassifyError(err); e.Code != CodeProtocolError || e.Retryable {
		t.Fatalf("classified %+v", e)
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("message: %v", err)
	}
}

func TestDoJSON_TransportErrorKeepsCause(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	res, err := DoJSON(ctx, JSONCall{Provider: "p", URL: srv.URL}, nil)
	if err == nil || res != nil {
		t.Fatalf("err=%v res=%+v", err, res)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cause lost: %v", err)
	}
	if e := ClassifyError(err); e.Code != CodeTimeout || !ShouldRetry(err) {
		t.Fatalf("classified %+v", e)
	}
}
