package main

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/messaging"
	"os"
)

func loadTaskInputsFile(filename string) ([]messaging.TaskInput, error) {
	if filename == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("inputs-file: %w", err)
	}
	inputs, err := messaging.DecodeTaskInputs(raw)
	if err != nil {
		return nil, fmt.Errorf("inputs-file %s: %w", filename, err)
	}
	return inputs, nil
}
