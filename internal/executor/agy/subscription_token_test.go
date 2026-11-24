package agy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// realShaped mirrors the key layout of a real agy 1.5.15 token file (values fake).
const realShaped = `{"token":{"access_token":"a","token_type":"Bearer","refresh_token":"r","expiry":"2026-11-24T12:14:57.227459+01:00","expires_in":3599},"auth_method":"oauth","id_token_email":"x@example.com"}`

func TestParseSubscriptionToken(t *testing.T) {
	if exp, err := ParseSubscriptionToken([]byte(realShaped)); err != nil || exp != "2026-11-24T12:14:57.227459+01:00" {
		t.Fatalf("real-shaped token: %q %v", exp, err)
	}
	for name, bad := range map[string]string{
		"empty":            "",
		"not json":         "nope",
		"no refresh token": `{"token":{"access_token":"a"},"auth_method":"oauth"}`,
		"no auth method":   `{"token":{"refresh_token":"r"}}`,
		"settings file":    `{"modelProvider":"gemini"}`,
	} {
		if _, err := ParseSubscriptionToken([]byte(bad)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestInstallSubscriptionToken(t *testing.T) {
	t.Setenv("AILANG_AGY_HOME_ROOT", "")
	if err := InstallSubscriptionToken([]byte(realShaped)); err == nil || !strings.Contains(err.Error(), "AILANG_AGY_HOME_ROOT") {
		t.Fatalf("no explicit HOME root must refuse, got %v", err)
	}
	root := setupHomes(t)
	if err := InstallSubscriptionToken([]byte(`{"modelProvider":"gemini"}`)); err == nil {
		t.Fatal("a non-token file was installed")
	}
	if err := InstallSubscriptionToken([]byte(realShaped)); err != nil {
		t.Fatal(err)
	}
	path, err := InstalledTokenPath()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "subscription", ".gemini", "antigravity-cli", "antigravity-oauth-token"); path != want {
		t.Fatalf("path = %s, want %s", path, want)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("token file mode: %v %v", fi, err)
	}
	if b, _ := os.ReadFile(path); string(b) != realShaped {
		t.Fatal("installed bytes differ")
	}
}

func TestRunQuota_EscapedGrandchildIsTerminationUnconfirmed(t *testing.T) {
	root := setupHomes(t)
	old := terminationGrace
	terminationGrace = 300 * time.Millisecond
	t.Cleanup(func() { terminationGrace = old })
	m := newMockAgy(t, ExpectedVersion, "", 0, `echo "Gemini Models	Weekly Limit Remaining	90%	2026-12-01T06:56:36Z"
perl -e 'use POSIX qw(setsid); setsid(); sleep 20' &
exit 0`)
	pinMockAsBinary(t, root, m)
	out, err := RunQuota(context.Background(), t.TempDir())
	if !errors.Is(err, executor.ErrProcessTerminationUnconfirmed) {
		t.Fatalf("err = %v, want ErrProcessTerminationUnconfirmed (out %q)", err, out)
	}
}

func TestRunQuota_Clean(t *testing.T) {
	root := setupHomes(t)
	m := newMockAgy(t, ExpectedVersion, "", 0, `echo "Gemini Models	Weekly Limit Remaining	90%	2026-12-01T06:56:36Z"`)
	pinMockAsBinary(t, root, m)
	out, err := RunQuota(context.Background(), t.TempDir())
	if err != nil || !strings.Contains(string(out), "Weekly Limit Remaining") {
		t.Fatalf("clean /quota: %q %v", out, err)
	}
}

// pinMockAsBinary installs the mock where defaultAgyPath looks first.
func pinMockAsBinary(t *testing.T, root string, m *mockAgy) {
	t.Helper()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(m.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "agy"), b, 0o755); err != nil {
		t.Fatal(err)
	}
}
