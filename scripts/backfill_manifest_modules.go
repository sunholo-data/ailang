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
// Authored fields and provenance are preserved; modules and statistics are derived.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/sunholo-data/ailang/internal/manifest"
	"os"
	"reflect"
	"sort"

	"github.com/sunholo-data/ailang/scripts/internal/importextract"
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
	manifestPath := flag.String("manifest", "examples/manifest.json", "Manifest to synchronize")
	examplesDir := flag.String("dir", "examples", "Examples directory")
	checkOnly := flag.Bool("check", false, "Report drift without writing")
	flag.Parse()

	raw, err := os.ReadFile(*manifestPath)
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
		full, ok := importextract.ResolvePath(*examplesDir, e.Path)
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

	// Decode independently of Load: stale statistics are precisely what we repair.
	var model manifest.Manifest
	if err := json.Unmarshal(raw, &model); err != nil {
		fmt.Fprintf(os.Stderr, "parse statistics: %v\n", err)
		os.Exit(1)
	}
	model.UpdateStatistics()
	var recorded map[string]interface{}
	if err := json.Unmarshal(d.Statistics, &recorded); err != nil {
		fmt.Fprintf(os.Stderr, "parse statistics: %v\n", err)
		os.Exit(1)
	}
	stats := map[string]interface{}{
		"total": float64(model.Statistics.Total), "working": float64(model.Statistics.Working),
		"aspirational": float64(model.Statistics.Aspirational), "broken": float64(model.Statistics.Broken),
		"coverage": model.Statistics.Coverage, "experimental": float64(model.Statistics.Experimental),
	}
	statsDrift := !reflect.DeepEqual(recorded, stats)
	// Retain the committed statistics key order.
	d.Statistics, err = json.Marshal(struct {
		Total        int     `json:"total"`
		Working      int     `json:"working"`
		Aspirational int     `json:"aspirational"`
		Broken       int     `json:"broken"`
		Coverage     float64 `json:"coverage"`
		Experimental int     `json:"experimental"`
	}{model.Statistics.Total, model.Statistics.Working, model.Statistics.Aspirational, model.Statistics.Broken, model.Statistics.Coverage, model.Statistics.Experimental})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *checkOnly {
		if changed > 0 || statsDrift {
			fmt.Printf("manifest drift: modules %d entr(ies), statistics drift: %t\n", changed, statsDrift)
			fmt.Println("regenerate with: make sync-examples-manifest")
			os.Exit(1)
		}
		fmt.Println("modules and statistics up to date")
		return
	}

	if changed == 0 && !statsDrift {
		fmt.Println("modules and statistics up to date")
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
	if err := os.WriteFile(*manifestPath, buf.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write manifest: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("backfilled modules for %d entr(ies) (%d skipped: missing/unparseable)\n", changed, skipped)
}
