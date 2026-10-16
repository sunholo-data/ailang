package effects

// CheckedSharedCache is an optional persistent-cache extension. Runtime callers
// use it to distinguish storage failures from missing keys or CAS contention.
// The original SharedCache contract remains available to existing brain clients.
type CheckedSharedCache interface {
	GetChecked(key string) ([]byte, bool, error)
	PutChecked(key string, value []byte) error
	DeleteChecked(key string) error
	CASChecked(key string, oldValue, newValue []byte) (bool, error)
	KeysChecked() ([]string, error)
}
