package coordinator

import (
	"fmt"
	"strings"
)

// ValidateAutoMergeCode refuses partial code authority; docs mode keeps its defaults.
func (a *AgentConfig) ValidateAutoMergeCode() error {
	if !a.AutoMergeCode {
		return nil
	}
	if !a.AutoMerge || a.SkipApproval {
		return fmt.Errorf("auto_merge_code requires auto_merge and a PR (skip_approval must be false)")
	}
	if strings.TrimSpace(a.AutoMergeApproverSecret) == "" || strings.TrimSpace(a.AutoMergeApproverIdentity) == "" {
		return fmt.Errorf("auto_merge_code requires approver secret name and expected identity")
	}
	for _, field := range []struct {
		name   string
		values []string
	}{
		{"required checks", a.AutoMergeRequiredChecks}, {"artifact patterns", a.ArtifactPatterns},
	} {
		name, values := field.name, field.values
		if len(values) == 0 {
			return fmt.Errorf("auto_merge_code requires non-empty %s", name)
		}
		for _, v := range values {
			if strings.TrimSpace(v) == "" || strings.ContainsAny(v, "\r\n") {
				return fmt.Errorf("auto_merge_code has empty or multiline %s", name)
			}
		}
	}
	return nil
}
