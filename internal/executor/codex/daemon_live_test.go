package codex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Opt-in real CLI ownership spike. All credentials are fake, every quota and
// refresh response is local, and the daemon belongs only to t.TempDir().
func TestDaemonRotatingTokenOwnership(t *testing.T) {
	if os.Getenv("AILANG_CODEX_OWNER_SPIKE") != "1" {
		t.Skip("set AILANG_CODEX_OWNER_SPIKE=1 for installed Codex fake-token ownership spike")
	}
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("Codex not installed")
	}
	var refresh atomic.Int32
	token := func(exp int64) string {
		data, _ := json.Marshal(map[string]any{"exp": exp, "https://api.openai.com/auth": map[string]any{"chatgpt_account_id": "fake-account", "chatgpt_plan_type": "plus"}})
		return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(data) + ".fake"
	}
	fresh := token(time.Now().Add(time.Hour).Unix())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "POST" && r.URL.Path == "/token" {
			refresh.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": fresh, "refresh_token": "fake-rotated", "id_token": fresh})
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+fresh {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"error":{"message":"fixture obsolete token"}}`)
			return
		}
		if r.URL.Path == "/wham/usage" || r.URL.Path == "/api/codex/usage" {
			_ = json.NewEncoder(w).Encode(map[string]any{"plan_type": "plus", "rate_limit": map[string]any{"allowed": true, "limit_reached": false, "primary_window": map[string]any{"used_percent": 5, "limit_window_seconds": 18000, "reset_after_seconds": 18000, "reset_at": time.Now().Add(5 * time.Hour).Unix()}}, "credits": map[string]any{"has_credits": false, "unlimited": false, "balance": "0"}})
			return
		}
		fmt.Fprint(w, `{"models":[]}`)
	}))
	defer server.Close()
	home := t.TempDir()
	cfg := fmt.Sprintf("cli_auth_credentials_store=\"file\"\nchatgpt_base_url=%q\n[analytics]\nenabled=false\n", server.URL)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]any{"auth_mode": "chatgpt", "OPENAI_API_KEY": nil, "last_refresh": time.Now().Add(-10 * 24 * time.Hour).Format(time.RFC3339), "tokens": map[string]any{"id_token": token(time.Now().Add(-time.Hour).Unix()), "access_token": "fake-old", "refresh_token": "fake-original", "account_id": "fake-account"}})
	if err := os.WriteFile(filepath.Join(home, "auth.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	owner := exec.CommandContext(context.Background(), "codex", "app-server", "-c", "chatgpt_base_url="+fmt.Sprintf("%q", server.URL), "--listen", "unix://")
	owner.Env = append(os.Environ(), "CODEX_HOME="+home, "CODEX_REFRESH_TOKEN_URL_OVERRIDE="+server.URL+"/token")
	configureProcessTree(owner)
	if err := owner.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { killProcessTree(owner); _ = owner.Wait() }()
	socket := filepath.Join(home, "app-server-control", "app-server-control.sock")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(socket); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated daemon socket did not become ready")
		}
		time.Sleep(50 * time.Millisecond)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_, err := CallDaemonRPC(ctx, home, "account/rateLimits/read", nil)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent read: %v", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := CallDaemonRPC(ctx, home, "account/rateLimits/read", nil); err != nil {
		t.Fatalf("later read: %v", err)
	}
	if got := refresh.Load(); got != 1 {
		t.Fatalf("refresh requests=%d want one owner rotation", got)
	}
	t.Log("8 concurrent daemon clients + later client succeeded with exactly one fake-token refresh")
}
