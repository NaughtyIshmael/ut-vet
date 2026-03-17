package rules

import "fmt"

// TautologicalAssertRule detects assertions that compare a variable to itself.
type TautologicalAssertRule struct{}

func (r *TautologicalAssertRule) ID() string { return "tautological-assert" }
func (r *TautologicalAssertRule) Description() string {
	return "Assertion compares a variable to itself"
}
func (r *TautologicalAssertRule) Severity() Severity { return SeverityP2 }

func (r *TautologicalAssertRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc
	var findings []Finding

	for _, call := range tf.CallExprs {
		if !isTestifyAssertionCall(call) {
			continue
		}

		// Skip the t parameter
		args := call.Args
		if len(args) > 0 && args[0].IsVariable && args[0].VarName == "t" {
			args = args[1:]
		}

		switch call.Function {
		case "Equal", "Exactly", "Same":
			// Check if both comparison args are the same variable
			if len(args) >= 2 && args[0].IsVariable && args[1].IsVariable && args[0].VarName == args[1].VarName {
				findings = append(findings, Finding{
					File:     ctx.File,
					Line:     call.Line,
					Rule:     r.ID(),
					Message:  fmt.Sprintf("%s compares %s to itself: %s", tf.Name, args[0].VarName, formatCall(call)),
					Severity: r.Severity(),
					TestName: tf.Name,
				})
			}
		}
	}

	return findings
}
