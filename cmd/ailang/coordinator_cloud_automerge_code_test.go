package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"testing"
)

type autoMergeTransport func(*http.Request) (*http.Response, error)

func (f autoMergeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func autoMergeAPI(t *testing.T, f func(*http.Request) (int, string)) {
	t.Helper()
	old := http.DefaultTransport
	http.DefaultTransport = autoMergeTransport(func(r *http.Request) (*http.Response, error) {
		status, body := f(r)
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = old })
}
func TestAutoMergeProductionScope(t *testing.T) {
	for _, tt := range []struct {
		name, file string
		patterns   []string
		mode       AutoMergeMode
		want       bool
	}{
		{"docs", "docs/note.md", []string{"docs/**"}, MergeModeDocs, true},
		{"docs code refused", "site/poster.html", []string{"**/*"}, MergeModeDocs, false},
		{"code html", "site/poster.html", []string{"site/**"}, MergeModeCode, true},
		{"code space filename", "site/summer poster.jpg", []string{"site/**"}, MergeModeCode, true},
		{"code image", "site/poster.jpg", []string{"site/**"}, MergeModeCode, true},
		{"out of scope", "private/x.md", []string{"site/**"}, MergeModeCode, false},
		{"empty declaration", "site/x.md", nil, MergeModeCode, false},
		{"empty diff", "", []string{"**/*"}, MergeModeCode, false},
		{"invalid mode", "site/x.md", []string{"**/*"}, AutoMergeMode(99), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := gitRepo(t)
			for _, args := range [][]string{{"update-ref", "refs/remotes/origin/main", "HEAD"}} {
				if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
					t.Fatalf("%s: %v", out, err)
				}
			}
			if tt.file != "" {
				write(t, dir, tt.file, "content")
				for _, args := range [][]string{{"add", "-A"}, {"commit", "-qm", "change"}} {
					if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
						t.Fatalf("%s: %v", out, err)
					}
				}
			}
			got, reason, err := branchIsAutoMergeable(context.Background(), dir, "main", tt.patterns, tt.mode)
			if err != nil || got != tt.want {
				t.Fatalf("got %v %s %v", got, reason, err)
			}
		})
	}
	if ok, _, err := branchIsAutoMergeable(context.Background(), t.TempDir(), "main", []string{"**/*"}, MergeModeCode); ok || err == nil {
		t.Fatal("git error accepted")
	}
}
func TestRequiredChecksPresentOnBase(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		status     int
		checks     []string
		wantErr    bool
		missing    int
	}{
		{"present", `{"total_count":2,"check_runs":[{"name":"site, build"},{"name":"image"}]}`, 200, []string{"site, build"}, false, 0},
		{"missing", `{"total_count":0,"check_runs":[]}`, 200, []string{"site"}, false, 1},
		{"empty config", `{}`, 200, nil, true, 0},
		{"malformed", `{}`, 200, []string{"site"}, true, 0},
		{"bad json", `bad`, 200, []string{"site"}, true, 0},
		{"api failure", `{"message":"no"}`, 403, []string{"site"}, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			autoMergeAPI(t, func(r *http.Request) (int, string) {
				if strings.Contains(r.URL.Path, "/check-runs") {
					if !strings.Contains(r.URL.Path, "/base-sha/") {
						t.Fatal("unresolved base", r.URL)
					}
					return tt.status, tt.body
				}
				return 200, `{"sha":"base-sha"}`
			})
			missing, err := requiredChecksPresentOnBase(context.Background(), "token", "owner", "repo", "main", tt.checks)
			if (err != nil) != tt.wantErr || len(missing) != tt.missing {
				t.Fatalf("missing %v err %v", missing, err)
			}
		})
	}
}
func TestRequiredCheckPagination(t *testing.T) {
	pages := 0
	autoMergeAPI(t, func(r *http.Request) (int, string) {
		if !strings.Contains(r.URL.Path, "check-runs") {
			return 200, `{"sha":"sha"}`
		}
		pages++
		runs := []map[string]string{}
		count := 100
		name := "first"
		if pages == 2 {
			count = 1
			name = "last"
		}
		for i := 0; i < count; i++ {
			runs = append(runs, map[string]string{"name": name})
		}
		b, _ := json.Marshal(map[string]interface{}{"total_count": 101, "check_runs": runs})
		return 200, string(b)
	})
	missing, err := requiredChecksPresentOnBase(context.Background(), "t", "o", "r", "main", []string{"last"})
	if err != nil || len(missing) != 0 || pages != 2 {
		t.Fatalf("pages=%d missing=%v err=%v", pages, missing, err)
	}
}

