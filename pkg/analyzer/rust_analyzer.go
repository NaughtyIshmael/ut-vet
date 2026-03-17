package analyzer

import (
	"regexp"
	"strings"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

// Rust assertion macros
var rustAssertMacros = map[string]bool{
	"assert!":          true,
	"assert_eq!":       true,
	"assert_ne!":       true,
	"debug_assert!":    true,
	"debug_assert_eq!": true,
	"debug_assert_ne!": true,
}

// Rust method calls that act as implicit assertions (panic on failure)
var rustAssertMethods = map[string]bool{
	"unwrap": true,
	"expect": true,
}

// Rust method calls that swallow errors (NOT assertions)
var rustErrorSwallowMethods = map[string]bool{
	"unwrap_or":         true,
	"unwrap_or_default": true,
	"unwrap_or_else":    true,
	"ok":                true,
}

// Rust log/print macros
var rustLogMacros = map[string]bool{
	"println!":  true,
	"print!":    true,
	"eprintln!": true,
	"eprint!":   true,
	"dbg!":      true,
	"log!":      true,
	"info!":     true,
	"debug!":    true,
	"warn!":     true,
	"error!":    true,
	"trace!":    true,
}

// Regex patterns for Rust test parsing
var (
	// Matches #[test], #[tokio::test], #[actix_web::test], etc.
	testAttrRe = regexp.MustCompile(`^\s*#\[(tokio::test|actix_web::test|test)\]`)
	// Matches #[should_panic] or #[should_panic(expected = "...")]
	shouldPanicRe = regexp.MustCompile(`^\s*#\[should_panic`)
	// Matches fn declarations: fn test_name() or async fn test_name()
	fnDeclRe = regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+(\w+)\s*\(`)
	// Matches macro calls: assert_eq!(args), println!(args), etc.
	macroCallRe = regexp.MustCompile(`\b(\w+!)\s*\(`)
	// Matches function calls: func_name(args) — not macros
	funcCallRe = regexp.MustCompile(`\b([a-z_]\w*)\s*\(`)
	// Matches method calls: expr.method(args)
	methodCallRe = regexp.MustCompile(`\.(\w+)\s*\(`)
	// Matches Rust comments
	commentRe = regexp.MustCompile(`^\s*//`)
)

// isRustAssertionCall returns true if the CallExpr is a Rust assertion.
func isRustAssertionCall(ce rules.CallExpr) bool {
	if rustAssertMacros[ce.Function] {
		return true
	}
	// .unwrap() and .expect() are implicit assertions (panic on failure)
	if rustAssertMethods[ce.Function] && ce.Receiver != "" {
		return true
	}
	return false
}

// ParseRustTestFile parses a Rust source file and extracts test functions.
func ParseRustTestFile(path string, src []byte) ([]*rules.TestFunc, error) {
	lines := strings.Split(string(src), "\n")
	var testFuncs []*rules.TestFunc

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Look for #[test] or #[tokio::test] attribute
		if !testAttrRe.MatchString(line) {
			i++
			continue
		}

		// Found a test attribute — collect additional attributes
		hasShouldPanic := false
		attrStart := i
		i++

		// Scan for more attributes and the fn declaration
		for i < len(lines) {
			l := strings.TrimSpace(lines[i])
			if shouldPanicRe.MatchString(lines[i]) {
				hasShouldPanic = true
				i++
				continue
			}
			if testAttrRe.MatchString(lines[i]) {
				i++
				continue
			}
			if strings.HasPrefix(l, "#[") {
				i++
				continue
			}
			break
		}

		if i >= len(lines) {
			break
		}

		// Now expect fn declaration
		fnMatch := fnDeclRe.FindStringSubmatch(lines[i])
		if fnMatch == nil {
			i++
			continue
		}

		fnName := fnMatch[1]
		fnLine := attrStart + 1 // 1-based line number of #[test] attribute

		// Extract function body by counting braces
		bodyLines, endIdx := extractRustFnBody(lines, i)
		// bodyStartLine is the 1-based line number of the first body line
		bodyStartLine := i + 2 // fn decl is at index i, body starts after {
		i = endIdx + 1

		tf := buildRustTestFunc(fnName, fnLine, bodyLines, bodyStartLine, hasShouldPanic)
		testFuncs = append(testFuncs, tf)
	}

	return testFuncs, nil
}

// extractRustFnBody extracts the body of a Rust function by counting braces.
// Skips braces inside string literals and comments.
// Returns the body lines (between outer braces) and the index of the closing brace.
func extractRustFnBody(lines []string, fnDeclLine int) ([]string, int) {
	braceCount := 0
	started := false
	var bodyLines []string
	bodyStart := -1

	for i := fnDeclLine; i < len(lines); i++ {
		line := lines[i]
		inString := false
		escaped := false

		for j := 0; j < len(line); j++ {
			ch := line[j]

			if escaped {
				escaped = false
				continue
			}

			if ch == '\\' && inString {
				escaped = true
				continue
			}

			// Toggle string context on unescaped quote
			if ch == '"' {
				inString = !inString
				continue
			}

			// Skip line comments
			if !inString && ch == '/' && j+1 < len(line) && line[j+1] == '/' {
				break // rest of line is a comment
			}

			if inString {
				continue
			}

			if ch == '{' {
				if !started {
					started = true
					bodyStart = i
				}
				braceCount++
			} else if ch == '}' {
				braceCount--
				if started && braceCount == 0 {
					if bodyStart >= 0 && i > bodyStart {
						bodyLines = lines[bodyStart+1 : i]
					} else if bodyStart == i {
						bodyLines = nil
					}
					return bodyLines, i
				}
			}
		}
	}

	if bodyStart >= 0 && bodyStart+1 < len(lines) {
		return lines[bodyStart+1:], len(lines) - 1
	}
	return nil, len(lines) - 1
}

// buildRustTestFunc constructs a TestFunc from parsed Rust function body.
func buildRustTestFunc(name string, line int, bodyLines []string, bodyStartLine int, hasShouldPanic bool) *rules.TestFunc {
	tf := &rules.TestFunc{
		Name:             name,
		Line:             line,
		HasBody:          false,
		BodyLength:       0,
		ErrorVarsChecked: make(map[string]bool),
	}

	// Build filtered lines with their original line numbers
	var codeLines []numberedLine
	for i, bl := range bodyLines {
		trimmed := strings.TrimSpace(bl)
		if trimmed == "" || commentRe.MatchString(bl) {
			continue
		}
		codeLines = append(codeLines, numberedLine{text: bl, lineNum: bodyStartLine + i})
	}

	tf.BodyLength = len(codeLines)
	tf.HasBody = tf.BodyLength > 0

	// Build body statements with correct line numbers
	for i, bl := range bodyLines {
		trimmed := strings.TrimSpace(bl)
		kind := rules.StmtOther
		if commentRe.MatchString(bl) {
			kind = rules.StmtComment
		} else if trimmed == "" {
			continue
		}
		tf.Body = append(tf.Body, rules.Statement{
			Line:    bodyStartLine + i,
			Kind:    kind,
			Content: trimmed,
		})
	}

	// Join multi-line macro calls before extracting call expressions
	joinedLines := joinMultiLineMacros(codeLines)

	tf.CallExprs = extractRustCallExprs(joinedLines)

	// Extract local function calls (excluding assertion/log macros and well-known stdlib)
	for _, ce := range tf.CallExprs {
		if ce.Receiver == "" && !rustAssertMacros[ce.Function] && !rustLogMacros[ce.Function] {
			funcName := strings.TrimSuffix(ce.Function, "!")
			if funcName != "" {
				tf.LocalFuncCalls = append(tf.LocalFuncCalls, funcName)
			}
		}
	}

	// Extract assignments with correct line numbers
	extractRustAssignments(joinedLines, tf)

	// If #[should_panic], inject a synthetic assertion call
	if hasShouldPanic {
		tf.CallExprs = append(tf.CallExprs, rules.CallExpr{
			Line:     line,
			Function: "assert!",
			FullName: "should_panic",
			Args:     []rules.Arg{{Value: "panic_expected", IsVariable: true, VarName: "panic_expected"}},
		})
	}

	// Extract terminating statements at top-level only (brace depth 0)
	braceDepth := 0
	for _, cl := range codeLines {
		trimmed := strings.TrimSpace(cl.text)
		// Track brace depth to avoid flagging returns inside if/match
		for _, ch := range trimmed {
			if ch == '{' {
				braceDepth++
			} else if ch == '}' {
				braceDepth--
			}
		}
		if braceDepth > 0 {
			continue
		}
		if strings.HasPrefix(trimmed, "panic!") || strings.HasPrefix(trimmed, "unreachable!") {
			tf.TerminatingStatements = append(tf.TerminatingStatements, rules.TerminatingStatement{
				Line: cl.lineNum,
				Kind: "panic!",
			})
		} else if strings.HasPrefix(trimmed, "return") {
			tf.TerminatingStatements = append(tf.TerminatingStatements, rules.TerminatingStatement{
				Line: cl.lineNum,
				Kind: "return",
			})
		}
	}

	return tf
}

// numberedLine pairs a line of text with its original file line number.
type numberedLine struct {
	text    string
	lineNum int
}

// joinMultiLineMacros joins consecutive lines when a macro call's parentheses
// are not closed on a single line.
func joinMultiLineMacros(lines []numberedLine) []numberedLine {
	var result []numberedLine
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i].text)
		// Check if this line has an unclosed paren from a macro/function call
		depth := countParenDepth(trimmed)
		if depth <= 0 {
			result = append(result, lines[i])
			i++
			continue
		}
		// Join subsequent lines until parens are balanced
		joined := trimmed
		lineNum := lines[i].lineNum
		i++
		for i < len(lines) && depth > 0 {
			nextTrimmed := strings.TrimSpace(lines[i].text)
			joined += " " + nextTrimmed
			depth += countParenDepth(nextTrimmed)
			i++
		}
		result = append(result, numberedLine{text: joined, lineNum: lineNum})
	}
	return result
}

// countParenDepth returns the net paren depth change for a line,
// ignoring parens inside string literals.
func countParenDepth(s string) int {
	depth := 0
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '/' && i+1 < len(s) && s[i+1] == '/' {
			break
		}
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
		}
	}
	return depth
}

