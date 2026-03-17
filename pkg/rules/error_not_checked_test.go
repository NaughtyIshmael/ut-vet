package rules

import (
	"testing"
)

func TestErrorNotCheckedRule_Detects(t *testing.T) {
	rule := &ErrorNotCheckedRule{}

	tests := []struct {
		name    string
		tf      *TestFunc
		wantHit bool
	}{
		{
			name: "error assigned to blank identifier",
			tf: &TestFunc{
				Name: "TestSave", Line: 10, HasBody: true, BodyLength: 2,
				Body: []Statement{
					{Kind: StmtAssign, Content: "_, _ = repo.Save(entity)"},
				},
				CallExprs: []CallExpr{
					{Receiver: "repo", Function: "Save", FullName: "repo.Save",
						Args: []Arg{{IsVariable: true, VarName: "entity"}}},
					{Receiver: "t", Function: "Log", FullName: "t.Log", IsTestingT: true},
				},
				Assignments: []Assignment{
					{
						LHS:           []string{"_", "_"},
						RHSCall:       &CallExpr{Receiver: "repo", Function: "Save", FullName: "repo.Save"},
						HasBlankError: true,
						ErrorVarName:  "_",
						Line:          11,
					},
				},
			},
			wantHit: true,
		},
		{
			name: "error assigned to variable but never checked",
			tf: &TestFunc{
				Name: "TestSave2", Line: 10, HasBody: true, BodyLength: 2,
				Assignments: []Assignment{
					{
						LHS:           []string{"result", "err"},
						RHSCall:       &CallExpr{Receiver: "repo", Function: "Save", FullName: "repo.Save"},
						HasBlankError: false,
						ErrorVarName:  "err",
						Line:          11,
					},
				},
				CallExprs: []CallExpr{
					{Receiver: "repo", Function: "Save", FullName: "repo.Save"},
					// No assertion on err
				},
				ErrorVarsChecked: map[string]bool{},
			},
			wantHit: true,
		},
		{
			name: "error checked with require.NoError",
			tf: &TestFunc{
				Name: "TestSaveChecked", Line: 10, HasBody: true, BodyLength: 3,
				Assignments: []Assignment{
					{
						LHS:           []string{"result", "err"},
						RHSCall:       &CallExpr{Receiver: "repo", Function: "Save", FullName: "repo.Save"},
						HasBlankError: false,
						ErrorVarName:  "err",
						Line:          11,
					},
				},
				CallExprs: []CallExpr{
					{Receiver: "repo", Function: "Save", FullName: "repo.Save"},
					{Receiver: "require", Function: "NoError", FullName: "require.NoError",
						Args: []Arg{
							{IsVariable: true, VarName: "t"},
							{IsVariable: true, VarName: "err"},
						}},
				},
				ErrorVarsChecked: map[string]bool{"err": true},
			},
			wantHit: false,
		},
		{
			name: "no assignments at all",
			tf: &TestFunc{
				Name: "TestSimple", Line: 10, HasBody: true, BodyLength: 2,
				CallExprs: []CallExpr{
					{Receiver: "t", Function: "Error", FullName: "t.Error", IsTestingT: true},
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
				if findings[0].Rule != "error-not-checked" {
					t.Errorf("expected rule 'error-not-checked', got %q", findings[0].Rule)
				}
			}
		})
	}
}

func TestErrorNotCheckedRule_Metadata(t *testing.T) {
	rule := &ErrorNotCheckedRule{}
	if rule.ID() != "error-not-checked" {
		t.Errorf("expected ID 'error-not-checked', got %q", rule.ID())
	}
	if rule.Severity() != SeverityP1 {
		t.Errorf("expected SeverityP1, got %v", rule.Severity())
	}
}
