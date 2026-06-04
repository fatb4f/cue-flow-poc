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
	Agent         AgentReport     `json:"agent"`
	Runner        RunnerReport    `json:"runner"`
	Flow          FlowReport      `json:"flow"`
	Tasks         []TaskReport    `json:"tasks"`
	FinalValue    json.RawMessage `json:"finalValue"`
}

type AgentReport struct {
	Role         string `json:"role"`
	ProposedFill bool   `json:"proposedFill"`
	OwnsPolicy   bool   `json:"ownsPolicy"`
}

type RunnerReport struct {
	Role           string `json:"role"`
	ValidatedFill  bool   `json:"validatedFill"`
	CalledTaskFill bool   `json:"calledTaskFill"`
	OwnsPolicy     bool   `json:"ownsPolicy"`
}

type FlowReport struct {
	Terminated             bool `json:"terminated"`
	FinalValueContainsFill bool `json:"finalValueContainsFill"`
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

	report := &Report{
		SourcePackage: "cuelang.org/go/tools/flow",
		SourceVersion: "v0.6.0",
		Root:          "flow",
		Agent: AgentReport{
			Role:       "semantic-runner",
			OwnsPolicy: false,
		},
		Runner: RunnerReport{
			Role:       "go-flow-runner",
			OwnsPolicy: false,
		},
	}

	controller := flow.New(&flow.Config{
		Root:       cue.ParsePath("flow"),
		InferTasks: false,
	}, root, TaskFuncFor(report))

	if err := controller.Run(context.Background()); err != nil {
		return nil, fmt.Errorf("run flow: %w", err)
	}

	final := controller.Value()
	finalJSON, err := final.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal final value: %w", err)
	}

	report.FinalValue = finalJSON

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
	report.Flow.Terminated = allTasksTerminated(report.Tasks)
	report.Flow.FinalValueContainsFill = finalValueContainsFill(final)

	return report, nil
}

func allTasksTerminated(tasks []TaskReport) bool {
	if len(tasks) == 0 {
		return false
	}
	for _, task := range tasks {
		if task.State != "Terminated" || task.Err != "" {
			return false
		}
	}
	return true
}

func finalValueContainsFill(v cue.Value) bool {
	firstMessage, err := v.LookupPath(cue.ParsePath("report.first.message")).String()
	if err != nil || firstMessage == "" {
		return false
	}
	firstOK, err := v.LookupPath(cue.ParsePath("report.first.ok")).Bool()
	if err != nil || !firstOK {
		return false
	}
	secondMessage, err := v.LookupPath(cue.ParsePath("report.second.message")).String()
	if err != nil || secondMessage == "" {
		return false
	}
	secondOK, err := v.LookupPath(cue.ParsePath("report.second.ok")).Bool()
	return err == nil && secondOK
}
