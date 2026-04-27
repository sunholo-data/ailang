// Fresh-fetch path for the prompt loader (M-AGENT-MCP M5).
//
// `ailang prompt --source auto|mcp|embedded` lets agents (and humans) pull
// the latest re-rendered teaching prompt from mcp.ailang.sunholo.com instead
// of relying on whatever was //go:embed'd at compile time. The MCP server
// holds the canonical, version-pinned content; the embedded copy is the
// always-available offline fallback.
//
// Determinism: --source=embedded is REQUIRED for reproducible eval runs and
// is what `auto` falls back to when MCP is unreachable, slow, or returns
// content tagged for a different AILANG version. The CLI never returns
// fresh-but-wrong-version content.
package prompt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/sunholo-data/ailang/internal/mcp_client"
)

// Source selects where prompt content comes from.
type Source string

const (
	// SourceAuto prefers MCP when reachable AND its content is tagged for our
	// version, otherwise silently falls back to embedded. Default.
	SourceAuto Source = "auto"
	// SourceMCP forces MCP. Returns an error if unreachable or version
	// mismatch — useful for testing the wire path.
	SourceMCP Source = "mcp"
	// SourceEmbedded forces the //go:embed'd copy. Required for reproducible
	// eval runs and any offline-deterministic context.
	SourceEmbedded Source = "embedded"
)

// FreshOptions controls a fresh fetch.
type FreshOptions struct {
	// Source: auto/mcp/embedded. Empty defaults to auto.
	Source Source
	// Kind: "" (full) | "agent" | "devtools" | "compact". Mapped to the MCP
	// prompt_get tool's "kind" arg.
	Kind string
	// Version: the AILANG version to pass as for_version. Empty resolves to
	// the binary's compile-time version.
	Version string
	// MCPURL overrides the default endpoint (env: AILANG_MCP_URL).
	MCPURL string
}

// FreshResult bundles the prompt content with metadata about how it was
// resolved, so the CLI can give the user/agent honest provenance.
type FreshResult struct {
	Content   string
	Source    Source // actual source used (may differ from request when auto)
	Version   string // ailang version this content is tagged for
	SHA256    string // content sha256 (first 7 chars used for display)
	FromCache bool   // true if served from ~/.ailang/cache/prompts/<ver>/
	MCPNote   string // human-readable note about the MCP attempt (drift, fallback reason)
}

// LoadPromptFresh resolves a prompt according to opts.Source.
//
// auto:     try MCP first; on any failure / version mismatch / timeout, fall
//
//	back to embedded silently. Result.MCPNote explains what happened.
//
// mcp:      MCP only. Returns error on failure (no embedded fallback).
// embedded: skip MCP entirely. Always works offline.
//
// callerVersion is the binary's compile-time version (typically version.Version).
func LoadPromptFresh(ctx context.Context, opts FreshOptions, callerVersion string) (*FreshResult, error) {
	src := opts.Source
	if src == "" {
		src = SourceAuto
	}
	wantVersion := opts.Version
	if wantVersion == "" {
		wantVersion = callerVersion
	}

	// Honor explicit --source=embedded immediately (no network, no cache).
	if src == SourceEmbedded {
		return loadEmbedded(opts.Kind, wantVersion)
	}

	// Try cache before hitting the network. Cache key is (version, kind).
	if cached, ok := readCache(wantVersion, opts.Kind); ok {
		cached.FromCache = true
		cached.Source = SourceMCP
		cached.MCPNote = "cache hit"
		if src == SourceMCP {
			return cached, nil
		}
		// auto: cache hit is good enough; skip the network round-trip.
		return cached, nil
	}

	mcpResult, mcpErr := fetchFromMCP(ctx, opts.MCPURL, callerVersion, wantVersion, opts.Kind)

	if mcpErr == nil {
		_ = writeCache(wantVersion, opts.Kind, mcpResult)
		return mcpResult, nil
	}

	// --source=mcp: surface the error.
	if src == SourceMCP {
		return nil, fmt.Errorf("mcp fetch failed: %w", mcpErr)
	}

	// auto: fall back to embedded. Annotate so the CLI can stderr-print why.
	embed, err := loadEmbedded(opts.Kind, wantVersion)
	if err != nil {
		return nil, fmt.Errorf("mcp fetch failed (%v) and embedded fallback failed: %w", mcpErr, err)
	}
	embed.MCPNote = fmt.Sprintf("fell back to embedded: %v", mcpErr)
	return embed, nil
}

