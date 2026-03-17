package rules

import "fmt"

// ZeroValueInputRule detects tests where functions are called with only
// zero-value arguments (nil, 0, "", false).
type ZeroValueInputRule struct{}

func (r *ZeroValueInputRule) ID() string { return "zero-value-input" }
func (r *ZeroValueInputRule) Description() string {
	return "Function called with only zero-value arguments"
}
func (r *ZeroValueInputRule) Severity() Severity { return SeverityP1 }

func (r *ZeroValueInputRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc
	var findings []Finding

	for _, call := range tf.CallExprs {
		// Skip assertion/testing calls
		if call.IsTestingT || assertionReceivers[call.Receiver] {
			continue
		}

		// Only check calls to local functions (code under test)
		isLocal := false
		for _, name := range tf.LocalFuncCalls {
			if name == call.Function {
				isLocal = true
				break
			}
		}
		if !isLocal {
			continue
		}

		// Must have at least one argument
		if len(call.Args) == 0 {
			continue
		}

		// Check if ALL arguments are zero-values
		allZero := true
		for _, arg := range call.Args {
			if !arg.IsZeroVal {
				allZero = false
				break
			}
		}

		if allZero {
			findings = append(findings, Finding{
				File:     ctx.File,
				Line:     call.Line,
				Rule:     r.ID(),
				Message:  fmt.Sprintf("%s calls %s with only zero-value arguments", tf.Name, call.FullName),
				Severity: r.Severity(),
				TestName: tf.Name,
			})
		}
	}

	return findings
}
