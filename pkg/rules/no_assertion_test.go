package rules

import (
	"testing"
)

func TestNoAssertionRule_Detects(t *testing.T) {
	rule := &NoAssertionRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "empty calls list",
			tf: &TestFunc{
				Name:       "TestNoAssert",
				Line:       10,
				HasBody:    true,
				BodyLength: 2,
				CallExprs:  []CallExpr{},
			},
			wantHit: true,
		},
		{
			name: "only non-assertion calls",
			tf: &TestFunc{
				Name:       "TestNoAssert",
				Line:       10,
				HasBody:    true,
				BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "fmt", Function: "Println", FullName: "fmt.Println"},
				},
			},
			wantHit: true,
		},
		{
			name: "has t.Errorf",
			tf: &TestFunc{
				Name:       "TestWithAssert",
				Line:       10,
				HasBody:    true,
				BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Errorf", FullName: "t.Errorf", IsTestingT: true},
				},
			},
			wantHit: false,
		},
		{
			name: "has t.Fatal",
			tf: &TestFunc{
				Name:       "TestWithFatal",
				Line:       10,
				HasBody:    true,
				BodyLength: 3,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Fatal", FullName: "t.Fatal", IsTestingT: true},
				},
			},
			wantHit: false,
		},
		{
			name: "has assert.Equal",
			tf: &TestFunc{
				Name:       "TestTestify",
				Line:       10,
				HasBody:    true,
				BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "assert", Function: "Equal", FullName: "assert.Equal"},
				},
			},
			wantHit: false,
		},
		{
			name: "has require.NoError",
			tf: &TestFunc{
				Name:       "TestRequire",
				Line:       10,
				HasBody:    true,
				BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "require", Function: "NoError", FullName: "require.NoError"},
				},
			},
			wantHit: false,
		},
		{
			name: "has t.Fail",
			tf: &TestFunc{
				Name:       "TestFail",
				Line:       10,
				HasBody:    true,
				BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Fail", FullName: "t.Fail", IsTestingT: true},
				},
			},
			wantHit: false,
		},
		{
			name: "empty body - skip (handled by empty-test rule)",
			tf: &TestFunc{
				Name:       "TestEmpty",
				Line:       10,
				HasBody:    false,
				BodyLength: 0,
			},
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &AnalysisContext{
				File:     "test.go",
				TestFunc: tt.tf,
			}
			findings := rule.Analyze(ctx)
			if tt.wantHit && len(findings) == 0 {
				t.Error("expected finding but got none")
			}
			if !tt.wantHit && len(findings) > 0 {
				t.Errorf("expected no findings but got: %v", findings)
			}
			if tt.wantHit && len(findings) > 0 {
				if findings[0].Rule != "no-assertion" {
					t.Errorf("expected rule 'no-assertion', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestNoAssertionRule_Metadata(t *testing.T) {
	rule := &NoAssertionRule{}
	if rule.ID() != "no-assertion" {
		t.Errorf("expected ID 'no-assertion', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP0 {
		t.Errorf("expected SeverityP0, got %v", rule.Severity())
	}
}
