package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/sunholo-data/ailang/internal/docsearch"
	"github.com/sunholo-data/ailang/std"
)

// stdlibFallbackMinScore is the minimum relevance score required for automatic
// offline stdlib fallback (without --stdlib). Requiring a module/symbol-name
// match (>= 0.76) lets `ailang docs search regex` or `ailang docs search split`
// succeed offline while ensuring non-stdlib design-doc queries (such as
// "needle" in confinement auth probes) still surface the GitHub auth error.
const stdlibFallbackMinScore = 0.76

type stdlibSearchEntry struct {
	path      string
	shortName string
	title     string
	searchTxt string
}

// searchEmbeddedStdlib searches the embedded std/*.ail modules, exported
// functions, exported types, and doc comments completely offline.
func searchEmbeddedStdlib(query string, limit int) ([]docsearch.SearchResult, docsearch.SearchStats, error) {
	start := time.Now()
	if limit <= 0 {
		limit = 10
	}

	modules := discoverModules(std.FS)
	entries := buildEmbeddedStdlibIndex(modules)

	qLower := strings.ToLower(strings.TrimSpace(query))
	tokens := splitSearchTokens(qLower)

	var hits []docsearch.SearchResult
	if qLower != "" {
		for _, e := range entries {
			score := scoreStdlibEntry(e, qLower, tokens)
			if score <= 0 {
				continue
			}
			hits = append(hits, docsearch.SearchResult{
				Path:  filepath.ToSlash(e.path),
				Title: e.title,
				Score: score,
			})
		}
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Path < hits[j].Path
	})

	if len(hits) > limit {
		hits = hits[:limit]
	}

	stats := docsearch.SearchStats{
		TotalDocs:         len(entries),
		SimHashCandidates: len(hits),
		SearchTimeMs:      time.Since(start).Milliseconds(),
	}
	return hits, stats, nil
}

func buildEmbeddedStdlibIndex(modules []moduleDoc) []stdlibSearchEntry {
	var entries []stdlibSearchEntry
	for _, mod := range modules {
		modPath := filepath.ToSlash(mod.Name + ".ail")
		shortMod := strings.TrimPrefix(mod.Name, "std/")
		header := strings.Join(mod.Header, " ")
		if header == "" {
			header = mod.Description
		}
		entries = append(entries, stdlibSearchEntry{
			path:      modPath,
			shortName: shortMod,
			title:     mod.Description,
			searchTxt: strings.ToLower(mod.Name + " " + shortMod + " " + mod.Description + " " + header),
		})

		astSigs, _, _ := parseExportSignatures(std.FS, mod.FileName)

		for _, t := range mod.Types {
			decl := strings.Join(t.Decl, " ")
			title := t.Name
			if t.DocLine != "" {
				title = t.Name + " -- " + t.DocLine
			}
			entries = append(entries, stdlibSearchEntry{
				path:      filepath.ToSlash(mod.Name + "." + t.Name),
				shortName: t.Name,
				title:     title,
				searchTxt: strings.ToLower(mod.Name + " " + t.Name + " " + decl + " " + t.DocLine),
			})
		}

		for _, exp := range mod.Exports {
			sig := astSigs[exp.Name]
			if sig == "" {
				sig = formatExportSignature(exp.Signature)
			}
			title := sig
			if exp.DocLine != "" {
				if title != "" {
					title += " -- " + exp.DocLine
				} else {
					title = exp.DocLine
				}
			}
			entries = append(entries, stdlibSearchEntry{
				path:      filepath.ToSlash(mod.Name + "." + exp.Name),
				shortName: exp.Name,
				title:     title,
				searchTxt: strings.ToLower(mod.Name + " " + exp.Name + " " + sig + " " + exp.DocLine),
			})
		}
	}
	return entries
}

func splitSearchTokens(s string) []string {
	raw := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	})
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, tok := range raw {
		if tok == "" || seen[tok] {
			continue
		}
		seen[tok] = true
		out = append(out, tok)
	}
	return out
}

func scoreStdlibEntry(e stdlibSearchEntry, qLower string, tokens []string) float64 {
	nameLower := strings.ToLower(e.shortName)
	pathLower := strings.ToLower(e.path)

	if nameLower == qLower || pathLower == qLower || strings.TrimSuffix(pathLower, ".ail") == qLower {
		if strings.HasSuffix(e.path, ".ail") {
			return 1.0
		}
		return 0.96
	}
	if strings.HasPrefix(nameLower, qLower) {
		return 0.88
	}
	if strings.Contains(nameLower, qLower) {
		return 0.80
	}
	if strings.Contains(pathLower, qLower) {
		return 0.76
	}
	if len(tokens) == 0 {
		return 0
	}

	matched := 0
	nameHits := 0
	for _, tok := range tokens {
		if strings.Contains(e.searchTxt, tok) {
			matched++
			if strings.Contains(nameLower, tok) || strings.Contains(pathLower, tok) {
				nameHits++
			}
		}
	}
	if matched == 0 {
		return 0
	}
	ratio := float64(matched) / float64(len(tokens))
	nameBoost := 0.15 * (float64(nameHits) / float64(len(tokens)))
	if matched == len(tokens) {
		if strings.Contains(e.searchTxt, qLower) {
			return 0.72 + nameBoost
		}
		return 0.60 + nameBoost
	}
	return 0.30*ratio + nameBoost
}

func printStdlibSearchResults(query string, results []docsearch.SearchResult, stats docsearch.SearchStats) {
	fmt.Printf("🔍 Embedded stdlib search: %q\n", query)
	fmt.Printf("   Indexed: %d stdlib entries\n\n", stats.TotalDocs)
	if len(results) == 0 {
		fmt.Println("No matching stdlib modules or exports found.")
		return
	}
	for i, r := range results {
		fmt.Printf("%d. %s (%.2f)\n", i+1, filepath.ToSlash(r.Path), r.Score)
		if r.Title != "" {
			fmt.Printf("   %s\n", r.Title)
		}
	}
}
