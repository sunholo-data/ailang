package executor

import (
	"errors"
	"fmt"
	"os"
)

// WritePromptFile writes a prompt to a fresh owner-only temp file, for CLIs
// that can read one (`claude --system-prompt-file`), and returns its path and
// a cleanup that removes it.
//
// Prompts never go on a child's argv. argv is world-readable — `ps`, and so
// `pkill -f`, match against it — and on 2026-10-27/28 an eval agent's
// `pkill -f "ailang run"` matched every sibling agent CLI whose argv carried
// the teaching prompt, SIGTERMing them all.
//
// The file is closed before WritePromptFile returns: Windows will not let a
// second process open a file this one still holds for writing. Call cleanup
// only after the child has exited. A failure is returned, never papered over
// by falling back to argv.
func WritePromptFile(pattern, content string) (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", nil, fmt.Errorf("prompt file: %w", err)
	}
	path = f.Name()
	_, werr := f.WriteString(content)
	if err := errors.Join(werr, f.Close()); err != nil {
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("prompt file: write %s: %w", path, err)
	}
	return path, func() { _ = os.Remove(path) }, nil
}
