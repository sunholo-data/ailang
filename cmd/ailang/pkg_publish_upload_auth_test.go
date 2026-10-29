package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestPublishUpload_FrozenHeaderAndReflection(t *testing.T) {
	isolatePublishCLI(t)
	t.Setenv(config.EnvRegistryAPIKey, "different-dummy-env")
	const key = "selected-dummy-header"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.Header.Get("X-API-Key") != key {
			t.Error("wrong frozen header")
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), key) {
			t.Error("key placed in multipart artifacts")
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "specific validation cause: "+key)
	}))
	defer server.Close()
	err := uploadTarball(server.URL, []byte("synthetic-package"), false, "", nil, key)
	if err == nil || !strings.Contains(err.Error(), "specific validation cause") || strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), "[REDACTED]") || calls != 1 {
		t.Fatalf("unsafe reflection or status: %v calls=%d", err, calls)
	}
}

func TestPublishUpload_NoAnonymousOrConfinedRequest(t *testing.T) {
	isolatePublishCLI(t)
	t.Setenv(config.EnvRegistryAPIKey, "unused-dummy-env")
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	if err := uploadTarball(server.URL, nil, false, "", nil, ""); err == nil {
		t.Fatal("anonymous request accepted")
	}
	t.Setenv(config.EnvAgentPolicy, "private-policy")
	err := uploadTarball(server.URL, nil, false, "", nil, "dummy-key")
	var refusal *pkg.ConfinedError
	if !errors.As(err, &refusal) || calls != 0 {
		t.Fatalf("authority/request ordering: %v calls=%d", err, calls)
	}
}
