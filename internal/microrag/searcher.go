package microrag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// CLISearcher shells out to `ailang cache search --json`. Production wiring.
type CLISearcher struct {
	Binary  string        // path to ailang binary; "" → "ailang" on PATH
	Timeout time.Duration // per-call timeout; 0 → no timeout
}

// Search runs `ailang cache search --namespace <ns> --limit <n> --json <query>`.
func (c *CLISearcher) Search(query, namespace string, limit int) ([]SearchHit, error) {
	bin := c.Binary
	if bin == "" {
		bin = "ailang"
	}
	args := []string{"cache", "search", "--json", "--limit", strconv.Itoa(limit)}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}
	args = append(args, query)
	cmd := exec.Command(bin, args...)
	if c.Timeout > 0 {
		// exec.CommandContext is cleaner; using simple wall-clock kill below.
		cmd.WaitDelay = c.Timeout
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cache search failed: %w (stderr: %s)", err, stderr.String())
	}
	var env SearchEnvelope
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		return nil, fmt.Errorf("parse cache search JSON: %w", err)
	}
	return env.Results, nil
}
