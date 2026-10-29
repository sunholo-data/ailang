package main

import (
	"strings"
	"testing"
)

// --process-* are refused without --caps Process rather than ignored: a
// timeout or allowlist that silently does nothing is a false guarantee.
func TestServeAPIProcessFlagsNeedProcessCap(t *testing.T) {
	for _, flag := range []string{"--process-timeout", "--process-allowlist"} {
		val := "90s"
		if flag == "--process-allowlist" {
			val = "ailang:run"
		}
		err := serveAPICommand([]string{"--caps", "IO", flag, val, t.TempDir()})
		if err == nil || !strings.Contains(err.Error(), "need --caps Process") {
			t.Errorf("%s without --caps Process: err = %v", flag, err)
		}
	}
}

func TestServeAPIProcessTimeoutInvalid(t *testing.T) {
	err := serveAPICommand([]string{"--caps", "Process", "--process-timeout", "soon", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "invalid --process-timeout") {
		t.Errorf("err = %v", err)
	}
}
