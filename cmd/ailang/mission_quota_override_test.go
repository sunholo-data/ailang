package main

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	storefs "github.com/sunholo-data/ailang/internal/storage/firestore"
)

type fakeOverrideStore struct {
	docs    map[string]storefs.QuotaOverride
	project string
}

func (f *fakeOverrideStore) Get(_ context.Context, b string) (*storefs.QuotaOverride, error) {
	if o, ok := f.docs[b]; ok {
		return &o, nil
	}
	return nil, nil
}
func (f *fakeOverrideStore) Put(_ context.Context, o storefs.QuotaOverride) error {
	f.docs[o.Bucket] = o
	return nil
}
func (f *fakeOverrideStore) Delete(_ context.Context, b string) error {
	delete(f.docs, b)
	return nil
}

func useFakeOverrideStore(t *testing.T) *fakeOverrideStore {
	t.Helper()
	f := &fakeOverrideStore{docs: map[string]storefs.QuotaOverride{}}
	old := openQuotaOverrideStore
	t.Cleanup(func() { openQuotaOverrideStore = old })
	openQuotaOverrideStore = func(_ context.Context, project string) (quotaOverrideStore, func(), error) {
		f.project = project
		return f, func() {}, nil
	}
	return f
}

func attended(t *testing.T) {
	t.Helper()
	t.Setenv("MISSION_CONTROL_ACTIVE", "")
	t.Setenv("MISSION_ROLE", "")
}

func TestMissionQuotaOverride_AllowAllThenRevoke(t *testing.T) {
	attended(t)
	f := useFakeOverrideStore(t)
	now := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	err := missionQuotaOverride(quotaOverrideRequest{allow: "all", project: "ailang-multivac-dev", reason: "smoke", duration: time.Hour, yes: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if f.project != "ailang-multivac-dev" {
		t.Fatalf("store opened in %q", f.project)
	}
	var got []string
	for b, o := range f.docs {
		got = append(got, b)
		if !o.Until.Equal(now.Add(time.Hour)) || o.Reason != "smoke" || o.GrantedBy == "" {
			t.Fatalf("%s: bad record %+v", b, o)
		}
	}
	sort.Strings(got)
	if strings.Join(got, ",") != "anthropic,codex,google" {
		t.Fatalf("buckets written: %v", got)
	}
	if err := missionQuotaOverride(quotaOverrideRequest{revoke: "codex", project: "ailang-multivac-dev"}, now); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.docs["codex"]; ok || len(f.docs) != 2 {
		t.Fatalf("revoke codex left %v", f.docs)
	}
}

func TestMissionQuotaOverride_Refusals(t *testing.T) {
	now := time.Now()
	ok := quotaOverrideRequest{allow: "codex", project: "p", reason: "r", duration: time.Hour, yes: true}
	cases := map[string]struct {
		mutate func(*quotaOverrideRequest)
		env    map[string]string
		want   string
	}{
		"inside a mission":    {env: map[string]string{"MISSION_CONTROL_ACTIVE": "1"}, want: "refused inside a mission iteration"},
		"pinned mission role": {env: map[string]string{"MISSION_ROLE": "executor"}, want: "refused inside a mission iteration"},
		"no project":          {mutate: func(r *quotaOverrideRequest) { r.project = "" }, want: "--project is required"},
		"no reason":           {mutate: func(r *quotaOverrideRequest) { r.reason = " " }, want: "--reason is required"},
		"no end":              {mutate: func(r *quotaOverrideRequest) { r.duration = 0 }, want: "--for must be"},
		"too long":            {mutate: func(r *quotaOverrideRequest) { r.duration = 25 * time.Hour }, want: "--for must be"},
		"unconfirmed":         {mutate: func(r *quotaOverrideRequest) { r.yes = false }, want: "--yes"},
		"unknown bucket":      {mutate: func(r *quotaOverrideRequest) { r.allow = "ollama" }, want: "unknown quota override bucket"},
		"allow and revoke":    {mutate: func(r *quotaOverrideRequest) { r.revoke = "codex" }, want: "not both"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			attended(t)
			for k, v := range c.env {
				t.Setenv(k, v)
			}
			f := useFakeOverrideStore(t)
			r := ok
			if c.mutate != nil {
				c.mutate(&r)
			}
			err := missionQuotaOverride(r, now)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
			if len(f.docs) != 0 {
				t.Fatalf("refused request wrote %v", f.docs)
			}
		})
	}
}

// The flags reach the override path through the real command.
func TestMissionQuota_AllowFlagsWired(t *testing.T) {
	attended(t)
	f := useFakeOverrideStore(t)
	err := missionQuota([]string{"--allow", "google", "--for", "30m", "--reason", "smoke", "--project", "ailang-multivac", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if o, ok := f.docs["google"]; !ok || o.Until.Sub(o.GrantedAt) != 30*time.Minute || f.project != "ailang-multivac" {
		t.Fatalf("override not written via flags: %+v project=%q", f.docs, f.project)
	}
}
