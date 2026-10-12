//go:build !unix

package proctree

import "fmt"

func KillOwnGroup() error { return fmt.Errorf("self process-group cancellation is unsupported") }
