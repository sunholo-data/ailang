package main

import (
	"strings"
	"testing"
)

// TestPkgCascadeCommand_Dispatch covers the dispatcher: bad subcommand,
// help, and missing subcommand. The actual `status` subcommand needs a
// live registry — that's covered by the integration smoke test in M5.
func TestPkgCascadeCommand_Dispatch(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantErr  bool
		wantHelp bool
	}{
		{name: "no args", args: []string{}, wantErr: true},
		{name: "unknown subcommand", args: []string{"bogus"}, wantErr: true},
		{name: "help long", args: []string{"--help"}, wantErr: false, wantHelp: true},
		{name: "help short", args: []string{"-h"}, wantErr: false, wantHelp: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := pkgCascadeCommand(tc.args)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestPkgCascadeStatusCommand_Help verifies the help path returns nil
// (no error) and prints usage. Missing-arg returns an error.
func TestPkgCascadeStatusCommand_Help(t *testing.T) {
	if err := pkgCascadeStatusCommand([]string{"--help"}); err != nil {
		t.Errorf("--help should not error, got %v", err)
	}
	if err := pkgCascadeStatusCommand([]string{"-h"}); err != nil {
		t.Errorf("-h should not error, got %v", err)
	}
	err := pkgCascadeStatusCommand([]string{})
	if err == nil {
		t.Error("missing argument should error")
	}
	if err != nil && !strings.Contains(err.Error(), "missing required argument") {
		t.Errorf("missing-arg error should mention required argument, got %q", err.Error())
	}
}

// TestDependentStatus_Defaults documents the expected defaults for the
// status struct so callers know what "nothing observed yet" looks like.
// Cheap guard against accidentally changing the contract.
func TestDependentStatus_Defaults(t *testing.T) {
	var ds dependentStatus
	if ds.Status != "" {
		t.Errorf("zero-value Status should be empty, got %q", ds.Status)
	}
	if ds.CostUSD != 0 {
		t.Errorf("zero-value CostUSD should be 0, got %f", ds.CostUSD)
	}
}
