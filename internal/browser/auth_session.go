package browser

import (
	"context"
	"errors"

	"github.com/sunholo-data/ailang/internal/browser/auth"
)

// AuthAttachment is the provider-neutral description of a leased authenticated
// identity. It is passed to the provider adapter, never to the model.
//
// StorageStatePath is a path, not contents: the path is safe to place in a child
// process's argv, the file it points at is not safe to log. Material carries the
// hosted-provider reference for backends that keep state server-side.
type AuthAttachment struct {
	Material         auth.SensitiveProfileMaterial
	Mode             auth.LeaseMode
	Persist          bool
	StorageStatePath string
}

// AuthenticatedProvider is the optional capability a provider implements when it
// can start a session from a leased profile. Providers that do not implement it
// are refused rather than silently downgraded to an unauthenticated session —
// a run that quietly starts logged out looks like a benchmark failure, not a
// configuration error.
type AuthenticatedProvider interface {
	SessionProvider
	CreateAuthenticated(ctx context.Context, spec SessionSpec, attachment AuthAttachment) (Session, error)
}

// AuthConfig attaches a profile to one run.
type AuthConfig struct {
	Broker  *auth.Broker
	Request auth.ProvisionRequest
}

// StartAuthenticated provisions a leased profile and starts a session already
// logged in.
//
// The ordering is the security property: the run id is validated, the profile is
// resolved and preflighted, and only then is a browser created. Everything that
// can refuse a run happens while there is still nothing to clean up.
//
// On any failure after provisioning, the profile is torn down before returning,
// because the caller has no handle to tear down with.
func (c *Controller) StartAuthenticated(ctx context.Context, spec SessionSpec, config AuthConfig) (*Run, error) {
	if c == nil || c.provider == nil {
		return nil, NewFailure(FailureProvision, "select provider", errors.New("nil provider"))
	}
	if config.Broker == nil {
		return nil, NewFailure(FailurePolicyDenied, "validate auth config", errors.New("no auth profile broker"))
	}
	if spec.RunID == "" {
		return nil, NewFailure(FailurePolicyDenied, "validate run id", errors.New("empty run id"))
	}

	provisioned, err := config.Broker.Provision(ctx, config.Request)
	if err != nil {
		return nil, err
	}

	authProvider, ok := c.provider.(AuthenticatedProvider)
	if !ok {
		_ = config.Broker.Teardown(ctx, provisioned)
		return nil, auth.NewFailureReason(auth.FailureScopeDenied, "start authenticated",
			"provider_cannot_authenticate: "+c.provider.Name())
	}

	// Stamp the RESOLVED identity onto the spec. "latest" never survives this
	// point, so the result records what actually ran.
	spec.ProfileRef = provisioned.Profile.Ref().String()
	spec.ProfileHash = provisioned.Profile.ProfileHash
	spec.AuthLeaseID = provisioned.Lease.SafeID
	spec.AuthMode = string(provisioned.Lease.Mode)

	attachment := AuthAttachment{
		Material:         provisioned.Material,
		Mode:             provisioned.Lease.Mode,
		Persist:          config.Request.RequestPersistence,
		StorageStatePath: provisioned.StorageStatePath,
	}

	session, err := authProvider.CreateAuthenticated(ctx, spec, attachment)
	if err != nil {
		_ = config.Broker.Teardown(ctx, provisioned)
		return nil, classify(err, FailureProvision, "create authenticated")
	}

	connection, err := c.provider.Connection(ctx, session)
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), c.options.CleanupTimeout)
		_, _ = c.provider.Stop(cleanupCtx, session)
		cancel()
		_ = config.Broker.Teardown(ctx, provisioned)
		return nil, classify(err, FailureConnect, "connection")
	}

	inspection, inspectErr := c.provider.Inspect(ctx, session)
	if inspectErr != nil {
		inspection = InspectionRef{}
	}

	return &Run{
		controller:      c,
		spec:            spec,
		session:         session,
		connection:      connection,
		inspection:      inspection,
		startedAt:       c.options.Now(),
		authBroker:      config.Broker,
		authProvisioned: provisioned,
	}, nil
}

// AuthRunOutcome separates the run's own failure from cleanup's, for the same
// reason auth.RunOutcome does: a cleanup failure must never mask why the run
// actually failed.
type AuthRunOutcome struct {
	Manifest   BrowserRunManifest
	Err        error
	CleanupErr error
}

// Primary returns the failure that should drive the result.
func (o AuthRunOutcome) Primary() error {
	if o.Err != nil {
		return o.Err
	}
	return o.CleanupErr
}

// Use runs fn and finishes on EVERY exit path — success, error, cancellation,
// deadline, and panic. A panic still tears the profile down and then continues
// unwinding: swallowing it would hide the failure that caused it.
func (r *Run) Use(ctx context.Context, termination Termination, fn func(context.Context) error) (outcome AuthRunOutcome) {
	defer func() {
		manifest, finishErr := r.Finish(context.Background(), termination)
		outcome.Manifest = manifest
		if finishErr != nil {
			outcome.CleanupErr = finishErr
		}
	}()
	if fn == nil {
		return outcome
	}
	outcome.Err = fn(ctx)
	return outcome
}

// teardownAuth releases the lease and destroys any decrypted state. It is called
// from Finish inside the once, so it runs exactly once per run.
func (r *Run) teardownAuth() error {
	if r.authBroker == nil || r.authProvisioned == nil {
		return nil
	}
	// Cleanup deliberately uses a fresh context: an already-cancelled run context
	// must not prevent the plaintext from being destroyed.
	cleanupCtx, cancel := context.WithTimeout(context.Background(), r.controller.options.CleanupTimeout)
	defer cancel()
	return r.authBroker.Teardown(cleanupCtx, r.authProvisioned)
}

// stampAuthIdentity copies the safe profile identity onto the manifest.
func (r *Run) stampAuthIdentity(manifest *BrowserRunManifest) {
	if r.authProvisioned == nil {
		return
	}
	manifest.AuthProfileAlias = r.authProvisioned.Profile.Alias
	manifest.AuthProfileVersion = r.authProvisioned.Profile.Version
	manifest.AuthLeaseID = r.authProvisioned.Lease.SafeID
	manifest.AuthMode = string(r.authProvisioned.Lease.Mode)
}
