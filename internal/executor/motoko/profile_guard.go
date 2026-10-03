package motoko

import (
	"fmt"
	"os"
	"path/filepath"
)

// requireProfileInRepo refuses a run whose profile does not exist in the motoko
// checkout. motoko resolves an unknown profile to defaults SILENTLY
// (extensions.order=[]), so a models.yml entry naming a profile the checkout
// lacks runs a different harness than it claims. Measured 2026-10-02: the
// ollama_fmt / ollama_docs / ollama_dp7 rows named profiles that ~/dev/mk-main
// no longer carried; any new trial on them would have banked defaults under the
// treatment's name.
//
// Only a checkout that has profiles at all (<repo>/.motoko/config) is judged.
// An undiscovered repo is HealthCheck's problem, not this guard's, and the
// unit tests' fake repos carry no config tree.
func requireProfileInRepo(repo, profile string) error {
	if repo == "" || profile == "" {
		return nil
	}
	configRoot := filepath.Join(repo, ".motoko", "config")
	if fi, err := os.Stat(configRoot); err != nil || !fi.IsDir() {
		return nil
	}
	profileConfig := filepath.Join(configRoot, profile, "config.json")
	if _, err := os.Stat(profileConfig); err != nil {
		return fmt.Errorf("motoko profile %q does not exist in %s (no %s); motoko would silently run defaults — add the profile to the motoko checkout or fix motoko_profile in models.yml",
			profile, repo, profileConfig)
	}
	return nil
}
