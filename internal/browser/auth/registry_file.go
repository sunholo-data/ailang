package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FileRegistry persists profile versions on the local filesystem.
//
// Layout, under a 0700 root:
//
//	<root>/<alias>/<version>.meta.json   safe metadata (0600)
//	<root>/<alias>/<version>.material    credential-grade material (0600)
//
// The material file holds whatever the caller published. For local profiles the
// broker has already sealed it with the deployment's key protector, so the bytes
// on disk are an AES-256-GCM envelope and this registry never sees plaintext.
// That layering is deliberate: at-rest encryption belongs to the key protector
// (an explicitly deferred deployment decision), not to the storage layout.
//
// Provider context references are the exception and are called out in the
// operator guide: they are stored as given, protected by file permissions
// rather than by encryption, because sealing them here would put the registry
// in the key-management business that KeyProtector already owns.
type FileRegistry struct {
	mu      sync.Mutex
	root    string
	nowFunc func() time.Time
}

const (
	profileDirMode  os.FileMode = 0o700
	profileFileMode os.FileMode = 0o600
)

func NewFileRegistry(root string) (*FileRegistry, error) {
	if root == "" {
		return nil, errors.New("browser auth file registry requires a root directory")
	}
	if err := os.MkdirAll(root, profileDirMode); err != nil {
		return nil, fmt.Errorf("create profile registry root: %w", err)
	}
	// Tighten an existing directory that was created with looser permissions.
	if err := os.Chmod(root, profileDirMode); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("secure profile registry root: %w", err)
	}
	return &FileRegistry{root: root, nowFunc: time.Now}, nil
}

func (r *FileRegistry) SetClock(now func() time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nowFunc = now
}

func (r *FileRegistry) now() time.Time {
	if r.nowFunc == nil {
		return time.Now()
	}
	return r.nowFunc()
}

func (r *FileRegistry) Root() string { return r.root }

// storedProfile is the on-disk shape. Material is stored beside it, never in it.
type storedProfile struct {
	Safe SafeProfile  `json:"safe"`
	Kind MaterialKind `json:"kind"`
	// ContextID is set only for hosted profiles; storage-state material lives
	// in the sibling .material file.
	ContextID string `json:"context_id,omitempty"`
}

func (r *FileRegistry) aliasDir(alias string) string { return filepath.Join(r.root, alias) }

func (r *FileRegistry) metaPath(alias, version string) string {
	return filepath.Join(r.aliasDir(alias), version+".meta.json")
}

func (r *FileRegistry) materialPath(alias, version string) string {
	return filepath.Join(r.aliasDir(alias), version+".material")
}

// loadAliasLocked reads every stored version of an alias, ordered by sequence.
func (r *FileRegistry) loadAliasLocked(alias string) ([]storedProfile, error) {
	entries, err := os.ReadDir(r.aliasDir(alias))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var profiles []storedProfile
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(r.aliasDir(alias), name))
		if err != nil {
			return nil, err
		}
		var stored storedProfile
		if err := json.Unmarshal(raw, &stored); err != nil {
			// A corrupt metadata file must fail loudly: silently skipping it
			// would make `latest` quietly resolve to an older version.
			return nil, fmt.Errorf("profile metadata %s is unreadable: %w", name, err)
		}
		profiles = append(profiles, stored)
	}
	sort.SliceStable(profiles, func(i, j int) bool { return profiles[i].Safe.Sequence < profiles[j].Safe.Sequence })
	return profiles, nil
}

func (r *FileRegistry) writeLocked(stored storedProfile, material []byte) error {
	dir := r.aliasDir(stored.Safe.Alias)
	if err := os.MkdirAll(dir, profileDirMode); err != nil {
		return err
	}
	if len(material) > 0 {
		if err := writeFileMode(r.materialPath(stored.Safe.Alias, stored.Safe.Version), material, profileFileMode); err != nil {
			return err
		}
	}
	encoded, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	return writeFileMode(r.metaPath(stored.Safe.Alias, stored.Safe.Version), encoded, profileFileMode)
}

