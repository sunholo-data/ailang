package mission

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var opencodeQuotaTime = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

func writeOpencodeQuotaFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
}

// A separate synthetic window makes the zero-token capacity control independent
// of the two spend rows while retaining the same real provider bucket.
func opencodeQuotaCache() Ledger {
	l := Ledger{Updated: opencodeQuotaTime}
	for _, w := range []Window{WindowShort, WindowLong} {
		l.Usage = append(l.Usage,
			BucketUsage{Bucket: "anthropic", Window: w, Tokens: 11, Stages: 1,
				WindowStart: opencodeQuotaTime.Add(-time.Hour), Capacity: 100,
				ProviderReset: opencodeQuotaTime.Add(time.Hour)},
			BucketUsage{Bucket: "opencode", Window: w})
	}
	l.Usage = append(l.Usage, BucketUsage{Bucket: "anthropic", Window: "capacity-control",
		Capacity: 321, ProviderReset: opencodeQuotaTime.Add(2 * time.Hour)})
	return l
}

func seedOpencodeQuotaCache(t *testing.T, p Paths, l Ledger) {
	t.Helper()
	body, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	writeOpencodeQuotaFile(t, LedgerPath(p), body)
}

func TestQuotaOpencodeCache(t *testing.T) {
	t.Run("phantom-removal", func(t *testing.T) {
		p := Paths{Home: t.TempDir()}
		fixture := opencodeQuotaCache()
		// Qualified harness caches must be dropped, never migrated as spend.
		fixture.Usage = append(fixture.Usage, BucketUsage{Bucket: "OpenCode:openrouter/x", Window: WindowLong, Tokens: 99})
		seedOpencodeQuotaCache(t, p, fixture)
		l, err := LoadLedger(p, opencodeQuotaTime)
		if err != nil {
			t.Fatal(err)
		}
		for _, u := range l.Usage {
			if strings.HasPrefix(strings.ToLower(u.Bucket), "opencode") || u.Bucket == "openrouter" {
				t.Errorf("harness cache survived or migrated: %+v", u)
			}
		}
	})
	t.Run("capacity-survives", func(t *testing.T) {
		p := Paths{Home: t.TempDir()}
		fixture := opencodeQuotaCache()
		seedOpencodeQuotaCache(t, p, fixture)
		l, err := LoadLedger(p, opencodeQuotaTime)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range fixture.Usage {
			if want.Bucket != "anthropic" {
				continue
			}
			found := false
			for _, got := range l.Usage {
				if got.Bucket == want.Bucket && got.Window == want.Window {
					found = true
					if got.Tokens != want.Tokens || got.Stages != want.Stages || got.Capacity != want.Capacity || !got.ProviderReset.Equal(want.ProviderReset) {
						t.Errorf("real bucket fields changed: got %+v, want %+v", got, want)
					}
				}
			}
			if !found {
				t.Errorf("real capacity row missing: %s/%s", want.Bucket, want.Window)
			}
		}
	})
}

func TestQuotaOpencodeJournal(t *testing.T) {
	for _, tc := range []struct {
		name, bucket string
		skipped      int
	}{
		{"unmappable", "opencode", 1},
		{"mapped", "opencode:openrouter/x", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := Paths{Home: t.TempDir()}
			writeOpencodeQuotaFile(t, journalPath(p), []byte(fmt.Sprintf("%d\t%s\t7\t1\n", opencodeQuotaTime.Unix(), tc.bucket)))
			l, skipped, err := rebuild(p, opencodeQuotaTime)
			if err != nil {
				t.Fatal(err)
			}
			if skipped != tc.skipped {
				t.Errorf("skipped = %d, want %d", skipped, tc.skipped)
			}
			if tc.skipped == 1 {
				if len(l.Usage) != 0 {
					t.Errorf("unmappable journal created rows: %+v", l.Usage)
				}
				return
			}
			if len(l.Usage) != 2 {
				t.Fatalf("mapped journal rows = %d, want 2", len(l.Usage))
			}
			for i, w := range []Window{WindowShort, WindowLong} {
				u := l.Usage[i]
				if u.Bucket != "openrouter" || u.Window != w || u.Tokens != 7 || u.Stages != 1 {
					t.Errorf("mapped journal row = %+v, want openrouter/%s with 7 tokens", u, w)
				}
			}
		})
	}
}

func TestQuotaOpencodeAppendRejects(t *testing.T) {
	for _, exists := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing=%t", exists), func(t *testing.T) {
			p := Paths{Home: t.TempDir()}
			var before []byte
			if exists {
				before = []byte(fmt.Sprintf("%d\tanthropic\t9\t1\n", opencodeQuotaTime.Unix()))
				writeOpencodeQuotaFile(t, journalPath(p), before)
			}
			err := AppendSpend(p, "opencode", 5, 1, opencodeQuotaTime)
			if err == nil || !strings.Contains(err.Error(), "opencode is a harness, not a quota bucket; record the backend bucket instead") {
				t.Errorf("AppendSpend must reject harness: %v", err)
			}
			after, readErr := os.ReadFile(journalPath(p))
			if !exists {
				if !os.IsNotExist(readErr) {
					t.Errorf("rejected append created journal: %v", readErr)
				}
			} else if readErr != nil || sha256.Sum256(before) != sha256.Sum256(after) {
				t.Errorf("rejected append changed journal: %v", readErr)
			}
		})
	}
}

func TestQuotaOpencodeAppendMaps(t *testing.T) {
	p := Paths{Home: t.TempDir()}
	if err := AppendSpend(p, "opencode:ollama/m:cloud", 3, 1, opencodeQuotaTime); err != nil {
		t.Fatal(err)
	}
	entries, skipped, err := readJournalFiles([]string{journalPath(p)})
	if err != nil || skipped != 0 || len(entries) != 1 {
		t.Fatalf("journal parse: entries=%v skipped=%d err=%v", entries, skipped, err)
	}
	if entries[0].Bucket != "ollama" || entries[0].Tokens != 3 {
		t.Fatalf("journal backend = %+v, want ollama with 3 tokens", entries[0])
	}
}
