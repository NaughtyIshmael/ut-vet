package rules

import "fmt"

// OnlyNilCheckRule detects tests that only assert err == nil without checking
// the actual return value. This is a common pattern in AI-generated tests.
type OnlyNilCheckRule struct{}

func (r *OnlyNilCheckRule) ID() string { return "only-nil-check" }
func (r *OnlyNilCheckRule) Description() string {
	return "Test only checks error is nil, ignoring actual result"
}
func (r *OnlyNilCheckRule) Severity() Severity { return SeverityP1 }

func (r *OnlyNilCheckRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc
	if !tf.HasBody || tf.BodyLength == 0 {
		return nil
	}

	var assertionCalls []CallExpr
	for _, ce := range tf.CallExprs {
		if IsAssertionCall(ce) {
			assertionCalls = append(assertionCalls, ce)
		}
	}

	if len(assertionCalls) == 0 {
		return nil
	}

	// Check if ALL assertions are error-related
	for _, ce := range assertionCalls {
		if !isErrorOnlyAssertion(ce) {
			return nil
		}
	}

	return []Finding{{
		File:     ctx.File,
		Line:     tf.Line,
		Rule:     r.ID(),
		Severity: r.Severity(),
		TestName: tf.Name,
		Message:  fmt.Sprintf("%s only checks error is nil — never validates the actual result", tf.Name),
	}}
}

// isErrorOnlyAssertion returns true if the assertion call only checks an error condition.
func isErrorOnlyAssertion(ce CallExpr) bool {
	if ce.Receiver == "assert" || ce.Receiver == "require" {
		switch ce.Function {
		case "NoError", "Error":
			return true
		case "Nil", "NotNil":
			// Only error-related if the argument is an error variable
			for _, arg := range ce.Args {
				if arg.IsVariable && isLikelyErrorVar(arg.VarName) {
					return true
				}
			}
		}
	}

	// t.Error/Fatal/Fatalf with an error variable as argument
	if ce.IsTestingT {
		switch ce.Function {
		case "Fatal", "Fatalf":
			for _, arg := range ce.Args {
				if arg.IsVariable && isLikelyErrorVar(arg.VarName) {
					return true
				}
			}
		}
	}

	return false
}

// isLikelyErrorVar returns true if the variable name looks like an error variable.
func isLikelyErrorVar(name string) bool {
	if name == "err" || name == "e" {
		return true
	}
	if len(name) > 3 && name[:3] == "err" && name[3] >= 'A' && name[3] <= 'Z' {
		return true
	}
	return false
}
