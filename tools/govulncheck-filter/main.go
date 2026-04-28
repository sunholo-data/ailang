// govulncheck-filter reads `govulncheck -format json ./...` output from
// stdin, compares findings against .govulncheck-allow.yml, and exits
// non-zero on any unallowlisted finding or any allowlisted entry past
// its expiry date.
//
// Usage:
//
//	govulncheck -format json ./... | govulncheck-filter
//	govulncheck -format json ./... | govulncheck-filter -allow=path/to/allow.yml
//
// Exit codes:
//
//	0 - no findings, or every finding has a non-expired allowlist entry
//	1 - one or more findings with no allowlist entry, or expired entry
//	2 - filter itself failed (bad input, missing allowlist, etc.)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

type allowEntry struct {
	ID      string `yaml:"id"`
	Reason  string `yaml:"reason"`
	Expires string `yaml:"expires"`
}

type allowlist struct {
	Allow []allowEntry `yaml:"allow"`
}

// govulncheck JSON streams a sequence of objects, one per line. We
// only care about the "finding" frames where Trace[0].Module is
// non-empty (those are real reachable findings — symbol-level rather
// than module-level scan artifacts).
type vulnFrame struct {
	Finding *struct {
		OSV   string `json:"osv"`
		Trace []struct {
			Module   string `json:"module"`
			Function string `json:"function"`
		} `json:"trace"`
	} `json:"finding"`
}

func main() {
	allowPath := flag.String("allow", ".govulncheck-allow.yml", "path to allowlist YAML")
	flag.Parse()

	allow, err := loadAllowlist(*allowPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck-filter: load allowlist: %v\n", err)
		os.Exit(2)
	}

	// Index allowlist by ID for O(1) lookup. Surface duplicate IDs
	// rather than silently masking them.
	byID := map[string]allowEntry{}
	for _, e := range allow.Allow {
		if _, dup := byID[e.ID]; dup {
			fmt.Fprintf(os.Stderr, "govulncheck-filter: duplicate allowlist entry for %s\n", e.ID)
			os.Exit(2)
		}
		byID[e.ID] = e
	}

	findings, err := readFindings(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "govulncheck-filter: parse stdin: %v\n", err)
		os.Exit(2)
	}

	now := time.Now().UTC()
	var (
		blocking []string // findings without allowlist entries
		expired  []string // allowlist entries whose expiry has passed
		seen     = map[string]bool{}
	)

	for _, id := range findings {
		if seen[id] {
			continue
		}
		seen[id] = true

		entry, ok := byID[id]
		if !ok {
			blocking = append(blocking, id)
			continue
		}
		t, err := time.Parse("2006-01-02", entry.Expires)
		if err != nil {
			fmt.Fprintf(os.Stderr, "govulncheck-filter: bad expires date %q for %s: %v\n",
				entry.Expires, id, err)
			os.Exit(2)
		}
		if !t.After(now) {
			expired = append(expired, fmt.Sprintf("%s (expired %s)", id, entry.Expires))
		}
	}

	// Surface allowlist entries that no longer match any finding —
	// they're dead weight and should be removed.
	var stale []string
	for _, e := range allow.Allow {
		if !seen[e.ID] {
			stale = append(stale, e.ID)
		}
	}

	sort.Strings(blocking)
	sort.Strings(expired)
	sort.Strings(stale)

	if len(blocking) == 0 && len(expired) == 0 {
		fmt.Printf("govulncheck-filter: %d finding(s), all allowlisted and unexpired.\n",
			len(findings))
		if len(stale) > 0 {
			fmt.Printf("Note: %d stale allowlist entr(ies) — consider removing: %v\n",
				len(stale), stale)
		}
		os.Exit(0)
	}

	if len(blocking) > 0 {
		fmt.Fprintf(os.Stderr, "govulncheck-filter: %d unallowlisted finding(s):\n",
			len(blocking))
		for _, id := range blocking {
			fmt.Fprintf(os.Stderr, "  - %s\n", id)
		}
		fmt.Fprintln(os.Stderr,
			"Either fix the underlying issue or add an entry to .govulncheck-allow.yml with a reason and expires date.")
	}
	if len(expired) > 0 {
		fmt.Fprintf(os.Stderr, "govulncheck-filter: %d expired allowlist entr(ies):\n",
			len(expired))
		for _, e := range expired {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
		fmt.Fprintln(os.Stderr,
			"Re-evaluate: bump expires forward, fix the underlying issue, or remove the dependency.")
	}
	os.Exit(1)
}

func loadAllowlist(path string) (*allowlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a allowlist
	if err := yaml.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// readFindings extracts unique OSV IDs from a govulncheck JSON stream.
// Each finding object has an "osv" field naming the vulnerability;
// govulncheck emits multiple frames per OSV (one per call trace) — we
// dedup at the OSV level since the allowlist is OSV-keyed.
//
// Note: govulncheck's JSON output is a stream of pretty-printed
// objects (NOT JSON Lines), so we use json.Decoder which handles
// arbitrary whitespace between top-level values.
func readFindings(r io.Reader) ([]string, error) {
	dec := json.NewDecoder(r)
	seen := map[string]bool{}
	var ids []string
	for {
		var frame vulnFrame
		if err := dec.Decode(&frame); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if frame.Finding == nil {
			continue
		}
		// Module-level frames have empty Trace[0].Function — those are
		// false positives at the symbol level. Keep only frames whose
		// trace reaches a function (= reachable finding).
		if len(frame.Finding.Trace) == 0 || frame.Finding.Trace[0].Function == "" {
			continue
		}
		if !seen[frame.Finding.OSV] {
			seen[frame.Finding.OSV] = true
			ids = append(ids, frame.Finding.OSV)
		}
	}
	return ids, nil
}