func TestApproverPreflight(t *testing.T) {
	for _, tt := range []struct {
		name, actual, expected, author string
		status                         int
		wantErr                        bool
	}{
		{"valid case insensitive", "Reviewer", "reviewer", "author", 200, false},
		{"wrong identity", "other", "reviewer", "author", 200, true},
		{"author collision", "Reviewer", "reviewer", "REVIEWER", 200, true},
		{"empty login", "", "reviewer", "author", 200, true},
		{"empty author", "reviewer", "reviewer", "", 200, true},
		{"api failure", "reviewer", "reviewer", "author", 401, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			autoMergeAPI(t, func(r *http.Request) (int, string) {
				if r.URL.Path == "/user" {
					return tt.status, `{"login":"` + tt.actual + `"}`
				}
				return 200, `{"user":{"login":"` + tt.author + `"},"head":{"sha":"head-sha"}}`
			})
			_, err := preflightCodeApprover(context.Background(), "approver-token", "fleet-token", "o", "r", 1, tt.expected)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v", err)
			}
			if err != nil && strings.Contains(err.Error(), "approver-token") {
				t.Fatal("token exposed")
			}
		})
	}
}

func TestAutoMergeCodeSequenceAndRollback(t *testing.T) {
	for _, tt := range []struct {
		name                                     string
		reviewStatus, disableStatus, labelStatus int
		wrongIdentity                            bool
		wantErr                                  bool
	}{
		{"success", 200, 200, 200, false, false},
		{"review failed", 403, 200, 200, false, true},
		{"cleanup failed", 403, 500, 200, false, true},
		{"label failed body persists", 200, 200, 403, false, false},
		{"wrong identity before enable", 200, 200, 200, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := []string{}
			prBody := "Original request\nRefs #1599"
			enabled := false
			approved := false
			disabled := false
			autoMergeAPI(t, func(r *http.Request) (int, string) {
				var data []byte
				if r.Body != nil {
					data, _ = io.ReadAll(r.Body)
				}
				path := r.URL.Path
				calls = append(calls, r.Method+" "+path)
				switch {
				case path == "/graphql":
					if strings.Contains(string(data), "enablePullRequestAutoMerge") {
						enabled = true
						return 200, `{"data":{"enablePullRequestAutoMerge":{"clientMutationId":null}}}`
					}
					if strings.Contains(string(data), "disablePullRequestAutoMerge") {
						disabled = true
						return tt.disableStatus, `{"data":{"disablePullRequestAutoMerge":{"clientMutationId":null}}}`
					}
					return 200, `{"data":{"repository":{"pullRequest":{"id":"node"}}}}`
				case path == "/user":
					login := "reviewer"
					if tt.wrongIdentity {
						login = "wrong"
					}
					return 200, `{"login":"` + login + `"}`
				case strings.HasSuffix(path, "/commits/main"):
					return 200, `{"sha":"base"}`
				case strings.HasSuffix(path, "/check-runs"):
					return 200, `{"total_count":1,"check_runs":[{"name":"site"}]}`
				case strings.HasSuffix(path, "/reviews"):
					if !enabled {
						t.Fatal("approval before enable")
					}
					if r.Header.Get("Authorization") != "Bearer review-token" {
						t.Fatal("wrong approval credential")
					}
					if !strings.Contains(string(data), `"commit_id":"head-sha"`) {
						t.Fatal("approval not pinned to final push", string(data))
					}
					approved = tt.reviewStatus == 200
					return tt.reviewStatus, `{"id":1,"state":"APPROVED"}`
				case strings.HasSuffix(path, "/labels"):
					return tt.labelStatus, `[]`
				case r.Method == "PATCH":
					var v struct {
						Body string `json:"body"`
					}
					if err := json.Unmarshal(data, &v); err != nil {
						t.Fatal(err)
					}
					prBody = v.Body
					return 200, `{}`
				default:
					b, _ := json.Marshal(map[string]interface{}{"body": prBody, "user": map[string]string{"login": "author"}, "head": map[string]string{"sha": "head-sha"}})
					return 200, string(b)
				}
			})
			s := codeMergeSettings{true, []string{"site"}, []string{"site/**"}, "secret-name", "reviewer", "head-sha"}
			err := executeCodeAutoMerge(context.Background(), "fleet-token", "o", "r", 1, "main", s, func(context.Context, string) (string, error) { return "review-token", nil })
			if (err != nil) != tt.wantErr {
				t.Fatalf("err=%v calls=%v", err, calls)
			}
			if tt.wrongIdentity {
				if enabled {
					t.Fatal("enabled before identity validation")
				}
				return
			}
			if !strings.Contains(prBody, "Original request") || !strings.Contains(prBody, "Refs #1599") || !strings.Contains(prBody, "required checks: site") {
				t.Fatal("lost durable audit", prBody)
			}
			if approved && !strings.Contains(prBody, "enabled and approved") {
				t.Fatal("success audit missing", prBody)
			}
			if !approved && (!disabled || strings.Contains(prBody, "enabled and approved")) {
				t.Fatal("partial failure reported success", prBody)
			}
			if tt.disableStatus == 500 && (err == nil || !strings.Contains(err.Error(), "cleanup failed")) {
				t.Fatal("cleanup failure hidden", err)
			}
			// Re-running on an existing PR preserves its original body and one audit section.
			if !tt.wantErr {
				if err := executeCodeAutoMerge(context.Background(), "fleet-token", "o", "r", 1, "main", s, func(context.Context, string) (string, error) { return "review-token", nil }); err != nil {
					t.Fatal(err)
				}
				if strings.Count(prBody, codeMergeAuditStart) != 1 {
					t.Fatal("duplicate audit", prBody)
				}
			}
		})
	}
}

