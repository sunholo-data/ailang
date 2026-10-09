package config

import "testing"

func TestAutoMergeCodeEnvironment(t *testing.T) {
	t.Setenv(EnvAutoMergeCode, "")
	if AutoMergeCode() {
		t.Fatal("absent code opt-in enabled")
	}
	t.Setenv(EnvAutoMergeCode, "1")
	t.Setenv(EnvAutoMergeRequiredChecks, " build, site\nimage-size\n")
	t.Setenv(EnvApproverSecret, " secret-name ")
	t.Setenv(EnvApproverIdentity, " reviewer ")
	checks := AutoMergeRequiredChecks()
	if !AutoMergeCode() || len(checks) != 2 || checks[0] != "build, site" || ApproverSecret() != "secret-name" || ApproverIdentity() != "reviewer" {
		t.Fatalf("config lost: %v", checks)
	}
}