// extractRustCallExprs extracts macro and function calls from Rust source lines.
func extractRustCallExprs(lines []numberedLine) []rules.CallExpr {
	var calls []rules.CallExpr
	seen := make(map[string]bool)

	for _, nl := range lines {
		lineNum := nl.lineNum
		trimmed := strings.TrimSpace(nl.text)

		// Find macro calls (assert!, println!, etc.)
		macroMatches := macroCallRe.FindAllStringSubmatchIndex(trimmed, -1)
		for _, loc := range macroMatches {
			macroName := trimmed[loc[2]:loc[3]]
			argsStart := loc[1]

			args := extractRustMacroArgs(trimmed, argsStart)
			ce := rules.CallExpr{
				Line:     lineNum,
				Function: macroName,
				FullName: macroName,
				Args:     args,
			}
			key := macroName + "@" + strings.Join(argValues(args), ",")
			if !seen[key] {
				calls = append(calls, ce)
				seen[key] = true
			}
		}

		// Find method calls: .method(args)
		methodMatchIndices := methodCallRe.FindAllStringSubmatchIndex(trimmed, -1)
		for _, loc := range methodMatchIndices {
			methodName := trimmed[loc[2]:loc[3]]
			if strings.Contains(methodName, "!") {
				continue
			}
			dotPos := loc[0]
			receiver := extractRustReceiver(trimmed, dotPos)
			argsStart := loc[1]
			args := extractRustMacroArgs(trimmed, argsStart)
			ce := rules.CallExpr{
				Line:     lineNum,
				Function: methodName,
				Receiver: receiver,
				FullName: receiver + "." + methodName,
				Args:     args,
			}
			key := receiver + "." + methodName + "@" + strings.Join(argValues(args), ",")
			if !seen[key] {
				calls = append(calls, ce)
				seen[key] = true
			}
		}

		// Find bare function calls (not macros, not methods)
		if !macroCallRe.MatchString(trimmed) {
			funcMatchIndices := funcCallRe.FindAllStringSubmatchIndex(trimmed, -1)
			for _, loc := range funcMatchIndices {
				funcName := trimmed[loc[2]:loc[3]]
				if funcName == "let" || funcName == "if" || funcName == "for" || funcName == "while" || funcName == "match" || funcName == "fn" || funcName == "use" || funcName == "mod" {
					continue
				}
				if loc[0] > 0 && trimmed[loc[0]-1] == '.' {
					continue
				}
				argsStart := loc[1]
				args := extractRustMacroArgs(trimmed, argsStart)
				ce := rules.CallExpr{
					Line:     lineNum,
					Function: funcName,
					FullName: funcName,
					Args:     args,
				}
				key := funcName + "@" + strings.Join(argValues(args), ",")
				if !seen[key] {
					calls = append(calls, ce)
					seen[key] = true
				}
			}
		}
	}

	return calls
}

