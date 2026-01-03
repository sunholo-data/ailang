package messaging

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestCheckGHInstalled(t *testing.T) {
	tests := []struct {
		name        string
		mockOutput  []byte
		mockErr     error
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "gh installed",
			mockOutput:  []byte("gh version 2.40.1 (2024-01-15)\nhttps://github.com/cli/cli/releases/tag/v2.40.1\n"),
			mockErr:     nil,
			wantVersion: "2.40.1",
			wantErr:     false,
		},
		{
			name:        "gh not installed",
			mockOutput:  nil,
			mockErr:     errors.New("executable file not found"),
			wantVersion: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitHubClient{
				execCommand: func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) > 0 && args[0] == "--version" {
						return tt.mockOutput, tt.mockErr
					}
					return nil, errors.New("unexpected command")
				},
			}

			version, err := client.CheckGHInstalled()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckGHInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if version != tt.wantVersion {
				t.Errorf("CheckGHInstalled() version = %v, want %v", version, tt.wantVersion)
			}
			if tt.wantErr && err != nil {
				// Verify error message includes install instructions
				if !strings.Contains(err.Error(), "brew install gh") {
					t.Errorf("error should include install instructions, got: %v", err)
				}
			}
		})
	}
}

func TestCheckGHAuth(t *testing.T) {
	tests := []struct {
		name       string
		mockOutput []byte
		mockErr    error
		wantUser   string
		wantErr    bool
	}{
		{
			name:       "authenticated with account pattern",
			mockOutput: []byte("github.com\n  Logged in to github.com account MarkEdmondson1234 (keyring)\n  Git operations protocol: https\n"),
			mockErr:    nil,
			wantUser:   "MarkEdmondson1234",
			wantErr:    false,
		},
		{
			name:       "authenticated with as pattern",
			mockOutput: []byte("github.com\n  Logged in to github.com as testuser\n"),
			mockErr:    nil,
			wantUser:   "testuser",
			wantErr:    false,
		},
		{
			name:       "not authenticated",
			mockOutput: []byte("You are not logged in to any GitHub hosts. Run gh auth login to authenticate.\n"),
			mockErr:    errors.New("exit status 1"),
			wantUser:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitHubClient{
				execCommand: func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
						return tt.mockOutput, tt.mockErr
					}
					return nil, errors.New("unexpected command")
				},
			}

			user, err := client.CheckGHAuth()
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckGHAuth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if user != tt.wantUser {
				t.Errorf("CheckGHAuth() user = %v, want %v", user, tt.wantUser)
			}
		})
	}
}

func TestValidateUser(t *testing.T) {
	tests := []struct {
		name        string
		config      *GitHubConfig
		authOutput  []byte
		authErr     error
		wantErr     bool
		errContains string
	}{
		{
			name:       "user matches",
			config:     &GitHubConfig{ExpectedUser: "MarkEdmondson1234"},
			authOutput: []byte("github.com\n  Logged in to github.com account MarkEdmondson1234 (keyring)\n"),
			authErr:    nil,
			wantErr:    false,
		},
		{
			name:        "user mismatch",
			config:      &GitHubConfig{ExpectedUser: "MarkEdmondson1234"},
			authOutput:  []byte("github.com\n  Logged in to github.com account rw-markedmondson (keyring)\n"),
			authErr:     nil,
			wantErr:     true,
			errContains: "account mismatch",
		},
		{
			name:        "no expected_user configured",
			config:      &GitHubConfig{},
			authOutput:  []byte("github.com\n  Logged in to github.com account someuser (keyring)\n"),
			authErr:     nil,
			wantErr:     true,
			errContains: "expected_user not configured",
		},
		{
			name:        "nil config",
			config:      nil,
			wantErr:     true,
			errContains: "config not loaded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitHubClient{
				config: tt.config,
				execCommand: func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) >= 2 && args[0] == "auth" && args[1] == "status" {
						return tt.authOutput, tt.authErr
					}
					return nil, errors.New("unexpected command")
				},
			}

			err := client.ValidateUser()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}

