package mission

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type quotaRoundTrip func(*http.Request) (*http.Response, error)

func (f quotaRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestOllamaQuotaRawUnitsRequireMetadata(t *testing.T) {
	now := time.Now()
	for _, body := range []string{`{}`, `{"limits":{"session":{"usage":0},"weekly":{}}}`, `{"limits":{"session":{"usage":-1},"weekly":{"usage":0}}}`, `{"limits":{"session":{"usage":"0"},"weekly":{"usage":0}}}`} {
		if o := parseOllamaUsage([]byte(body), now); !o.Blocked() || o.WeeklyUsage != nil {
			t.Fatalf("accepted malformed: %+v", o)
		}
	}
	o := parseOllamaUsage([]byte(`{"limits":{"session":{"usage":0.011},"weekly":{"usage":0.356}}}`), now)
	if !o.Blocked() || o.WeeklyUsage == nil || *o.WeeklyUsage != 0.356 || len(o.Windows) != 0 {
		t.Fatal(o)
	}
}
func TestOllamaQuotaVerifiedLimits(t *testing.T) {
	now := time.Now()
	o := parseOllamaUsage([]byte(`{"limits":{"session":{"usage":0},"weekly":{"usage":0.356}}}`), now)
	limits := OllamaQuotaLimits{SessionCapacity: 1, WeeklyCapacity: 1, SessionResetsAt: now.Add(time.Hour), WeeklyResetsAt: now.Add(6 * 24 * time.Hour)}
	v := evaluateOllamaQuota(o, limits, now)
	if v.State != "over" || v.Windows[1].UsedPercent != 35.6 {
		t.Fatal(v)
	}
	limits.WeeklyCapacity = 10
	v = evaluateOllamaQuota(o, limits, now)
	if v.Blocked() {
		t.Fatal(v)
	}
	limits.WeeklyResetsAt = now.Add(-time.Second)
	v = evaluateOllamaQuota(o, limits, now)
	if !v.Blocked() {
		t.Fatal("expired metadata admitted")
	}
}
func TestOllamaQuotaHTTPAndCredentialBinding(t *testing.T) {
	paths := Paths{Home: t.TempDir()}
	now := time.Now()
	key := "fixture-only-key"
	calls := 0
	client := &http.Client{Transport: quotaRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://ollama.com/api/usage" || r.Header.Get("Authorization") != "Bearer "+key {
			t.Fatal("wrong destination or auth")
		}
		if _, ok := r.Context().Deadline(); !ok {
			t.Fatal("missing deadline")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"limits":{"session":{"usage":0},"weekly":{"usage":0}}}`))}, nil
	})}
	if o := observeOllamaQuota(paths, "", now, client); !o.Blocked() || calls != 0 {
		t.Fatal(o)
	}
	if o := observeOllamaQuota(paths, key, now, client); !o.Blocked() || o.WeeklyUsage == nil {
		t.Fatal(o)
	}
	dir := filepath.Join(paths.Home, ".ailang", "state")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(key))
	limits := OllamaQuotaLimits{CredentialSHA256: hex.EncodeToString(sum[:]), Source: "fixture account page", SessionCapacity: 1, WeeklyCapacity: 1, SessionResetsAt: now.Add(time.Hour), WeeklyResetsAt: now.Add(6 * 24 * time.Hour)}
	data, _ := json.Marshal(limits)
	if err := os.WriteFile(filepath.Join(dir, "ollama-quota-limits.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if o := observeOllamaQuota(paths, key, now, client); o.Blocked() {
		t.Fatal(o)
	}
	limits.CredentialSHA256 = "other-account"
	data, _ = json.Marshal(limits)
	_ = os.WriteFile(filepath.Join(dir, "ollama-quota-limits.json"), data, 0600)
	if o := observeOllamaQuota(paths, key, now, client); !o.Blocked() {
		t.Fatal("cross-account metadata admitted")
	}
}
func TestOllamaQuotaHTTPFailureDoesNotExposeBody(t *testing.T) {
	for _, status := range []int{302, 401, 429, 500} {
		client := &http.Client{Transport: quotaRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("sensitive diagnostic"))}, nil
		})}
		o := observeOllamaQuota(Paths{Home: t.TempDir()}, "key", time.Now(), client)
		if !o.Blocked() || strings.Contains(o.Reason, "sensitive") {
			t.Fatal(o)
		}
	}
}

func TestOllamaQuotaRejectsOversizedResponse(t *testing.T) {
	client := &http.Client{Transport: quotaRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(strings.Repeat(" ", (1<<20)+1)))}, nil
	})}
	o := observeOllamaQuota(Paths{Home: t.TempDir()}, "key", time.Now(), client)
	if !o.Blocked() || !strings.Contains(o.Reason, "1 MiB") {
		t.Fatal(o)
	}
}
