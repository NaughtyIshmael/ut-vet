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
		if !isTestifyAssertionCall(call) && !isRustAssertMacro(call) {
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
		return len(args) >= 1 && args[0].IsLiteral && args[0].Value == "true"

	case "False":
		return len(args) >= 1 && args[0].IsLiteral && (args[0].Value == "false")

	case "Nil":
		return len(args) >= 1 && args[0].IsNil

	case "NotNil":
		return len(args) >= 1 && args[0].IsLiteral && !args[0].IsNil

	case "Equal", "Exactly":
		if len(args) >= 2 {
			return args[0].IsLiteral && args[1].IsLiteral && args[0].Value == args[1].Value
		}

	case "NotEqual":
		if len(args) >= 2 {
			return args[0].IsLiteral && args[1].IsLiteral
		}

	// Rust macros
	case "assert!":
		return len(args) >= 1 && args[0].IsLiteral && args[0].Value == "true"

	case "assert_eq!", "debug_assert_eq!":
		if len(args) >= 2 {
			return args[0].IsLiteral && args[1].IsLiteral && args[0].Value == args[1].Value
		}

	case "assert_ne!", "debug_assert_ne!":
		if len(args) >= 2 {
			return args[0].IsLiteral && args[1].IsLiteral
		}
	}

	return false
}

func isTestifyAssertionCall(call CallExpr) bool {
	return call.Receiver == "assert" || call.Receiver == "require"
}

func isRustAssertMacro(call CallExpr) bool {
	switch call.Function {
	case "assert!", "assert_eq!", "assert_ne!", "debug_assert!", "debug_assert_eq!", "debug_assert_ne!":
		return true
	}
	return false
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
