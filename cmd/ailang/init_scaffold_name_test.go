package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `ailang init web-app --help` created a directory called "--help" instead of
// printing help (one landed in this repo on 2026-09-18 01:55). flag.Parse stops
// at the first non-flag argument, so everything after the type is positional
// and "--help" was read as the project NAME.
func TestCheckScaffoldName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"the incident", "--help", true},
		{"single dash help", "-h", true},
		{"any other flag", "--verbose", true},
		{"empty (unset shell var)", "", true},
		{"ordinary name", "myproject", false},
		{"default name", "my-ailang-app", false},
		// Only a LEADING dash is a flag. Interior dashes are ordinary, and
		// rejecting them would break the command's own documented default.
		{"interior dashes fine", "my-cool-app-2", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkScaffoldName(tc.input)
			if tc.wantErr && err == nil {
				t.Fatalf("checkScaffoldName(%q) = nil, want an error", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("checkScaffoldName(%q) = %v, want nil", tc.input, err)
			}
		})
	}
}

// The guard is worth little if the scaffolder does not consult it, so assert
// against the filesystem: the call site is what the incident exercised.
func TestInitWebApp_RefusesFlagNameAndWritesNothing(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	if err := initWebApp("--help"); err == nil {
		t.Fatal("initWebApp(\"--help\") returned nil — it scaffolded a flag-named directory")
	} else if !strings.Contains(err.Error(), "starts with a dash") {
		t.Errorf("error should explain the dash, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "--help")); !os.IsNotExist(err) {
		t.Error("a directory named \"--help\" was created despite the refusal")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("refused scaffold left %d entries behind, want 0: %v", len(entries), entries)
	}
}
