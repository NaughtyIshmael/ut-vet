package rules

import (
	"fmt"
	"strings"
)

// HappyPathOnlyRule detects tests that call fallible functions
// but only test the success path, never exercising error conditions.
type HappyPathOnlyRule struct{}

func (r *HappyPathOnlyRule) ID() string { return "happy-path-only" }
func (r *HappyPathOnlyRule) Description() string {
	return "Test only exercises the success path of a fallible function"
}
func (r *HappyPathOnlyRule) Severity() Severity { return SeverityP2 }

// Go assertion functions that test error conditions.
var errorPathAssertions = map[string]bool{
	"Error":           true,
	"EqualError":      true,
	"ErrorContains":   true,
	"ErrorIs":         true,
	"ErrorAs":         true,
	"Panics":          true,
	"PanicsWithValue": true,
}

func (r *HappyPathOnlyRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc

	if !tf.HasBody || tf.BodyLength == 0 {
		return nil
	}

	// Must have assertions — otherwise no-assertion rule handles it
	hasAssertion := false
	for _, call := range tf.CallExprs {
		if IsAssertionCall(call) {
			hasAssertion = true
			break
		}
	}
	if !hasAssertion {
		return nil
	}

	// Must have local function calls — otherwise no-code-under-test handles it
	if len(tf.LocalFuncCalls) == 0 {
		return nil
	}

	// Check if test has fallible function calls.
	hasFallible := false
	hasBlankError := false

	for _, a := range tf.Assignments {
		if a.HasBlankError {
			hasBlankError = true
		}
		for _, name := range a.LHS {
			if name == "err" || (strings.HasPrefix(name, "err") && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z') {
				hasFallible = true
				break
			}
		}
	}

	// Skip if error is assigned to blank — error-not-checked rule handles it
	if hasBlankError && !hasFallible {
		return nil
	}

	// Rust: .unwrap() or .expect() implies fallible
	if !hasFallible {
		for _, call := range tf.CallExprs {
			if (call.Function == "unwrap" || call.Function == "expect") && call.Receiver != "" {
				hasFallible = true
				break
			}
		}
	}

	if !hasFallible {
		return nil
	}

	// Skip if test only checks error (only-nil-check handles it).
	// Check: does the test have any assertion that is NOT about error status?
	hasNonErrorAssertion := false
	for _, call := range tf.CallExprs {
		if !IsAssertionCall(call) {
			continue
		}
		// Go: NoError/Error/Nil on error var are error-status assertions
		if assertionReceivers[call.Receiver] {
			fn := call.Function
			if fn == "NoError" || fn == "Error" || fn == "EqualError" || fn == "ErrorContains" || fn == "ErrorIs" || fn == "ErrorAs" {
				continue
			}
			if (fn == "Nil" || fn == "NotNil") && len(call.Args) >= 2 {
				argName := call.Args[1].VarName
				if argName == "err" || strings.HasPrefix(argName, "err") {
					continue
				}
			}
			hasNonErrorAssertion = true
			break
		}
		// Rust: assert!(result.is_ok()) / assert!(result.is_err()) are error-status
		if rustAssertionMacros[call.Function] {
			isErrorCheck := false
			for _, arg := range call.Args {
				v := arg.VarName
				if v == "" {
					v = arg.Value
				}
				if strings.Contains(v, ".is_ok()") || strings.Contains(v, ".is_err()") {
					isErrorCheck = true
					break
				}
			}
			if !isErrorCheck {
				hasNonErrorAssertion = true
				break
			}
			continue
		}
		// .unwrap()/.expect() are error-status-like (they just unwrap)
		if rustAssertionMethods[call.Function] {
			continue
		}
		hasNonErrorAssertion = true
		break
	}
	if !hasNonErrorAssertion {
		return nil // only-nil-check handles this
	}

	// Check if test has subtests (t.Run) — skip if so
	for _, call := range tf.CallExprs {
		if call.IsTestingT && call.Function == "Run" {
			return nil
		}
	}

	// Check if test has any error-path assertions.
	for _, call := range tf.CallExprs {
		if assertionReceivers[call.Receiver] && errorPathAssertions[call.Function] {
			return nil
		}
		if rustAssertionMacros[call.Function] {
			for _, arg := range call.Args {
				v := arg.Value
				if arg.IsVariable {
					v = arg.VarName
				}
				if strings.Contains(v, ".is_err()") || strings.Contains(v, "is_err") {
					return nil
				}
			}
		}
		if call.Function == "unwrap_err" {
			return nil
		}
	}

	for _, stmt := range tf.Body {
		content := stmt.Content
		if strings.Contains(content, "err != nil") && !strings.Contains(content, "err == nil") {
			return nil
		}
		if strings.Contains(content, "Err(") && (strings.Contains(content, "match") || strings.Contains(content, "=>")) {
			return nil
		}
		if strings.Contains(content, "if let Err") {
			return nil
		}
	}

	return []Finding{
		{
			File:     ctx.File,
			Line:     tf.Line,
			Rule:     r.ID(),
			Message:  fmt.Sprintf("%s only tests the success path of a fallible function", tf.Name),
			Severity: r.Severity(),
			TestName: tf.Name,
		},
	}
}
