package main

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor/agy"
	"github.com/sunholo-data/ailang/internal/executor/codex"
)

// codexCredentialSpec: a cloud codex job runs on the ChatGPT SUBSCRIPTION (Mark,
// 2026-09-30: cloud executors bill OAuth, never the metered API; OpenAI has
// confirmed this use). No secret is a hard failure — the old behaviour of quietly
// writing an api-key auth.json from OPENAI_API_KEY is how every cloud codex run
// billed the API. AILANG_AUTH_MODE=apikey no longer selects a metered key here: the
// per-request API-key jobs were retired 2026-11-10, and the dispatcher only sends
// that marker on the guarded credit lane, which is Claude-only.
var codexCredentialSpec = &rotatingCredentialSpec{
	label:     "Codex",
	secretEnv: config.EnvCodexAuthSecret,
	secret:    config.CodexAuthSecret,
	precheck: func() error {
		if config.CodexHome() == "" {
			return fmt.Errorf("cloud subscription execution requires an explicit isolated CODEX_HOME")
		}
		return nil
	},
	validate: func(b []byte) error { _, err := codex.ParseSubscriptionAuth(b); return err },
	install:  codex.InstallSubscriptionAuth,
	path:     codex.InstalledAuthPath,
	billing:  "ChatGPT subscription",
}

// agyCredentialSpec: a cloud agy job runs on the Google AI (Ultra) subscription
// through a cloud-only sign-in (M-AGY-CLOUD D1; tools/agy/signin.sh --profile
// cloud). Subscription only, by ruling (Mark, 2026-11-24): metered Gemini in the
// cloud is gemini_agents, so there is no API-key agy job.
var agyCredentialSpec = &rotatingCredentialSpec{
	label:     "agy",
	secretEnv: config.EnvAgyAuthSecret,
	secret:    config.AgyAuthSecret,
	precheck: func() error {
		if config.AgyHomeRoot() == "" {
			return fmt.Errorf("cloud subscription execution requires an explicit isolated %s", config.EnvAgyHomeRoot)
		}
		return nil
	},
	validate: func(b []byte) error { _, err := agy.ParseSubscriptionToken(b); return err },
	install:  agy.InstallSubscriptionToken,
	path:     agy.InstalledTokenPath,
	billing:  "Google AI subscription",
}
