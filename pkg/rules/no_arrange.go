package rules

import "fmt"

// NoArrangeRule detects tests that have assertions but no meaningful setup.
// Specifically, it flags tests where all local function calls use only
// zero-value/nil arguments.
type NoArrangeRule struct{}

func (r *NoArrangeRule) ID() string { return "no-arrange" }
func (r *NoArrangeRule) Description() string {
	return "Test has no meaningful setup — all function args are zero/nil"
}
func (r *NoArrangeRule) Severity() Severity { return SeverityP2 }

func (r *NoArrangeRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc

	if !tf.HasBody || tf.BodyLength == 0 {
		return nil
	}

	// Must have at least one local function call with args
	if len(tf.LocalFuncCalls) == 0 {
		return nil
	}

	// Check if ALL local function calls use only zero-value args
	hasLocalCallWithArgs := false
	allZero := true

	for _, call := range tf.CallExprs {
		if call.IsTestingT || assertionReceivers[call.Receiver] {
			continue
		}

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

		if len(call.Args) == 0 {
			continue
		}

		hasLocalCallWithArgs = true
		for _, arg := range call.Args {
			if !arg.IsZeroVal {
				allZero = false
				break
			}
		}
		if !allZero {
			break
		}
	}

	if hasLocalCallWithArgs && allZero {
		return []Finding{
			{
				File:     ctx.File,
				Line:     tf.Line,
				Rule:     r.ID(),
				Message:  fmt.Sprintf("%s has no meaningful setup — all function arguments are zero-values/nil", tf.Name),
				Severity: r.Severity(),
				TestName: tf.Name,
			},
		}
	}

	return nil
}