func TestCreateIssue(t *testing.T) {
	tests := []struct {
		name          string
		input         CreateIssueInput
		config        *GitHubConfig
		createOutput  string
		createErr     error
		wantIssueNum  int
		wantErr       bool
		wantTitleArgs string // substring that should be in title arg
		wantLabels    []string
	}{
		{
			name: "create issue with all options",
			input: CreateIssueInput{
				Title:     "Test bug",
				Body:      "This is a test",
				FromAgent: "ailang-core",
				Category:  "bug",
				Repo:      "sunholo-data/ailang",
			},
			config:        &GitHubConfig{ExpectedUser: "testuser"},
			createOutput:  "https://github.com/sunholo-data/ailang/issues/42\n",
			createErr:     nil,
			wantIssueNum:  42,
			wantErr:       false,
			wantTitleArgs: "[ailang-core] Test bug",
			wantLabels:    []string{"from:ailang-core", "bug"},
		},
		{
			name: "use default repo from config",
			input: CreateIssueInput{
				Title:     "Feature request",
				Body:      "Add this feature",
				FromAgent: "stapledon",
				Category:  "feature",
			},
			config: &GitHubConfig{
				ExpectedUser: "testuser",
				DefaultRepo:  "sunholo-data/ailang",
				CreateLabels: []string{"ailang-message"},
			},
			createOutput:  "https://github.com/sunholo-data/ailang/issues/123\n",
			createErr:     nil,
			wantIssueNum:  123,
			wantErr:       false,
			wantTitleArgs: "[stapledon] Feature request",
			wantLabels:    []string{"from:stapledon", "feature", "ailang-message"},
		},
		{
			name: "no repo specified",
			input: CreateIssueInput{
				Title:     "Test",
				Body:      "Test",
				FromAgent: "test",
			},
			config:       &GitHubConfig{ExpectedUser: "testuser"},
			createOutput: "",
			createErr:    nil,
			wantIssueNum: 0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedArgs []string
			client := &GitHubClient{
				config: tt.config,
				execCommand: func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) > 0 {
						switch args[0] {
						case "--version":
							return []byte("gh version 2.40.1\n"), nil
						case "auth":
							return []byte("Logged in to github.com account testuser\n"), nil
						case "issue":
							if args[1] == "create" {
								capturedArgs = args
								return []byte(tt.createOutput), tt.createErr
							}
						}
					}
					return nil, errors.New("unexpected command: " + name + " " + strings.Join(args, " "))
				},
			}

			issueNum, err := client.CreateIssue(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if issueNum != tt.wantIssueNum {
				t.Errorf("CreateIssue() issueNum = %v, want %v", issueNum, tt.wantIssueNum)
			}

			// Verify command arguments if we expected success
			if !tt.wantErr && len(capturedArgs) > 0 {
				argsStr := strings.Join(capturedArgs, " ")
				if !strings.Contains(argsStr, tt.wantTitleArgs) {
					t.Errorf("expected title %q in args, got: %v", tt.wantTitleArgs, argsStr)
				}
				for _, label := range tt.wantLabels {
					if !strings.Contains(argsStr, label) {
						t.Errorf("expected label %q in args, got: %v", label, argsStr)
					}
				}
			}
		})
	}
}

