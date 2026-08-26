package auth

import (
	"encoding/json"
	"fmt"
)

// MaterialKind names the shape of a canonical profile's sensitive material.
type MaterialKind string

const (
	// MaterialStorageState is a Playwright storage-state blob: cookies, local
	// storage, IndexedDB entries, and optional virtual credentials.
	MaterialStorageState MaterialKind = "storage_state"
	// MaterialProviderContext is an opaque hosted-provider context reference,
	// such as a Browserbase Context ID.
	MaterialProviderContext MaterialKind = "provider_context"
)

// SensitiveProfileMaterial keeps credential-grade browser state out of fmt,
// JSON, logs, and errors. It follows the browser.SensitiveConnection precedent:
// no exported fields, every presentation redacts, and Materialize is the only
// extraction API.
//
// It deliberately implements Error() as well as String() so that a value
// accidentally wrapped as an error still redacts, and GoString() so that %#v
// cannot fall back to Go-syntax printing of the unexported fields.
type SensitiveProfileMaterial struct {
	kind      MaterialKind
	state     []byte
	contextID string
}

// NewStorageStateMaterial copies state so the caller cannot mutate registry-held
// bytes afterwards.
func NewStorageStateMaterial(state []byte) SensitiveProfileMaterial {
	return SensitiveProfileMaterial{
		kind:  MaterialStorageState,
		state: append([]byte(nil), state...),
	}
}

func NewProviderContextMaterial(contextID string) SensitiveProfileMaterial {
	return SensitiveProfileMaterial{
		kind:      MaterialProviderContext,
		contextID: contextID,
	}
}

func (m SensitiveProfileMaterial) Kind() MaterialKind { return m.kind }

// Empty reports whether there is any material at all. It does not reveal size.
func (m SensitiveProfileMaterial) Empty() bool {
	return len(m.state) == 0 && m.contextID == ""
}

func (m SensitiveProfileMaterial) String() string {
	kind := m.kind
	if kind == "" {
		kind = "unset"
	}
	return fmt.Sprintf("browser auth material %s (%s)", Redacted, kind)
}

func (m SensitiveProfileMaterial) GoString() string { return m.String() }

func (m SensitiveProfileMaterial) Error() string { return m.String() }

func (m SensitiveProfileMaterial) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Kind     MaterialKind `json:"kind"`
		Material string       `json:"material"`
	}{Kind: m.kind, Material: Redacted})
}

// Materialize is the sole explicit extraction API. It returns copies so a caller
// cannot reach back into the registry's storage.
func (m SensitiveProfileMaterial) Materialize() (MaterialKind, []byte, string) {
	return m.kind, append([]byte(nil), m.state...), m.contextID
}
