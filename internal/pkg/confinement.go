package pkg

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/config"
)

// Confined reports whether the host has installed an agent policy.
func Confined() bool { return config.AgentPolicy() != "" }

// RefuseIfConfined guards registry authority before any network or cache write.
func RefuseIfConfined(what string) error {
	if Confined() {
		return fmt.Errorf("confined by AILANG_AGENT_POLICY: %s is operator authority — provision AILANG_PACKAGE_ROOT or run 'ailang install' outside the sandbox", what)
	}
	return nil
}

// EnsureRegistryCacheDir creates the writable HOME registry cache for operators.
// Package reads must use RegistryCacheDir, which does not create anything.
func EnsureRegistryCacheDir() (string, error) {
	if err := RefuseIfConfined("writing the registry cache"); err != nil {
		return "", err
	}
	dir, err := RegistryCacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}
