package rules

import "fmt"

// TrivialAssertRule detects assertions on literal/constant expressions.
type TrivialAssertRule struct{}

func (r *TrivialAssertRule) ID() string { return "trivial-assertion" }
func (r *TrivialAssertRule) Description() string {
	return "Assertion on a literal or constant expression"
}
func (r *TrivialAssertRule) Severity() Severity { return SeverityP0 }

func (r *TrivialAssertRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc
	var findings []Finding

	for _, call := range tf.CallExprs {
		if !isTestifyAssertionCall(call) {
			continue
		}

		if isTrivial := r.checkTrivial(call); isTrivial {
			findings = append(findings, Finding{
				File:     ctx.File,
				Line:     call.Line,
				Rule:     r.ID(),
				Message:  fmt.Sprintf("%s asserts a constant expression: %s", tf.Name, formatCall(call)),
				Severity: r.Severity(),
				TestName: tf.Name,
			})
		}
	}

	return findings
}

func (r *TrivialAssertRule) checkTrivial(call CallExpr) bool {
	// Skip the first arg (t *testing.T) for testify-style calls
	args := call.Args
	if len(args) > 0 && args[0].IsVariable && args[0].VarName == "t" {
		args = args[1:]
	}

	switch call.Function {
	case "True":
		// assert.True(t, true) — trivial
		return len(args) >= 1 && args[0].IsLiteral && args[0].Value == "true"

	case "False":
		// assert.False(t, false) — trivial
		return len(args) >= 1 && args[0].IsLiteral && (args[0].Value == "false")

	case "Nil":
		// assert.Nil(t, nil) — trivial
		return len(args) >= 1 && args[0].IsNil

	case "NotNil":
		// assert.NotNil(t, <literal>) — trivial if literal is non-nil
		return len(args) >= 1 && args[0].IsLiteral && !args[0].IsNil

	case "Equal", "Exactly":
		// assert.Equal(t, X, X) — trivial if both args are identical literals
		if len(args) >= 2 {
			return args[0].IsLiteral && args[1].IsLiteral && args[0].Value == args[1].Value
		}

	case "NotEqual":
		// assert.NotEqual(t, 1, 2) with both literals — trivially true
		if len(args) >= 2 {
			return args[0].IsLiteral && args[1].IsLiteral
		}
	}

	return false
}

func isTestifyAssertionCall(call CallExpr) bool {
	return call.Receiver == "assert" || call.Receiver == "require"
}

func formatCall(call CallExpr) string {
	s := call.FullName + "("
	for i, arg := range call.Args {
		if i > 0 {
			s += ", "
		}
		s += arg.Value
	}
	s += ")"
	return s
}
