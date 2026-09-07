package mission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// CodexQuotaObservation is provider account usage, never a token-capacity estimate.
type CodexQuotaObservation struct {
	State      string             `json:"state"`
	Reason     string             `json:"reason"`
	ObservedAt time.Time          `json:"observed_at,omitempty"`
	Source     string             `json:"source,omitempty"`
	Windows    []CodexQuotaWindow `json:"windows,omitempty"`
}
type CodexQuotaWindow struct {
	UsedPercent      float64   `json:"used_percent"`
	WindowMinutes    int64     `json:"window_minutes"`
	ResetsAt         time.Time `json:"resets_at"`
	AllowancePercent float64   `json:"allowance_percent"`
}

func (o CodexQuotaObservation) Blocked() bool { return o.State != "ok" }

type codexRateWindow struct {
	Used    *float64 `json:"used_percent"`
	Minutes int64    `json:"window_minutes"`
	Reset   int64    `json:"resets_at"`
}
type codexQuotaEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Payload   struct {
		Type       string `json:"type"`
		RateLimits *struct {
			LimitID   string           `json:"limit_id"`
			Primary   *codexRateWindow `json:"primary"`
			Secondary *codexRateWindow `json:"secondary"`
		} `json:"rate_limits"`
	} `json:"payload"`
}

func parseCodexQuota(line []byte, now time.Time) *CodexQuotaObservation {
	// Avoid decoding unrelated prompt/tool records.
	if !bytes.Contains(line, []byte(`"rate_limits"`)) {
		return nil
	}
	var e codexQuotaEvent
	if err := json.Unmarshal(line, &e); err != nil {
		// An identifiable malformed quota record must not uncover an older "ok".
		if bytes.Contains(line, []byte(`"codex"`)) && bytes.Contains(line, []byte(`"token_count"`)) {
			at := e.Timestamp
			if at.IsZero() || at.After(now.Add(time.Minute)) {
				at = now
			}
			return &CodexQuotaObservation{ObservedAt: at, State: "unknown", Reason: "malformed Codex provider observation"}
		}
		return nil
	}
	if e.Type != "event_msg" || e.Payload.Type != "token_count" {
		return nil
	}
	r := e.Payload.RateLimits
	if r == nil || r.LimitID != "codex" {
		return nil
	}
	if e.Timestamp.IsZero() || e.Timestamp.After(now.Add(time.Minute)) {
		return &CodexQuotaObservation{ObservedAt: now, State: "unknown", Reason: "invalid Codex observation timestamp"}
	}
	o := &CodexQuotaObservation{ObservedAt: e.Timestamp, State: "unknown", Reason: "no valid long-window Codex provider observation"}
	for _, w := range []*codexRateWindow{r.Primary, r.Secondary} {
		if w == nil {
			continue
		}
		if w.Used == nil || *w.Used < 0 || *w.Used > 100 || w.Minutes <= 0 || w.Minutes > 31*24*60 || w.Reset <= 0 {
			o.Windows = nil
			return o
		}
		reset := time.Unix(w.Reset, 0).UTC()
		duration := time.Duration(w.Minutes) * time.Minute
		if !reset.After(e.Timestamp) || reset.Sub(e.Timestamp) > duration+time.Minute {
			o.Windows = nil
			return o
		}
		o.Windows = append(o.Windows, CodexQuotaWindow{UsedPercent: *w.Used, WindowMinutes: w.Minutes, ResetsAt: reset})
	}
	return o
}

func (o *CodexQuotaObservation) evaluate(now time.Time) {
	if now.Sub(o.ObservedAt) > 15*time.Minute {
		o.State = "stale"
		o.Reason = "Codex provider observation is older than 15 minutes; new Codex routing blocked"
		return
	}
	hasLong := false
	over := false
	for i := range o.Windows {
		w := &o.Windows[i]
		if !w.ResetsAt.After(now) {
			o.State = "expired"
			o.Reason = "Codex provider window expired; refresh required before routing"
			return
		}
		w.AllowancePercent = 100
		if w.WindowMinutes > 24*60 {
			hasLong = true
			start := w.ResetsAt.Add(-time.Duration(w.WindowMinutes) * time.Minute)
			w.AllowancePercent = 100 * DailyRationFraction * now.Sub(start).Hours() / 24
			if w.AllowancePercent < 100*DailyRationFraction {
				w.AllowancePercent = 100 * DailyRationFraction
			}
			if w.AllowancePercent > 100 {
				w.AllowancePercent = 100
			}
		}
		if w.UsedPercent >= 100 || w.UsedPercent > w.AllowancePercent {
			over = true
		}
	}
	if !hasLong {
		o.State = "unknown"
		o.Reason = "Codex long-window usage missing; new Codex routing blocked"
		return
	}
	o.State = "ok"
	o.Reason = "provider-reported Codex account usage is within ration"
	if over {
		o.State = "over"
		o.Reason = "provider-reported Codex account usage exceeds ration or exhausts a window"
	}
}

// ObserveCodexQuota reads a bounded tail of recent local sessions. It makes no
// provider request and does not add account percentages to the token ledger.
func ObserveCodexQuota(codexHome string, now time.Time) CodexQuotaObservation {
	unknown := CodexQuotaObservation{State: "unknown", Reason: "no recent Codex provider observation; new Codex routing blocked"}
	type candidate struct {
		path     string
		modified time.Time
	}
	var files []candidate
	for day := 0; day < 8; day++ {
		dir := filepath.Join(codexHome, "sessions", now.UTC().AddDate(0, 0, -day).Format("2006/01/02"))
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			unknown.Reason = fmt.Sprintf("cannot read Codex session directory: %v", err)
			return unknown
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() || !bytes.HasPrefix([]byte(entry.Name()), []byte("rollout-")) || filepath.Ext(entry.Name()) != ".jsonl" {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				unknown.Reason = "cannot stat Codex session record"
				return unknown
			}
			if now.Sub(info.ModTime()) <= 24*time.Hour {
				files = append(files, candidate{filepath.Join(dir, entry.Name()), info.ModTime()})
			}
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].modified.After(files[j].modified) })
	var omittedNewest time.Time
	if len(files) > 128 {
		omittedNewest = files[128].modified
		files = files[:128]
	}
	var latest *CodexQuotaObservation
	for _, file := range files {
		data, err := codexQuotaTail(file.path)
		if err != nil {
			unknown.Reason = fmt.Sprintf("cannot read Codex session tail: %v", err)
			return unknown
		}
		for _, line := range bytes.Split(data, []byte{'\n'}) {
			o := parseCodexQuota(line, now)
			if o != nil && (latest == nil || o.ObservedAt.After(latest.ObservedAt)) {
				o.Source = filepath.Base(file.path)
				latest = o
			}
		}
	}
	if latest == nil {
		return unknown
	}
	if omittedNewest.After(latest.ObservedAt) {
		unknown.Reason = "Codex scan bound excludes files newer than the observation; new routing blocked"
		return unknown
	}
	latest.evaluate(now)
	return *latest
}

func codexQuotaTail(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	const limit int64 = 2 << 20
	offset := info.Size() - limit
	if offset < 0 {
		offset = 0
	}
	if _, err = f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return nil, err
	}
	if offset > 0 {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		} else {
			return nil, nil
		}
	}
	// Ignore an in-flight partial last record.
	if i := bytes.LastIndexByte(data, '\n'); i >= 0 {
		data = data[:i]
	} else {
		data = nil
	}
	return data, nil
}
