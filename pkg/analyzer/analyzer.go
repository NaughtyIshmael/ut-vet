package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

// Analyzer orchestrates the analysis of test files.
type Analyzer struct {
	Rules   []rules.Rule
	Exclude []string // glob patterns to exclude
}

// NewAnalyzer creates an analyzer with the given rules.
func NewAnalyzer(enabledRules []rules.Rule) *Analyzer {
	return &Analyzer{
		Rules: enabledRules,
	}
}

// DefaultRules returns all P0 rules.
func DefaultRules() []rules.Rule {
	return []rules.Rule{
		&rules.EmptyTestRule{},
		&rules.NoAssertionRule{},
		&rules.LogOnlyRule{},
		&rules.TrivialAssertRule{},
	}
}

// AllRules returns all available rules.
func AllRules() []rules.Rule {
	return []rules.Rule{
		// P0
		&rules.EmptyTestRule{},
		&rules.NoAssertionRule{},
		&rules.LogOnlyRule{},
		&rules.TrivialAssertRule{},
		// P1
		&rules.ErrorNotCheckedRule{},
		&rules.NoCodeUnderTestRule{},
		&rules.ZeroValueInputRule{},
		&rules.OnlyNilCheckRule{},
		// P2
		&rules.TautologicalAssertRule{},
		&rules.DeadAssertionRule{},
		&rules.NoArrangeRule{},
	}
}

// AnalyzePaths analyzes Go test files at the given paths.
func (a *Analyzer) AnalyzePaths(paths []string) ([]rules.Finding, error) {
	var allFindings []rules.Finding

	for _, p := range paths {
		findings, err := a.analyzePath(p)
		if err != nil {
			return nil, fmt.Errorf("analyzing %s: %w", p, err)
		}
		allFindings = append(allFindings, findings...)
	}

	return allFindings, nil
}

func (a *Analyzer) analyzePath(path string) ([]rules.Finding, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.IsDir() {
		return a.analyzeDir(path)
	}

	if isGoTestFile(path) {
		return a.analyzeGoFile(path)
	}

	if isRustTestFile(path) {
		return a.analyzeRustFile(path)
	}

	return nil, nil
}

func (a *Analyzer) analyzeDir(dir string) ([]rules.Finding, error) {
	var allFindings []rules.Finding

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip hidden directories and vendor
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "vendor" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		if !isGoTestFile(path) && !isRustTestFile(path) {
			return nil
		}

		if a.isExcluded(path) {
			return nil
		}

		findings, findErr := a.analyzeFile(path)
		if findErr != nil {
			return fmt.Errorf("analyzing %s: %w", path, findErr)
		}
		allFindings = append(allFindings, findings...)
		return nil
	})

	return allFindings, err
}

func (a *Analyzer) analyzeGoFile(path string) ([]rules.Finding, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	testFuncs, err := ParseGoTestFile(path, src)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return a.runRules(path, testFuncs), nil
}

func (a *Analyzer) analyzeRustFile(path string) ([]rules.Finding, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	testFuncs, err := ParseRustTestFile(path, src)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return a.runRules(path, testFuncs), nil
}

func (a *Analyzer) analyzeFile(path string) ([]rules.Finding, error) {
	if isGoTestFile(path) {
		return a.analyzeGoFile(path)
	}
	if isRustTestFile(path) {
		return a.analyzeRustFile(path)
	}
	return nil, nil
}

func (a *Analyzer) runRules(path string, testFuncs []*rules.TestFunc) []rules.Finding {
	var findings []rules.Finding
	for _, tf := range testFuncs {
		ctx := &rules.AnalysisContext{
			File:     path,
			TestFunc: tf,
		}
		for _, rule := range a.Rules {
			findings = append(findings, rule.Analyze(ctx)...)
		}
	}
	return findings
}

func (a *Analyzer) isExcluded(path string) bool {
	for _, pattern := range a.Exclude {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	return false
}

func isGoTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go")
}
