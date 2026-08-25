package main

import "fmt"

func validateBrowserEvalFlags(agent bool, provider string) error {
	if provider == "" {
		return nil
	}
	if !agent {
		return fmt.Errorf("--browser-provider requires --agent")
	}
	if provider != "local-playwright" && provider != "browserbase" {
		return fmt.Errorf("invalid --browser-provider %q (want local-playwright or browserbase)", provider)
	}
	return nil
}
