package rules

import (
	"testing"
)

func TestNoCodeUnderTestRule_Detects(t *testing.T) {
	rule := &NoCodeUnderTestRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "only stdlib and assertion calls",
			tf: &TestFunc{
				Name: "TestNothing", Line: 10, HasBody: true, BodyLength: 3,
				PackageName: "mypackage",
				CallExprs: []CallExpr{
					{Receiver: "strings", Function: "ToUpper", FullName: "strings.ToUpper"},
					{Receiver: "assert", Function: "Equal", FullName: "assert.Equal"},
				},
			},
			wantHit: true,
		},
		{
			name: "only testing.T calls",
			tf: &TestFunc{
				Name: "TestOnlyT", Line: 10, HasBody: true, BodyLength: 2,
				PackageName: "mypackage",
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Errorf", FullName: "t.Errorf", IsTestingT: true},
				},
			},
			wantHit: true,
		},
		{
			name: "calls function from same package (no receiver)",
			tf: &TestFunc{
				Name: "TestWithPkgFunc", Line: 10, HasBody: true, BodyLength: 3,
				PackageName: "mypackage",
				CallExprs: []CallExpr{
					{Function: "Calculate", FullName: "Calculate"},
					{Receiver: "t", Function: "Errorf", FullName: "t.Errorf", IsTestingT: true},
				},
				LocalFuncCalls: []string{"Calculate"},
			},
			wantHit: false,
		},
		{
			name: "calls method on locally created object",
			tf: &TestFunc{
				Name: "TestWithMethod", Line: 10, HasBody: true, BodyLength: 4,
				PackageName: "mypackage",
				CallExprs: []CallExpr{
					{Function: "NewService", FullName: "NewService"},
					{Receiver: "svc", Function: "Process", FullName: "svc.Process"},
					{Receiver: "assert", Function: "NoError", FullName: "assert.NoError"},
				},
				LocalFuncCalls: []string{"NewService"},
			},
			wantHit: false,
		},
		{
			name: "empty test — skip",
			tf: &TestFunc{
				Name: "TestEmpty", Line: 10, HasBody: false, BodyLength: 0,
				PackageName: "mypackage",
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
				if findings[0].Rule != "no-code-under-test" {
					t.Errorf("expected rule 'no-code-under-test', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestNoCodeUnderTestRule_Metadata(t *testing.T) {
	rule := &NoCodeUnderTestRule{}
	if rule.ID() != "no-code-under-test" {
		t.Errorf("expected ID 'no-code-under-test', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP1 {
		t.Errorf("expected SeverityP1, got %v", rule.Severity())
	}
}
