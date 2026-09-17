package config

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseByteSize parses a human byte count: a plain integer, or an integer
// with a K/M/G/T suffix in any of the common spellings (K, KB, KiB; case
// insensitive; optional space). It is THE size parser for every cap the CLI
// and the runtime take — --max-memory, --fs-max-bytes, AILANG_EVAL_MAX_RSS —
// so an operator learns one spelling. Fractions are accepted for the suffixed
// forms ("1.5GB"). A negative or malformed value is an error, never a
// fallback: a misconfigured safety cap must fail loudly (CLAUDE.md §2).
func ParseByteSize(s string) (int64, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return 0, fmt.Errorf("empty size")
	}
	lower := strings.ToLower(raw)
	for _, u := range []struct {
		suffix string
		mult   int64
	}{
		{"tib", 1 << 40}, {"tb", 1 << 40}, {"t", 1 << 40},
		{"gib", 1 << 30}, {"gb", 1 << 30}, {"g", 1 << 30},
		{"mib", 1 << 20}, {"mb", 1 << 20}, {"m", 1 << 20},
		{"kib", 1 << 10}, {"kb", 1 << 10}, {"k", 1 << 10},
	} {
		if strings.HasSuffix(lower, u.suffix) {
			num := strings.TrimSpace(raw[:len(raw)-len(u.suffix)])
			f, err := strconv.ParseFloat(num, 64)
			if err != nil || f < 0 {
				return 0, fmt.Errorf("invalid size %q: want an integer, optionally with a K/M/G/T suffix (e.g. 256MB, 8G)", s)
			}
			return int64(f * float64(u.mult)), nil
		}
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid size %q: want an integer, optionally with a K/M/G/T suffix (e.g. 256MB, 8G)", s)
	}
	return n, nil
}
