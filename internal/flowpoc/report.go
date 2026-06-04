package flowpoc

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/tools/flow"
)

type Report struct {
	SourcePackage string          `json:"sourcePackage"`
	SourceVersion string          `json:"sourceVersion"`
	Root          string          `json:"root"`
	Tasks         []TaskReport    `json:"tasks"`
	FinalValue    json.RawMessage `json:"finalValue"`
}

type TaskReport struct {
	Index        int      `json:"index"`
	Path         string   `json:"path"`
	State        string   `json:"state"`
	Dependencies []string `json:"dependencies"`
	Err          string   `json:"err,omitempty"`
}

func Run(repoRoot string) (*Report, error) {
	if repoRoot == "" {
		return nil, fmt.Errorf("repo root required")
	}

	appDir := filepath.Join(repoRoot, "cue", "flow", "app")
	insts := load.Instances([]string{"."}, &load.Config{Dir: appDir})
	if len(insts) != 1 {
		return nil, fmt.Errorf("expected one CUE instance, got %d", len(insts))
	}
	inst := insts[0]
	if err := inst.Err; err != nil {
		return nil, fmt.Errorf("load CUE app: %w", err)
	}

	ctx := cuecontext.New()
	root := ctx.BuildInstance(inst)
	if err := root.Err(); err != nil {
		return nil, fmt.Errorf("build CUE app: %w", err)
	}

	controller := flow.New(&flow.Config{
		Root:       cue.ParsePath("flow"),
		InferTasks: false,
	}, root, TaskFunc)

	if err := controller.Run(context.Background()); err != nil {
		return nil, fmt.Errorf("run flow: %w", err)
	}

	final := controller.Value()
	finalJSON, err := final.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal final value: %w", err)
	}

	report := &Report{
		SourcePackage: "cuelang.org/go/tools/flow",
		SourceVersion: "v0.6.0",
		Root:          "flow",
		FinalValue:    finalJSON,
	}

	for _, task := range controller.Tasks() {
		tr := TaskReport{
			Index: task.Index(),
			Path:  task.Path().String(),
			State: task.State().String(),
		}
		for _, dep := range task.Dependencies() {
			tr.Dependencies = append(tr.Dependencies, dep.Path().String())
		}
		if err := task.Err(); err != nil {
			tr.Err = err.Error()
		}
		report.Tasks = append(report.Tasks, tr)
	}

	return report, nil
}
