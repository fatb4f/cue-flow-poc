package flowpoc

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/tools/flow"
)

type FillProposal struct {
	Output map[string]any
}

type StubAgent struct{}

func (StubAgent) Propose(v cue.Value) (FillProposal, error) {
	message, err := v.LookupPath(cue.ParsePath("input.message")).String()
	if err != nil {
		return FillProposal{}, fmt.Errorf("read input.message for %s: %w", v.Path(), err)
	}
	return FillProposal{
		Output: map[string]any{
			"message": message,
			"ok":      true,
		},
	}, nil
}

// EchoRunner is the mechanical flow.Runner boundary. The stub agent proposes
// the semantic result; the runner validates the proposal and calls Task.Fill.
type EchoRunner struct {
	Agent  StubAgent
	Report *Report
}

func (r EchoRunner) Run(t *flow.Task, depErr error) error {
	if depErr != nil {
		return depErr
	}

	proposal, err := r.Agent.Propose(t.Value())
	if err != nil {
		return err
	}
	if r.Report != nil {
		r.Report.Agent.ProposedFill = true
	}

	if err := validateProposal(proposal); err != nil {
		return err
	}
	if r.Report != nil {
		r.Report.Runner.ValidatedFill = true
	}

	if err := t.Fill(map[string]any{"output": proposal.Output}); err != nil {
		return err
	}
	if r.Report != nil {
		r.Report.Runner.CalledTaskFill = true
	}
	return nil
}

func validateProposal(proposal FillProposal) error {
	message, ok := proposal.Output["message"].(string)
	if !ok || message == "" {
		return fmt.Errorf("agent proposal must include output.message")
	}
	okValue, ok := proposal.Output["ok"].(bool)
	if !ok || !okValue {
		return fmt.Errorf("agent proposal must include output.ok=true")
	}
	return nil
}
