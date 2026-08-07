package testing

import (
	"bytes"
	"crypto/rand" // nolint:depguard // entropy source for --random-seed
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
)

// SeedMode distinguishes an explicit master seed from the default derived mode.
type SeedMode string

const (
	// SeedModeDerived is used when no seed flag was given; the master is int64(0).
	SeedModeDerived SeedMode = "derived"
	// SeedModeMaster is used when --seed N (including 0) or --random-seed was given.
	SeedModeMaster SeedMode = "master"
)

// SeedDerivationV1 is a REPRODUCIBILITY CONTRACT. Changing the framing, hash,
// byte order or identity below is a breaking change and REQUIRES a new tag.
const SeedDerivationV1 = "ailang-property-seed-v1"

// TestConfig carries the workspace and seed policy for a property-testing run.
type TestConfig struct {
	WorkspaceRoot string // absolute; the CLI's initial os.Getwd()
	SeedMode      SeedMode
	MasterSeed    int64
}

// Validate returns an error when the config is not usable for seed derivation.
func (c TestConfig) Validate() error {
	if c.WorkspaceRoot == "" {
		return fmt.Errorf("test config: WorkspaceRoot is empty")
	}
	if !filepath.IsAbs(c.WorkspaceRoot) {
		return fmt.Errorf("test config: WorkspaceRoot %q is not absolute", c.WorkspaceRoot)
	}
	switch c.SeedMode {
	case SeedModeDerived, SeedModeMaster:
		return nil
	}
	return fmt.Errorf("test config: unknown SeedMode %q", c.SeedMode)
}

// DeriveSeedV1 derives a per-property seed from a master seed and the property's
// module identity and name. It is FROZEN and byte-exact (see SeedDerivationV1).
func DeriveSeedV1(master int64, moduleIdentity, propertyName string) int64 {
	var b bytes.Buffer
	b.WriteString(SeedDerivationV1)
	b.WriteByte(0)
	b.WriteString(strconv.FormatInt(master, 10))
	b.WriteByte(0)
	b.WriteString(moduleIdentity)
	b.WriteByte(0)
	b.WriteString(propertyName)
	sum := sha256.Sum256(b.Bytes())
	return int64(binary.LittleEndian.Uint64(sum[0:8]))
}

// ResolveModuleIdentity returns the stable identity used to derive a seed for a
// module. A declared module is path-independent. Otherwise the identity is the
// input path made workspace-relative. It never silently falls back: an error
// that cannot be resolved returns a loud error rather than a plausible wrong
// answer (the value decides a seed).
func ResolveModuleIdentity(workspaceRoot, inputPath, declaredModule string) (string, error) {
	if declaredModule != "" {
		return declaredModule, nil
	}
	abs, err := filepath.Abs(inputPath)
	if err != nil {
		return "", fmt.Errorf("resolve module identity: cannot make input path absolute: %w", err)
	}
	rel, err := filepath.Rel(workspaceRoot, abs)
	if err != nil {
		return "", fmt.Errorf("resolve module identity: cannot make %q relative to workspace root %q: %w", abs, workspaceRoot, err)
	}
	return filepath.ToSlash(filepath.Clean(rel)), nil
}

// EntropyReader is the entropy source for NewRandomMasterSeed. Tests swap it to
// inject failure; production uses crypto/rand.
var EntropyReader io.Reader = rand.Reader

// NewRandomMasterSeed reads exactly 8 bytes of entropy and returns them as a
// master seed. On failure it returns an error and seed 0 — never a clock,
// constant, or per-property fallback.
func NewRandomMasterSeed() (int64, error) {
	var b [8]byte
	if _, err := io.ReadFull(EntropyReader, b[:]); err != nil {
		return 0, fmt.Errorf("--random-seed: failed to read 8 bytes of entropy: %w", err)
	}
	return int64(binary.LittleEndian.Uint64(b[:])), nil
}