// extractRustMacroArgs parses the arguments from a macro call.
func extractRustMacroArgs(line string, startIdx int) []rules.Arg {
	depth := 1
	end := startIdx
	inString := false
	escaped := false

	for end < len(line) && depth > 0 {
		ch := line[end]
		if escaped {
			escaped = false
			end++
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			end++
			continue
		}
		if ch == '"' {
			inString = !inString
			end++
			continue
		}
		if !inString {
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
			}
		}
		if depth > 0 {
			end++
		}
	}

	if end <= startIdx {
		return nil
	}

	argStr := strings.TrimSpace(line[startIdx:end])
	if argStr == "" {
		return nil
	}

	// Split by comma, respecting parentheses
	parts := splitRustArgs(argStr)
	var args []rules.Arg
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		args = append(args, classifyRustArg(part))
	}
	return args
}

// splitRustArgs splits a comma-separated argument string, respecting parens,
// braces, and string literals.
func splitRustArgs(s string) []string {
	var parts []string
	depth := 0
	start := 0
	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inString {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// classifyRustArg determines if an argument is a literal, variable, nil, or zero-value.
func classifyRustArg(s string) rules.Arg {
	s = strings.TrimSpace(s)
	arg := rules.Arg{Value: s}

	// Check for format strings (starts with " and contains {})
	if strings.HasPrefix(s, "\"") || strings.HasPrefix(s, "r\"") || strings.HasPrefix(s, "r#\"") {
		arg.IsLiteral = true
		// Check for empty string
		if s == `""` {
			arg.IsZeroVal = true
		}
		return arg
	}

	// Numeric literals
	if isRustNumericLiteral(s) {
		arg.IsLiteral = true
		if s == "0" || s == "0.0" || s == "0_i32" || s == "0_u32" || s == "0_usize" {
			arg.IsZeroVal = true
		}
		return arg
	}

	// Boolean literals
	if s == "true" || s == "false" {
		arg.IsLiteral = true
		if s == "false" {
			arg.IsZeroVal = true
		}
		return arg
	}

	// None (Rust's nil equivalent)
	if s == "None" {
		arg.IsNil = true
		arg.IsZeroVal = true
		return arg
	}

	// Simple variable reference (identifier)
	if isRustIdentifier(s) {
		arg.IsVariable = true
		arg.VarName = s
		return arg
	}

	// Method chain on a variable: result.name, result.unwrap()
	if parts := strings.SplitN(s, ".", 2); len(parts) == 2 && isRustIdentifier(parts[0]) {
		arg.IsVariable = true
		arg.VarName = s
		return arg
	}

	return arg
}

// isRustNumericLiteral checks if a string is a Rust numeric literal.
func isRustNumericLiteral(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Simple check: starts with digit or minus+digit
	start := 0
	if s[0] == '-' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	if s[start] < '0' || s[start] > '9' {
		return false
	}
	// Rest can be digits, underscores, dots, or type suffixes
	for i := start + 1; i < len(s); i++ {
		ch := s[i]
		if (ch >= '0' && ch <= '9') || ch == '_' || ch == '.' || (ch >= 'a' && ch <= 'z') {
			continue
		}
		return false
	}
	return true
}

// isRustIdentifier checks if a string is a valid Rust identifier.
func isRustIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, ch := range s {
		if i == 0 {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
				return false
			}
		} else {
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
				return false
			}
		}
	}
	return true
}

