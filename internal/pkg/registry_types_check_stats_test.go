package pkg

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidationCheckStats(t *testing.T) {
	var legacy ValidationResult
	if err := json.Unmarshal([]byte(`{"compiles":true}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.CheckPeakRSS != 0 || legacy.CheckWallMS != 0 {
		t.Fatal(legacy)
	}
	data, _ := json.Marshal(legacy)
	if strings.Contains(string(data), "check_") {
		t.Fatal(string(data))
	}
	want := ValidationResult{Compiles: true, CheckPeakRSS: 123456789, CheckWallMS: 987}
	data, _ = json.Marshal(want)
	var got ValidationResult
	if err := json.Unmarshal(data, &got); err != nil || got != want {
		t.Fatalf("%s: %+v %v", data, got, err)
	}
	for _, field := range []string{"check_peak_rss", "check_wall_ms"} {
		if !strings.Contains(string(data), field) {
			t.Fatal(string(data))
		}
	}
}
