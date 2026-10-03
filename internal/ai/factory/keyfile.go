package factory

import (
	"fmt"
	"os"
	"strings"
)

// maxKeyFileBytes bounds what ReadKeyFile accepts. Provider keys are well
// under 1 KiB; anything larger is the wrong file, and refusing it keeps a
// mis-pointed path from being sent to a provider as a header.
const maxKeyFileBytes = 16 * 1024

// ReadKeyFile reads an API key from path (#1499 --ai-key-file /
// AILANG_AI_KEY_FILE): surrounding whitespace is trimmed and the remainder
// must be a single non-empty line. Errors name the path, never the contents,
// so a key can reach a provider without ever passing through argv, the
// environment, a log line or an error message.
func ReadKeyFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("--ai-key-file %s: %w", path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("--ai-key-file %s: is a directory", path)
	}
	if info.Size() > maxKeyFileBytes {
		return "", fmt.Errorf("--ai-key-file %s: %d bytes is too large for an API key (limit %d)", path, info.Size(), maxKeyFileBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("--ai-key-file %s: %w", path, err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("--ai-key-file %s: file is empty", path)
	}
	if strings.ContainsAny(key, "\r\n") {
		return "", fmt.Errorf("--ai-key-file %s: expected a single line holding the key, found several (contents not shown)", path)
	}
	return key, nil
}
