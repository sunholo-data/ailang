package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
)

type publishCredential struct {
	key     string
	fromEnv bool
}

// Resolve once before package subprocesses, after the operator authority gate.
// Help/dry-run never inspect the file or load a credential for an unused upload.
func resolvePublishCredential(path string, selected, dry bool) (publishCredential, error) {
	if dry {
		return publishCredential{}, nil
	}
	if err := pkg.Require(pkg.AuthNetwork, "reading publisher credentials", pkg.OperatorRemedy); err != nil {
		return publishCredential{}, err
	}
	env := config.RegistryAPIKey()
	if selected {
		if path == "" {
			return publishCredential{}, fmt.Errorf("publish --key-file path is empty")
		}
		if env != "" {
			return publishCredential{}, fmt.Errorf("publish cannot use both --key-file and AILANG_REGISTRY_API_KEY; unset the environment key to select the file")
		}
		absolute, err := filepath.Abs(path)
		if err != nil {
			return publishCredential{}, fmt.Errorf("publish --key-file path: %w", err)
		}
		key, err := pkg.ReadPublishKeyFile(absolute)
		if err != nil {
			return publishCredential{}, err
		}
		return publishCredential{key: key}, nil
	}
	if env == "" {
		return publishCredential{}, fmt.Errorf("publish requires --key-file or a per-command AILANG_REGISTRY_API_KEY")
	}
	return publishCredential{key: env, fromEnv: true}, nil
}

// Redact only an exact selected-key echo at the publish error boundary.
func redactPublishError(err error, key string) error {
	if err == nil || key == "" {
		return err
	}
	safe := strings.ReplaceAll(err.Error(), key, "[REDACTED]")
	if safe == err.Error() {
		return err
	}
	return fmt.Errorf("%s", safe)
}