func TestAutoMergeCodeRefusesBeforeEnable(t *testing.T) {
	for _, name := range []string{"invalid config", "missing check", "base API failure", "empty base SHA", "secret failure", "empty secret", "intent audit failure", "malformed checks", "incomplete checks", "local head mismatch"} {
		t.Run(name, func(t *testing.T) {
			enabled := false
			autoMergeAPI(t, func(r *http.Request) (int, string) {
				switch {
				case r.URL.Path == "/graphql":
					enabled = true
					return 200, `{}`
				case strings.HasSuffix(r.URL.Path, "/commits/main"):
					if name == "base API failure" {
						return 403, `{}`
					}
					if name == "empty base SHA" {
						return 200, `{}`
					}
					return 200, `{"sha":"base"}`
				case strings.HasSuffix(r.URL.Path, "/check-runs"):
					if name == "missing check" {
						return 200, `{"total_count":0,"check_runs":[]}`
					}
					if name == "malformed checks" {
						return 200, `{"total_count":0}`
					}
					if name == "incomplete checks" {
						return 200, `{"total_count":2,"check_runs":[{"name":"site"}]}`
					}
					return 200, `{"total_count":1,"check_runs":[{"name":"site"}]}`
				case r.URL.Path == "/user":
					return 200, `{"login":"reviewer"}`
				case r.Method == "PATCH":
					return 500, `{}`
				default:
					return 200, `{"user":{"login":"author"},"head":{"sha":"head"},"body":"Original"}`
				}
			})
			s := codeMergeSettings{true, []string{"site"}, []string{"site/**"}, "secret", "reviewer", "head"}
			if name == "local head mismatch" {
				s.localHead = "other"
			}
			if name == "invalid config" {
				s.autoMerge = false
			}
			err := executeCodeAutoMerge(context.Background(), "fleet", "o", "r", 1, "main", s, func(context.Context, string) (string, error) {
				if name == "secret failure" {
					return "", fmt.Errorf("sensitive-token")
				}
				if name == "empty secret" {
					return " ", nil
				}
				return "review-token", nil
			})
			if err == nil || enabled {
				t.Fatalf("refusal missing: enabled=%v err=%v", enabled, err)
			}
			if name == "local head mismatch" && !strings.Contains(err.Error(), "does not match the local HEAD") {
				t.Fatalf("head mismatch not the refusal reason: %v", err)
			}
			if strings.Contains(err.Error(), "sensitive-token") {
				t.Fatal("secret error exposed")
			}
		})
	}
}
func TestApprovingReviewRequiresConfirmation(t *testing.T) {
	for _, body := range []string{`{}`, `{"id":1,"state":"COMMENTED"}`, `broken`} {
		t.Run(body, func(t *testing.T) {
			autoMergeAPI(t, func(*http.Request) (int, string) { return 200, body })
			if err := approvePRAsNonAuthor(context.Background(), "token", "o", "r", 1, "head", "audit"); err == nil {
				t.Fatal("unconfirmed review accepted")
			}
		})
	}
}

