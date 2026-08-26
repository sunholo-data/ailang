package gatelint

type allowReason struct {
	text string
}

func mustReason(text string) allowReason {
	if text == "" {
		panic("gatelint allowlist reasons must not be empty")
	}
	return allowReason{text: text}
}

var ruleAllowlist = map[Rule]map[string]allowReason{
	RuleR2: {
		"internal/coordinator/provider_script_test.go": mustReason("this is a Unix shell/grandchild signal semantics test, not a live-network test, so the network opt-in helper would misstate its requirement. CI runners skip the known flaky shell behavior while local Unix runs retain coverage."),
	},
	RuleR3: {
		"internal/coordinator/agent_registry_test.go": mustReason("ailang-packages appears only in workspace-path fixtures; this test performs no live network call"),
		"internal/coordinator/execution_lane_test.go": mustReason("ailang-packages appears only in workspace-shape fixtures for execution-lane inference (M-MESSAGE-PLANE-FAIL-LOUD M3); this test performs no live network call"),
		"internal/parser/cli_integration_test.go":     mustReason("httpbin.org appears only in expected CLI diagnostic fixture text; this test performs no live network call"),
		"internal/parser/suggestion_errors_test.go":   mustReason("httpbin.org appears only in parser suggestion/error fixtures; this test performs no live network call"),
		"internal/messaging/config_test.go":           mustReason("ailang-packages appears only in registry-mapping configuration fixtures; this test performs no live network call"),
		"internal/pkg/manifest_test.go":               mustReason("ailang-packages appears only in manifest parsing fixture URLs; this test performs no live network call"),
	},
}

func isAllowlisted(rule Rule, path string) bool {
	_, ok := ruleAllowlist[rule][path]
	return ok
}
