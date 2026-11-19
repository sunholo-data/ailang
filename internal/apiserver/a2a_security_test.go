package apiserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

// fakeIssuer serves RFC 8414 metadata; name overrides the issuer it claims.
func fakeIssuer(t *testing.T, name string, hits *int32) *httptest.Server {
	t.Helper()
	var hs *httptest.Server
	hs = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != issuerMetaPath {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(hits, 1)
		iss := hs.URL
		if name != "" {
			iss = name
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": iss, "authorization_endpoint": hs.URL + "/oauth/authorize",
			"token_endpoint": hs.URL + "/oauth/token", "code_challenge_methods_supported": []string{"S256"},
		})
	}))
	t.Cleanup(hs.Close)
	return hs
}

func getCard(t *testing.T, url string) map[string]any {
	t.Helper()
	resp, err := http.Get(url + "/.well-known/agent-card.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var card map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatal(err)
	}
	return card
}

func TestA2ACardOAuthSchemeFromIssuer(t *testing.T) {
	var hits int32
	iss := fakeIssuer(t, "", &hits)
	hs := a2aGatedHTTP(t, Config{OAuthIssuer: iss.URL})

	card := getCard(t, hs.URL)
	oauth, ok := card["securitySchemes"].(map[string]any)["oauth"].(map[string]any)
	if !ok {
		t.Fatalf("card has no securitySchemes.oauth: %#v", card["securitySchemes"])
	}
	flow := oauth["flows"].(map[string]any)["authorizationCode"].(map[string]any)
	if oauth["type"] != "oauth2" || flow["authorizationUrl"] != iss.URL+"/oauth/authorize" || flow["tokenUrl"] != iss.URL+"/oauth/token" {
		t.Fatalf("scheme = %#v, want the issuer's authorize and token endpoints", oauth)
	}
	if oauth["oauth2MetadataUrl"] != iss.URL+issuerMetaPath {
		t.Fatalf("oauth2MetadataUrl = %v", oauth["oauth2MetadataUrl"])
	}
	if _, ok := flow["scopes"].(map[string]any); !ok {
		t.Fatalf("scopes must be an object (A2A requires it): %#v", flow["scopes"])
	}
	sec, _ := card["security"].([]any)
	if len(sec) != 1 || sec[0].(map[string]any)["oauth"] == nil {
		t.Fatalf("security = %#v, want [{oauth: []}]", card["security"])
	}
	getCard(t, hs.URL)
	if n := atomic.LoadInt32(&hits); n != 1 {
		t.Fatalf("issuer metadata read %d times for two cards, want 1 (cached)", n)
	}
}

func TestA2ACardOAuthSchemeOmittedWhenIssuerUnusable(t *testing.T) {
	var hits int32
	liar := fakeIssuer(t, "https://someone-else.example", &hits)
	down := httptest.NewServer(http.NotFoundHandler())
	down.Close()
	for name, issuer := range map[string]string{"issuer mismatch": liar.URL, "issuer down": down.URL} {
		hs := a2aGatedHTTP(t, Config{OAuthIssuer: issuer})
		card := getCard(t, hs.URL)
		if _, ok := card["securitySchemes"]; ok {
			t.Errorf("%s: card advertises a scheme: %#v", name, card["securitySchemes"])
		}
		if card["protocolVersion"] != a2aProtocolVersion || card["url"] == nil {
			t.Errorf("%s: the card itself must still be served: %#v", name, card)
		}
	}
}

func TestA2ACardNoOAuthSchemeWithoutGate(t *testing.T) {
	// An issuer is configured and reachable, but /a2a/ is open (not
	// --mcp-connect-only): the card must not claim sign-in is needed.
	var hits int32
	iss := fakeIssuer(t, "", &hits)
	eff := effects.NewEffContext(nil)
	srv, ts := newSurfaceServer(t, Config{A2A: true, A2AMessage: "reply", EffCtx: eff, OAuthIssuer: iss.URL},
		map[string]string{"agent.ail": a2aAgentSrc})
	if err := srv.ValidateA2A(); err != nil {
		t.Fatal(err)
	}
	if card := getCard(t, ts.URL); card["securitySchemes"] != nil || card["security"] != nil {
		t.Fatalf("an open /a2a/ card advertises sign-in: %#v", card)
	}
}

func TestSameSchemeURL(t *testing.T) {
	for _, c := range []struct {
		issuer, u string
		ok        bool
	}{
		{"https://a", "https://a/x", true},
		{"https://a", "http://a/x", false},
		{"http://localhost", "http://localhost/x", true},
		{"https://a", "javascript:alert(1)", false},
		{"https://a", "", false},
	} {
		if got := sameSchemeURL(c.issuer, c.u); got != c.ok {
			t.Errorf("sameSchemeURL(%q, %q) = %v, want %v", c.issuer, c.u, got, c.ok)
		}
	}
}
