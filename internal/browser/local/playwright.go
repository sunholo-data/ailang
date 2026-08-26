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
	"github.com/sunholo-data/ailang/internal/browser/auth"
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
	// storageState maps session ID to a materialized storage-state path. Absent
	// means an unauthenticated session, which is the default.
	storageState map[string]string
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
		baseDir:      config.BaseDir,
		npxPath:      config.NpxPath,
		mcpVersion:   config.MCPVersion,
		now:          config.Now,
		started:      make(map[string]time.Time),
		stopped:      make(map[string]bool),
		specs:        make(map[string]browser.SessionSpec),
		storageState: make(map[string]string),
	}, nil
}

func (p *Provider) Name() string { return ProviderName }

// StorageStateAttachment carries an already-materialized Playwright
// storage-state file into a session.
//
// It holds a PATH, never the state itself: the file is owned by the auth
// broker's disposable materialization, which is responsible for destroying it.
// The provider only reads the path to build argv and never opens the file, so
// it cannot leak the contents into a connection, a manifest, or a log line.
//
// This is a provider-local type rather than a field on browser.SessionSpec
// because SessionSpec crosses the serialization boundary and a filesystem path
// to decrypted credentials should not travel with it.
type StorageStateAttachment struct {
	// Path is the materialized storage-state file. It must be non-empty and
	// must already exist.
	Path string
}

// CreateWithStorageState creates a session that starts from an authenticated
// storage state. Ordinary unauthenticated sessions use Create.
//
// Write-back is deliberately not implemented here: an ordinary run must not be
// able to publish a new canonical version, so the state file is an input only.
func (p *Provider) CreateWithStorageState(ctx context.Context, spec browser.SessionSpec, attachment StorageStateAttachment) (browser.Session, error) {
	if attachment.Path == "" {
		return browser.Session{}, browser.NewFailure(browser.FailureProvision, "attach storage state", fmt.Errorf("empty path"))
	}
	info, err := os.Stat(attachment.Path)
	if err != nil {
		// Failing here rather than at browser launch keeps a missing or already
		// destroyed materialization from looking like an unauthenticated run
		// that merely failed to log in.
		return browser.Session{}, browser.NewFailure(browser.FailureProvision, "attach storage state", err)
	}
	if info.IsDir() {
		return browser.Session{}, browser.NewFailure(browser.FailureProvision, "attach storage state", fmt.Errorf("path is a directory"))
	}

	session, err := p.Create(ctx, spec)
	if err != nil {
		return browser.Session{}, err
	}
	p.mu.Lock()
	p.storageState[session.ID] = attachment.Path
	p.mu.Unlock()
	return session, nil
}

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
	args = append(args, "--isolated")
	// --storage-state seeds the isolated context from the disposable file. It
	// travels next to --isolated so the pairing is obvious in a process listing:
	// authenticated state is loaded INTO a throwaway profile, never persisted
	// back into one. Only the path is passed; Playwright reads the file itself.
	if statePath := p.storageStatePath(session); statePath != "" {
		args = append(args, "--storage-state", statePath)
	}
	args = append(args, "--output-dir", session.ArtifactDir, "--save-session")
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

func (p *Provider) storageStatePath(session browser.Session) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.storageState[session.ID]
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
	// Drop the reference only. The materialized file belongs to the auth
	// broker's Materialization handle, which destroys it on every exit path;
	// deleting it from here would be a second owner racing the first.
	delete(p.storageState, session.ID)
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

// CreateAuthenticated implements browser.AuthenticatedProvider by adapting the
// neutral attachment to this provider's storage-state input.
//
// A hosted-context attachment is refused rather than ignored: silently starting
// an unauthenticated session would look like a failed login in the results.
func (p *Provider) CreateAuthenticated(ctx context.Context, spec browser.SessionSpec, attachment browser.AuthAttachment) (browser.Session, error) {
	if attachment.StorageStatePath == "" {
		return browser.Session{}, auth.NewFailureReason(auth.FailureMaterializeFailed,
			"attach storage state", "no_storage_state_path")
	}
	if attachment.Persist {
		// Local profiles have no write-back path at all; only the trusted
		// refresh workflow publishes a new version, and it does not run here.
		return browser.Session{}, auth.NewFailureReason(auth.FailureWritebackDenied,
			"attach storage state", "local_provider_never_persists")
	}
	return p.CreateWithStorageState(ctx, spec, StorageStateAttachment{Path: attachment.StorageStatePath})
}

var _ browser.AuthenticatedProvider = (*Provider)(nil)