func TestAutoMergeEnableRequiresConfirmation(t *testing.T) {
	autoMergeAPI(t, func(r *http.Request) (int, string) {
		data, _ := io.ReadAll(r.Body)
		if strings.Contains(string(data), "enablePullRequestAutoMerge") {
			return 200, `{}`
		}
		return 200, `{"data":{"repository":{"pullRequest":{"id":"node"}}}}`
	})
	if err := enableGitHubAutoMerge(context.Background(), "token", "o", "r", 1); err == nil {
		t.Fatal("unconfirmed native enable accepted")
	}
}

func TestAutoMergeWrapperOptInAndRefusals(t *testing.T) {
	for _, tt := range []struct {
		name, file, checks, patterns string
		code, auto                   bool
		wantAPI                      bool
	}{
		{"docs floor", "site/x.html", "site", "site/**", false, true, false},
		{"code opt in missing check", "site/x.html", "site", "site/**", true, true, true},
		{"code out of scope", "private/x.html", "site", "site/**", true, true, false},
		{"code empty scope", "site/x.html", "site", "", true, true, false},
		{"code incomplete checks", "site/x.html", "", "site/**", true, true, false},
		{"code requires auto merge", "site/x.html", "site", "site/**", true, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AILANG_AUTO_MERGE_CODE", "0")
			if tt.code {
				t.Setenv("AILANG_AUTO_MERGE_CODE", "1")
			}
			t.Setenv("AILANG_AUTO_MERGE", "0")
			if tt.auto {
				t.Setenv("AILANG_AUTO_MERGE", "1")
			}
			t.Setenv("AILANG_AUTO_MERGE_REQUIRED_CHECKS", tt.checks)
			t.Setenv("AILANG_ARTIFACT_PATTERNS", tt.patterns)
			t.Setenv("AILANG_APPROVER_SECRET", "secret")
			t.Setenv("AILANG_APPROVER_IDENTITY", "reviewer")
			dir := gitRepo(t)
			for _, args := range [][]string{{"update-ref", "refs/remotes/origin/main", "HEAD"}} {
				if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
					t.Fatalf("%s: %v", out, err)
				}
			}
			write(t, dir, tt.file, "content")
			for _, args := range [][]string{{"add", "-A"}, {"commit", "-qm", "change"}} {
				if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
					t.Fatalf("%s: %v", out, err)
				}
			}
			called := false
			autoMergeAPI(t, func(r *http.Request) (int, string) {
				called = true
				if r.URL.Path == "/graphql" || r.URL.Path == "/user" {
					t.Fatal("refused PR reached enable or secret identity API")
				}
				if strings.HasSuffix(r.URL.Path, "/check-runs") {
					return 200, `{"total_count":0,"check_runs":[]}`
				}
				return 200, `{"sha":"base"}`
			})
			maybeEnableAutoMerge(context.Background(), "fleet", "o", "r", 1, dir, "main")
			if called != tt.wantAPI {
				t.Fatalf("API called=%v want=%v", called, tt.wantAPI)
			}
		})
	}
}
