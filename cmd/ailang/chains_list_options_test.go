package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/observatory"
)

func TestChainsListFlagsDefaults(t *testing.T) {
	got, err := parseChainsListFlags(nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	want := chainsListFlags{Query: observatory.ChainListOptions{Limit: 20}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v; want %+v", got, want)
	}
}

func TestChainsListFlagsQueryAndPresentation(t *testing.T) {
	got, err := parseChainsListFlags([]string{
		"--status", "completed", "--source", "message", "--agent", "builder",
		"--workspace", "workspace-a", "--repo", "owner/repo", "--since", "2026-09-01",
		"--limit", "7", "--offset", "14", "--remote", "gcp", "--json", "--full",
	}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	want := chainsListFlags{Query: observatory.ChainListOptions{
		Status: "completed", SourceType: "message", AgentID: "builder",
		WorkspaceID: "workspace-a", GitHubRepo: "owner/repo", CreatedAfter: &after,
		Limit: 7, Offset: 14,
	}, Remote: "gcp", JSON: true, FullIDs: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v; want %+v", got, want)
	}
}

func TestChainsListFlagsInvalidRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		message string
	}{
		{"zero limit", []string{"--limit", "0"}, "--limit must be positive"},
		{"negative limit", []string{"--limit=-1"}, "--limit must be positive"},
		{"negative offset", []string{"--offset=-1"}, "--offset must be non-negative"},
		{"noninteger offset", []string{"--offset", "one"}, "invalid value"},
		{"overflow limit", []string{"--limit", "9999999999999999999999999"}, "invalid value"},
		{"invalid since", []string{"--since", "yesterday"}, "invalid --since"},
		{"unknown flag", []string{"--offsets", "2"}, "flag provided but not defined"},
		{"positional argument", []string{"unexpected", "--offset", "1"}, "unexpected argument"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseChainsListFlags(tc.args, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("got %v; want %q", err, tc.message)
			}
		})
	}
}

func TestChainsListFlagsRelativeSince(t *testing.T) {
	for _, tc := range []struct {
		value string
		age   time.Duration
	}{{"24h", 24 * time.Hour}, {"7d", 7 * 24 * time.Hour}, {"90m", 90 * time.Minute}} {
		t.Run(tc.value, func(t *testing.T) {
			before := time.Now().Add(-tc.age)
			got, err := parseChainsListFlags([]string{"--since", tc.value}, io.Discard)
			after := time.Now().Add(-tc.age)
			if err != nil {
				t.Fatal(err)
			}
			if got.Query.CreatedAfter == nil || got.Query.CreatedAfter.Before(before) || got.Query.CreatedAfter.After(after) {
				t.Fatalf("created_after %v outside [%v, %v]", got.Query.CreatedAfter, before, after)
			}
		})
	}
}

func TestChainsListFlagsHelp(t *testing.T) {
	var output bytes.Buffer
	_, err := parseChainsListFlags([]string{"--help"}, &output)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("got %v; want help", err)
	}
	for _, name := range []string{"-offset", "-limit", "-workspace", "-repo", "-remote", "-since", "-json", "-full"} {
		if !strings.Contains(output.String(), name) {
			t.Errorf("help missing %s", name)
		}
	}
}
