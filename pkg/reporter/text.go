package reporter

import (
	"fmt"
	"strings"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

// TextReporter outputs findings in a human-readable format.
type TextReporter struct {
	Verbose bool
}

func (r *TextReporter) Report(findings []rules.Finding) (string, error) {
	if len(findings) == 0 {
		if r.Verbose {
			return "✅ No issues found\n", nil
		}
		return "", nil
	}

	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "%s:%d: [%s] %s\n", f.File, f.Line, f.Rule, f.Message)
	}

	if r.Verbose {
		fmt.Fprintf(&b, "\n⚠️  %d issue(s) found\n", len(findings))
	}

	return b.String(), nil
}
