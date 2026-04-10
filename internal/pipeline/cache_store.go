package pipeline

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo/ailang/internal/core"
	_ "github.com/sunholo/ailang/internal/core" // registers gob types
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/types"
)

// M-PERF6 M3: Disk-based compilation cache for module interfaces.
//
// Stores a manifest mapping module IDs to their cache keys. When a module's
// source + dependency digests haven't changed, the pipeline can skip
// recompilation for that module.
//
// Cache location: .ailang/cache/compile/manifest.json

// CacheStore manages the on-disk compilation cache.
type CacheStore struct {
	dir      string
	manifest *CacheManifest
}

// CacheManifest tracks cached module compilation state.
type CacheManifest struct {
	Version string                 `json:"version"`
	Entries map[string]*CacheEntry `json:"entries"`
}

// CacheEntry represents a single cached module.
type CacheEntry struct {
	CacheKey      string    `json:"cache_key"`
	IfaceDigest   string    `json:"iface_digest"`
	IfaceJSON     []byte    `json:"iface_json,omitempty"`
	CompileTimeMs int64     `json:"compile_time_ms"`
	Timestamp     time.Time `json:"timestamp"`
}

// NewCacheStore creates or loads a cache store from the given directory.
func NewCacheStore(projectDir string) (*CacheStore, error) {
	dir := filepath.Join(projectDir, ".ailang", "cache", "compile")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}

	cs := &CacheStore{dir: dir}
	if err := cs.load(); err != nil {
		// Corrupted cache — start fresh
		cs.manifest = &CacheManifest{
			Version: cacheKeyVersion,
			Entries: make(map[string]*CacheEntry),
		}
	}
	return cs, nil
}

// Lookup checks if a module has a valid cache entry for the given key.
func (cs *CacheStore) Lookup(moduleID, cacheKey string) (*CacheEntry, bool) {
	entry, ok := cs.manifest.Entries[moduleID]
	if !ok || entry.CacheKey != cacheKey {
		return nil, false
	}
	return entry, true
}

// Store writes a cache entry for a module.
func (cs *CacheStore) Store(moduleID string, entry *CacheEntry) {
	cs.manifest.Entries[moduleID] = entry
}

