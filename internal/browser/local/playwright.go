// Package local implements isolated local Playwright MCP browser sessions.
package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/browser"
)

const (
	ProviderName      = "local-playwright"
	DefaultMCPVersion = "0.0.79"
	// DefaultBrowserVersion is the exact Playwright dependency declared by
	// @playwright/mcp@0.0.79; Playwright resolves its bundled Chromium revision.
	DefaultBrowserVersion = "playwright@1.63.0-alpha-2026-08-05/bundled-chromium"
)

type Config struct {
	BaseDir    string
	NpxPath    string
	MCPVersion string
	LookPath   func(string) (string, error)
	Now        func() time.Time
}

type Provider struct {
	baseDir    string
	npxPath    string
	mcpVersion string
	now        func() time.Time
	mu         sync.Mutex
	started    map[string]time.Time
	stopped    map[string]bool
	specs      map[string]browser.SessionSpec
}

func New(config Config) (*Provider, error) {
	if config.LookPath == nil {
		config.LookPath = exec.LookPath
	}
	if config.NpxPath == "" {
		path, err := config.LookPath("npx")
		if err != nil {
			return nil, browser.NewFailure(browser.FailureProvision, "find npx", err)
		}
		config.NpxPath = path
	}
	if config.MCPVersion == "" {
		config.MCPVersion = DefaultMCPVersion
	}
	if config.BaseDir == "" {
		config.BaseDir = filepath.Join(os.TempDir(), "ailang-browser")
	}
	if err := os.MkdirAll(config.BaseDir, 0700); err != nil {
		return nil, browser.NewFailure(browser.FailureProvision, "create browser root", err)
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Provider{
		baseDir:    config.BaseDir,
		npxPath:    config.NpxPath,
		mcpVersion: config.MCPVersion,
		now:        config.Now,
		started:    make(map[string]time.Time),
		stopped:    make(map[string]bool),
		specs:      make(map[string]browser.SessionSpec),
	}, nil
}

func (p *Provider) Name() string { return ProviderName }

func (p *Provider) Create(ctx context.Context, spec browser.SessionSpec) (browser.Session, error) {
	if err := ctx.Err(); err != nil {
		return browser.Session{}, browser.NewFailure(browser.FailureProvision, "create local session", err)
	}
	id := uuid.NewString()
	stateDir, err := os.MkdirTemp(p.baseDir, "state-"+id[:8]+"-")
	if err != nil {
		return browser.Session{}, browser.NewFailure(browser.FailureProvision, "create state directory", err)
	}
	artifactBase := p.baseDir
	if spec.ArtifactDir != "" {
		artifactBase = spec.ArtifactDir
		if err := os.MkdirAll(artifactBase, 0700); err != nil {
			_ = os.RemoveAll(stateDir)
			return browser.Session{}, browser.NewFailure(browser.FailureProvision, "create artifact root", err)
		}
	}
	artifactDir, err := os.MkdirTemp(artifactBase, "artifacts-"+id[:8]+"-")
	if err != nil {
		_ = os.RemoveAll(stateDir)
		return browser.Session{}, browser.NewFailure(browser.FailureProvision, "create artifact directory", err)
	}
	started := p.now()
	p.mu.Lock()
	p.started[id] = started
	p.specs[id] = spec
	p.mu.Unlock()
	return browser.Session{
		ID: id, Provider: p.Name(), CreatedAt: started,
		StateDir: stateDir, ArtifactDir: artifactDir,
	}, nil
}

func (p *Provider) Connection(ctx context.Context, session browser.Session) (browser.SensitiveConnection, error) {
	if err := ctx.Err(); err != nil {
		return browser.SensitiveConnection{}, browser.NewFailure(browser.FailureConnect, "build local connection", err)
	}
	if session.Provider != p.Name() || session.ID == "" {
		return browser.SensitiveConnection{}, browser.NewFailure(browser.FailureConnect, "validate local session", fmt.Errorf("invalid session"))
	}
	args := []string{"-y", "@playwright/mcp@" + p.mcpVersion}
	spec := p.spec(session)
	if spec.Headless {
		args = append(args, "--headless")
	}
	args = append(args, "--isolated", "--output-dir", session.ArtifactDir, "--save-session")
	viewportWidth, viewportHeight := 1280, 720
	if width, height := p.viewport(session); width > 0 && height > 0 {
		viewportWidth, viewportHeight = width, height
	}
	args = append(args, "--viewport-size", fmt.Sprintf("%dx%d", viewportWidth, viewportHeight))
	if actionTimeout := p.actionTimeout(session); actionTimeout > 0 {
		args = append(args, "--timeout-action", strconv.FormatInt(actionTimeout.Milliseconds(), 10))
	}
	return browser.NewSensitiveConnection(browser.MCPServerSpec{
		Name: "playwright", Command: p.npxPath, Args: args, Required: true,
	}, nil), nil
}

func (p *Provider) viewport(session browser.Session) (int, int) {
	spec := p.spec(session)
	return spec.ViewportWidth, spec.ViewportHeight
}

func (p *Provider) actionTimeout(session browser.Session) time.Duration {
	return p.spec(session).ActionTimeout
}

func (p *Provider) spec(session browser.Session) browser.SessionSpec {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.specs[session.ID]
}

func (p *Provider) Inspect(context.Context, browser.Session) (browser.InspectionRef, error) {
	return browser.InspectionRef{Available: false}, nil
}

func (p *Provider) Export(ctx context.Context, session browser.Session, _ string) (browser.ArtifactManifest, error) {
	refs := make([]browser.ArtifactRef, 0)
	err := filepath.WalkDir(session.ArtifactDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		digest, err := hashFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(session.ArtifactDir, path)
		if err != nil {
			return err
		}
		refs = append(refs, browser.ArtifactRef{Kind: artifactKind(path), Path: relative, SHA256: digest})
		return nil
	})
	if err != nil {
		return browser.ArtifactManifest{}, browser.NewFailure(browser.FailureArtifactExport, "index local artifacts", err)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	return browser.ArtifactManifest{Complete: true, Refs: refs}, nil
}

func (p *Provider) Stop(_ context.Context, session browser.Session) (browser.Usage, error) {
	p.mu.Lock()
	if p.stopped[session.ID] {
		started := p.started[session.ID]
		p.mu.Unlock()
		return browser.Usage{DurationMS: p.now().Sub(started).Milliseconds()}, nil
	}
	p.stopped[session.ID] = true
	started := p.started[session.ID]
	delete(p.specs, session.ID)
	p.mu.Unlock()
	if err := os.RemoveAll(session.StateDir); err != nil {
		return browser.Usage{}, browser.NewFailure(browser.FailureCleanup, "remove local state", err)
	}
	return browser.Usage{DurationMS: p.now().Sub(started).Milliseconds()}, nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func artifactKind(path string) string {
	switch filepath.Ext(path) {
	case ".webm":
		return "video"
	case ".zip":
		return "trace"
	case ".png", ".jpg", ".jpeg":
		return "screenshot"
	case ".har":
		return "har"
	default:
		return "playwright-output"
	}
}
