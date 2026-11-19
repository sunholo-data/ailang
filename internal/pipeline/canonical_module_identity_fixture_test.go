package pipeline

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// Downstream regression (motoko-fork tools/code-graph fixtures): inside a
// prefixed root package, a fixture that declares itself by its path (`module
// stub_step`, imported as `stub_step` from its own directory) keeps its path
// identity — it is not renamed to the root module src/…/stub_step, which its
// module line would then contradict (MOD010).
func TestScope_RootFixtureDeclaringItsPathKeepsIt(t *testing.T) {
	root := prefixpkg.Tree(t, map[string]string{
		"app/ailang.toml":            prefixpkg.Manifest("local/agent", "src", nil, nil, nil),
		"app/tools/fx/stub_step.ail": "module stub_step\nexport func dispatch_step() -> int { 1 }\n",
		"app/tools/fx/loop.ail":      "module loop\nimport stub_step (dispatch_step)\nexport func run() -> int { dispatch_step() }\n",
	})
	dir := filepath.Join(root, "app", "tools", "fx")
	t.Chdir(dir)
	result := checkFixture(t, filepath.Join(dir, "loop.ail"))
	if _, ok := result.Modules["stub_step"]; !ok {
		t.Fatalf("stub_step must keep its path identity; modules %v", sortedKeys(result.Modules))
	}
}
