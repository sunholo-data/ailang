package config

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// The ONE storage plane switch and its per-store overrides
// (M-V1-SIMPLIFY-S3 M3, program §2.5). Nothing else reads these names.
const (
	// EnvStorage selects the plane every store follows unless overridden.
	EnvStorage = "AILANG_STORAGE"
	// EnvStorageMessaging, EnvStorageCoordinator and EnvStorageObservatory
	// override ONE store. Each takes local or gcp.
	EnvStorageMessaging   = "AILANG_STORAGE_MESSAGING"
	EnvStorageCoordinator = "AILANG_STORAGE_COORDINATOR"
	EnvStorageObservatory = "AILANG_STORAGE_OBSERVATORY"
	// EnvCoordinatorMode is the coordinator daemon's execution mode. It is
	// validated against the plane here — see CoordinatorMode.
	EnvCoordinatorMode = "COORDINATOR_MODE"
)

// Plane is the value of AILANG_STORAGE.
type Plane string

// The planes.
const (
	// PlaneLocal keeps every store in SQLite under statedir.Dir().
	PlaneLocal Plane = "local"
	// PlaneGCP keeps every store in Firestore in the cloud project.
	PlaneGCP Plane = "gcp"
	// PlaneHybrid keeps every store in SQLite but joins the SHARED plane: the
	// cloud project is required, the coordinator publishes approvals to
	// Pub/Sub and the secret approver is attached. (The observatory was meant
	// to go to BigQuery under hybrid; it never did, and this package reports
	// what is opened, not what was planned.)
	PlaneHybrid Plane = "hybrid"
)

// StoreMode is where ONE store lives: local (SQLite) or gcp (Firestore).
// hybrid is a plane value, never a per-store value.
type StoreMode string

// The per-store modes.
const (
	StoreLocal StoreMode = "local"
	StoreGCP   StoreMode = "gcp"
)

// StoreName identifies one of the three stores.
type StoreName string

// The three stores.
const (
	StoreMessaging   StoreName = "messaging"
	StoreCoordinator StoreName = "coordinator"
	StoreObservatory StoreName = "observatory"
)

// SourceDefault is the Source of a value nothing set.
const SourceDefault Source = "default"

// StoreSelection is one store's resolved mode and where it came from.
type StoreSelection struct {
	Store  StoreName
	Mode   StoreMode
	Source Source // the override var, AILANG_STORAGE, or SourceDefault
}

// Storage is the fully resolved plane: the plane itself and each store.
type Storage struct {
	Plane       Plane
	PlaneSource Source
	Messaging   StoreSelection
	Coordinator StoreSelection
	Observatory StoreSelection
}

// Shared reports whether this process joins the shared cloud plane — gcp or
// hybrid — which is what requires a cloud project, a Pub/Sub publisher on the
// coordinator and the secret approver. A local plane with a per-store gcp
// override is NOT shared: only that store moved.
func (s Storage) Shared() bool { return s.Plane != PlaneLocal }

// AnyGCP reports whether at least one store is in Firestore, i.e. whether a
// Firestore client (and so a cloud project) is needed to open the backends.
func (s Storage) AnyGCP() bool {
	return s.Messaging.Mode == StoreGCP || s.Coordinator.Mode == StoreGCP || s.Observatory.Mode == StoreGCP
}

// Select returns the selection for one store.
func (s Storage) Select(name StoreName) StoreSelection {
	switch name {
	case StoreMessaging:
		return s.Messaging
	case StoreCoordinator:
		return s.Coordinator
	default:
		return s.Observatory
	}
}

// ErrRemovedEnv is wrapped by the error StoragePlane returns when one of the
// retired selectors is set. Match it with errors.Is.
var ErrRemovedEnv = errors.New("removed environment variable")

// ErrBadPlane is wrapped when AILANG_STORAGE or an override holds a value
// that is not one of its valid values.
var ErrBadPlane = errors.New("invalid storage selection")

