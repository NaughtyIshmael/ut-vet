package rules

import "testing"

func TestHappyPathOnlyRule_Detects(t *testing.T) {
	rule := &HappyPathOnlyRule{}

	tests := []struct {
		name     string
		tf       TestFunc
		wantHit  bool
		wantRule string
	}{
		{
			name: "fallible function, only success check — detected",
			tf: TestFunc{
				Name:       "TestCreateUser",
				Line:       10,
				HasBody:    true,
				BodyLength: 3,
				CallExprs: []CallExpr{
					{Function: "CreateUser", FullName: "CreateUser", Args: []Arg{{Value: "john", IsLiteral: true}}},
					{Function: "NoError", Receiver: "require", FullName: "require.NoError",
						Args: []Arg{{Value: "err", IsVariable: true, VarName: "err"}}},
					{Function: "Equal", Receiver: "assert", FullName: "assert.Equal",
						Args: []Arg{{Value: "t"}, {Value: "john", IsLiteral: true}, {Value: "user.Name", IsVariable: true, VarName: "user.Name"}}},
				},
				Assignments: []Assignment{
					{LHS: []string{"user", "err"}},
				},
				ErrorVarsChecked: map[string]bool{"err": true},
				LocalFuncCalls:   []string{"CreateUser"},
			},
			wantHit:  true,
			wantRule: "happy-path-only",
		},
		{
			name: "has error-path assertion (assert.Error) — NOT detected",
			tf: TestFunc{
				Name:       "TestCreateUser_Error",
				Line:       20,
				HasBody:    true,
				BodyLength: 3,
				CallExprs: []CallExpr{
					{Function: "CreateUser", FullName: "CreateUser"},
					{Function: "Error", Receiver: "assert", FullName: "assert.Error",
						Args: []Arg{{Value: "err", IsVariable: true, VarName: "err"}}},
				},
				Assignments: []Assignment{
					{LHS: []string{"_", "err"}},
				},
				ErrorVarsChecked: map[string]bool{"err": true},
				LocalFuncCalls:   []string{"CreateUser"},
			},
			wantHit: false,
		},
		{
			name: "no fallible functions — NOT detected",
			tf: TestFunc{
				Name:       "TestAdd",
				Line:       30,
				HasBody:    true,
				BodyLength: 2,
				CallExprs: []CallExpr{
					{Function: "Add", FullName: "Add"},
					{Function: "Equal", Receiver: "assert", FullName: "assert.Equal"},
				},
				Assignments: []Assignment{
					{LHS: []string{"result"}},
				},
				ErrorVarsChecked: map[string]bool{},
				LocalFuncCalls:   []string{"Add"},
			},
			wantHit: false,
		},
		{
			name: "has subtests with t.Run — NOT detected",
			tf: TestFunc{
				Name:       "TestCreateUser",
				Line:       40,
				HasBody:    true,
				BodyLength: 5,
				CallExprs: []CallExpr{
					{Function: "Run", IsTestingT: true, FullName: "t.Run"},
					{Function: "CreateUser", FullName: "CreateUser"},
					{Function: "NoError", Receiver: "require", FullName: "require.NoError"},
				},
				Assignments: []Assignment{
					{LHS: []string{"user", "err"}},
				},
				ErrorVarsChecked: map[string]bool{"err": true},
				LocalFuncCalls:   []string{"CreateUser"},
			},
			wantHit: false,
		},
		{
			name: "empty test — skip",
			tf: TestFunc{
				Name:       "TestEmpty",
				Line:       50,
				HasBody:    false,
				BodyLength: 0,
			},
			wantHit: false,
		},
		{
			name: "Rust unwrap only — skip (only-nil-check handles it)",
			tf: TestFunc{
				Name:       "test_create_user",
				Line:       60,
				HasBody:    true,
				BodyLength: 2,
				CallExprs: []CallExpr{
					{Function: "create_user", FullName: "create_user"},
					{Function: "unwrap", Receiver: "result", FullName: "result.unwrap"},
				},
				Assignments: []Assignment{
					{LHS: []string{"result"}},
				},
				ErrorVarsChecked: map[string]bool{},
				LocalFuncCalls:   []string{"create_user"},
			},
			wantHit: false,
		},
		{
			name: "Rust unwrap + meaningful assert — detected",
			tf: TestFunc{
				Name:       "test_create_user",
				Line:       60,
				HasBody:    true,
				BodyLength: 3,
				CallExprs: []CallExpr{
					{Function: "create_user", FullName: "create_user"},
					{Function: "unwrap", Receiver: "result", FullName: "result.unwrap"},
					{Function: "assert_eq!", FullName: "assert_eq!",
						Args: []Arg{{Value: "user.name", IsVariable: true, VarName: "user.name"}, {Value: "john", IsLiteral: true}}},
				},
				Assignments: []Assignment{
					{LHS: []string{"result"}},
				},
				ErrorVarsChecked: map[string]bool{},
				LocalFuncCalls:   []string{"create_user"},
			},
			wantHit:  true,
			wantRule: "happy-path-only",
		},
		{
			name: "Rust is_err assertion — NOT detected",
			tf: TestFunc{
				Name:       "test_create_user_error",
				Line:       70,
				HasBody:    true,
				BodyLength: 2,
				CallExprs: []CallExpr{
					{Function: "create_user", FullName: "create_user"},
					{Function: "assert!", FullName: "assert!",
						Args: []Arg{{Value: "result.is_err()", IsVariable: true, VarName: "result.is_err()"}}},
				},
				Assignments: []Assignment{
					{LHS: []string{"result"}},
				},
				LocalFuncCalls: []string{"create_user"},
			},
			wantHit: false,
		},
		{
			name: "Go EqualError assertion — NOT detected",
			tf: TestFunc{
				Name:       "TestCreateUser_EqualError",
				Line:       80,
				HasBody:    true,
				BodyLength: 3,
				CallExprs: []CallExpr{
					{Function: "CreateUser", FullName: "CreateUser"},
					{Function: "EqualError", Receiver: "assert", FullName: "assert.EqualError",
						Args: []Arg{{Value: "err", IsVariable: true, VarName: "err"}, {Value: "invalid", IsLiteral: true}}},
				},
				Assignments: []Assignment{
					{LHS: []string{"_", "err"}},
				},
				ErrorVarsChecked: map[string]bool{"err": true},
				LocalFuncCalls:   []string{"CreateUser"},
			},
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &AnalysisContext{
				File:     "test.go",
				TestFunc: &tt.tf,
			}
			findings := rule.Analyze(ctx)
			if tt.wantHit && len(findings) == 0 {
				t.Errorf("expected finding but got none")
			}
			if !tt.wantHit && len(findings) > 0 {
				t.Errorf("expected no finding but got: %v", findings)
			}
			if tt.wantHit && len(findings) > 0 && findings[0].Rule != tt.wantRule {
				t.Errorf("got rule %q, want %q", findings[0].Rule, tt.wantRule)
			}
		})
	}
}

func TestHappyPathOnlyRule_Metadata(t *testing.T) {
	rule := &HappyPathOnlyRule{}
	if rule.ID() != "happy-path-only" {
		t.Errorf("got ID %q, want %q", rule.ID(), "happy-path-only")
	}
	if rule.Severity() != SeverityP2 {
		t.Errorf("got severity %v, want P2", rule.Severity())
	}
}
