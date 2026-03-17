package rules

import (
	"testing"
)

func TestLogOnlyRule_Detects(t *testing.T) {
	rule := &LogOnlyRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "t.Log only",
			tf: &TestFunc{
				Name: "TestLogOnly", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Log", FullName: "t.Log", IsTestingT: true},
				},
			},
			wantHit: true,
		},
		{
			name: "t.Logf only",
			tf: &TestFunc{
				Name: "TestLogfOnly", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Logf", FullName: "t.Logf", IsTestingT: true},
				},
			},
			wantHit: true,
		},
		{
			name: "fmt.Println only",
			tf: &TestFunc{
				Name: "TestFmtPrint", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "fmt", Function: "Println", FullName: "fmt.Println"},
				},
			},
			wantHit: true,
		},
		{
			name: "log.Println only",
			tf: &TestFunc{
				Name: "TestLogPrint", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "log", Function: "Println", FullName: "log.Println"},
				},
			},
			wantHit: true,
		},
		{
			name: "t.Log + t.Errorf — has assertion, not log-only",
			tf: &TestFunc{
				Name: "TestLogAndAssert", Line: 10, HasBody: true, BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Log", FullName: "t.Log", IsTestingT: true},
					{Receiver: "t", Function: "Errorf", FullName: "t.Errorf", IsTestingT: true},
				},
			},
			wantHit: false,
		},
		{
			name: "no calls at all — not log-only (that's no-assertion)",
			tf: &TestFunc{
				Name: "TestNoCalls", Line: 10, HasBody: true, BodyLength: 1,
				CallExprs: []CallExpr{},
			},
			wantHit: false,
		},
		{
			name: "empty body — not log-only",
			tf: &TestFunc{
				Name: "TestEmpty", Line: 10, HasBody: false, BodyLength: 0,
			},
			wantHit: false,
		},
		{
			name: "fmt.Printf + fmt.Println — multiple log calls, still log-only",
			tf: &TestFunc{
				Name: "TestMultiLog", Line: 10, HasBody: true, BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "fmt", Function: "Printf", FullName: "fmt.Printf"},
					{Receiver: "fmt", Function: "Println", FullName: "fmt.Println"},
				},
			},
			wantHit: true,
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
				if findings[0].Rule != "log-only-test" {
					t.Errorf("expected rule 'log-only-test', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestLogOnlyRule_Metadata(t *testing.T) {
	rule := &LogOnlyRule{}
	if rule.ID() != "log-only-test" {
		t.Errorf("expected ID 'log-only-test', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP0 {
		t.Errorf("expected SeverityP0, got %v", rule.Severity())
	}
}
