//go:build !unix

package proctree

// ReapWorkspace is unsupported off Unix: there is no cwd scan, so it finds nothing.
func ReapWorkspace(_ string) ([]int, error) { return nil, nil }
