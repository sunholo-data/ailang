package main

// Splitting a live fault from old debt.
//
// `messages health` printed one number per bucket and no age, so fifteen
// undelivered handoffs — the newest 35 hours old, the oldest nine days — read
// exactly like fifteen that arrived this morning and went nowhere. Measured
// 2026-09-16: the banner said DEGRADED while the plane had, in fact, dispatched
// everything it received for 24 hours. A counter that cannot go to zero teaches
// the reader to stop looking at it, which is the failure this command exists to
// prevent.
//
// So the verdict is now about a WINDOW, and the backlog is reported separately
// and by age. Both numbers are real; only one of them is news.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// defaultHealthWindow is what "is the plane working?" usually means: today.
const defaultHealthWindow = 24 * time.Hour

// healthMsg is the only thing the summary needs from a message.
type healthMsg struct {
	Inbox  string
	Type   string
	Age    time.Duration
	Bucket inboxBucket
}

// healthRow is one inbox's contribution to a bucket.
type healthRow struct {
	Inbox   string        `json:"inbox"`
	Total   int           `json:"total"`
	New     int           `json:"new"`
	Backlog int           `json:"backlog"`
	Oldest  time.Duration `json:"-"`
	OldestH float64       `json:"oldest_hours"`
}

// bucketStat is one bucket, split by the window.
type bucketStat struct {
	Total   int         `json:"total"`
	New     int         `json:"new"`
	Backlog int         `json:"backlog"`
	Rows    []healthRow `json:"rows,omitempty"`
}

// healthSummary is the whole judgement, computed from data rather than printed
// as it goes — so the verdict and the rows cannot disagree, and so it can be
// rendered as text or JSON from one source.
type healthSummary struct {
	WindowHours float64                     `json:"window_hours"` // 0 = no window: judge everything
	Unread      int                         `json:"unread_total"`
	Buckets     map[inboxBucket]*bucketStat `json:"-"`
	Routable    *bucketStat                 `json:"routable"`
	Unroutable  *bucketStat                 `json:"unroutable"`
	Result      *bucketStat                 `json:"agent_output"`
	Triage      *bucketStat                 `json:"human_triage"`
}

// summarizeHealth buckets the unread messages and splits each bucket by age.
//
// window == 0 means "no window": everything counts as new, which is the
// all-time judgement `--since 0` asks for.
func summarizeHealth(msgs []healthMsg, window time.Duration) *healthSummary {
	s := &healthSummary{
		WindowHours: window.Hours(),
		Unread:      len(msgs),
		Buckets:     map[inboxBucket]*bucketStat{},
	}
	rows := map[inboxBucket]map[string]*healthRow{}
	for _, b := range []inboxBucket{bucketRoutable, bucketTriage, bucketUnroutable, bucketResult} {
		s.Buckets[b] = &bucketStat{}
		rows[b] = map[string]*healthRow{}
	}
	for _, m := range msgs {
		st := s.Buckets[m.Bucket]
		st.Total++
		isNew := window == 0 || m.Age < window
		if isNew {
			st.New++
		} else {
			st.Backlog++
		}
		r := rows[m.Bucket][m.Inbox]
		if r == nil {
			r = &healthRow{Inbox: m.Inbox}
			rows[m.Bucket][m.Inbox] = r
		}
		r.Total++
		if isNew {
			r.New++
		} else {
			r.Backlog++
		}
		if m.Age > r.Oldest {
			r.Oldest = m.Age
			r.OldestH = m.Age.Hours()
		}
	}
	for b, byInbox := range rows {
		out := make([]healthRow, 0, len(byInbox))
		for _, r := range byInbox {
			out = append(out, *r)
		}
		// New first, then volume, then name: the rows that need acting on today
		// must not be below nine days of backlog.
		sort.Slice(out, func(i, j int) bool {
			if out[i].New != out[j].New {
				return out[i].New > out[j].New
			}
			if out[i].Total != out[j].Total {
				return out[i].Total > out[j].Total
			}
			return out[i].Inbox < out[j].Inbox
		})
		s.Buckets[b].Rows = out
	}
	s.Routable, s.Unroutable = s.Buckets[bucketRoutable], s.Buckets[bucketUnroutable]
	s.Result, s.Triage = s.Buckets[bucketResult], s.Buckets[bucketTriage]
	return s
}

// LiveFaults counts what went wrong INSIDE the window: work that arrived and
// was never dispatched, plus sends that had nowhere to go.
func (s *healthSummary) LiveFaults() int { return s.Routable.New + s.Unroutable.New }

// Backlog counts the same two classes from before the window.
func (s *healthSummary) Backlog() int { return s.Routable.Backlog + s.Unroutable.Backlog }

// parseHealthWindow reads --since. "0" / "all" means judge everything.
//
// Accepts a `d` suffix because the honest answers here are "today" and "this
// week", and Go's duration parser has no day.
func parseHealthWindow(s string) (time.Duration, error) {
	t := strings.TrimSpace(strings.ToLower(s))
	switch t {
	case "", "24h":
		return defaultHealthWindow, nil
	case "0", "all", "everything":
		return 0, nil
	}
	if strings.HasSuffix(t, "d") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(t, "d"), 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("--since %q: expected a duration like 24h, 7d, 90m, or 0 for all time", s)
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	}
	d, err := time.ParseDuration(t)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("--since %q: expected a duration like 24h, 7d, 90m, or 0 for all time", s)
	}
	return d, nil
}

// humanAge renders an age the way a reader compares it to "today".
func humanAge(d time.Duration) string {
	switch {
	case d <= 0:
		return "-"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%.1fh", d.Hours())
	default:
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
}

// humanWindow names the window in the same units the flag accepts.
func humanWindow(d time.Duration) string {
	if d == 0 {
		return "all time"
	}
	if d >= 48*time.Hour {
		return fmt.Sprintf("last %.0fd", d.Hours()/24)
	}
	return fmt.Sprintf("last %.0fh", d.Hours())
}

// stripANSI removes colour codes so a JSON field carries text, not escapes.
//
// The send-path line is built for a terminal; a hook that greps it for "BROKEN"
// would otherwise have to know about escape sequences to find the word.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				i = j + 1
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// briefError keeps a multi-line backend error from taking over the report.
// firstLine already exists for the un-truncated case; this one also bounds the
// width, because a Firestore index error arrives as one 400-character line.
func briefError(s string) string {
	s = firstLine(s)
	if len(s) > 120 {
		s = s[:117] + "..."
	}
	return s
}
