//go:build !darwin && !linux

package activation

import "errors"

func (m *Manager) lock() (func(), error) {
	return nil, errors.New("local mission activation requires macOS or Linux host locking")
}
