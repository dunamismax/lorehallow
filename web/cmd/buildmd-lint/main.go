// Command buildmd-lint fails on any BUILD.md format-contract violation.
//
// Usage: buildmd-lint [path/to/BUILD.md]   (default ../BUILD.md, run from web/)
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dunamismax/lorehallow/web/internal/buildmd"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	path := "../BUILD.md"
	if len(args) > 0 {
		path = args[0]
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "buildmd-lint: %v\n", err)
		return 1
	}
	plan := buildmd.Parse(string(data))
	for _, p := range plan.Problems {
		fmt.Fprintf(os.Stderr, "%s:%d: %s\n", path, p.Line, p.Message)
	}
	if len(plan.Phases) == 0 {
		fmt.Fprintf(os.Stderr, "%s: no phases found\n", path)
		return 1
	}
	if len(plan.Problems) > 0 {
		fmt.Fprintf(os.Stderr, "buildmd-lint: %d problem(s)\n", len(plan.Problems))
		return 1
	}
	pr := plan.Progress()
	msg := fmt.Sprintf("buildmd-lint: ok, %d phases, %d/%d items checked", len(plan.Phases), pr.Done, pr.Total)
	if c := plan.Current(); c != nil {
		msg += fmt.Sprintf(", current phase %d (%s)", c.Number, c.Title)
	} else {
		msg += ", all phases done"
	}
	fmt.Println(msg)
	return 0
}
