package eval_analysis

import "encoding/json"

func evidenceIdentity(model, mode, harness string) string {
	data, _ := json.Marshal([3]string{model, mode, harness}) // strings always marshal
	return string(data)
}
