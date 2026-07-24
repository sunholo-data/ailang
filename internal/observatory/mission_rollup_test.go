package observatory

import "testing"

func TestMissionKey(t *testing.T) {
	cases := map[string]string{
		"mission:v1/iter-42":         "mission:v1",
		"mission:world/iter-3":       "mission:world",
		"mission:v1":                 "mission:v1", // no slash → unchanged
		"eval_suite:something":       "eval_suite:something",
		"mission:v1/iter-1/substage": "mission:v1", // first slash wins
	}
	for in, want := range cases {
		if got := missionKey(in); got != want {
			t.Errorf("missionKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseQuotaBucket(t *testing.T) {
	cases := map[string]string{
		"sprint-executor (quota:opus)": "opus",
		"quota:sonnet":                 "sonnet",
		"controller (quota:fable) x":   "fable",
		"eval-agent":                   "", // no bucket marker
		"":                             "",
		"quota:":                       "",
	}
	for in, want := range cases {
		if got := parseQuotaBucket(in); got != want {
			t.Errorf("parseQuotaBucket(%q) = %q, want %q", in, got, want)
		}
	}
}
