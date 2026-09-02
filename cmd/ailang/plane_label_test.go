package main

import "testing"

func TestPlaneLabelNamesTheEnvironment(t *testing.T) {
	cases := map[string]string{
		"ailang-multivac":      "prod",
		"ailang-multivac-dev":  "dev",
		"ailang-multivac-test": "test",
	}
	for project, want := range cases {
		if got := planeLabel(project); got != want {
			t.Errorf("planeLabel(%q) = %q, want %q", project, got, want)
		}
	}
}

// An unknown project must produce NOTHING rather than a guess. Labelling an
// unrecognised project "prod" would be worse than saying nothing at all: the
// label exists to stop people drawing conclusions about the wrong plane.
func TestUnknownProjectIsNotLabelled(t *testing.T) {
	for _, p := range []string{"", "some-other-project", "ailang-multivac-staging", "AILANG-MULTIVAC"} {
		if got := planeLabel(p); got != "" {
			t.Errorf("planeLabel(%q) = %q, want empty — never guess a plane", p, got)
		}
	}
}

// The label must agree with the topic prefix derivation, which is the other
// place a project is mapped onto an environment. If these two ever disagree, one
// of them is routing to the wrong plane.
func TestPlaneLabelAgreesWithTopicPrefixMapping(t *testing.T) {
	for _, p := range []string{"ailang-multivac", "ailang-multivac-dev", "ailang-multivac-test"} {
		_, prefixKnown := topicPrefixForProject(p)
		if labelled := planeLabel(p) != ""; labelled != prefixKnown {
			t.Errorf("project %q: planeLabel knows it = %v but topicPrefixForProject knows it = %v",
				p, labelled, prefixKnown)
		}
	}
	// And an unknown project must be unknown to BOTH.
	_, prefixKnown := topicPrefixForProject("nope")
	if planeLabel("nope") != "" || prefixKnown {
		t.Error("an unknown project must be unknown to both mappings")
	}
}
