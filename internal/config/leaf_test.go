package config_test

import (
	"os/exec"
	"strings"
	"testing"
)

// config is a LEAF: standard library plus gopkg.in/yaml.v3, nothing under
// internal/. internal/storage, internal/coordinator, internal/messaging,
// internal/observatory and every internal/ai provider resolve their project
// through this package; the moment config imports any of them the graph has a
// cycle. The compiler enforces the cycle; this test enforces the stricter rule
// and names why, which a build error does not.
func TestConfigIsALeaf(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list -deps: %v", err)
	}
	deps := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Control: `go list -deps .` always lists the package itself, so an
	// absence assertion passes vacuously on a build that did not resolve.
	// yaml.v3 is the one third-party dependency the package genuinely has.
	const mustSee = "gopkg.in/yaml.v3"
	sawControl := false
	for _, d := range deps {
		if d == mustSee {
			sawControl = true
			break
		}
	}
	if !sawControl {
		t.Fatalf("instrument check failed: %d deps returned but %s absent; config parses "+
			"pubsub.project_id from YAML, so this build did not resolve and the assertion "+
			"below would pass vacuously", len(deps), mustSee)
	}

	const module = "github.com/sunholo-data/ailang/"
	const self = module + "internal/config"
	for _, d := range deps {
		if d == self {
			continue
		}
		if strings.HasPrefix(d, module) {
			t.Errorf("config must be a leaf but depends on %s; move the needed symbol DOWN, never import up", d)
			continue
		}
		first, _, _ := strings.Cut(d, "/")
		if strings.Contains(first, ".") && d != mustSee {
			t.Errorf("config may depend on the standard library and %s only, but depends on %s", mustSee, d)
		}
	}
}
