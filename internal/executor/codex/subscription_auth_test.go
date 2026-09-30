package codex

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

func chatgptAuth(lastRefresh string) []byte {
	return []byte(`{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"last_refresh":"` + lastRefresh +
		`","tokens":{"access_token":"a","id_token":"i","refresh_token":"r","account_id":"x"}}`)
}

func TestParseSubscriptionAuth_RejectsAPIKeyFile(t *testing.T) {
	_, err := ParseSubscriptionAuth([]byte(`{"auth_mode":"apikey","OPENAI_API_KEY":"sk-x"}`))
	if err == nil || !strings.Contains(err.Error(), "metered") {
		t.Fatalf("an api-key auth.json must be refused on the subscription path, got %v", err)
	}
}

func TestParseSubscriptionAuth_RequiresRefreshToken(t *testing.T) {
	_, err := ParseSubscriptionAuth([]byte(`{"auth_mode":"chatgpt","last_refresh":"2026-09-30T10:00:00Z","tokens":{}}`))
	if err == nil {
		t.Fatal("a subscription file without a refresh_token cannot survive its first refresh")
	}
}

func TestNewerRefresh(t *testing.T) {
	old := chatgptAuth("2026-09-30T10:00:00.000000Z")
	newer := chatgptAuth("2026-10-08T10:00:00.000000Z")
	if !NewerRefresh(old, newer) {
		t.Error("a later last_refresh must be written back")
	}
	if NewerRefresh(old, old) {
		t.Error("an unchanged file must not be written back")
	}
	if NewerRefresh(newer, old) {
		t.Error("an older file must never replace a newer stored one")
	}
	if NewerRefresh(old, []byte(`{"auth_mode":"apikey","OPENAI_API_KEY":"sk"}`)) {
		t.Error("an api-key file must never be written back over the subscription secret")
	}
}

func TestInstallSubscriptionAuth_WritesOnceAndNeverOverwrites(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	data := chatgptAuth("2026-09-30T10:00:00Z")
	if err := InstallSubscriptionAuth(data); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, err := os.ReadFile(authPath(home))
	if err != nil || string(got) != string(data) {
		t.Fatalf("installed file = %q, %v", got, err)
	}
	// Windows has no POSIX mode bits (it reports 0666); same guard as the api-key test.
	if info, _ := os.Stat(authPath(home)); runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
	if err := InstallSubscriptionAuth(chatgptAuth("2026-10-01T10:00:00Z")); err == nil {
		t.Fatal("a second install must refuse to overwrite the credential in use")
	}
}
