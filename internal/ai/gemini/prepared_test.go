package gemini

import "testing"

func TestPreparedVertexConfiguration_FreezesConfiguredOverrides(t *testing.T) {
	t.Setenv("AILANG_CLOUD_PROJECT", "selected-project")
	t.Setenv("GOOGLE_CLOUD_LOCATION", "selected-location")
	prepared := WithPreparedConfiguration()
	t.Setenv("AILANG_CLOUD_PROJECT", "late-project")
	t.Setenv("GOOGLE_CLOUD_LOCATION", "late-location")
	client, err := NewVertexAIClient("", prepared)
	if err != nil {
		t.Fatal(err)
	}
	if client.projectID != "selected-project" || client.location != "selected-location" {
		t.Fatalf("late mutable configuration: project=%s location=%s", client.projectID, client.location)
	}
}
