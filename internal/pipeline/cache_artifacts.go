package pipeline

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

const (
	artifactStampName        = "artifacts.json"
	artifactCoreName         = "core.gob"
	artifactCoreTIName       = "coretypeinfo.gob"
	artifactIfaceName        = "iface.json"
	artifactConstructorsName = "constructors.json"

	maxArtifactBlobBytes   int64 = 16 << 20
	maxArtifactStampBytes  int64 = 64 << 10
	maxModuleArtifactBytes int64 = 32 << 20

	artifactInvalidReason  = "ARTIFACT_INVALID"
	artifactTooLargeReason = "ARTIFACT_TOO_LARGE"
)

type artifactStamp struct {
	Version  string            `json:"version"`
	ModuleID string            `json:"module_id"`
	CacheKey string            `json:"cache_key"`
	SHA256   map[string]string `json:"sha256"`
}

type artifactLimits struct {
	blob   int64
	stamp  int64
	module int64
}

func productionArtifactLimits() artifactLimits {
	return artifactLimits{
		blob:   maxArtifactBlobBytes,
		stamp:  maxArtifactStampBytes,
		module: maxModuleArtifactBytes,
	}
}

type artifactReadFile interface {
	io.Reader
	Stat() (fs.FileInfo, error)
	Close() error
}

type artifactTempFile interface {
	io.Writer
	Close() error
	Name() string
}

type cacheArtifactIO struct {
	open       func(string) (artifactReadFile, error)
	mkdirAll   func(string, fs.FileMode) error
	writeFile  func(string, []byte, fs.FileMode) error
	createTemp func(string, string) (artifactTempFile, error)
	rename     func(string, string) error
	remove     func(string) error
	removeAll  func(string) error
}

func productionCacheArtifactIO() cacheArtifactIO {
	return cacheArtifactIO{
		open: func(path string) (artifactReadFile, error) {
			return os.Open(path)
		},
		mkdirAll:  os.MkdirAll,
		writeFile: os.WriteFile,
		createTemp: func(dir, pattern string) (artifactTempFile, error) {
			return os.CreateTemp(dir, pattern)
		},
		rename:    os.Rename,
		remove:    os.Remove,
		removeAll: os.RemoveAll,
	}
}

type cacheArtifactCodec struct {
	encodeCore         func(*core.Program) ([]byte, error)
	encodeCoreTI       func(types.CoreTypeInfo) ([]byte, error)
	encodeIface        func(*iface.Iface) ([]byte, error)
	encodeConstructors func(map[string]*ConstructorInfo) ([]byte, error)
	decodeCore         func([]byte) (*core.Program, error)
	decodeCoreTI       func([]byte) (types.CoreTypeInfo, error)
	decodeIface        func([]byte) (*iface.Iface, error)
	decodeConstructors func([]byte) (map[string]*ConstructorInfo, error)
}

func productionCacheArtifactCodec() cacheArtifactCodec {
	return cacheArtifactCodec{
		encodeCore: func(program *core.Program) ([]byte, error) {
			var buf bytes.Buffer
			err := gob.NewEncoder(&buf).Encode(program)
			return buf.Bytes(), err
		},
		encodeCoreTI: func(info types.CoreTypeInfo) ([]byte, error) {
			var buf bytes.Buffer
			err := gob.NewEncoder(&buf).Encode(info)
			return buf.Bytes(), err
		},
		encodeIface:        marshalIfaceFull,
		encodeConstructors: marshalConstructors,
		decodeCore: func(data []byte) (*core.Program, error) {
			var program core.Program
			err := gob.NewDecoder(bytes.NewReader(data)).Decode(&program)
			return &program, err
		},
		decodeCoreTI: func(data []byte) (types.CoreTypeInfo, error) {
			var info types.CoreTypeInfo
			err := gob.NewDecoder(bytes.NewReader(data)).Decode(&info)
			return info, err
		},
		decodeIface:        unmarshalIfaceFull,
		decodeConstructors: unmarshalConstructors,
	}
}

type cacheArtifactError struct {
	Reason     string
	Stage      string
	Path       string
	Scope      string
	LimitBytes int64
	Err        error
}

func (e *cacheArtifactError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.Reason, e.Path)
	}
	return fmt.Sprintf("%s: %s: %v", e.Reason, e.Path, e.Err)
}

func (e *cacheArtifactError) Unwrap() error { return e.Err }

func artifactFailure(stage, path string, err error) error {
	return &cacheArtifactError{
		Reason: artifactInvalidReason,
		Stage:  stage,
		Path:   path,
		Err:    err,
	}
}

