package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/NaughtyIshmael/ut-vet/pkg/analyzer"
	"github.com/NaughtyIshmael/ut-vet/pkg/mutate"
	"github.com/NaughtyIshmael/ut-vet/pkg/reporter"
	"github.com/NaughtyIshmael/ut-vet/pkg/rules"
)

var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) > 1 && os.Args[1] == "mutate" {
		return runMutate(os.Args[2:])
	}

	format := flag.String("format", "text", "Output format: text or json")
	rulesFlag := flag.String("rules", "", "Comma-separated rule IDs to enable (default: all)")
	exclude := flag.String("exclude", "", "Comma-separated glob patterns to exclude files")
	severity := flag.String("severity", "p0", "Minimum severity to report: p0, p1, or p2")
	listRules := flag.Bool("list-rules", false, "List all available rules")
	showVersion := flag.Bool("version", false, "Show version")
	verbose := flag.Bool("v", false, "Verbose output")
	quiet := flag.Bool("q", false, "Quiet mode (findings only)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ut-vet version %s\n", version)
		return 0
	}

	allRules := analyzer.AllRules()

	if *listRules {
		for _, r := range allRules {
			fmt.Printf("  %-25s [%s] %s\n", r.ID(), r.Severity(), r.Description())
		}
		return 0
	}

	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	maxSeverity, err := parseSeverity(*severity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	var enabledRules []rules.Rule
	if *rulesFlag != "" {
		ids := strings.Split(*rulesFlag, ",")
		idSet := make(map[string]bool)
		for _, id := range ids {
			idSet[strings.TrimSpace(id)] = true
		}
		for _, r := range allRules {
			if idSet[r.ID()] {
				enabledRules = append(enabledRules, r)
			}
		}
	} else {
		for _, r := range allRules {
			if r.Severity() <= maxSeverity {
				enabledRules = append(enabledRules, r)
			}
		}
	}

	a := analyzer.NewAnalyzer(enabledRules)
	if *exclude != "" {
		a.Exclude = strings.Split(*exclude, ",")
	}

	if *verbose && !*quiet {
		fmt.Fprintf(os.Stderr, "ut-vet: analyzing %s\n", strings.Join(paths, ", "))
	}

	findings, err := a.AnalyzePaths(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	var rep reporter.Reporter
	switch *format {
	case "json":
		rep = &reporter.JSONReporter{}
	default:
		rep = &reporter.TextReporter{Verbose: *verbose && !*quiet}
	}

	output, err := rep.Report(findings)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	fmt.Print(output)

	if len(findings) > 0 {
		return 1
	}
	return 0
}

func runMutate(args []string) int {
	fs := flag.NewFlagSet("mutate", flag.ExitOnError)
	tool := fs.String("tool", "", "Mutation tool: gremlins or cargo-mutants (auto-detected if omitted)")
	threshold := fs.Float64("threshold", 0, "Minimum mutation score (0.0-1.0)")
	jsonOutput := fs.Bool("json", false, "Output results as JSON")
	verbose := fs.Bool("v", false, "Verbose output (show killed mutants)")
	fs.Parse(args)

	path := "."
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}

	if *tool != "" {
		if err := mutate.CheckToolInstalled(*tool); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 2
		}
	}

	opts := mutate.Options{
		Tool:      *tool,
		Threshold: *threshold,
		Verbose:   *verbose,
		JSON:      *jsonOutput,
	}

	result, err := mutate.Run(path, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}

	if *jsonOutput {
		out, err := mutate.FormatJSON(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 2
		}
		fmt.Println(out)
	} else {
		fmt.Print(mutate.FormatText(result, *verbose))
	}

	if *threshold > 0 && result.Score < *threshold {
		return 1
	}
	if result.Survived > 0 {
		return 1
	}
	return 0
}

func parseSeverity(s string) (rules.Severity, error) {
	switch strings.ToLower(s) {
	case "p0":
		return rules.SeverityP0, nil
	case "p1":
		return rules.SeverityP1, nil
	case "p2":
		return rules.SeverityP2, nil
	default:
		return 0, fmt.Errorf("unknown severity: %q (use p0, p1, or p2)", s)
	}
}
