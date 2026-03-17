package rules

import "fmt"

// EmptyTestRule detects test functions with empty bodies or only comments.
type EmptyTestRule struct{}

func (r *EmptyTestRule) ID() string { return "empty-test" }
func (r *EmptyTestRule) Description() string {
	return "Test function body is empty or contains only comments"
}
func (r *EmptyTestRule) Severity() Severity { return SeverityP0 }

func (r *EmptyTestRule) Analyze(ctx *AnalysisContext) []Finding {
	tf := ctx.TestFunc

	if !tf.HasBody || tf.BodyLength == 0 {
		return []Finding{
			{
				File:     ctx.File,
				Line:     tf.Line,
				Rule:     r.ID(),
				Message:  fmt.Sprintf("%s is empty", tf.Name),
				Severity: r.Severity(),
				TestName: tf.Name,
			},
		}
	}

	return nil
}
