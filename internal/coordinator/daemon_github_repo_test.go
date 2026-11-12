package coordinator

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"
)

// syncRepoIssues passes the repo into a subprocess argv, and on the webhook
// path the repo comes from the payload. Anything that is not an exact
// owner/repo must be refused, loudly, before the subprocess is built.
func TestSyncRepoIssuesRefusesMalformedRepo(t *testing.T) {
	// A cancelled context guarantees no subprocess can start even if the
	// guard regresses: exec.CommandContext refuses to start on a done context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for _, repo := range []string{"--upload-pack=touch /tmp/x", "owner/repo/extra", "../etc", "owner/..", "", "owner repo/x"} {
		t.Run(repo, func(t *testing.T) {
			var buf bytes.Buffer
			d := &Daemon{ctx: ctx, logger: log.New(&buf, "", 0)}
			d.syncRepoIssues(RepoSyncConfig{Repo: repo, Enabled: true})
			out := buf.String()
			if !strings.Contains(out, "GitHub sync REFUSED") {
				t.Fatalf("repo %q: want a REFUSED log line, got %q", repo, out)
			}
			if strings.Contains(out, "Syncing GitHub issues") {
				t.Fatalf("repo %q: sync went ahead: %q", repo, out)
			}
		})
	}
}
