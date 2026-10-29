package pkg

import (
	"fmt"
	"strings"
)

// ReadPublishKeyFile reads one private publishing key without changing the
// environment. A selected unsafe/invalid file has no alternate credential lane.
func ReadPublishKeyFile(path string) (string, error) {
	data, err := readPublishKeyBytes(path)
	if err != nil {
		return "", err
	}
	key, err := parsePublishKey(data)
	if err != nil {
		return "", fmt.Errorf("publish --key-file %q: %w", path, err)
	}
	return key, nil
}

func parsePublishKey(data []byte) (string, error) {
	if len(data) > maxPublishKeyBytes {
		return "", fmt.Errorf("key file exceeds %d bytes (contents not shown)", maxPublishKeyBytes)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("key file is empty (contents not shown)")
	}
	for i := 0; i < len(key); i++ {
		if key[i] < 0x20 || key[i] == 0x7f {
			return "", fmt.Errorf("expected one line without header control bytes (contents not shown)")
		}
	}
	return key, nil
}
