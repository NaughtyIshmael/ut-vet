package rules

import "fmt"

// Severity represents the priority level of a finding.
type Severity int

const (
	SeverityP0 Severity = iota // Critical — must fix
	SeverityP1                 // High value
	SeverityP2                 // Advanced / nice-to-have
)

func (s Severity) String() string {
	switch s {
	case SeverityP0:
		return "P0"
	case SeverityP1:
		return "P1"
	case SeverityP2:
		return "P2"
	default:
		return fmt.Sprintf("P%d", int(s))
	}
}

// Finding represents a single issue detected in a test function.
type Finding struct {
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Rule     string   `json:"rule"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	TestName string   `json:"test_name"`
}

func (f Finding) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s", f.File, f.Line, f.Rule, f.Message)
}

// Rule is the interface that all detection rules must implement.
type Rule interface {
	// ID returns the unique identifier for this rule (e.g., "no-assertion").
	ID() string

	// Description returns a human-readable description of what this rule detects.
	Description() string

	// Severity returns the severity level of findings from this rule.
	Severity() Severity

	// Analyze inspects a test function and returns any findings.
	Analyze(ctx *AnalysisContext) []Finding
}

// AnalysisContext provides the information a rule needs to analyze a test function.
type AnalysisContext struct {
	File     string
	TestFunc *TestFunc
}

// TestFunc represents a parsed test function.
type TestFunc struct {
	Name       string
	Line       int
	Body       []Statement
	CallExprs  []CallExpr
	HasBody    bool // false if body is empty (no statements at all)
	BodyLength int  // number of non-comment statements

	// P1 fields
	PackageName      string          // package name of the test file
	LocalFuncCalls   []string        // function calls without a receiver (same-package functions)
	Assignments      []Assignment    // variable assignments in the test body
	ErrorVarsChecked map[string]bool // error variables that are passed to an assertion

	// P2 fields
	TerminatingStatements []TerminatingStatement // t.Fatal, t.FailNow, return, etc.
}

// TerminatingStatement represents a statement that stops test execution.
type TerminatingStatement struct {
	Line int
	Kind string // "t.Fatal", "t.Fatalf", "t.FailNow", "return", "os.Exit"
}

// Assignment represents a variable assignment, e.g. `result, err := foo()`.
type Assignment struct {
	LHS           []string  // left-hand side variable names
	RHSCall       *CallExpr // the call on the right-hand side, if any
	HasBlankError bool      // true if error position is assigned to `_`
	ErrorVarName  string    // name of the error variable (or "_")
	Line          int
}

// Statement represents a statement in a test function body.
type Statement struct {
	Line    int
	Kind    StatementKind
	Content string // raw text representation
}

type StatementKind int

const (
	StmtComment StatementKind = iota
	StmtCall
	StmtAssign
	StmtExpr
	StmtOther
)

// CallExpr represents a function/method call in a test function.
type CallExpr struct {
	Line       int
	Receiver   string // e.g., "t", "assert", "require", "fmt"
	Function   string // e.g., "Error", "Equal", "Println"
	Args       []Arg  // the arguments passed to the call
	FullName   string // e.g., "t.Error", "assert.Equal", "fmt.Println"
	IsTestingT bool   // true if receiver is the *testing.T param
}

// Arg represents an argument passed to a function call.
type Arg struct {
	IsLiteral  bool // true if arg is a literal value (number, string, bool)
	IsNil      bool
	IsZeroVal  bool   // true if arg is a zero-value (0, "", false, nil)
	Value      string // string representation
	IsVariable bool   // true if arg is a simple variable reference
	VarName    string // variable name if IsVariable
}
