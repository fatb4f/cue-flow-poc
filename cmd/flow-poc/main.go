package main

import (
	"encoding/json"
	"fmt"
	"os"

	"cue-flow-poc/internal/flowpoc"
)

func main() {
	repoRoot := "."
	if len(os.Args) > 1 {
		repoRoot = os.Args[1]
	}

	report, err := flowpoc.Run(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "flow-poc: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal report: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
