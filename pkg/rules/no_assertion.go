package rules

import "fmt"

// NoAssertionRule detects test functions that have no assertion calls.
type NoAssertionRule struct{}

func (r *NoAssertionRule) ID() string          { return "no-assertion" }
func (r *NoAssertionRule) Description() string { return "Test function has no assertion calls" }
func (r *NoAssertionRule) Severity() Severity  { return SeverityP0 }

// assertionFunctions lists testing.T methods that count as assertions.
var assertionFunctions = map[string]bool{
	"Error":   true,
	"Errorf":  true,
	"Fatal":   true,
	"Fatalf":  true,
	"Fail":    true,
	"FailNow": true,
}

// assertionReceivers lists package-level receivers that are assertion libraries.
var assertionReceivers = map[string]bool{
	"assert":  true,
	"require": true,
}

func (r *NoAssertionRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc

	// Skip empty tests — handled by empty-test rule
	if !tf.HasBody || tf.BodyLength == 0 {
		return nil
	}

	for _, call := range tf.CallExprs {
		if IsAssertionCall(call) {
			return nil
		}
	}

	return []Finding{
		{
			File:     ctx.File,
			Line:     tf.Line,
			Rule:     r.ID(),
			Message:  fmt.Sprintf("%s has no assertion calls", tf.Name),
			Severity: r.Severity(),
			TestName: tf.Name,
		},
	}
}

// Rust assertion macros
var rustAssertionMacros = map[string]bool{
	"assert!":          true,
	"assert_eq!":       true,
	"assert_ne!":       true,
	"debug_assert!":    true,
	"debug_assert_eq!": true,
	"debug_assert_ne!": true,
}

// Rust method calls that act as implicit assertions (panic on failure)
var rustAssertionMethods = map[string]bool{
	"unwrap": true,
	"expect": true,
}

// IsAssertionCall returns true if the call expression is an assertion.
func IsAssertionCall(call CallExpr) bool {
	// testing.T assertion methods (Go)
	if call.IsTestingT && assertionFunctions[call.Function] {
		return true
	}
	// testify and similar assertion packages (Go)
	if assertionReceivers[call.Receiver] {
		return true
	}
	// Rust assertion macros
	if rustAssertionMacros[call.Function] {
		return true
	}
	// Rust assertion methods (.unwrap(), .expect())
	if rustAssertionMethods[call.Function] && call.Receiver != "" {
		return true
	}
	return false
}
