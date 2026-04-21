package eval_analysis

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadSuiteEvents reads benchmarks/events.yml (or any path) and returns
// the parsed timeline annotations. Missing file returns an empty slice
// rather than an error — events are optional.
func LoadSuiteEvents(path string) ([]SuiteEvent, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read events.yml: %w", err)
	}

	var events []SuiteEvent
	if err := yaml.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("parse events.yml: %w", err)
	}

	for i := range events {
		if events[i].Version == "" || events[i].Label == "" {
			return nil, fmt.Errorf("event %d missing required version/label", i)
		}
	}

	return events, nil
}
