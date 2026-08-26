package eval_harness

import "testing"

// TestIsOllamaCloudRoute pins ollama's own suffix grammar. Getting this wrong
// is not a loud failure: omitting the suffix yields
// "404 model 'kimi-k3' not found", which reads as a missing model rather than a
// misrouted request (M-OLLAMA-CLOUD-PROVIDER V38).
func TestIsOllamaCloudRoute(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		// Untagged model: the whole suffix is "cloud".
		{"untagged cloud", "kimi-k3:cloud", true},
		{"untagged cloud, agent form", "ollama/kimi-k3:cloud", true},
		// Tagged model: "-cloud" appends to the TAG.
		{"tagged cloud", "deepseek-v4-flash:0731-cloud", true},
		{"tagged cloud, size tag", "gpt-oss:20b-cloud", true},
		{"tagged cloud, agent form", "ollama/deepseek-v4-flash:0731-cloud", true},
		{"case insensitive", "GPT-OSS:20B-CLOUD", true},

		// Local models must NOT be mistaken for cloud — a false positive here
		// would drop a genuine GPU job out of the rig lock and cause the thrash
		// the lock exists to prevent.
		{"local tagged", "qwen3.8:27b-mxfp8", false},
		{"local untagged", "gemma4", false},
		{"local agent form", "ollama/qwen3.6:35b-a3b-mxfp8", false},
		{"bare cloud is not a suffix", "cloud", false},
		{"substring only", "kimi-k3-cloudy", false},
		{"cloud in the name, not the tag", "cloudmodel:latest", false},
		{"empty", "", false},

		// A vendor/name path in the suffix position is not a route marker.
		{"slash in suffix", "vendor:some/thing-cloud", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsOllamaCloudRoute(c.in); got != c.want {
				t.Errorf("IsOllamaCloudRoute(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