// extractRustAssignments extracts let bindings from Rust source and tracks error patterns.
func extractRustAssignments(lines []numberedLine, tf *rules.TestFunc) {
	letRe := regexp.MustCompile(`^\s*let\s+(?:mut\s+)?(_|\w+)\s*=\s*(.+?)\s*;?\s*$`)
	destructRe := regexp.MustCompile(`^\s*let\s+\(([^)]+)\)\s*=\s*(.+?)\s*;?\s*$`)

	for _, nl := range lines {
		line := nl.text
		lineNum := nl.lineNum

		// Destructuring let
		if dm := destructRe.FindStringSubmatch(line); dm != nil {
			vars := strings.Split(dm[1], ",")
			var lhs []string
			for _, v := range vars {
				lhs = append(lhs, strings.TrimSpace(v))
			}
			rhs := dm[2]
			a := rules.Assignment{
				LHS:  lhs,
				Line: lineNum,
			}
			if fcMatch := funcCallRe.FindStringSubmatch(rhs); fcMatch != nil {
				a.RHSCall = &rules.CallExpr{
					Function: fcMatch[1],
					FullName: fcMatch[1],
				}
			}
			tf.Assignments = append(tf.Assignments, a)
			continue
		}

		// Simple let
		m := letRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		varName := m[1]
		rhs := m[2]

		a := rules.Assignment{
			LHS:  []string{varName},
			Line: lineNum,
		}

		// Check if RHS is a function call
		if fcMatch := funcCallRe.FindStringSubmatch(rhs); fcMatch != nil {
			a.RHSCall = &rules.CallExpr{
				Function: fcMatch[1],
				FullName: fcMatch[1],
			}
		}

		// Track error swallowing patterns
		if varName == "_" {
			// let _ = fallible() — error/result discarded
			a.HasBlankError = true
			a.ErrorVarName = "_"
		}

		// Track .unwrap_or_default(), .ok(), etc. as error-swallowing
		for method := range rustErrorSwallowMethods {
			if strings.Contains(rhs, "."+method+"(") {
				a.HasBlankError = true
				a.ErrorVarName = "_"
			}
		}

		tf.Assignments = append(tf.Assignments, a)
	}

	// Track which error-like variables are checked in assertions
	extractRustErrorVarChecks(tf)
}

