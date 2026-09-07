package mission

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// OllamaQuotaObservation retains the legacy session/weekly fractional usage gauge.
type OllamaQuotaObservation struct {
	GaugeStatus  string             `json:"gauge_status,omitempty"`
	State        string             `json:"state"`
	Reason       string             `json:"reason"`
	ObservedAt   time.Time          `json:"observed_at"`
	SessionUsage *float64           `json:"session_usage,omitempty"`
	WeeklyUsage  *float64           `json:"weekly_usage,omitempty"`
	Windows      []CodexQuotaWindow `json:"windows,omitempty"`
}

func (o OllamaQuotaObservation) Blocked() bool { return o.State != "ok" }

// OllamaQuotaLimits is explicit account metadata, never inferred from exhaustion.
type OllamaQuotaLimits struct {
	CredentialSHA256 string    `json:"credential_sha256"`
	Source           string    `json:"source"`
	SessionCapacity  float64   `json:"session_capacity"`
	WeeklyCapacity   float64   `json:"weekly_capacity"`
	SessionResetsAt  time.Time `json:"session_resets_at"`
	WeeklyResetsAt   time.Time `json:"weekly_resets_at"`
}

// ObserveOllamaQuota makes a bounded usage-only call; it never makes an inference call.
func ObserveOllamaQuota(paths Paths, key string, now time.Time) OllamaQuotaObservation {
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return observeOllamaQuota(paths, key, now, client)
}
func observeOllamaQuota(paths Paths, key string, now time.Time, client *http.Client) OllamaQuotaObservation {
	o := OllamaQuotaObservation{State: "unknown", Reason: "Ollama Cloud quota unavailable; new cloud routing blocked", ObservedAt: now}
	if key == "" {
		o.Reason = "OLLAMA_API_KEY missing; cannot read Ollama Cloud quota"
		return o
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ollama.com/api/usage", nil)
	if err != nil {
		return o
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := client.Do(req)
	if err != nil {
		o.Reason = "Ollama usage request failed or timed out"
		return o
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		o.Reason = fmt.Sprintf("Ollama usage endpoint returned HTTP %d", resp.StatusCode)
		return o
	}
	const limit = 1 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || len(data) > limit {
		o.Reason = "Ollama usage response unreadable or exceeds 1 MiB"
		return o
	}
	o = parseOllamaUsage(data, now)
	if o.SessionUsage == nil || o.WeeklyUsage == nil {
		return o
	}
	metadataPath := filepath.Join(paths.Home, ".ailang", "state", "ollama-quota-limits.json")
	info, statErr := os.Lstat(metadataPath)
	if statErr == nil && (!info.Mode().IsRegular() || info.Size() > 16<<10) {
		o.Reason = "Ollama quota metadata must be a regular file no larger than 16 KiB"
		return o
	}
	f, err := os.Open(metadataPath)
	if os.IsNotExist(err) {
		return evaluateOllamaGauge(o)
	}
	if err != nil {
		o.Reason = "Ollama quota metadata cannot be read"
		return o
	}
	defer f.Close()
	var limits OllamaQuotaLimits
	decoder := json.NewDecoder(io.LimitReader(f, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&limits); err != nil {
		o.Reason = "invalid Ollama quota account metadata"
		return o
	}
	if decoder.Decode(new(any)) != io.EOF {
		o.Reason = "invalid trailing Ollama quota account metadata"
		return o
	}
	sum := sha256.Sum256([]byte(key))
	if limits.CredentialSHA256 != hex.EncodeToString(sum[:]) || limits.Source == "" {
		o.Reason = "Ollama quota metadata is not bound to this credential with provenance"
		return o
	}
	return evaluateOllamaQuota(o, limits, now)
}
func parseOllamaUsage(data []byte, now time.Time) OllamaQuotaObservation {
	o := OllamaQuotaObservation{State: "unknown", Reason: "unsupported or malformed Ollama quota response", ObservedAt: now}
	var body struct {
		Limits struct {
			Session struct {
				Usage *float64 `json:"usage"`
			} `json:"session"`
			Weekly struct {
				Usage *float64 `json:"usage"`
			} `json:"weekly"`
		} `json:"limits"`
	}
	if json.Unmarshal(data, &body) != nil {
		return o
	}
	for _, v := range []*float64{body.Limits.Session.Usage, body.Limits.Weekly.Usage} {
		if v == nil || *v < 0 || math.IsNaN(*v) || math.IsInf(*v, 0) {
			return o
		}
	}
	o.SessionUsage = body.Limits.Session.Usage
	o.WeeklyUsage = body.Limits.Weekly.Usage
	o.Reason = "provider usage units recorded; capacity and reset metadata required"
	return o
}

// evaluateOllamaGauge uses the existing Pi extension policy: V50 measured session
// exhaustion at 1.0; weekly uses Pi's same fractional interpretation. Reset times
// are not supplied by this endpoint, so this verdict does not claim daily pacing.
func evaluateOllamaGauge(o OllamaQuotaObservation) OllamaQuotaObservation {
	if o.SessionUsage == nil || o.WeeklyUsage == nil {
		return o
	}
	o.State, o.GaugeStatus = "ok", "OK"
	o.Reason = "Ollama Cloud gauge below 80%; reset-aware pacing unavailable without metadata"
	for _, usage := range []float64{*o.SessionUsage, *o.WeeklyUsage} {
		if usage >= 0.95 {
			o.State, o.GaugeStatus = "over", "CRITICAL"
			o.Reason = "Ollama Cloud session or weekly gauge reached 95%; new cloud routing blocked"
			return o
		}
		if usage >= 0.8 {
			o.GaugeStatus = "WARN"
			o.Reason = "Ollama Cloud session or weekly gauge reached 80%; admission allowed below 95%; reset-aware pacing unavailable without metadata"
		}
	}
	return o
}

func evaluateOllamaQuota(o OllamaQuotaObservation, limits OllamaQuotaLimits, now time.Time) OllamaQuotaObservation {
	gauge := evaluateOllamaGauge(o)
	if gauge.Blocked() {
		return gauge
	}
	o.GaugeStatus = gauge.GaugeStatus
	if o.SessionUsage == nil || o.WeeklyUsage == nil {
		return o
	}
	for _, c := range []float64{limits.SessionCapacity, limits.WeeklyCapacity} {
		if c <= 0 || math.IsNaN(c) || math.IsInf(c, 0) {
			o.Reason = "invalid Ollama quota capacities"
			return o
		}
	}
	for _, w := range []struct {
		reset    time.Time
		duration time.Duration
	}{{limits.SessionResetsAt, 5 * time.Hour}, {limits.WeeklyResetsAt, 7 * 24 * time.Hour}} {
		if !w.reset.After(now) || w.reset.Sub(now) > w.duration {
			o.Reason = "Ollama quota reset metadata expired or outside its verified window"
			return o
		}
	}
	o.Windows = []CodexQuotaWindow{{UsedPercent: 100 * (*o.SessionUsage / limits.SessionCapacity), WindowMinutes: 300, ResetsAt: limits.SessionResetsAt}, {UsedPercent: 100 * (*o.WeeklyUsage / limits.WeeklyCapacity), WindowMinutes: 10080, ResetsAt: limits.WeeklyResetsAt}}
	for _, w := range o.Windows {
		if math.IsNaN(w.UsedPercent) || math.IsInf(w.UsedPercent, 0) {
			o.Windows = nil
			o.Reason = "Ollama usage/capacity ratio is not finite"
			return o
		}
	}
	// Reuse identical percentage pacing; keep Ollama identity in the public reason.
	verdict := CodexQuotaObservation{ObservedAt: now, Windows: o.Windows}
	verdict.evaluate(now)
	o.State = verdict.State
	o.Windows = verdict.Windows
	o.Reason = "Ollama Cloud usage is within the verified account ration; gauge " + o.GaugeStatus
	if verdict.Blocked() {
		o.Reason = "Ollama Cloud usage exceeds its verified ration or exhausts a window"
	}
	return o
}