// removedEnv maps each selector retired by M-V1-SIMPLIFY-S3 M3 to the
// replacement its error names. They are hard errors for one release: the old
// names used to disagree silently with AILANG_STORAGE (`ailang storage
// status` read only the plane var), so a value that is now ignored must not
// LOOK honoured.
var removedEnv = map[string]string{
	"AILANG_MESSAGES_STORE":     EnvStorageMessaging + "=gcp",
	"AILANG_COORDINATOR_REMOTE": EnvStorageCoordinator + "=gcp",
	"AILANG_CHAINS_READ":        EnvStorageObservatory + "=gcp",
	"AILANG_CHAINS_CLOUD":       EnvStorageObservatory + "=gcp",
}

// RemovedEnvNames lists the retired selector names, for documentation and
// for the metric that counts backend switches.
func RemovedEnvNames() []string {
	names := make([]string, 0, len(removedEnv))
	for n := range removedEnv {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// checkRemovedEnv scans the environment for a retired selector. It reads
// os.Environ rather than each name, because a retired name is a REJECTION,
// not a configuration route, and must not count as one.
func checkRemovedEnv() error {
	for _, kv := range os.Environ() {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || value == "" {
			continue
		}
		if repl, removed := removedEnv[name]; removed {
			return fmt.Errorf("%w: %s was removed in v1.0.0 (M-V1-SIMPLIFY-S3); set %s (it is %s=%s here). "+
				"The one plane switch is %s=local|gcp|hybrid with per-store overrides %s, %s, %s",
				ErrRemovedEnv, name, repl, name, value,
				EnvStorage, EnvStorageMessaging, EnvStorageCoordinator, EnvStorageObservatory)
		}
	}
	return nil
}

// StoragePlane resolves the plane and every store from the environment. It
// is the ONLY reader of AILANG_STORAGE and the three overrides; every
// backend constructor and every command that names a store calls it.
//
// Resolution:
//
//   - AILANG_STORAGE=local|gcp|hybrid, default local. local and hybrid put
//     every store in SQLite; gcp puts every store in Firestore.
//   - AILANG_STORAGE_MESSAGING / _COORDINATOR / _OBSERVATORY=local|gcp move
//     ONE store, and are reported with that variable as the Source.
//   - Any retired selector that is set is an error naming the replacement.
//   - An unknown value anywhere is an error naming the variable.
func StoragePlane() (Storage, error) {
	if err := checkRemovedEnv(); err != nil {
		return Storage{}, err
	}
	plane, planeSrc, err := readPlane()
	if err != nil {
		return Storage{}, err
	}
	s := StorageForPlane(plane)
	s.PlaneSource = planeSrc
	s.Messaging.Source, s.Coordinator.Source, s.Observatory.Source = planeSrc, planeSrc, planeSrc
	for _, o := range []struct {
		env string
		sel *StoreSelection
	}{
		{EnvStorageMessaging, &s.Messaging},
		{EnvStorageCoordinator, &s.Coordinator},
		{EnvStorageObservatory, &s.Observatory},
	} {
		v := strings.TrimSpace(os.Getenv(o.env))
		if v == "" {
			continue
		}
		switch StoreMode(v) {
		case StoreLocal, StoreGCP:
			o.sel.Mode = StoreMode(v)
			o.sel.Source = Source(o.env)
		case StoreMode(PlaneHybrid):
			return Storage{}, fmt.Errorf("%w: %s=hybrid; hybrid is a plane value for %s, a per-store override takes local or gcp",
				ErrBadPlane, o.env, EnvStorage)
		default:
			return Storage{}, fmt.Errorf("%w: %s=%q (valid: local, gcp)", ErrBadPlane, o.env, v)
		}
	}
	return s, nil
}

// StorageForPlane is the resolution for an EXPLICIT plane with no overrides —
// what a caller that names a plane itself (`--remote gcp`, the mission loop's
// second observatory) gets. It reads nothing from the environment.
func StorageForPlane(plane Plane) Storage {
	mode := StoreLocal
	if plane == PlaneGCP {
		mode = StoreGCP
	}
	src := Source(EnvStorage)
	return Storage{
		Plane:       plane,
		PlaneSource: src,
		Messaging:   StoreSelection{Store: StoreMessaging, Mode: mode, Source: src},
		Coordinator: StoreSelection{Store: StoreCoordinator, Mode: mode, Source: src},
		Observatory: StoreSelection{Store: StoreObservatory, Mode: mode, Source: src},
	}
}

// ParsePlane validates a plane value given on a command line.
func ParsePlane(v string) (Plane, error) {
	switch Plane(strings.TrimSpace(v)) {
	case PlaneLocal, "":
		return PlaneLocal, nil
	case PlaneGCP:
		return PlaneGCP, nil
	case PlaneHybrid:
		return PlaneHybrid, nil
	default:
		return "", fmt.Errorf("%w: %q (valid: local, gcp, hybrid)", ErrBadPlane, v)
	}
}

func readPlane() (Plane, Source, error) {
	v := strings.TrimSpace(os.Getenv(EnvStorage))
	if v == "" {
		return PlaneLocal, SourceDefault, nil
	}
	p, err := ParsePlane(v)
	if err != nil {
		return "", "", fmt.Errorf("%s=%q: %w", EnvStorage, v, err)
	}
	return p, Source(EnvStorage), nil
}

// Coordinator execution modes. The daemon runs work either on THIS host, in
// git worktrees, polling its stores (local) — or as Cloud Run Jobs, fed by
// Pub/Sub push and broadcasting to Pub/Sub (cloud).
const (
	CoordinatorModeLocal = "local"
	CoordinatorModeCloud = "cloud"
)

// ErrCoordinatorModeDisagrees is wrapped when COORDINATOR_MODE contradicts
// the plane.
var ErrCoordinatorModeDisagrees = errors.New(EnvCoordinatorMode + " contradicts the storage plane")

// CoordinatorMode resolves the coordinator daemon's execution mode and
// validates it against the plane.
//
// The mode is NOT derived from the plane, because the plane does not
// determine it: the rig's coordinator runs AILANG_STORAGE=gcp with no
// COORDINATOR_MODE — a local-execution daemon (worktrees on this host) on the
// shared Firestore plane — while the Cloud Run coordinator runs the same
// plane with COORDINATOR_MODE=cloud. Deriving cloud from gcp would flip the
// rig's daemon into Cloud Run Jobs dispatch.
//
// What IS enforced is the disagreement that used to be silent: cloud mode
// with a coordinator or messaging store in SQLite would run Pub/Sub push
// intake against a database no Cloud Run Job can see. That is an error
// naming what to set. Unset means local.
func CoordinatorMode() (string, Source, error) {
	s, err := StoragePlane()
	if err != nil {
		return "", "", err
	}
	v := strings.TrimSpace(os.Getenv(EnvCoordinatorMode))
	switch v {
	case "":
		return CoordinatorModeLocal, SourceDefault, nil
	case CoordinatorModeLocal:
		return CoordinatorModeLocal, Source(EnvCoordinatorMode), nil
	case CoordinatorModeCloud:
		if s.Coordinator.Mode != StoreGCP || s.Messaging.Mode != StoreGCP {
			return "", "", fmt.Errorf("%w: %s=cloud needs the coordinator and messaging stores in Firestore, but coordinator is %s (%s) and messaging is %s (%s); set %s=gcp",
				ErrCoordinatorModeDisagrees, EnvCoordinatorMode,
				s.Coordinator.Mode, s.Coordinator.Source, s.Messaging.Mode, s.Messaging.Source, EnvStorage)
		}
		return CoordinatorModeCloud, Source(EnvCoordinatorMode), nil
	default:
		return "", "", fmt.Errorf("%w: %s=%q (valid: local, cloud)", ErrBadPlane, EnvCoordinatorMode, v)
	}
}
