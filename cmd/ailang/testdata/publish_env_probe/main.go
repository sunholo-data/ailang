// This acceptance probe reports presence only, never credential contents.
package main

import (
	"os"
)

func main() {
	sentinel := "KEY_ABSENT"
	if _, present := os.LookupEnv("AILANG_REGISTRY_API_KEY"); present {
		sentinel = "KEY_PRESENT"
	}
	if sentinel != os.Getenv("PUBLISH_PROBE_EXPECT") {
		os.Exit(41)
	}
	if path := os.Getenv("PUBLISH_PROBE_REPLACE"); path != "" {
		if err := os.WriteFile(path, []byte("replacement-dummy"), 0600); err != nil {
			os.Exit(42)
		}
	}
	if err := os.WriteFile(os.Getenv("PUBLISH_PROBE_MARKER"), []byte(sentinel), 0600); err != nil {
		os.Exit(43)
	}
}
