package rules

import (
	"testing"
)

func TestEmptyTestRule_Detects(t *testing.T) {
	rule := &EmptyTestRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "empty body (no statements)",
			tf: &TestFunc{
				Name:       "TestEmpty",
				Line:       10,
				HasBody:    false,
				BodyLength: 0,
			},
			wantHit: true,
		},
		{
			name: "only comments",
			tf: &TestFunc{
				Name:       "TestOnlyComments",
				Line:       15,
				HasBody:    true,
				BodyLength: 0, // all statements are comments
				Body: []Statement{
					{Kind: StmtComment, Content: "// TODO"},
				},
			},
			wantHit: true,
		},
		{
			name: "has real statements",
			tf: &TestFunc{
				Name:       "TestReal",
				Line:       20,
				HasBody:    true,
				BodyLength: 3,
				Body: []Statement{
					{Kind: StmtAssign, Content: "x := 1"},
					{Kind: StmtCall, Content: "t.Error(x)"},
				},
			},
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &AnalysisContext{File: "test.go", TestFunc: tt.tf}
			findings := rule.Analyze(ctx)
			if tt.wantHit && len(findings) == 0 {
				t.Error("expected finding but got none")
			}
			if !tt.wantHit && len(findings) > 0 {
				t.Errorf("expected no findings but got: %v", findings)
			}
			if tt.wantHit && len(findings) > 0 {
				if findings[0].Rule != "empty-test" {
					t.Errorf("expected rule 'empty-test', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestEmptyTestRule_Metadata(t *testing.T) {
	rule := &EmptyTestRule{}
	if rule.ID() != "empty-test" {
		t.Errorf("expected ID 'empty-test', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP0 {
		t.Errorf("expected SeverityP0, got %v", rule.Severity())
	}
}
