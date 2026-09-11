package observatory

import "errors"

// ErrNotFound distinguishes an absent record from a failed storage query.
// Backends may also return (nil, nil) for an absent optional lookup.
var ErrNotFound = errors.New("observatory record not found")
