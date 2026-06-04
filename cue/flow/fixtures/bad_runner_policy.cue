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

_badRunner: {
	id:      "runner.bad-policy"
	adapter: "local-go"

	executesTask:  true
	mayFill:       true
	validatesFill: true
	callsTaskFill: true
	ownsPolicy:    true
}

badRunnerPolicy: flow.#FlowRunContract & {
	config: {
		root: "flow"
	}

	taskFunc: _tf

	tasks: first: {
		id:   "first"
		kind: "run_step"
		path: "flow.first"
		input: {message: "hello"}
		output: {message: "hello"}
		taskFunc: _tf
		runner:   _badRunner
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
			ambiguity: []
			accepted: true
		}
		ambiguity: []
		flowTerminated:    true
		outputAccepted:    true
		authorityAccepted: true
		clear:             true
	}]
}
