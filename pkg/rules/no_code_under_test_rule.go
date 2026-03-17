package rules

import "fmt"

// NoCodeUnderTestRule detects tests that never call any function
// from the package being tested.
type NoCodeUnderTestRule struct{}

func (r *NoCodeUnderTestRule) ID() string { return "no-code-under-test" }
func (r *NoCodeUnderTestRule) Description() string {
	return "Test never calls functions from the tested package"
}
func (r *NoCodeUnderTestRule) Severity() Severity { return SeverityP1 }

// wellKnownReceivers are receivers that are NOT code-under-test.
var wellKnownReceivers = map[string]bool{
	// Go stdlib
	"assert":   true,
	"require":  true,
	"fmt":      true,
	"log":      true,
	"strings":  true,
	"strconv":  true,
	"os":       true,
	"io":       true,
	"bytes":    true,
	"context":  true,
	"time":     true,
	"sync":     true,
	"http":     true,
	"json":     true,
	"math":     true,
	"sort":     true,
	"regexp":   true,
	"reflect":  true,
	"errors":   true,
	"path":     true,
	"filepath": true,
	// Rust stdlib / well-known types
	"std":     true,
	"Vec":     true,
	"String":  true,
	"HashMap": true,
	"HashSet": true,
	"Option":  true,
	"Result":  true,
	"Box":     true,
	"Rc":      true,
	"Arc":     true,
}

func (r *NoCodeUnderTestRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc

	if !tf.HasBody || tf.BodyLength == 0 {
		return nil
	}

	// If there are local function calls (no receiver = same package), it's testing real code
	if len(tf.LocalFuncCalls) > 0 {
		return nil
	}

	// Check if any call goes to a non-well-known, non-testing receiver
	for _, call := range tf.CallExprs {
		if call.IsTestingT {
			continue
		}
		if assertionReceivers[call.Receiver] {
			continue
		}
		if wellKnownReceivers[call.Receiver] {
			continue
		}
		// Skip Rust assertion macros (empty receiver, but not code-under-test)
		if IsAssertionCall(call) {
			continue
		}
		// Skip Rust log macros
		if rustLogMacroSet[call.Function] {
			continue
		}
		// A call to an unknown receiver or a plain function — likely code under test
		if call.Receiver == "" && call.Function != "" {
			return nil
		}
		if call.Receiver != "" && !wellKnownReceivers[call.Receiver] {
			return nil
		}
	}

	return []Finding{
		{
			File:     ctx.File,
			Line:     tf.Line,
			Rule:     r.ID(),
			Message:  fmt.Sprintf("%s never calls any function from the tested package", tf.Name),
			Severity: r.Severity(),
			TestName: tf.Name,
		},
	}
}
