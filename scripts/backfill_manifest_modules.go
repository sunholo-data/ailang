//go:build ignore
// +build ignore

// backfill_manifest_modules.go populates the `modules` field on every
// examples/manifest.json entry using the shared, parser-backed extractor
// (scripts/internal/importextract). Run once at development time and COMMIT the
// result — `ailang docs --examples <module>` reads the committed field, never a
// runtime scan. The SAME extractor backs the validate_manifest drift assertion,
// so the CI authority cannot disagree with the language.
//
//	go run ./scripts/backfill_manifest_modules.go            # write in place
//	go run ./scripts/backfill_manifest_modules.go --check    # report drift only
//
// All manifest fields (run_mode, run_flags, requires, broken, statistics,
// milestones, schema…) are preserved; only each entry's `modules` is (re)set.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/sunholo-data/ailang/scripts/internal/importextract"
)

const (
	manifestPath = "examples/manifest.json"
	examplesDir  = "examples"
)

// entry mirrors every field present in examples/manifest.json so a round-trip
// loses nothing. Field order here defines the serialized key order.
type entry struct {
	Path        string          `json:"path"`
	Status      string          `json:"status"`
	Tags        []string        `json:"tags,omitempty"`
	Description string          `json:"description,omitempty"`
	Modules     []string        `json:"modules,omitempty"`
	Expected    json.RawMessage `json:"expected,omitempty"`
	Broken      json.RawMessage `json:"broken,omitempty"`
	Requires    json.RawMessage `json:"requires,omitempty"`
	RunMode     json.RawMessage `json:"run_mode,omitempty"`
	RunFlags    json.RawMessage `json:"run_flags,omitempty"`
	SkipReason  string          `json:"skip_reason,omitempty"`
}

// doc mirrors the top-level manifest, preserving unknown blocks verbatim.
type doc struct {
	Schema        json.RawMessage `json:"schema,omitempty"`
	SchemaVersion json.RawMessage `json:"schema_version,omitempty"`
	GeneratedAt   json.RawMessage `json:"generated_at,omitempty"`
	Generator     json.RawMessage `json:"generator,omitempty"`
	Examples      []entry         `json:"examples"`
	Statistics    json.RawMessage `json:"statistics,omitempty"`
	Milestones    json.RawMessage `json:"milestones,omitempty"`
}

func main() {
	checkOnly := false
	for _, a := range os.Args[1:] {
		if a == "--check" {
			checkOnly = true
		}
	}

	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read manifest: %v\n", err)
		os.Exit(1)
	}

	var d doc
	if err := json.Unmarshal(raw, &d); err != nil {
		fmt.Fprintf(os.Stderr, "parse manifest: %v\n", err)
		os.Exit(1)
	}

	changed, skipped := 0, 0
	for i := range d.Examples {
		e := &d.Examples[i]
		full, ok := importextract.ResolvePath(examplesDir, e.Path)
		if !ok {
			skipped++ // missing file — validate_manifest flags it separately
			continue
		}
		mods, err := importextract.ExtractModules(full)
		if err != nil {
			skipped++ // unparseable — already red via the verify gate; never guess
			continue
		}
		if mods == nil {
			mods = []string{}
		}
		sort.Strings(mods)
		if !importextract.Equal(e.Modules, mods) {
			changed++
		}
		if len(mods) == 0 {
			e.Modules = nil // omitempty: keep entries without std imports clean
		} else {
			e.Modules = mods
		}
	}

	if checkOnly {
		if changed > 0 {
			fmt.Printf("modules drift: %d entr(ies) out of date\n", changed)
			fmt.Println("regenerate with: go run ./scripts/backfill_manifest_modules.go")
			os.Exit(1)
		}
		fmt.Println("modules field up to date")
		return
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&d); err != nil {
		fmt.Fprintf(os.Stderr, "encode manifest: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(manifestPath, buf.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write manifest: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("backfilled modules for %d entr(ies) (%d skipped: missing/unparseable)\n", changed, skipped)
}
