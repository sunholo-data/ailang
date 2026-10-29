//go:build !darwin && !linux

package pkg

import "os"

func openPublishKeyFile(_ string) (*os.File, error) {
	if err := Require(AuthNetwork, "reading publisher credentials", OperatorRemedy); err != nil {
		return nil, err
	}
	return nil, ErrPublishKeyFileUnsupported
}
