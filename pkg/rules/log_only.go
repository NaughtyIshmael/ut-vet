package rules

import "fmt"

// LogOnlyRule detects tests that only log/print but never assert.
type LogOnlyRule struct{}

func (r *LogOnlyRule) ID() string          { return "log-only-test" }
func (r *LogOnlyRule) Description() string { return "Test only logs/prints but has no assertions" }
func (r *LogOnlyRule) Severity() Severity  { return SeverityP0 }

// logFunctions identifies logging calls.
var logFunctions = map[string]map[string]bool{
	"t": {
		"Log":  true,
		"Logf": true,
	},
	"fmt": {
		"Print":    true,
		"Printf":   true,
		"Println":  true,
		"Fprint":   true,
		"Fprintf":  true,
		"Fprintln": true,
		"Sprint":   true,
		"Sprintf":  true,
		"Sprintln": true,
	},
	"log": {
		"Print":   true,
		"Printf":  true,
		"Println": true,
		"Fatal":   true,
		"Fatalf":  true,
		"Fatalln": true,
	},
}

func (r *LogOnlyRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc

	// Skip empty tests
	if !tf.HasBody || tf.BodyLength == 0 {
		return nil
	}

	hasLog := false
	hasAssertion := false

	for _, call := range tf.CallExprs {
		if IsAssertionCall(call) {
			hasAssertion = true
			break
		}
		if isLogCall(call) {
			hasLog = true
		}
	}

	if hasLog && !hasAssertion {
		return []Finding{
			{
				File:     ctx.File,
				Line:     tf.Line,
				Rule:     r.ID(),
				Message:  fmt.Sprintf("%s only logs/prints but has no assertions", tf.Name),
				Severity: r.Severity(),
				TestName: tf.Name,
			},
		}
	}

	return nil
}

func isLogCall(call CallExpr) bool {
	receiver := call.Receiver
	// For t.Log/t.Logf, use "t" as the receiver key
	if call.IsTestingT {
		receiver = "t"
	}
	if funcs, ok := logFunctions[receiver]; ok {
		return funcs[call.Function]
	}
	return false
}
