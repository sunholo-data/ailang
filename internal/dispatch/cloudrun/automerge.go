package cloudrun

import (
	"strings"

	runpb "cloud.google.com/go/run/apiv2/runpb"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
)

// codeAutoMergeEnv forwards trusted registry values only, never secret material.
func codeAutoMergeEnv(p coordinator.DispatchParams) []*runpb.EnvVar {
	if !p.AutoMergeCode {
		return nil
	}
	return []*runpb.EnvVar{
		{Name: config.EnvAutoMergeCode, Values: &runpb.EnvVar_Value{Value: "1"}},
		{Name: config.EnvAutoMergeRequiredChecks, Values: &runpb.EnvVar_Value{Value: strings.Join(p.AutoMergeRequiredChecks, "\n")}},
		{Name: config.EnvApproverSecret, Values: &runpb.EnvVar_Value{Value: p.AutoMergeApproverSecret}},
		{Name: config.EnvApproverIdentity, Values: &runpb.EnvVar_Value{Value: p.AutoMergeApproverIdentity}},
	}
}
