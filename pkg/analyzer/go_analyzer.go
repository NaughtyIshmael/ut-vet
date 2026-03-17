package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

// ParseGoTestFile parses a Go test file and extracts test functions.
func ParseGoTestFile(filename string, src []byte) ([]*rules.TestFunc, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var testFuncs []*rules.TestFunc
	pkgName := ""
	if file.Name != nil {
		pkgName = file.Name.Name
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		if !isTestFunc(fn) {
			continue
		}

		tf := extractTestFunc(fset, fn, src)
		tf.PackageName = pkgName
		testFuncs = append(testFuncs, tf)
	}

	return testFuncs, nil
}

// isTestFunc checks if a function declaration is a Go test function.
func isTestFunc(fn *ast.FuncDecl) bool {
	name := fn.Name.Name
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	// Must have exactly one parameter of type *testing.T
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	for _, param := range fn.Type.Params.List {
		if isTestingTType(param.Type) {
			return true
		}
	}
	return false
}

// isTestingTType checks if a type expression is *testing.T.
func isTestingTType(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "testing" && sel.Sel.Name == "T"
}

// extractTestFunc builds a TestFunc from an AST function declaration.
func extractTestFunc(fset *token.FileSet, fn *ast.FuncDecl, src []byte) *rules.TestFunc {
	tf := &rules.TestFunc{
		Name: fn.Name.Name,
		Line: fset.Position(fn.Pos()).Line,
	}

	if fn.Body == nil || len(fn.Body.List) == 0 {
		tf.HasBody = false
		tf.BodyLength = 0
		return tf
	}

	tf.HasBody = true

	// Find the testing.T parameter name
	tParamName := findTestingTParamName(fn)

	nonCommentCount := 0
	for _, stmt := range fn.Body.List {
		s := extractStatement(fset, stmt, src)
		tf.Body = append(tf.Body, s)
		if s.Kind != rules.StmtComment {
			nonCommentCount++
		}
	}
	tf.BodyLength = nonCommentCount

	// Extract all call expressions recursively
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ce := extractCallExpr(fset, call, tParamName, src)
		tf.CallExprs = append(tf.CallExprs, ce)
		return true
	})

	// P1: Extract local function calls (no receiver = same-package)
	for _, ce := range tf.CallExprs {
		if ce.Receiver == "" && ce.Function != "" {
			tf.LocalFuncCalls = append(tf.LocalFuncCalls, ce.Function)
		}
	}

	// P1: Extract assignments and error variable tracking
	tf.ErrorVarsChecked = make(map[string]bool)
	extractAssignments(fset, fn.Body, tParamName, src, tf)
	extractErrorVarChecks(tf)

	return tf
}

// findTestingTParamName returns the name of the *testing.T parameter.
func findTestingTParamName(fn *ast.FuncDecl) string {
	if fn.Type.Params == nil {
		return "t"
	}
	for _, param := range fn.Type.Params.List {
		if isTestingTType(param.Type) && len(param.Names) > 0 {
			return param.Names[0].Name
		}
	}
	return "t"
}

// extractStatement creates a Statement from an AST statement node.
func extractStatement(fset *token.FileSet, stmt ast.Stmt, src []byte) rules.Statement {
	pos := fset.Position(stmt.Pos())
	s := rules.Statement{
		Line: pos.Line,
	}

	switch stmt.(type) {
	case *ast.ExprStmt:
		s.Kind = rules.StmtExpr
	case *ast.AssignStmt:
		s.Kind = rules.StmtAssign
	default:
		s.Kind = rules.StmtOther
	}

	// Extract source text for the statement
	start := fset.Position(stmt.Pos()).Offset
	end := fset.Position(stmt.End()).Offset
	if start >= 0 && end <= len(src) && start < end {
		s.Content = string(src[start:end])
	}

	return s
}

