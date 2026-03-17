package reporter

import (
	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

// Reporter formats and outputs findings.
type Reporter interface {
	Report(findings []rules.Finding) (string, error)
}
