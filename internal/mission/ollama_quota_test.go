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

// ollamaPaceFixtureNow pins the tests that expect an ADMITTED verdict. The weekly allowance
// is the weekday pace line (WeekdayPacePercent), and both fixtures open their weekly window
// one day before now, so on the wall clock the allowance swung from 20% midweek to 0% on a
// Sunday. With the 3-point ollama start margin (quota_margin.go) the 3.56% and 0% fixtures
// fell short of headroom from Saturday ~08:00 to Monday ~08:00 UTC, and CI went red
// every weekend after the margin landed (#1524; PRs #1578-#1582). Wednesday noon: 20% allowance.
var ollamaPaceFixtureNow = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)

func TestOllamaQuotaVerifiedLimits(t *testing.T) {
	now := ollamaPaceFixtureNow
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
	now := ollamaPaceFixtureNow
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
	if o := observeOllamaQuota(paths, key, now, client); o.Blocked() || o.WeeklyUsage == nil {
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

func TestOllamaGaugeThresholdsWithoutMetadata(t *testing.T) {
	for _, tc := range []struct {
		session, weekly float64
		status          string
		blocked         bool
	}{
		{0.006, 0.357, "OK", false}, {0.799, 0, "OK", false},
		{0.8, 0, "WARN", false}, {0, 0.8, "WARN", false},
		{0.949, 0.949, "WARN", false}, {0.95, 0, "CRITICAL", true},
		{0, 0.95, "CRITICAL", true}, {1, 0, "CRITICAL", true}, {0, 1.1, "CRITICAL", true},
	} {
		body, _ := json.Marshal(map[string]any{"limits": map[string]any{"session": map[string]any{"usage": tc.session}, "weekly": map[string]any{"usage": tc.weekly}}})
		client := &http.Client{Transport: quotaRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
		})}
		o := observeOllamaQuota(Paths{Home: t.TempDir()}, "fixture", time.Now(), client)
		if o.Blocked() != tc.blocked || o.GaugeStatus != tc.status || len(o.Windows) != 0 {
			t.Fatalf("%+v: %+v", tc, o)
		}
	}
}

func TestOllamaMetadataCannotRelaxGaugeCutoff(t *testing.T) {
	now := time.Now()
	o := parseOllamaUsage([]byte(`{"limits":{"session":{"usage":0.95},"weekly":{"usage":0}}}`), now)
	limits := OllamaQuotaLimits{SessionCapacity: 100, WeeklyCapacity: 100, SessionResetsAt: now.Add(time.Hour), WeeklyResetsAt: now.Add(time.Hour)}
	if v := evaluateOllamaQuota(o, limits, now); !v.Blocked() || v.GaugeStatus != "CRITICAL" {
		t.Fatal(v)
	}
}

// The /api/usage shape since ~2026-10-07 (live response, counts trimmed): request counts only.
// Admitted unrationed (Mark, attended 2026-10-08, option A); anything unrecognised still blocks.
func TestOllamaQuotaRequestCountShapeIsAdmittedUnrationed(t *testing.T) {
	body := `{"range":"7d","scope":"self","granularity":"day","from":"2026-10-01T00:00:00Z","until":"2026-10-08T08:30:33Z",
	  "totals":{"request_count":923},
	  "buckets":[{"from":"2026-10-01T00:00:00Z","until":"2026-10-02T00:00:00Z","request_count":292},
	             {"from":"2026-10-08T00:00:00Z","until":"2026-10-08T08:30:33Z","partial":true,"request_count":8}]}`
	client := &http.Client{Transport: quotaRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	o := observeOllamaQuota(Paths{Home: t.TempDir()}, "key", time.Now(), client)
	if o.Blocked() || o.GaugeStatus != "UNMEASURED" || o.RequestCount == nil || *o.RequestCount != 923 {
		t.Fatalf("request-count shape: %+v, want admitted UNMEASURED with 923 requests", o)
	}
	if !strings.Contains(o.Reason, "923 in 7d") {
		t.Errorf("reason = %q, want the count and range", o.Reason)
	}
	for _, bad := range []string{`{}`, `{"range":"7d","buckets":[]}`, `{"range":"7d","totals":{"request_count":-1},"buckets":[]}`, `{"totals":{"request_count":5},"buckets":[]}`} {
		client := &http.Client{Transport: quotaRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(bad))}, nil
		})}
		if o := observeOllamaQuota(Paths{Home: t.TempDir()}, "key", time.Now(), client); !o.Blocked() {
			t.Errorf("%s admitted: %+v", bad, o)
		}
	}
}