// extractCallExpr creates a CallExpr from an AST call expression.
func extractCallExpr(fset *token.FileSet, call *ast.CallExpr, tParamName string, src []byte) rules.CallExpr {
	ce := rules.CallExpr{
		Line: fset.Position(call.Pos()).Line,
	}

	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		// Method call: receiver.Method(...)
		if ident, ok := fn.X.(*ast.Ident); ok {
			ce.Receiver = ident.Name
			ce.Function = fn.Sel.Name
			ce.FullName = ident.Name + "." + fn.Sel.Name
			ce.IsTestingT = (ident.Name == tParamName)
		}
	case *ast.Ident:
		// Plain function call: funcName(...)
		ce.Function = fn.Name
		ce.FullName = fn.Name
	}

	// Extract arguments
	for _, arg := range call.Args {
		ce.Args = append(ce.Args, extractArg(arg, src, fset))
	}

	return ce
}

// extractArg analyzes a call argument.
func extractArg(expr ast.Expr, src []byte, fset *token.FileSet) rules.Arg {
	a := rules.Arg{}

	// Get string representation
	start := fset.Position(expr.Pos()).Offset
	end := fset.Position(expr.End()).Offset
	if start >= 0 && end <= len(src) && start < end {
		a.Value = string(src[start:end])
	}

	switch e := expr.(type) {
	case *ast.BasicLit:
		a.IsLiteral = true
		a.Value = e.Value
		switch e.Kind {
		case token.INT:
			a.IsZeroVal = e.Value == "0"
		case token.FLOAT:
			a.IsZeroVal = e.Value == "0.0" || e.Value == "0."
		case token.STRING:
			a.IsZeroVal = e.Value == `""` || e.Value == "``"
		}
	case *ast.Ident:
		a.IsVariable = true
		a.VarName = e.Name
		if e.Name == "nil" {
			a.IsNil = true
			a.IsZeroVal = true
			a.IsLiteral = true
		} else if e.Name == "true" || e.Name == "false" {
			a.IsLiteral = true
			a.IsZeroVal = e.Name == "false"
		}
	}

	return a
}

// extractAssignments finds multi-value assignments like `result, err := foo()`.
func extractAssignments(fset *token.FileSet, body *ast.BlockStmt, tParamName string, src []byte, tf *rules.TestFunc) {
	for _, stmt := range body.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok {
			continue
		}

		// Only interested in assignments from function calls
		if len(assign.Rhs) != 1 {
			continue
		}

		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			continue
		}

		ce := extractCallExpr(fset, call, tParamName, src)

		a := rules.Assignment{
			RHSCall: &ce,
			Line:    fset.Position(assign.Pos()).Line,
		}

		for _, lhs := range assign.Lhs {
			if ident, ok := lhs.(*ast.Ident); ok {
				a.LHS = append(a.LHS, ident.Name)
			}
		}

		// Heuristic: last variable is often the error
		if len(a.LHS) >= 2 {
			lastVar := a.LHS[len(a.LHS)-1]
			a.ErrorVarName = lastVar
			if lastVar == "_" {
				a.HasBlankError = true
			}
		}

		tf.Assignments = append(tf.Assignments, a)
	}
}

// extractErrorVarChecks determines which error variables are checked via assertions.
func extractErrorVarChecks(tf *rules.TestFunc) {
	for _, call := range tf.CallExprs {
		// Check assertion calls that reference error variables
		for _, arg := range call.Args {
			if arg.IsVariable {
				for _, assign := range tf.Assignments {
					if assign.ErrorVarName == arg.VarName && assign.ErrorVarName != "_" {
						if rules.IsAssertionCall(call) || isErrorCheckCall(call) {
							tf.ErrorVarsChecked[arg.VarName] = true
						}
					}
				}
			}
		}
	}
}

func isErrorCheckCall(call rules.CallExpr) bool {
	if call.Receiver == "assert" || call.Receiver == "require" {
		switch call.Function {
		case "NoError", "Error", "EqualError", "ErrorIs", "ErrorAs",
			"ErrorContains", "Nil", "NotNil":
			return true
		}
	}
	return false
}
