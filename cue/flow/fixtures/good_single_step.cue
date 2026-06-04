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


goodSingleStep: flow.#FlowRunContract & {
	config: {
		root:            "flow"
		inferTasks:      false
		ignoreConcrete: false
		findHiddenTasks: false
	}

	taskFunc: _tf

	tasks: first: {
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

		state: "Terminated"

		referenceDependencies: []
	}

	referenceGraph: {
		cyclic: false
		edges:  []
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
	}]
}
