package firestore

import (
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/messaging"
)

func decodeStoredInputs(value any) ([]messaging.TaskInput, error) {
	if value == nil {
		return nil, nil
	}
	if raw, ok := value.(string); ok {
		return messaging.DecodeTaskInputs([]byte(raw))
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("inputs: %w", err)
	}
	return messaging.DecodeTaskInputs(raw)
}
