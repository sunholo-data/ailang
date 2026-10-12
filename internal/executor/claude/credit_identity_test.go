package claude

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

type creditIdentityTransport func(*http.Request) (*http.Response, error)

func (f creditIdentityTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCreditChildIdentityAuthenticationBoundary(t *testing.T) {
	const gateway = "https://credit-gateway.example"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	claims, _ := json.Marshal(map[string]any{"aud": gateway, "exp": time.Now().Add(time.Hour).Unix()})
	token := "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(claims) + ".fixture-signature"
	var calls atomic.Int32
	var deny atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if deny.Load() {
			http.Error(w, "fixture token exchange denied", http.StatusForbidden)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		parts := strings.Split(r.Form.Get("assertion"), ".")
		if len(parts) != 3 {
			t.Error("identity exchange must carry a signed service-account assertion")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		var assertion map[string]any
		if err != nil || json.Unmarshal(payload, &assertion) != nil || assertion["target_audience"] != gateway {
			t.Error("job identity request was not bound to the credit gateway audience")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id_token": token, "token_type": "Bearer", "expires_in": 3600})
	}))
	defer server.Close()
	credential := map[string]string{
		"type": "service_account", "client_email": "fixture@example.iam.gserviceaccount.com",
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})),
		"token_uri":   server.URL,
	}
	data, _ := json.Marshal(credential)
	credentialPath := filepath.Join(t.TempDir(), "fixture-credentials.json")
	if err := os.WriteFile(credentialPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	// Explicit synthetic ADC and a transport that cannot reach Google or any
	// other host keep both authentication success and failure entirely local.
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialPath)
	client := &http.Client{Transport: creditIdentityTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != server.URL {
			return nil, fmt.Errorf("fixture refuses nonlocal authentication route")
		}
		return server.Client().Transport.RoundTrip(r)
	})}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	t.Setenv("AILANG_CLAUDE_CREDIT_ACCOUNT", "anthropic-api-credits")
	t.Setenv("ANTHROPIC_BASE_URL", gateway)
	env, err := prepareCreditChildEnvironment(ctx, []string{"ANTHROPIC_API_KEY=task-capability", "ANTHROPIC_CUSTOM_HEADERS=Authorization: forged", "CLAUDE_CODE_USE_GATEWAY=1"}, 0, 1800)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		if _, duplicate := values[name]; duplicate {
			t.Fatalf("ambiguous child environment entry %s", name)
		}
		values[name] = value
	}
	if values["ANTHROPIC_CUSTOM_HEADERS"] != "Authorization: Bearer "+token+"\nX-Serverless-Authorization: Bearer "+token || values["ANTHROPIC_API_KEY"] != "task-capability" || values["ANTHROPIC_BASE_URL"] != gateway || values["CLAUDE_CODE_USE_GATEWAY"] != "" || calls.Load() != 1 {
		t.Fatal("child must receive only the audience-bound identity and guarded task capability")
	}
	for _, timeout := range []time.Duration{0, -time.Second, 51 * time.Minute} {
		if _, err := creditJobIdentityHeader(ctx, gateway, timeout); err == nil {
			t.Errorf("unbounded credential lifetime admitted for timeout %v", timeout)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("invalid timeout reached credential exchange")
	}
	deny.Store(true)
	if env, err := prepareCreditChildEnvironment(ctx, nil, time.Minute, 1800); err == nil || env != nil {
		t.Fatal("denied authentication must prevent child launch")
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing-fixture.json"))
	if env, err := prepareCreditChildEnvironment(ctx, nil, time.Minute, 1800); err == nil || env != nil {
		t.Fatal("missing authentication must prevent child launch")
	}
}

func TestCreditEnvironmentRejectsIncoherentOriginsAndCapabilities(t *testing.T) {
	t.Setenv("AILANG_CLAUDE_CREDIT_ACCOUNT", "anthropic-api-credits")
	t.Setenv("AILANG_AUTH_MODE", "apikey")
	t.Setenv("AILANG_MAX_COST_USD", "2")
	t.Setenv("ANTHROPIC_API_KEY", "task-capability")
	for _, origin := range []string{"http://gateway", "https://user@gateway", "https://gateway/path", "https://gateway?route=other", "https://gateway#fragment", "%invalid", "https:///missing-host"} {
		t.Setenv("ANTHROPIC_BASE_URL", origin)
		if err := validateCreditEnvironment("claude-haiku-5-5"); err == nil {
			t.Errorf("incoherent credit origin admitted: %s", origin)
		}
	}
	t.Setenv("ANTHROPIC_BASE_URL", "https://gateway")
	for _, capability := range []string{"", "ENC:not-decrypted"} {
		t.Setenv("ANTHROPIC_API_KEY", capability)
		if err := validateCreditEnvironment("claude-haiku-5-5"); err == nil {
			t.Error("unusable task capability admitted")
		}
	}
	t.Setenv("AILANG_CLAUDE_CREDIT_ACCOUNT", "")
	if err := validateCreditEnvironment("opus"); err != nil {
		t.Fatal("legacy lane was subjected to credit validation")
	}
}
