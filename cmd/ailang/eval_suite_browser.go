package main

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/browser/auth"
)

func validateBrowserEvalFlags(agent bool, provider, profile string) error {
	if provider == "" {
		if profile != "" {
			return fmt.Errorf("--browser-profile requires --browser-provider")
		}
		return nil
	}
	if !agent {
		return fmt.Errorf("--browser-provider requires --agent")
	}
	if provider != "local-playwright" && provider != "browserbase" {
		return fmt.Errorf("invalid --browser-provider %q (want local-playwright or browserbase)", provider)
	}
	if profile == "" {
		return nil
	}
	// Parse eagerly so a malformed reference fails before any model is billed
	// and before a browser is provisioned.
	if _, err := auth.ParseRef(profile); err != nil {
		return fmt.Errorf("invalid --browser-profile: %w", err)
	}
	return nil
}
