package pkg

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

func TestPublishKeyFile_AuthorityBeforePath(t *testing.T) {
	t.Setenv(config.EnvAgentPolicy, "private-policy")
	_, err := readPublishKeyBytes(filepath.Join(t.TempDir(), "nonexistent"))
	var refusal *ConfinedError
	if !errors.As(err, &refusal) || refusal.Authority != AuthNetwork {
		t.Fatalf("authority must precede path lookup: %v", err)
	}
}

func TestPublishKeyFile_UnsupportedPlatform(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		t.Skip("protected Unix reader has separate executable controls")
	}
	t.Setenv(config.EnvAgentPolicy, "")
	_, err := readPublishKeyBytes(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("must refuse before file lookup: %v", err)
	}
}
