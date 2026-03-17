package rules

import "fmt"

// DeadAssertionRule detects assertions that appear after terminating statements.
type DeadAssertionRule struct{}

func (r *DeadAssertionRule) ID() string { return "dead-assertion" }
func (r *DeadAssertionRule) Description() string {
	return "Assertion appears after a terminating statement (unreachable)"
}
func (r *DeadAssertionRule) Severity() Severity { return SeverityP2 }

func (r *DeadAssertionRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc

	if len(tf.TerminatingStatements) == 0 {
		return nil
	}

	// Find the earliest unconditional terminating statement at the top level
	earliestTermLine := tf.TerminatingStatements[0].Line
	for _, ts := range tf.TerminatingStatements {
		if ts.Line < earliestTermLine {
			earliestTermLine = ts.Line
		}
	}

	var findings []Finding
	for _, call := range tf.CallExprs {
		if call.Line <= earliestTermLine {
			continue
		}
		if IsAssertionCall(call) || isTestifyAssertionCall(call) {
			findings = append(findings, Finding{
				File:     ctx.File,
				Line:     call.Line,
				Rule:     r.ID(),
				Message:  fmt.Sprintf("%s has unreachable assertion at line %d (after terminating statement at line %d)", tf.Name, call.Line, earliestTermLine),
				Severity: r.Severity(),
				TestName: tf.Name,
			})
		}
	}

	return findings
}