func artifactTooLarge(stage, path, scope string, limit int64) error {
	return &cacheArtifactError{
		Reason:     artifactTooLargeReason,
		Stage:      stage,
		Path:       path,
		Scope:      scope,
		LimitBytes: limit,
		Err:        fmt.Errorf("artifact exceeds %s byte limit", scope),
	}
}

type encodedArtifact struct {
	name string
	data []byte
}

func (cs *CacheStore) encodeArtifacts(moduleID string, cm *CachedModule) ([]encodedArtifact, error) {
	encoders := []struct {
		name string
		fn   func() ([]byte, error)
	}{
		{artifactCoreName, func() ([]byte, error) { return cs.artifactCodec.encodeCore(cm.Core) }},
		{artifactCoreTIName, func() ([]byte, error) { return cs.artifactCodec.encodeCoreTI(cm.CoreTI) }},
		{artifactIfaceName, func() ([]byte, error) { return cs.artifactCodec.encodeIface(cm.Iface) }},
		{artifactConstructorsName, func() ([]byte, error) { return cs.artifactCodec.encodeConstructors(cm.Constructors) }},
	}

	artifacts := make([]encodedArtifact, 0, len(encoders))
	var accepted int64
	for _, encoder := range encoders {
		data, err := encoder.fn()
		path := filepath.Join(cs.moduleArtifactDir(moduleID), encoder.name)
		if err != nil {
			return nil, artifactFailure("encoding", path, err)
		}
		if err := cs.checkArtifactSize("encoding", path, "blob", int64(len(data)), &accepted); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, encodedArtifact{name: encoder.name, data: data})
	}
	return artifacts, nil
}

func (cs *CacheStore) checkArtifactSize(stage, path, fileScope string, size int64, accepted *int64) error {
	remaining := cs.artifactLimits.module - *accepted
	limit, scope := bindingArtifactLimit(cs.artifactLimits.blob, remaining, fileScope)
	if fileScope == "stamp" {
		limit, scope = bindingArtifactLimit(cs.artifactLimits.stamp, remaining, fileScope)
	}
	if size > limit {
		return artifactTooLarge(stage, path, scope, limit)
	}
	*accepted += size
	return nil
}

func bindingArtifactLimit(fileLimit, remaining int64, fileScope string) (int64, string) {
	if remaining < 0 {
		remaining = 0
	}
	if fileLimit <= remaining {
		return fileLimit, fileScope
	}
	return remaining, "module"
}

