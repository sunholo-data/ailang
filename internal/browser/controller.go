package browser

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type SessionProvider interface {
	Name() string
	Create(context.Context, SessionSpec) (Session, error)
	Connection(context.Context, Session) (SensitiveConnection, error)
	Inspect(context.Context, Session) (InspectionRef, error)
	Export(context.Context, Session, string) (ArtifactManifest, error)
	Stop(context.Context, Session) (Usage, error)
}

type ControllerOptions struct {
	CleanupTimeout time.Duration
	Now            func() time.Time
}

type Controller struct {
	provider SessionProvider
	options  ControllerOptions
}

func NewController(provider SessionProvider, options ControllerOptions) *Controller {
	if options.CleanupTimeout <= 0 {
		options.CleanupTimeout = 30 * time.Second
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Controller{provider: provider, options: options}
}

type Run struct {
	controller *Controller
	spec       SessionSpec
	session    Session
	connection SensitiveConnection
	inspection InspectionRef
	startedAt  time.Time
	once       sync.Once
	manifest   BrowserRunManifest
	err        error
}

func (c *Controller) Start(ctx context.Context, spec SessionSpec) (*Run, error) {
	if c == nil || c.provider == nil {
		return nil, NewFailure(FailureProvision, "select provider", errors.New("nil provider"))
	}
	if spec.RunID == "" {
		return nil, NewFailure(FailurePolicyDenied, "validate run id", errors.New("empty run id"))
	}
	session, err := c.provider.Create(ctx, spec)
	if err != nil {
		return nil, classify(err, FailureProvision, "create")
	}
	connection, err := c.provider.Connection(ctx, session)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), c.options.CleanupTimeout)
		defer cancel()
		_, _ = c.provider.Stop(cleanupCtx, session)
		return nil, classify(err, FailureConnect, "connection")
	}
	inspection, inspectErr := c.provider.Inspect(ctx, session)
	if inspectErr != nil {
		inspection = InspectionRef{}
	}
	return &Run{
		controller: c,
		spec:       spec,
		session:    session,
		connection: connection,
		inspection: inspection,
		startedAt:  c.options.Now(),
	}, nil
}

func (r *Run) Connection() (MCPServerSpec, map[string]string) {
	if r == nil {
		return MCPServerSpec{}, nil
	}
	return r.connection.Materialize()
}

func (r *Run) Session() Session { return r.session }

func (r *Run) Finish(ctx context.Context, termination Termination) (BrowserRunManifest, error) {
	if r == nil || r.controller == nil {
		return BrowserRunManifest{}, NewFailure(FailureCleanup, "finish", errors.New("nil run"))
	}
	r.once.Do(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), r.controller.options.CleanupTimeout)
		defer cancel()
		artifacts, exportErr := r.controller.provider.Export(cleanupCtx, r.session, r.spec.ArtifactDir)
		usage, stopErr := r.controller.provider.Stop(cleanupCtx, r.session)
		r.manifest = BrowserRunManifest{
			RunID:             r.spec.RunID,
			ChainID:           r.spec.ChainID,
			StageID:           r.spec.StageID,
			Provider:          r.controller.provider.Name(),
			ProviderSessionID: r.session.ID,
			ToolSurface:       "playwright-mcp",
			Browser:           r.spec.Browser,
			BrowserVersion:    r.spec.BrowserVersion,
			MCPVersion:        r.spec.MCPVersion,
			PolicyVersion:     r.spec.PolicyVersion,
			ProfileHash:       r.spec.ProfileHash,
			ViewportWidth:     r.spec.ViewportWidth,
			ViewportHeight:    r.spec.ViewportHeight,
			Locale:            r.spec.Locale,
			Timezone:          r.spec.Timezone,
			Headless:          r.spec.Headless,
			StartedAt:         r.startedAt,
			EndedAt:           r.controller.options.Now(),
			Termination:       termination,
			Usage:             usage,
			Cost:              Cost{USD: nil, Currency: "USD", Source: "provider-unpriced"},
			Artifacts:         artifacts,
			Inspection:        r.inspection,
			Comparable:        !r.spec.HumanTakeover,
		}
		if r.spec.HumanTakeover {
			r.manifest.NonComparableReason = "human_takeover"
		}
		switch r.controller.provider.Name() {
		case "local-playwright":
			r.manifest.Cost.Source = "local-resource-unpriced"
		case "browserbase":
			r.manifest.Cost.Source = "browserbase-billing-not-joined"
		}
		if exportErr != nil {
			r.manifest.ArtifactErrorCategory = FailureArtifactExport
			r.err = classify(exportErr, FailureArtifactExport, "export")
		}
		if stopErr != nil {
			r.manifest.CleanupErrorCategory = FailureCleanup
			if r.err == nil {
				r.err = classify(stopErr, FailureCleanup, "stop")
			}
		}
		_ = ctx // cleanup deliberately outlives an already-cancelled run context.
	})
	return r.manifest, r.err
}

func classify(err error, fallback FailureCategory, op string) error {
	if err == nil {
		return nil
	}
	var failure *Failure
	if errors.As(err, &failure) {
		return failure
	}
	return NewFailure(fallback, op, err)
}

type Registry struct {
	mu        sync.RWMutex
	providers map[string]SessionProvider
}

func NewRegistry() *Registry { return &Registry{providers: make(map[string]SessionProvider)} }

func (r *Registry) Register(provider SessionProvider) error {
	if provider == nil || provider.Name() == "" {
		return fmt.Errorf("browser provider name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[provider.Name()]; exists {
		return fmt.Errorf("browser provider %q already registered", provider.Name())
	}
	r.providers[provider.Name()] = provider
	return nil
}

func (r *Registry) Get(name string) (SessionProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("browser provider %q is not registered", name)
	}
	return provider, nil
}