func TestListIssuesByLabel(t *testing.T) {
	mockIssues := []ghIssueResponse{
		{
			Number: 1,
			Title:  "[ailang-core] Bug report",
			Body:   "This is a bug",
			State:  "open",
			Labels: []struct {
				Name string `json:"name"`
			}{{Name: "bug"}, {Name: "from:ailang-core"}},
			CreatedAt: "2025-01-01T00:00:00Z",
			Author: struct {
				Login string `json:"login"`
			}{Login: "testuser"},
			URL: "https://github.com/test/repo/issues/1",
		},
		{
			Number: 2,
			Title:  "[stapledon] Feature request",
			Body:   "Add this feature",
			State:  "open",
			Labels: []struct {
				Name string `json:"name"`
			}{{Name: "feature"}, {Name: "from:stapledon"}},
			CreatedAt: "2025-01-02T00:00:00Z",
			Author: struct {
				Login string `json:"login"`
			}{Login: "testuser"},
			URL: "https://github.com/test/repo/issues/2",
		},
	}

	mockJSON, _ := json.Marshal(mockIssues)

	client := &GitHubClient{
		config: &GitHubConfig{
			ExpectedUser: "testuser",
			DefaultRepo:  "test/repo",
			WatchLabels:  []string{"ailang-message"},
		},
		execCommand: func(name string, args ...string) ([]byte, error) {
			if name == "gh" && len(args) > 0 {
				switch args[0] {
				case "--version":
					return []byte("gh version 2.40.1\n"), nil
				case "auth":
					return []byte("Logged in to github.com account testuser\n"), nil
				case "issue":
					if args[1] == "list" {
						return mockJSON, nil
					}
				}
			}
			return nil, errors.New("unexpected command")
		},
	}

	issues, err := client.ListIssuesByLabel("", nil)
	if err != nil {
		t.Fatalf("ListIssuesByLabel() error = %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}

	if issues[0].Number != 1 {
		t.Errorf("expected issue number 1, got %d", issues[0].Number)
	}

	if issues[0].Author != "testuser" {
		t.Errorf("expected author testuser, got %s", issues[0].Author)
	}

	if len(issues[0].Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(issues[0].Labels))
	}
}

func TestGetIssue(t *testing.T) {
	mockIssue := ghIssueResponse{
		Number: 42,
		Title:  "[ailang-core] Test issue",
		Body:   "Issue body",
		State:  "open",
		Labels: []struct {
			Name string `json:"name"`
		}{{Name: "bug"}},
		CreatedAt: "2025-01-01T00:00:00Z",
		Author: struct {
			Login string `json:"login"`
		}{Login: "testuser"},
		URL: "https://github.com/test/repo/issues/42",
	}

	mockJSON, _ := json.Marshal(mockIssue)

	client := &GitHubClient{
		config: &GitHubConfig{
			ExpectedUser: "testuser",
			DefaultRepo:  "test/repo",
		},
		execCommand: func(name string, args ...string) ([]byte, error) {
			if name == "gh" && len(args) > 0 {
				switch args[0] {
				case "--version":
					return []byte("gh version 2.40.1\n"), nil
				case "auth":
					return []byte("Logged in to github.com account testuser\n"), nil
				case "issue":
					if args[1] == "view" {
						return mockJSON, nil
					}
				}
			}
			return nil, errors.New("unexpected command")
		},
	}

	issue, err := client.GetIssue("", 42)
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}

	if issue.Number != 42 {
		t.Errorf("expected issue number 42, got %d", issue.Number)
	}

	if issue.Title != "[ailang-core] Test issue" {
		t.Errorf("expected title '[ailang-core] Test issue', got %s", issue.Title)
	}
}

func TestParseIssueNumberFromURL(t *testing.T) {
	tests := []struct {
		url     string
		want    int
		wantErr bool
	}{
		{"https://github.com/owner/repo/issues/123", 123, false},
		{"https://github.com/sunholo-data/ailang/issues/42", 42, false},
		{"https://github.com/owner/repo/issues/1", 1, false},
		{"https://github.com/owner/repo/pull/123", 0, true},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := parseIssueNumberFromURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseIssueNumberFromURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseIssueNumberFromURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestPreFlightChecks(t *testing.T) {
	tests := []struct {
		name        string
		config      *GitHubConfig
		ghInstalled bool
		ghAuth      bool
		userMatch   bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "all checks pass",
			config:      &GitHubConfig{ExpectedUser: "testuser"},
			ghInstalled: true,
			ghAuth:      true,
			userMatch:   true,
			wantErr:     false,
		},
		{
			name:        "gh not installed",
			config:      &GitHubConfig{ExpectedUser: "testuser"},
			ghInstalled: false,
			ghAuth:      true,
			userMatch:   true,
			wantErr:     true,
			errContains: "gh CLI not installed",
		},
		{
			name:        "not authenticated",
			config:      &GitHubConfig{ExpectedUser: "testuser"},
			ghInstalled: true,
			ghAuth:      false,
			userMatch:   false,
			wantErr:     true,
			errContains: "not authenticated",
		},
		{
			name:        "user mismatch",
			config:      &GitHubConfig{ExpectedUser: "testuser"},
			ghInstalled: true,
			ghAuth:      true,
			userMatch:   false,
			wantErr:     true,
			errContains: "account mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitHubClient{
				config: tt.config,
				execCommand: func(name string, args ...string) ([]byte, error) {
					if name == "gh" && len(args) > 0 {
						switch args[0] {
						case "--version":
							if tt.ghInstalled {
								return []byte("gh version 2.40.1\n"), nil
							}
							return nil, errors.New("command not found")
						case "auth":
							if !tt.ghAuth {
								return []byte("not logged in"), errors.New("exit status 1")
							}
							if tt.userMatch {
								return []byte("Logged in to github.com account testuser\n"), nil
							}
							return []byte("Logged in to github.com account wronguser\n"), nil
						}
					}
					return nil, errors.New("unexpected command")
				},
			}

			err := client.PreFlightChecks()
			if (err != nil) != tt.wantErr {
				t.Errorf("PreFlightChecks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}
