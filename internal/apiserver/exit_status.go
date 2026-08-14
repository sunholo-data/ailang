package apiserver

import (
	"errors"

	"github.com/sunholo-data/ailang/internal/embed"
)

// isCleanExit reports whether an embedded handler deliberately called exit(0).
//
// D-17 resolves #706 by giving API hosts the same clean-exit semantics as the
// CLI batch path. The embed layer deliberately preserves exit(0) as a typed
// error (#691), so each host must classify it. Non-zero exits remain errors.
func isCleanExit(err error) bool {
	var exitErr *embed.ExitError
	return errors.As(err, &exitErr) && exitErr.Code == 0
}
