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
	return rustAssertMacros[ce.Function]
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
		fnLine := attrStart + 1 // 1-based line number

		// Extract function body by counting braces
		bodyLines, endIdx := extractRustFnBody(lines, i)
		i = endIdx + 1

		tf := buildRustTestFunc(fnName, fnLine, bodyLines, hasShouldPanic)
		testFuncs = append(testFuncs, tf)
	}

	return testFuncs, nil
}

// extractRustFnBody extracts the body of a Rust function by counting braces.
// Returns the body lines (between outer braces) and the index of the closing brace.
func extractRustFnBody(lines []string, fnDeclLine int) ([]string, int) {
	braceCount := 0
	started := false
	var bodyLines []string
	bodyStart := -1

	for i := fnDeclLine; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			if ch == '{' {
				if !started {
					started = true
					bodyStart = i
				}
				braceCount++
			} else if ch == '}' {
				braceCount--
				if started && braceCount == 0 {
					// Collect lines between opening and closing brace
					if bodyStart >= 0 && i > bodyStart {
						bodyLines = lines[bodyStart+1 : i]
					} else if bodyStart == i {
						// Single-line fn: fn test() {}
						bodyLines = nil
					}
					return bodyLines, i
				}
			}
		}
	}

	// Unclosed brace — return what we have
	if bodyStart >= 0 && bodyStart+1 < len(lines) {
		return lines[bodyStart+1:], len(lines) - 1
	}
	return nil, len(lines) - 1
}

// buildRustTestFunc constructs a TestFunc from parsed Rust function body.
func buildRustTestFunc(name string, line int, bodyLines []string, hasShouldPanic bool) *rules.TestFunc {
	tf := &rules.TestFunc{
		Name:             name,
		Line:             line,
		HasBody:          false,
		BodyLength:       0,
		ErrorVarsChecked: make(map[string]bool),
	}

	// Count non-comment, non-empty lines
	var nonCommentLines []string
	for _, bl := range bodyLines {
		trimmed := strings.TrimSpace(bl)
		if trimmed == "" || commentRe.MatchString(bl) {
			continue
		}
		nonCommentLines = append(nonCommentLines, bl)
	}

	tf.BodyLength = len(nonCommentLines)
	tf.HasBody = tf.BodyLength > 0

	// Build body statements
	for i, bl := range bodyLines {
		trimmed := strings.TrimSpace(bl)
		kind := rules.StmtOther
		if commentRe.MatchString(bl) {
			kind = rules.StmtComment
		} else if trimmed == "" {
			continue
		}
		tf.Body = append(tf.Body, rules.Statement{
			Line:    line + i + 1,
			Kind:    kind,
			Content: trimmed,
		})
	}

	// Extract call expressions from body
	fullBody := strings.Join(bodyLines, "\n")
	tf.CallExprs = extractRustCallExprs(nonCommentLines, line)

	// Extract local function calls
	for _, ce := range tf.CallExprs {
		if ce.Receiver == "" && !rustAssertMacros[ce.Function] && !rustLogMacros[ce.Function] {
			// Strip trailing ! for macro-style local calls
			funcName := strings.TrimSuffix(ce.Function, "!")
			tf.LocalFuncCalls = append(tf.LocalFuncCalls, funcName)
		}
	}

	// Extract assignments
	extractRustAssignments(nonCommentLines, line, tf)

	// If #[should_panic], inject a synthetic assertion call
	if hasShouldPanic {
		tf.CallExprs = append(tf.CallExprs, rules.CallExpr{
			Line:     line,
			Function: "assert!",
			FullName: "should_panic",
			Args:     []rules.Arg{{Value: "panic_expected", IsVariable: true, VarName: "panic_expected"}},
		})
	}

	// Extract terminating statements (panic!, return, unreachable!)
	for _, bl := range nonCommentLines {
		trimmed := strings.TrimSpace(bl)
		if strings.HasPrefix(trimmed, "panic!") {
			tf.TerminatingStatements = append(tf.TerminatingStatements, rules.TerminatingStatement{
				Line: line,
				Kind: "panic!",
			})
		} else if strings.HasPrefix(trimmed, "return") {
			tf.TerminatingStatements = append(tf.TerminatingStatements, rules.TerminatingStatement{
				Line: line,
				Kind: "return",
			})
		}
	}

	_ = fullBody
	return tf
}

// extractRustCallExprs extracts macro and function calls from Rust source lines.
func extractRustCallExprs(lines []string, baseLine int) []rules.CallExpr {
	var calls []rules.CallExpr
	seen := make(map[string]bool)

	for i, line := range lines {
		lineNum := baseLine + i + 1
		trimmed := strings.TrimSpace(line)

		// Find macro calls (assert!, println!, etc.)
		macroMatches := macroCallRe.FindAllStringSubmatchIndex(trimmed, -1)
		for _, loc := range macroMatches {
			macroName := trimmed[loc[2]:loc[3]]
			argsStart := loc[1] // position after the opening paren (end of full match)

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

		// Find method calls: .method(args) — gives receiver context
		methodMatches := methodCallRe.FindAllStringSubmatch(trimmed, -1)
		for _, m := range methodMatches {
			methodName := m[1]
			// Skip if it's a macro (already handled)
			if strings.Contains(methodName, "!") {
				continue
			}
			ce := rules.CallExpr{
				Line:     lineNum,
				Function: methodName,
				Receiver: "self", // simplified
				FullName: "." + methodName,
			}
			key := "." + methodName
			if !seen[key] {
				calls = append(calls, ce)
				seen[key] = true
			}
		}

		// Find bare function calls (not macros, not methods)
		// Only if line is not dominated by a macro call
		if !macroCallRe.MatchString(trimmed) {
			funcMatches := funcCallRe.FindAllStringSubmatch(trimmed, -1)
			for _, m := range funcMatches {
				funcName := m[1]
				if funcName == "let" || funcName == "if" || funcName == "for" || funcName == "while" || funcName == "match" || funcName == "fn" {
					continue
				}
				ce := rules.CallExpr{
					Line:     lineNum,
					Function: funcName,
					FullName: funcName,
				}
				key := funcName
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
	// Find the matching closing paren
	depth := 1
	end := startIdx
	for end < len(line) && depth > 0 {
		if line[end] == '(' {
			depth++
		} else if line[end] == ')' {
			depth--
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

// splitRustArgs splits a comma-separated argument string, respecting parens and braces.
func splitRustArgs(s string) []string {
	var parts []string
	depth := 0
	start := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
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

// extractRustAssignments extracts let bindings from Rust source.
func extractRustAssignments(lines []string, baseLine int, tf *rules.TestFunc) {
	letRe := regexp.MustCompile(`^\s*let\s+(?:mut\s+)?(_|\w+)\s*=\s*(.+?)\s*;?\s*$`)

	for i, line := range lines {
		lineNum := baseLine + i + 1
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

		tf.Assignments = append(tf.Assignments, a)
	}
}

func argValues(args []rules.Arg) []string {
	var vals []string
	for _, a := range args {
		vals = append(vals, a.Value)
	}
	return vals
}

// isRustTestFile checks if a file path is a Rust file.
func isRustTestFile(path string) bool {
	return strings.HasSuffix(path, ".rs")
}