func (cs *CacheStore) storeArtifacts(moduleID, cacheKey string, cm *CachedModule) error {
	if cacheKey == "" {
		return artifactFailure("encoding", cs.moduleArtifactDir(moduleID), fmt.Errorf("empty cache key"))
	}
	artifacts, err := cs.encodeArtifacts(moduleID, cm)
	if err != nil {
		return err
	}

	digests := make(map[string]string, len(artifacts))
	var accepted int64
	for _, artifact := range artifacts {
		accepted += int64(len(artifact.data))
		sum := sha256.Sum256(artifact.data)
		digests[artifact.name] = hex.EncodeToString(sum[:])
	}
	stampData, err := json.MarshalIndent(artifactStamp{
		Version:  cacheKeyVersion,
		ModuleID: moduleID,
		CacheKey: cacheKey,
		SHA256:   digests,
	}, "", "  ")
	stampPath := filepath.Join(cs.moduleArtifactDir(moduleID), artifactStampName)
	if err != nil {
		return artifactFailure("encoding", stampPath, err)
	}
	if err := cs.checkArtifactSize("encoding", stampPath, "stamp", int64(len(stampData)), &accepted); err != nil {
		return err
	}

	modDir := cs.moduleArtifactDir(moduleID)
	if err := cs.artifactIO.mkdirAll(modDir, 0o755); err != nil {
		return artifactFailure("publication", modDir, err)
	}
	for _, artifact := range artifacts {
		path := filepath.Join(modDir, artifact.name)
		if err := cs.artifactIO.writeFile(path, artifact.data, 0o644); err != nil {
			return artifactFailure("publication", path, err)
		}
	}

	tmp, err := cs.artifactIO.createTemp(modDir, ".artifacts-*.tmp")
	if err != nil {
		return artifactFailure("publication", stampPath, err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = cs.artifactIO.remove(tmpPath) }
	if n, writeErr := tmp.Write(stampData); writeErr != nil || n != len(stampData) {
		if writeErr == nil {
			writeErr = io.ErrShortWrite
		}
		_ = tmp.Close()
		cleanup()
		return artifactFailure("publication", tmpPath, writeErr)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return artifactFailure("publication", tmpPath, err)
	}
	if err := cs.artifactIO.rename(tmpPath, stampPath); err != nil {
		cleanup()
		return artifactFailure("publication", stampPath, err)
	}
	return nil
}

func (cs *CacheStore) loadArtifacts(moduleID, expectedCacheKey string) (*CachedModule, error) {
	modDir := cs.moduleArtifactDir(moduleID)
	stampPath := filepath.Join(modDir, artifactStampName)
	var accepted int64
	stampData, err := cs.readBoundedArtifact(stampPath, cs.artifactLimits.stamp, "stamp", &accepted)
	if err != nil {
		return nil, err
	}
	var stamp artifactStamp
	if err := json.Unmarshal(stampData, &stamp); err != nil {
		return nil, artifactFailure("verification", stampPath, err)
	}
	if expectedCacheKey == "" || stamp.Version != cacheKeyVersion || stamp.ModuleID != moduleID || stamp.CacheKey != expectedCacheKey {
		return nil, artifactFailure("verification", stampPath, fmt.Errorf("stamp authorization mismatch"))
	}
	if err := validateArtifactDigests(stamp.SHA256); err != nil {
		return nil, artifactFailure("verification", stampPath, err)
	}

	artifacts := make(map[string][]byte, 4)
	for _, name := range artifactPayloadNames() {
		path := filepath.Join(modDir, name)
		data, readErr := cs.readBoundedArtifact(path, cs.artifactLimits.blob, "blob", &accepted)
		if readErr != nil {
			return nil, readErr
		}
		artifacts[name] = data
	}
	for _, name := range artifactPayloadNames() {
		sum := sha256.Sum256(artifacts[name])
		if hex.EncodeToString(sum[:]) != stamp.SHA256[name] {
			return nil, artifactFailure("verification", filepath.Join(modDir, name), fmt.Errorf("SHA-256 mismatch"))
		}
	}

	program, err := cs.artifactCodec.decodeCore(artifacts[artifactCoreName])
	if err != nil {
		return nil, artifactFailure("decoding", filepath.Join(modDir, artifactCoreName), err)
	}
	coreTI, err := cs.artifactCodec.decodeCoreTI(artifacts[artifactCoreTIName])
	if err != nil {
		return nil, artifactFailure("decoding", filepath.Join(modDir, artifactCoreTIName), err)
	}
	ifc, err := cs.artifactCodec.decodeIface(artifacts[artifactIfaceName])
	if err != nil {
		return nil, artifactFailure("decoding", filepath.Join(modDir, artifactIfaceName), err)
	}
	constructors, err := cs.artifactCodec.decodeConstructors(artifacts[artifactConstructorsName])
	if err != nil {
		return nil, artifactFailure("decoding", filepath.Join(modDir, artifactConstructorsName), err)
	}
	return &CachedModule{Core: program, CoreTI: coreTI, Iface: ifc, Constructors: constructors}, nil
}

func (cs *CacheStore) readBoundedArtifact(path string, fileLimit int64, fileScope string, accepted *int64) ([]byte, error) {
	remaining := cs.artifactLimits.module - *accepted
	limit, scope := bindingArtifactLimit(fileLimit, remaining, fileScope)
	file, err := cs.artifactIO.open(path)
	if err != nil {
		return nil, artifactFailure("verification", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, artifactFailure("verification", path, err)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, artifactFailure("verification", path, fmt.Errorf("not a regular file"))
	}
	if info.Size() > limit {
		_ = file.Close()
		return nil, artifactTooLarge("verification", path, scope, limit)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	if readErr != nil {
		return nil, artifactFailure("verification", path, readErr)
	}
	if closeErr != nil {
		return nil, artifactFailure("verification", path, closeErr)
	}
	if int64(len(data)) > limit {
		return nil, artifactTooLarge("verification", path, scope, limit)
	}
	*accepted += int64(len(data))
	return data, nil
}

func validateArtifactDigests(digests map[string]string) error {
	if len(digests) != 4 {
		return fmt.Errorf("stamp must contain exactly four payload digests")
	}
	for _, name := range artifactPayloadNames() {
		digest, ok := digests[name]
		if !ok || len(digest) != sha256.Size*2 {
			return fmt.Errorf("missing or malformed digest for %s", name)
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return fmt.Errorf("malformed digest for %s: %w", name, err)
		}
	}
	return nil
}

func artifactPayloadNames() [4]string {
	return [4]string{artifactCoreName, artifactCoreTIName, artifactIfaceName, artifactConstructorsName}
}

func (cs *CacheStore) moduleArtifactDir(moduleID string) string {
	return filepath.Join(cs.dir, "modules", sanitizeModuleID(moduleID))
}
