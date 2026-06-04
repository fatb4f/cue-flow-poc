package flowpoc

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/tools/flow"
)

// TaskFuncFor classifies only values marked with $id. CUE owns admissibility;
// this adapter only binds supported task IDs to mechanical runners.
func TaskFuncFor(run *RunReport) flow.TaskFunc {
	return func(v cue.Value) (flow.Runner, error) {
		idValue := v.LookupPath(cue.MakePath(cue.Str("$id")))
		id, err := idValue.String()
		if err != nil {
			if v.LookupPath(cue.MakePath(cue.Str("$id"))).Exists() {
				return nil, fmt.Errorf("task $id must be a concrete string at %s: %w", v.Path(), err)
			}
			return nil, nil
		}

		runner, ok := runnerForID(id, run)
		if !ok {
			return nil, fmt.Errorf("unsupported task $id %q at %s", id, v.Path())
		}
		return runner, nil
	}
}

func runnerForID(id string, run *RunReport) (flow.Runner, bool) {
	switch id {
	case "discover_root":
		return flow.RunnerFunc(func(t *flow.Task) error {
			return runTask(t, run, discoverRoot)
		}), true
	case "scan_surfaces":
		return flow.RunnerFunc(func(t *flow.Task) error {
			return runTask(t, run, scanSurfaces)
		}), true
	case "classify_surfaces":
		return flow.RunnerFunc(func(t *flow.Task) error {
			return runTask(t, run, classifySurfaces)
		}), true
	case "assess_ssot":
		return flow.RunnerFunc(func(t *flow.Task) error {
			return runTask(t, run, assessSSOT)
		}), true
	case "emit_report":
		return flow.RunnerFunc(func(t *flow.Task) error {
			return runTask(t, run, emitReport)
		}), true
	default:
		return nil, false
	}
}