// extractRustErrorVarChecks marks error variables that appear in assertion calls.
func extractRustErrorVarChecks(tf *rules.TestFunc) {
	for _, ce := range tf.CallExprs {
		if !rules.IsAssertionCall(ce) {
			continue
		}
		for _, arg := range ce.Args {
			if arg.IsVariable {
				tf.ErrorVarsChecked[arg.VarName] = true
			}
		}
	}
}

func argValues(args []rules.Arg) []string {
	var vals []string
	for _, a := range args {
		vals = append(vals, a.Value)
	}
	return vals
}

// extractRustReceiver extracts the receiver expression before a dot at position dotPos.
func extractRustReceiver(line string, dotPos int) string {
	if dotPos <= 0 {
		return "self"
	}
	// Walk backwards to find the receiver identifier
	end := dotPos
	// Skip closing parens (for chained calls like foo().method())
	if end > 0 && line[end-1] == ')' {
		depth := 1
		end--
		for end > 0 && depth > 0 {
			end--
			if line[end] == ')' {
				depth++
			} else if line[end] == '(' {
				depth--
			}
		}
	}
	// Now find the identifier
	i := end - 1
	for i >= 0 && (isIdentChar(line[i])) {
		i--
	}
	receiver := line[i+1 : end]
	if receiver == "" {
		return "self"
	}
	return receiver
}

func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// isRustTestFile checks if a file path is a Rust file.
func isRustTestFile(path string) bool {
	return strings.HasSuffix(path, ".rs")
}
