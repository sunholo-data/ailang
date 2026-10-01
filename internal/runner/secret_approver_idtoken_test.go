package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

// TestIDTokenMinter_LazyOnceNormalizedAudience: the credential lookup happens
// on the first secret(), once, for the approval URL as the dashboard
// normalizes it (no trailing slash) — and the minted token is what is sent.
func TestIDTokenMinter_LazyOnceNormalizedAudience(t *testing.T) {
	calls := 0
	var gotAudience string
	mint := idTokenMinter("https://dash.example/ ", func(_ context.Context, aud string, _ ...idtoken.ClientOption) (oauth2.TokenSource, error) {
		calls++
		gotAudience = aud
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "eyJ.minted.id"}), nil
	})
	if calls != 0 {
		t.Fatal("building the minter must not look up credentials")
	}
	for i := 0; i < 2; i++ {
		tok, err := mint(context.Background())
		if err != nil || tok != "eyJ.minted.id" {
			t.Fatalf("mint %d: tok=%q err=%v", i, tok, err)
		}
	}
	if calls != 1 {
		t.Errorf("expected one token source, built %d", calls)
	}
	if gotAudience != "https://dash.example" {
		t.Errorf("audience should be the normalized approval URL, got %q", gotAudience)
	}
}

func TestIDTokenMinter_SourceErrorSurfaces(t *testing.T) {
	mint := idTokenMinter("https://dash.example", func(context.Context, string, ...idtoken.ClientOption) (oauth2.TokenSource, error) {
		return nil, errors.New("no credentials")
	})
	if _, err := mint(context.Background()); err == nil || !strings.Contains(err.Error(), "no credentials") {
		t.Fatalf("expected the source error, got %v", err)
	}
}

// TestIDTokenMinter_UserADCCannotMint pins the real library's behaviour for a
// laptop's `gcloud auth application-default login` credential: it cannot mint
// an ID token, so local/CLI users authenticate with AILANG_APPROVAL_TOKEN.
func TestIDTokenMinter_UserADCCannotMint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "adc.json")
	adc := `{"type":"authorized_user","client_id":"x.apps.googleusercontent.com","client_secret":"s","refresh_token":"r"}`
	if err := os.WriteFile(path, []byte(adc), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", path)
	if _, err := idTokenMinter("https://dash.example", idtoken.NewTokenSource)(context.Background()); err == nil {
		t.Fatal("expected user ADC to be refused for ID-token minting")
	}
}