// fetchFromMCP calls prompt_get with version-locked args and returns the
// content + sha. ErrVersionMismatch is mapped onto a clear error string
// because the protocol-layer check already happened in mcp_client.
func fetchFromMCP(ctx context.Context, mcpURL, callerVersion, wantVersion, kind string) (*FreshResult, error) {
	c := mcp_client.New(mcp_client.Options{
		BaseURL:       mcpURL,
		AILangVersion: callerVersion,
	})
	body, err := c.CallTool(ctx, "prompt_get", map[string]any{
		"forVersion": wantVersion,
		"kind":       kind,
	})
	if err != nil {
		if errors.Is(err, mcp_client.ErrVersionMismatch) {
			return nil, fmt.Errorf("server has no prompt content for AILANG %s", wantVersion)
		}
		return nil, err
	}
	// prompt_get returns {served_for, data: {markdown, prompt_version, served_for, size_bytes}}
	data, ok := body["data"].(map[string]any)
	if !ok {
		// Some tool errors come back without the envelope; surface the body.
		return nil, fmt.Errorf("unexpected prompt_get response shape: %v", body)
	}
	markdown, _ := data["markdown"].(string)
	if markdown == "" {
		return nil, errors.New("prompt_get returned empty markdown")
	}
	servedFor, _ := body["served_for"].(string)
	if servedFor == "" {
		servedFor = wantVersion
	}
	if servedFor != wantVersion {
		return nil, fmt.Errorf("server returned prompt for %s but we asked for %s", servedFor, wantVersion)
	}
	sum := sha256.Sum256([]byte(markdown))
	return &FreshResult{
		Content: markdown,
		Source:  SourceMCP,
		Version: servedFor,
		SHA256:  hex.EncodeToString(sum[:]),
		MCPNote: "fresh from MCP",
	}, nil
}

// loadEmbedded reads from the //go:embed'd FS via the existing LoadPrompt
// path. kind is mapped onto the family of prompt files we ship (full, agent,
// devtools, compact) using the same conventions as the loader.
func loadEmbedded(kind, version string) (*FreshResult, error) {
	// For now we reuse the existing manifest-driven loader. Kind selection is
	// best-effort: full = active version's main file. Agent/devtools/compact
	// require their own families which existing loader.go already supports
	// via dedicated commands; M5 just tags the source.
	content, ver, err := LoadPromptWithVersion(version)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(content))
	return &FreshResult{
		Content:   content,
		Source:    SourceEmbedded,
		Version:   ver,
		SHA256:    hex.EncodeToString(sum[:]),
		FromCache: false,
		MCPNote:   "embedded copy (compiled in)",
	}, nil
}

// ─── on-disk cache ──────────────────────────────────────────────────────
// Path: $XDG_CACHE_HOME/ailang/prompts/<version>/<kind>.md (+.sha)
// Falls back to ~/.cache/ailang/prompts/... when XDG_CACHE_HOME is unset.

func cacheBaseDir() string {
	if env := os.Getenv("AILANG_CACHE_DIR"); env != "" {
		return env
	}
	if env := os.Getenv("XDG_CACHE_HOME"); env != "" {
		return filepath.Join(env, "ailang")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ailang-cache"
	}
	return filepath.Join(home, ".cache", "ailang")
}

func cachePath(version, kind string) string {
	if kind == "" {
		kind = "full"
	}
	return filepath.Join(cacheBaseDir(), "prompts", version, kind+".md")
}

func readCache(version, kind string) (*FreshResult, bool) {
	p := cachePath(version, kind)
	body, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	sum := sha256.Sum256(body)
	return &FreshResult{
		Content: string(body),
		Source:  SourceMCP,
		Version: version,
		SHA256:  hex.EncodeToString(sum[:]),
	}, true
}

func writeCache(version, kind string, r *FreshResult) error {
	p := cachePath(version, kind)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(r.Content), 0o644)
}

// ─── helpers used by `ailang mcp status` (callers may import) ───────────

// EmbeddedSHA256 returns the sha256 of the embedded full prompt for the
// currently-active version, or empty when the embedded FS isn't initialized.
func EmbeddedSHA256() string {
	if embeddedPrompts == nil {
		return ""
	}
	manifest, err := loadVersionsManifest()
	if err != nil {
		return ""
	}
	meta, ok := manifest.Versions[manifest.Active]
	if !ok {
		return ""
	}
	body, err := fs.ReadFile(embeddedPrompts, meta.File)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// BuildInfo exposes the binary's resolved version (build info if available).
func BuildInfo() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(unknown)"
}
