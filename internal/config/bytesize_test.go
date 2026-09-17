package config

import "testing"

func TestParseByteSize(t *testing.T) {
	cases := map[string]int64{
		"0": 0, "1024": 1024, "8G": 8 << 30, "8g": 8 << 30, "256MB": 256 << 20, "256 MiB": 256 << 20,
		"1.5GB": 3 << 29, "64k": 64 << 10, "2TiB": 2 << 40,
	}
	for in, want := range cases {
		got, err := ParseByteSize(in)
		if err != nil || got != want {
			t.Errorf("%q: got %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "-1", "abc", "1.5", "MB", "10 x"} {
		if _, err := ParseByteSize(bad); err == nil {
			t.Errorf("%q: want error", bad)
		}
	}
}