// writeFileMode writes with explicit permissions, then chmods, because
// os.WriteFile only applies the mode when it creates the file.
func writeFileMode(path string, data []byte, mode os.FileMode) error {
	if err := os.WriteFile(path, data, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func (r *FileRegistry) Publish(_ context.Context, record Record) (SafeProfile, error) {
	if err := validateAlias(record.Alias); err != nil {
		return SafeProfile{}, err
	}
	if err := validateVersion(record.Version); err != nil {
		return SafeProfile{}, err
	}
	if record.Provider == "" {
		return SafeProfile{}, fmt.Errorf("profile %s@%s has no provider", record.Alias, record.Version)
	}
	if record.Material.Empty() {
		return SafeProfile{}, fmt.Errorf("profile %s@%s has no material", record.Alias, record.Version)
	}
	if err := record.Policy.Validate(); err != nil {
		return SafeProfile{}, fmt.Errorf("profile %s@%s: %w", record.Alias, record.Version, err)
	}
	policy, err := record.Policy.Normalized()
	if err != nil {
		return SafeProfile{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.loadAliasLocked(record.Alias)
	if err != nil {
		return SafeProfile{}, err
	}
	for _, candidate := range existing {
		if candidate.Safe.Version == record.Version {
			return SafeProfile{}, fmt.Errorf("profile %s@%s already exists; versions are immutable", record.Alias, record.Version)
		}
	}

	previousVersion := ""
	sequence := 1
	if len(existing) > 0 {
		last := existing[len(existing)-1]
		previousVersion = last.Safe.Version
		sequence = last.Safe.Sequence + 1
	}

	kind, sealedBytes, contextID := record.Material.Materialize()
	stored := storedProfile{
		Kind:      kind,
		ContextID: contextID,
		Safe: SafeProfile{
			Alias:           record.Alias,
			Version:         record.Version,
			Sequence:        sequence,
			ProfileHash:     profileHash(record.Material),
			Provider:        record.Provider,
			Policy:          policy,
			CreatedAt:       r.now().UTC(),
			PreviousVersion: previousVersion,
			ExpiresAtOrZero: policy.ExpiresAt,
		},
	}
	if err := r.writeLocked(stored, sealedBytes); err != nil {
		return SafeProfile{}, err
	}
	return stored.Safe, nil
}

func (r *FileRegistry) findLocked(ref AuthProfileRef) (storedProfile, bool, error) {
	profiles, err := r.loadAliasLocked(ref.Alias)
	if err != nil {
		return storedProfile{}, false, err
	}
	if len(profiles) == 0 {
		return storedProfile{}, false, nil
	}
	if !ref.IsLatest() {
		for _, candidate := range profiles {
			if candidate.Safe.Version == ref.Version {
				return candidate, true, nil
			}
		}
		return storedProfile{}, false, nil
	}
	var best storedProfile
	found := false
	for _, candidate := range profiles {
		if candidate.Safe.Revoked() || candidate.Safe.Retired() {
			continue
		}
		if !found || candidate.Safe.Sequence > best.Safe.Sequence {
			best, found = candidate, true
		}
	}
	return best, found, nil
}

func (r *FileRegistry) checkLocked(stored storedProfile, found bool, op string, ref AuthProfileRef) error {
	if !found {
		return NewFailureReason(FailureProfileNotFound, op, "unknown "+ref.String())
	}
	if stored.Safe.Revoked() {
		return NewFailureReason(FailureProfileRevoked, op, "revoked "+stored.Safe.Ref().String())
	}
	if stored.Safe.Expired(r.now()) {
		return NewFailureReason(FailureProfileExpired, op, "expired "+stored.Safe.Ref().String())
	}
	return nil
}

func (r *FileRegistry) Resolve(_ context.Context, ref AuthProfileRef) (SafeProfile, error) {
	if err := validateAlias(ref.Alias); err != nil {
		return SafeProfile{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, found, err := r.findLocked(ref)
	if err != nil {
		return SafeProfile{}, err
	}
	if err := r.checkLocked(stored, found, "resolve", ref); err != nil {
		return SafeProfile{}, err
	}
	return stored.Safe, nil
}

func (r *FileRegistry) Open(_ context.Context, ref AuthProfileRef) (Record, error) {
	if err := validateAlias(ref.Alias); err != nil {
		return Record{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, found, err := r.findLocked(ref)
	if err != nil {
		return Record{}, err
	}
	if err := r.checkLocked(stored, found, "open", ref); err != nil {
		return Record{}, err
	}

	var material SensitiveProfileMaterial
	switch stored.Kind {
	case MaterialProviderContext:
		material = NewProviderContextMaterial(stored.ContextID)
	case MaterialStorageState:
		raw, err := os.ReadFile(r.materialPath(stored.Safe.Alias, stored.Safe.Version))
		if err != nil {
			return Record{}, NewFailure(FailureMaterializeFailed, "read profile material", err)
		}
		material = NewStorageStateMaterial(raw)
	default:
		return Record{}, NewFailureReason(FailureMaterializeFailed, "open", "unknown_material_kind")
	}

	return Record{
		Alias:    stored.Safe.Alias,
		Version:  stored.Safe.Version,
		Provider: stored.Safe.Provider,
		Policy:   stored.Safe.Policy,
		Material: material,
		Safe:     stored.Safe,
	}, nil
}

func (r *FileRegistry) List(_ context.Context, alias string) ([]SafeProfile, error) {
	if err := validateAlias(alias); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	profiles, err := r.loadAliasLocked(alias)
	if err != nil {
		return nil, err
	}
	out := make([]SafeProfile, 0, len(profiles))
	for _, stored := range profiles {
		out = append(out, stored.Safe)
	}
	return out, nil
}

// Aliases lists every alias the registry holds, for `browser-profile inspect`.
func (r *FileRegistry) Aliases() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries, err := os.ReadDir(r.root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var aliases []string
	for _, entry := range entries {
		if entry.IsDir() {
			aliases = append(aliases, entry.Name())
		}
	}
	sort.Strings(aliases)
	return aliases, nil
}

func (r *FileRegistry) mutate(ref AuthProfileRef, op string, apply func(*storedProfile)) error {
	if ref.IsLatest() {
		return fmt.Errorf("%s requires a concrete version, got %s", op, ref)
	}
	if err := validateAlias(ref.Alias); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, found, err := r.findLocked(ref)
	if err != nil {
		return err
	}
	if !found {
		return NewFailureReason(FailureProfileNotFound, op, "unknown "+ref.String())
	}
	apply(&stored)
	return r.writeLocked(stored, nil)
}

func (r *FileRegistry) Revoke(_ context.Context, ref AuthProfileRef, reason string) error {
	now := r.now().UTC()
	return r.mutate(ref, "revoke", func(stored *storedProfile) {
		if stored.Safe.Revoked() {
			return // idempotent: incident response may revoke twice
		}
		stored.Safe.RevokedAt = now
		stored.Safe.RevocationReason = reason
	})
}

func (r *FileRegistry) Retire(_ context.Context, ref AuthProfileRef) error {
	now := r.now().UTC()
	return r.mutate(ref, "retire", func(stored *storedProfile) {
		if stored.Safe.Retired() {
			return
		}
		stored.Safe.RetiredAt = now
	})
}

// Purge permanently deletes a revoked version's material. Revocation alone
// leaves the ciphertext on disk so an incident can still be investigated;
// purging is the separate, deliberate act of destroying it.
func (r *FileRegistry) Purge(ctx context.Context, ref AuthProfileRef) error {
	if ref.IsLatest() {
		return fmt.Errorf("purge requires a concrete version, got %s", ref)
	}
	r.mu.Lock()
	stored, found, err := r.findLocked(ref)
	r.mu.Unlock()
	if err != nil {
		return err
	}
	if !found {
		return NewFailureReason(FailureProfileNotFound, "purge", "unknown "+ref.String())
	}
	if !stored.Safe.Revoked() {
		return NewFailureReason(FailureWritebackDenied, "purge", "purge_requires_revocation_first")
	}
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := os.Remove(r.materialPath(ref.Alias, ref.Version)); err != nil && !os.IsNotExist(err) {
		return NewFailure(FailureCleanupFailed, "purge profile material", err)
	}
	return nil
}

var _ Registry = (*FileRegistry)(nil)
