package github

import "github.com/sunholo-data/ailang/internal/pkg"

// ConfinementRemedy keeps docs search local inside an agent policy.
const ConfinementRemedy = "use --path with a local documentation corpus, or run GitHub docs search outside the sandbox as the operator"

func requireNetwork() error {
	return pkg.Require(pkg.AuthNetwork, "GitHub docs search", ConfinementRemedy)
}

func requireCacheWrite() error {
	return pkg.Require(pkg.AuthHomeWrite, "writing the GitHub docs search cache", ConfinementRemedy)
}
