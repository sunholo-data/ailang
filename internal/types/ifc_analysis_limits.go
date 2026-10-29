package types

import "fmt"

// Defensive analysis limits are failures, never permission to omit a demand.
// Normal convergence is by summary equality and repeated application states.
const (
	ifcAnalysisDepth                    = 1000
	ifcAnalysisWork                     = 1000000
	IFCAnalysisLimitError TypeErrorKind = "ifc_analysis_limit"
)

type ifcAnalysisLimit struct{ reason string }

func (e ifcAnalysisLimit) diagnostic() *TypeCheckError {
	return &TypeCheckError{Kind: IFCAnalysisLimitError, Message: fmt.Sprintf("information-flow analysis limit exceeded (%s); callable requirements could not be verified", e.reason), Suggestion: "simplify the nested function-value expression before checking again"}
}

func (c *ifcChecker) analysisStep() {
	c.analysisRemaining--
	if c.analysisRemaining < 0 {
		panic(ifcAnalysisLimit{reason: "summary work budget"})
	}
}

// Recover only the typed defensive limit. Programming errors retain their panic.
func guardIFCAnalysis(run func() []*TypeCheckError) (errs []*TypeCheckError) {
	defer func() {
		if p := recover(); p != nil {
			if limit, ok := p.(ifcAnalysisLimit); ok {
				errs = []*TypeCheckError{limit.diagnostic()}
			} else {
				panic(p)
			}
		}
	}()
	return run()
}
