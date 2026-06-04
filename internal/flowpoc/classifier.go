package flowpoc

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/tools/flow"
)

// TaskFunc is the only place where CUE values are classified as executable
// tasks. It does not own policy. The CUE contract owns admissible task shape.
func TaskFunc(v cue.Value) (flow.Runner, error) {
	kindValue := v.LookupPath(cue.ParsePath("kind"))
	if !kindValue.Exists() {
		return nil, nil
	}

	kind, err := kindValue.String()
	if err != nil {
		return nil, fmt.Errorf("task kind must be a concrete string at %s: %w", v.Path(), err)
	}

	switch kind {
	case "echo":
		return EchoRunner{}, nil
	default:
		return nil, fmt.Errorf("unsupported task kind %q at %s", kind, v.Path())
	}
}
