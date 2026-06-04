package fixtures

import flow "cue-flow-poc.local/cue/flow"

_tf: {
	id:      "taskfunc.basic"
	adapter: "local-go"

	classifiesCueValue: true
	createsRunner:      true
	ownsPolicy:         false
}

_echoRunner: {
	id:      "runner.echo"
	adapter: "local-go"

	executesTask:  true
	mayFill:       true
	validatesFill: true
	callsTaskFill: true
	ownsPolicy:    false
}

_agent: {
	kind:                "agent-runner"
	agentExecutesTask:   true
	agentMayProposeFill: true
	agentMayCallRawFill: false
	agentOwnsPolicy:     false
}

badAmbiguity: flow.#FlowRunContract & {
	config: {
		root:            "flow"
		inferTasks:      false
		ignoreConcrete:  false
		findHiddenTasks: false
	}

	taskFunc: _tf

	tasks: first: {
		id:   "first"
		kind: "run_step"
		path: "flow.first"
		input: {message: "hello"}
		output: {message: "hello"}
		taskFunc: _tf
		runner:   _echoRunner
		agent:    _agent
		state:    "Terminated"
		referenceDependencies: []
	}

	referenceGraph: {
		cyclic: false
		edges: []
	}

	steps: [{
		id:   "step.first"
		task: tasks.first
		fillGate: {
			taskPath:   "flow.first"
			proposedBy: "agent"
			appliedBy:  "go-flow-runner"
			payload: {output: tasks.first.output}
			outputAccepted:    true
			authorityAccepted: true
			ambiguity: [{
				kind:     "runner_unbound"
				path:     "flow.first"
				reason:   "intentional bad fixture: ambiguity cannot coexist with clear=true"
				severity: "blocker"
			}]
			accepted: true
		}
		ambiguity: [{
			kind:     "runner_unbound"
			path:     "flow.first"
			reason:   "intentional bad fixture: ambiguity cannot coexist with clear=true"
			severity: "blocker"
		}]
		flowTerminated:    true
		outputAccepted:    true
		authorityAccepted: true
		clear:             true
	}]
}
