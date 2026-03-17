package reporter

import (
	"encoding/json"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

// JSONReporter outputs findings as JSON.
type JSONReporter struct{}

type jsonOutput struct {
	Findings []rules.Finding `json:"findings"`
	Total    int             `json:"total"`
}

func (r *JSONReporter) Report(findings []rules.Finding) (string, error) {
	out := jsonOutput{
		Findings: findings,
		Total:    len(findings),
	}
	if out.Findings == nil {
		out.Findings = []rules.Finding{}
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}
