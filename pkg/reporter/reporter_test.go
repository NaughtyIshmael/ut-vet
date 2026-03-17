package reporter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

func sampleFindings() []rules.Finding {
	return []rules.Finding{
		{
			File:     "handler_test.go",
			Line:     42,
			Rule:     "no-assertion",
			Message:  "TestCreateUser has no assertion calls",
			Severity: rules.SeverityP0,
			TestName: "TestCreateUser",
		},
		{
			File:     "repo_test.go",
			Line:     15,
			Rule:     "empty-test",
			Message:  "TestGetAll is empty",
			Severity: rules.SeverityP0,
			TestName: "TestGetAll",
		},
	}
}

func TestTextReporter_WithFindings(t *testing.T) {
	r := &TextReporter{Verbose: false}
	out, err := r.Report(sampleFindings())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "handler_test.go:42: [no-assertion]") {
		t.Errorf("expected no-assertion finding in output, got:\n%s", out)
	}
	if !strings.Contains(out, "repo_test.go:15: [empty-test]") {
		t.Errorf("expected empty-test finding in output, got:\n%s", out)
	}
}

func TestTextReporter_NoFindings(t *testing.T) {
	r := &TextReporter{Verbose: false}
	out, err := r.Report(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for no findings, got: %q", out)
	}
}

func TestTextReporter_Verbose(t *testing.T) {
	r := &TextReporter{Verbose: true}
	out, err := r.Report(sampleFindings())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "2 issue(s) found") {
		t.Errorf("expected summary in verbose mode, got:\n%s", out)
	}
}

func TestTextReporter_VerboseNoFindings(t *testing.T) {
	r := &TextReporter{Verbose: true}
	out, err := r.Report(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No issues found") {
		t.Errorf("expected 'No issues found' in verbose mode, got: %q", out)
	}
}

func TestJSONReporter_WithFindings(t *testing.T) {
	r := &JSONReporter{}
	out, err := r.Report(sampleFindings())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Findings []rules.Finding `json:"findings"`
		Total    int             `json:"total"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
	if len(result.Findings) != 2 {
		t.Errorf("expected 2 findings, got %d", len(result.Findings))
	}
}

func TestJSONReporter_NoFindings(t *testing.T) {
	r := &JSONReporter{}
	out, err := r.Report(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result struct {
		Findings []rules.Finding `json:"findings"`
		Total    int             `json:"total"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
	if result.Findings == nil {
		t.Error("findings should be empty array, not null")
	}
}