// Save persists the manifest to disk.
func (cs *CacheStore) Save() error {
	data, err := json.MarshalIndent(cs.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	path := filepath.Join(cs.dir, "manifest.json")
	return os.WriteFile(path, data, 0644)
}

// Clear removes all cache entries.
func (cs *CacheStore) Clear() error {
	cs.manifest = &CacheManifest{
		Version: cacheKeyVersion,
		Entries: make(map[string]*CacheEntry),
	}
	return cs.Save()
}

// Stats returns cache statistics.
func (cs *CacheStore) Stats() (entries int, totalCompileMs int64) {
	for _, e := range cs.manifest.Entries {
		entries++
		totalCompileMs += e.CompileTimeMs
	}
	return
}

// CachedModule holds the full compiled state for a module, enabling
// compilation skip on cache hit. M-INCREMENTAL-TYPECHECK.
type CachedModule struct {
	Core         *core.Program               // Core IR (gob-encoded on disk)
	CoreTI       types.CoreTypeInfo          // Type info for Core nodes (JSON-encoded)
	Iface        *iface.Iface                // Module interface
	Constructors map[string]*ConstructorInfo // ADT constructors
}

// StoreArtifacts serializes a CachedModule to disk alongside the manifest.
// Core program is gob-encoded; CoreTypeInfo, Iface, and Constructors are JSON-encoded.
func (cs *CacheStore) StoreArtifacts(moduleID string, cm *CachedModule) error {
	modDir := filepath.Join(cs.dir, "modules", sanitizeModuleID(moduleID))
	if err := os.MkdirAll(modDir, 0755); err != nil {
		return fmt.Errorf("create module cache dir: %w", err)
	}

	// 1. Gob-encode core.Program
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(cm.Core); err != nil {
		return fmt.Errorf("gob encode core.Program for %s: %w", moduleID, err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "core.gob"), buf.Bytes(), 0644); err != nil {
		return err
	}

	// 2. Gob-encode CoreTypeInfo (M-PERF6B: replaced JSON for 3-5x faster deserialization)
	var ctiBuf bytes.Buffer
	ctiEnc := gob.NewEncoder(&ctiBuf)
	if err := ctiEnc.Encode(cm.CoreTI); err != nil {
		return fmt.Errorf("gob encode CoreTypeInfo for %s: %w", moduleID, err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "coretypeinfo.gob"), ctiBuf.Bytes(), 0644); err != nil {
		return err
	}

	// 3. JSON-encode Iface (full reconstruction, not just digest)
	ifaceData, err := marshalIfaceFull(cm.Iface)
	if err != nil {
		return fmt.Errorf("marshal Iface for %s: %w", moduleID, err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "iface.json"), ifaceData, 0644); err != nil {
		return err
	}

	// 4. JSON-encode Constructors
	ctorData, err := marshalConstructors(cm.Constructors)
	if err != nil {
		return fmt.Errorf("marshal Constructors for %s: %w", moduleID, err)
	}
	if err := os.WriteFile(filepath.Join(modDir, "constructors.json"), ctorData, 0644); err != nil {
		return err
	}

	return nil
}

// LoadArtifacts deserializes a CachedModule from disk.
func (cs *CacheStore) LoadArtifacts(moduleID string) (*CachedModule, error) {
	modDir := filepath.Join(cs.dir, "modules", sanitizeModuleID(moduleID))

	// 1. Gob-decode core.Program
	coreData, err := os.ReadFile(filepath.Join(modDir, "core.gob"))
	if err != nil {
		return nil, fmt.Errorf("read core.gob for %s: %w", moduleID, err)
	}
	var prog core.Program
	dec := gob.NewDecoder(bytes.NewReader(coreData))
	if err := dec.Decode(&prog); err != nil {
		return nil, fmt.Errorf("gob decode core.Program for %s: %w", moduleID, err)
	}

	// 2. Gob-decode CoreTypeInfo (M-PERF6B: replaced JSON for 3-5x faster deserialization)
	ctiData, err := os.ReadFile(filepath.Join(modDir, "coretypeinfo.gob"))
	if err != nil {
		return nil, fmt.Errorf("read coretypeinfo.gob for %s: %w", moduleID, err)
	}
	var cti types.CoreTypeInfo
	ctiDec := gob.NewDecoder(bytes.NewReader(ctiData))
	if err := ctiDec.Decode(&cti); err != nil {
		return nil, fmt.Errorf("gob decode CoreTypeInfo for %s: %w", moduleID, err)
	}

	// 3. JSON-decode Iface
	ifaceData, err := os.ReadFile(filepath.Join(modDir, "iface.json"))
	if err != nil {
		return nil, fmt.Errorf("read iface.json for %s: %w", moduleID, err)
	}
	ifc, err := unmarshalIfaceFull(ifaceData)
	if err != nil {
		return nil, fmt.Errorf("unmarshal Iface for %s: %w", moduleID, err)
	}

	// 4. JSON-decode Constructors
	ctorData, err := os.ReadFile(filepath.Join(modDir, "constructors.json"))
	if err != nil {
		return nil, fmt.Errorf("read constructors.json for %s: %w", moduleID, err)
	}
	ctors, err := unmarshalConstructors(ctorData)
	if err != nil {
		return nil, fmt.Errorf("unmarshal Constructors for %s: %w", moduleID, err)
	}

	return &CachedModule{
		Core:         &prog,
		CoreTI:       cti,
		Iface:        ifc,
		Constructors: ctors,
	}, nil
}

// sanitizeModuleID converts a module ID (e.g., "std/list") to a safe directory name.
func sanitizeModuleID(moduleID string) string {
	// Replace path separators with double underscores
	result := make([]byte, 0, len(moduleID))
	for i := 0; i < len(moduleID); i++ {
		switch moduleID[i] {
		case '/', '\\':
			result = append(result, '_', '_')
		default:
			result = append(result, moduleID[i])
		}
	}
	return string(result)
}

// --- Iface full JSON serialization ---

type ifaceFullJSON struct {
	Module       string                       `json:"module"`
	Schema       string                       `json:"schema"`
	Digest       string                       `json:"digest"`
	Exports      map[string]*ifaceItemJSON    `json:"exports"`
	Constructors map[string]*ctorSchemeJSON   `json:"constructors"`
	Types        map[string]*iface.TypeExport `json:"types"`
	TypeAliases  map[string]json.RawMessage   `json:"type_aliases,omitempty"`
}

type ifaceItemJSON struct {
	Name   string          `json:"name"`
	Type   json.RawMessage `json:"type"`
	Purity bool            `json:"purity"`
	Module string          `json:"ref_module"`
	Symbol string          `json:"ref_symbol"`
}

type ctorSchemeJSON struct {
	TypeName   string            `json:"type_name"`
	CtorName   string            `json:"ctor_name"`
	FieldTypes []json.RawMessage `json:"field_types"`
	ResultType json.RawMessage   `json:"result_type"`
	Arity      int               `json:"arity"`
}

func marshalIfaceFull(ifc *iface.Iface) ([]byte, error) {
	result := ifaceFullJSON{
		Module:       ifc.Module,
		Schema:       ifc.Schema,
		Digest:       ifc.Digest,
		Exports:      make(map[string]*ifaceItemJSON),
		Constructors: make(map[string]*ctorSchemeJSON),
		Types:        ifc.Types,
	}

	// Exports
	for name, item := range ifc.Exports {
		schemeBytes, err := types.MarshalScheme(item.Type)
		if err != nil {
			return nil, fmt.Errorf("marshal scheme for %s: %w", name, err)
		}
		result.Exports[name] = &ifaceItemJSON{
			Name:   item.Name,
			Type:   schemeBytes,
			Purity: item.Purity,
			Module: item.Ref.Module,
			Symbol: item.Ref.Name,
		}
	}

	// Constructors
	for name, cs := range ifc.Constructors {
		fieldTypes := make([]json.RawMessage, len(cs.FieldTypes))
		for i, ft := range cs.FieldTypes {
			raw, err := json.Marshal(ft)
			if err != nil {
				return nil, fmt.Errorf("marshal constructor field type: %w", err)
			}
			fieldTypes[i] = raw
		}
		resultType, err := json.Marshal(cs.ResultType)
		if err != nil {
			return nil, fmt.Errorf("marshal constructor result type: %w", err)
		}
		result.Constructors[name] = &ctorSchemeJSON{
			TypeName:   cs.TypeName,
			CtorName:   cs.CtorName,
			FieldTypes: fieldTypes,
			ResultType: resultType,
			Arity:      cs.Arity,
		}
	}

	// TypeAliases
	if len(ifc.TypeAliases) > 0 {
		result.TypeAliases = make(map[string]json.RawMessage)
		for name, t := range ifc.TypeAliases {
			raw, err := json.Marshal(t)
			if err != nil {
				return nil, fmt.Errorf("marshal type alias %s: %w", name, err)
			}
			result.TypeAliases[name] = raw
		}
	}

	return json.Marshal(result)
}

func unmarshalIfaceFull(data []byte) (*iface.Iface, error) {
	var d ifaceFullJSON
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	ifc := &iface.Iface{
		Module:       d.Module,
		Schema:       d.Schema,
		Digest:       d.Digest,
		Exports:      make(map[string]*iface.IfaceItem),
		Constructors: make(map[string]*iface.ConstructorScheme),
		Types:        d.Types,
		TypeAliases:  make(map[string]types.Type),
	}

	// Exports
	for name, item := range d.Exports {
		scheme, err := types.UnmarshalScheme(item.Type)
		if err != nil {
			return nil, fmt.Errorf("unmarshal scheme for %s: %w", name, err)
		}
		ifc.Exports[name] = &iface.IfaceItem{
			Name:   item.Name,
			Type:   scheme,
			Purity: item.Purity,
			Ref:    core.GlobalRef{Module: item.Module, Name: item.Symbol},
		}
	}

	// Constructors
	for name, cs := range d.Constructors {
		fieldTypes := make([]types.Type, len(cs.FieldTypes))
		for i, raw := range cs.FieldTypes {
			t, err := types.UnmarshalType(raw)
			if err != nil {
				return nil, fmt.Errorf("unmarshal constructor field type: %w", err)
			}
			fieldTypes[i] = t
		}
		resultType, err := types.UnmarshalType(cs.ResultType)
		if err != nil {
			return nil, fmt.Errorf("unmarshal constructor result type: %w", err)
		}
		ifc.Constructors[name] = &iface.ConstructorScheme{
			TypeName:   cs.TypeName,
			CtorName:   cs.CtorName,
			FieldTypes: fieldTypes,
			ResultType: resultType,
			Arity:      cs.Arity,
		}
	}

	// TypeAliases
	for name, raw := range d.TypeAliases {
		t, err := types.UnmarshalType(raw)
		if err != nil {
			return nil, fmt.Errorf("unmarshal type alias %s: %w", name, err)
		}
		ifc.TypeAliases[name] = t
	}

	return ifc, nil
}

// --- ConstructorInfo JSON serialization ---

type constructorInfoJSON struct {
	TypeName       string            `json:"type_name"`
	CtorName       string            `json:"ctor_name"`
	Arity          int               `json:"arity"`
	TypeParamCount int               `json:"type_param_count"`
	TypeParamNames []string          `json:"type_param_names,omitempty"`
	FieldTypes     []json.RawMessage `json:"field_types,omitempty"`
}

func marshalConstructors(ctors map[string]*ConstructorInfo) ([]byte, error) {
	m := make(map[string]*constructorInfoJSON)
	for name, ci := range ctors {
		fieldTypes := make([]json.RawMessage, len(ci.InternalFieldTypes))
		for i, ft := range ci.InternalFieldTypes {
			raw, err := json.Marshal(ft)
			if err != nil {
				return nil, err
			}
			fieldTypes[i] = raw
		}
		m[name] = &constructorInfoJSON{
			TypeName:       ci.TypeName,
			CtorName:       ci.CtorName,
			Arity:          ci.Arity,
			TypeParamCount: ci.TypeParamCount,
			TypeParamNames: ci.TypeParamNames,
			FieldTypes:     fieldTypes,
		}
	}
	return json.Marshal(m)
}

func unmarshalConstructors(data []byte) (map[string]*ConstructorInfo, error) {
	var m map[string]*constructorInfoJSON
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	result := make(map[string]*ConstructorInfo)
	for name, cij := range m {
		fieldTypes := make([]types.Type, len(cij.FieldTypes))
		for i, raw := range cij.FieldTypes {
			t, err := types.UnmarshalType(raw)
			if err != nil {
				return nil, err
			}
			fieldTypes[i] = t
		}
		result[name] = &ConstructorInfo{
			TypeName:           cij.TypeName,
			CtorName:           cij.CtorName,
			Arity:              cij.Arity,
			TypeParamCount:     cij.TypeParamCount,
			TypeParamNames:     cij.TypeParamNames,
			InternalFieldTypes: fieldTypes,
			// Note: FieldTypes ([]ast.Type) is not cached — it's only used during
			// interface building, which we skip on cache hit.
		}
	}
	return result, nil
}

func (cs *CacheStore) load() error {
	path := filepath.Join(cs.dir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest CacheManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	if manifest.Version != cacheKeyVersion {
		return fmt.Errorf("cache version mismatch: %s != %s", manifest.Version, cacheKeyVersion)
	}
	if manifest.Entries == nil {
		manifest.Entries = make(map[string]*CacheEntry)
	}
	cs.manifest = &manifest
	return nil
}
