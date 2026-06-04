package flowpoc

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/tools/flow"
)

// EchoRunner is intentionally boring: it reads input.message and Fills output.
// It does not decide whether its output is authoritative or clear.
type EchoRunner struct{}

func (EchoRunner) Run(t *flow.Task, depErr error) error {
	if depErr != nil {
		return depErr
	}

	v := t.Value()
	message, err := v.LookupPath(cue.ParsePath("input.message")).String()
	if err != nil {
		return fmt.Errorf("read input.message for %s: %w", t.Path(), err)
	}

	return t.Fill(map[string]any{
		"output": map[string]any{
			"message": message,
			"ok":      true,
		},
	})
}
