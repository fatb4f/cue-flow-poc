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

_edgeFirstToSecond: {
	fromTaskPath:  "flow.second"
	fromValuePath: "flow.second.input.message"
	toTaskPath:    "flow.first"
	derivedBy:     "cue-reference-analysis"
}

goodTwoStepReferenceChain: flow.#FlowRunContract & {
	config: {
		root:            "flow"
		inferTasks:      false
		ignoreConcrete:  false
		findHiddenTasks: false
	}

	taskFunc: _tf

	tasks: {
		first: {
			id:   "first"
			kind: "run_step"
			path: "flow.first"

			input: {message: "hello"}
			output: {
				message: "hello"
				ok:      true
			}

			taskFunc: _tf
			runner:   _echoRunner
			agent:    _agent
			state:    "Terminated"

			referenceDependencies: []
		}

		second: {
			id:   "second"
			kind: "run_step"
			path: "flow.second"

			input: {message: tasks.first.output.message}
			output: {
				message: tasks.first.output.message
				ok:      true
			}

			taskFunc: _tf
			runner:   _echoRunner
			agent:    _agent
			state:    "Terminated"

			referenceDependencies: [_edgeFirstToSecond]

			// Projection/cache only; real authority remains cue-reference-analysis.
			dependsOnProjection: ["first"]
			dependsOnProjectionAuthority: "projection-only"
		}
	}

	referenceGraph: {
		cyclic: false
		edges: [_edgeFirstToSecond]
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
	}, {
		id:   "step.second"
		task: tasks.second
		fillGate: {
			taskPath:   "flow.second"
			proposedBy: "agent"
			appliedBy:  "go-flow-runner"
			payload: {output: tasks.second.output}
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
