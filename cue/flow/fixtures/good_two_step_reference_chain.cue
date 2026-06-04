package fixtures

import flow "cue-flow-poc/cue/flow"

_tf: {
	id:      "taskfunc.basic"
	adapter: "local-go"

	classifiesCueValue: true
	createsRunner:     true
	ownsPolicy:        false
}

_echoRunner: {
	id:      "runner.echo"
	adapter: "local-go"

	executesTask: true
	mayFill:      true
	ownsPolicy:   false
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
		ignoreConcrete: false
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
			state:    "Terminated"

			referenceDependencies: [_edgeFirstToSecond]

			// Projection/cache only; real authority remains cue-reference-analysis.
			dependsOnProjection:          ["first"]
			dependsOnProjectionAuthority: "projection-only"
		}
	}

	referenceGraph: {
		cyclic: false
		edges:  [_edgeFirstToSecond]
	}

	steps: [{
		id:   "step.first"
		task: tasks.first
		fill: {
			taskPath: "flow.first"
			payload: {output: tasks.first.output}
			accepted: true
		}
		ambiguity: []
		flowTerminated: true
		outputAccepted: true
		authorityAccepted: true
		clear: true
	}, {
		id:   "step.second"
		task: tasks.second
		fill: {
			taskPath: "flow.second"
			payload: {output: tasks.second.output}
			accepted: true
		}
		ambiguity: []
		flowTerminated: true
		outputAccepted: true
		authorityAccepted: true
		clear: true
	}]
}
