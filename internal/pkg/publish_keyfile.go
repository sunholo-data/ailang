package pkg

import (
	"errors"
	"fmt"
	"io"
)

const maxPublishKeyBytes = 16 * 1024

// ErrPublishKeyFileUnsupported refuses a security guarantee this host cannot prove.
var ErrPublishKeyFileUnsupported = errors.New("publish --key-file is unsupported on this platform; protected file verification requires macOS or Linux")

func readPublishKeyBytes(path string) ([]byte, error) {
	if err := Require(AuthNetwork, "reading publisher credentials", OperatorRemedy); err != nil {
		return nil, err
	}
	f, err := openPublishKeyFile(path)
	if err != nil {
		return nil, fmt.Errorf("publish --key-file %q: %w", path, err)
	}
	defer f.Close()
	data, err := readPublishKeyStream(f)
	if err != nil {
		return nil, fmt.Errorf("publish --key-file %q: %w", path, err)
	}
	return data, nil
}

// Bound the stream itself: stat size alone cannot constrain a growing file.
func readPublishKeyStream(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxPublishKeyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPublishKeyBytes {
		return nil, fmt.Errorf("key file exceeds %d bytes (contents not shown)", maxPublishKeyBytes)
	}
	return data, nil
}
