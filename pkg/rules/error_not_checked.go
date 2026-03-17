package rules

import "fmt"

// ErrorNotCheckedRule detects when a function returns an error
// but the test ignores it.
type ErrorNotCheckedRule struct{}

func (r *ErrorNotCheckedRule) ID() string { return "error-not-checked" }
func (r *ErrorNotCheckedRule) Description() string {
	return "Returned error is ignored or assigned to _"
}
func (r *ErrorNotCheckedRule) Severity() Severity { return SeverityP1 }

func (r *ErrorNotCheckedRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc
	var findings []Finding

	for _, assign := range tf.Assignments {
		if assign.RHSCall == nil {
			continue
		}

		// Case 1: error assigned to blank identifier
		if assign.HasBlankError {
			findings = append(findings, Finding{
				File:     ctx.File,
				Line:     assign.Line,
				Rule:     r.ID(),
				Message:  fmt.Sprintf("%s ignores error returned by %s", tf.Name, assign.RHSCall.FullName),
				Severity: r.Severity(),
				TestName: tf.Name,
			})
			continue
		}

		// Case 2: error variable exists but is never checked
		if assign.ErrorVarName != "" && assign.ErrorVarName != "_" {
			if tf.ErrorVarsChecked == nil || !tf.ErrorVarsChecked[assign.ErrorVarName] {
				findings = append(findings, Finding{
					File:     ctx.File,
					Line:     assign.Line,
					Rule:     r.ID(),
					Message:  fmt.Sprintf("%s does not check error %q returned by %s", tf.Name, assign.ErrorVarName, assign.RHSCall.FullName),
					Severity: r.Severity(),
					TestName: tf.Name,
				})
			}
		}
	}

	return findings
}
