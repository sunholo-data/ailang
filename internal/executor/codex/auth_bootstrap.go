package codex

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Codex reads its credential from a FILE, so an env var alone is no credential
// (2026-09-05).
//
// `codex` authenticates from ~/.codex/auth.json, written by `codex login`. It
// does not consult OPENAI_API_KEY — probe-verified 2026-07-30, and the reason
// authLane reads the file rather than the environment.
//
// The Cloud Run codex jobs were given OPENAI_API_KEY from Secret Manager and no
// auth.json, so codex ran with no credential at all and every turn 401'd against
// wss://api.openai.com/v1/responses. The key itself was fine the whole time: the
// same secret returns 200 from /v1/models and from /v1/responses on gpt-5.6-sol.
// Nothing was wrong with the credential — it was in a place codex never looks.
//
// OpenAI documents an API key as the right way to authenticate automation, so
// materialising the file the CLI expects is the sanctioned fix; it needs no
// subscription token, and so carries neither the "one machine or serialized job
// stream" condition nor the ~8-day refresh that a ChatGPT auth.json would.

// authFileMode keeps the credential owner-only.
const authFileMode = 0o600

// apiKeyAuth is the on-disk shape `codex login --api-key` writes.
type apiKeyAuth struct {
	AuthMode string `json:"auth_mode"`
	APIKey   string `json:"OPENAI_API_KEY"`
}

// EnsureAPIKeyAuth writes an api-key auth.json when, and only when, there is no
// auth.json already and OPENAI_API_KEY is set.
//
// It NEVER overwrites an existing file. The rig authenticates codex with a
// ChatGPT-subscription auth.json (auth_mode "chatgpt") and also keeps
// OPENAI_API_KEY exported for other tools; overwriting there would silently move
// those runs onto a metered key — turning a flat-rate lane into a billed one
// with nothing in the output to say so.
//
// Returns true when it wrote a file.
func EnsureAPIKeyAuth() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("locate home directory for codex auth: %w", err)
	}
	dir := filepath.Join(home, ".codex")
	path := filepath.Join(dir, "auth.json")

	// Any existing credential wins, including one this process cannot parse —
	// "unreadable" is not "absent", and guessing wrong costs real money.
	if _, statErr := os.Stat(path); statErr == nil {
		return false, nil
	} else if !os.IsNotExist(statErr) {
		return false, fmt.Errorf("stat codex auth.json: %w", statErr)
	}

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		// Nothing to write. Left to HealthCheck/the run to fail loudly, rather
		// than inventing a credential here.
		return false, nil
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, fmt.Errorf("create %s: %w", dir, err)
	}
	data, err := json.Marshal(apiKeyAuth{AuthMode: "apikey", APIKey: key})
	if err != nil {
		return false, fmt.Errorf("encode codex auth: %w", err)
	}
	// O_EXCL so a concurrent writer cannot be clobbered by this one.
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, authFileMode)
	if err != nil {
		if os.IsExist(err) {
			return false, nil // someone else got there first; their file stands
		}
		return false, fmt.Errorf("create codex auth.json: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(data); err != nil {
		return false, fmt.Errorf("write codex auth.json: %w", err)
	}
	return true, nil
}
