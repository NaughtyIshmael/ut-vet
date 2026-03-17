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

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		if !isTestFunc(fn) {
			continue
		}

		tf := extractTestFunc(fset, fn, src)
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
