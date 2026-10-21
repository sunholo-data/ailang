package rolemodels

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestRoutingScriptAllowlistParity(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "mission_routing_models_check.sh"))
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?m)^ALLOWED_OPENAI='([^']*)'$`).FindStringSubmatch(string(body))
	if len(match) != 2 {
		t.Fatal("script must declare one literal ALLOWED_OPENAI list")
	}
	keys := func(ids []string) []string {
		out := make([]string, len(ids))
		for i, id := range ids {
			out[i] = CanonicalKey(id)
		}
		sort.Strings(out)
		return out
	}
	got, want := keys(strings.Fields(match[1])), keys(AllowedOpenAI)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shell allowlist drift: canonical shell %v != Go %v", got, want)
	}
}
